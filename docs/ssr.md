# Server-Side Rendering (SSR)

Server-side rendering (SSR) allows you to render your Inertia.js pages on the server before sending them to the browser.
This improves search engine optimization (SEO) and initial page load performance.

## Prerequisites

To enable SSR, you must first configure your frontend (Vite, React/Vue/Svelte) to support server-side rendering.

Please follow the [official Inertia.js SSR documentation](https://inertiajs.com/docs/v2/advanced/server-side-rendering)
for detailed
instructions on:

- Installing frontend dependencies.
- Creating the `ssr.js` entry point.
- Configuring Vite for SSR builds.

## Go Configuration

Once your Node.js SSR server is ready, configure `goinertia` to communicate with it.

### 1. Initialize with SSR

Use `WithSSRConfig` to provide the SSR server URL and other optional settings.

```go
inertiaAdapter := goinertia.Must(goinertia.NewWithValidation("http://localhost:3000",
// ... other options
goinertia.WithSSRConfig(goinertia.SSRConfig{
// The URL of your Node.js SSR server (default port is 13714)
URL: "http://127.0.0.1:13714/render",
// Optional: request timeout
Timeout: 3 * time.Second,
// Optional: SSR cache settings
CacheTTL: 5 * time.Minute,
CacheMaxEntries: 1024,
}),
))
```

### Configuration Options (`SSRConfig`)

| Field             | Type                | Description                                                                                   |
|-------------------|---------------------|-----------------------------------------------------------------------------------------------|
| `URL`             | `string`            | The full URL to your SSR server's render endpoint (e.g., `http://127.0.0.1:13714/render`).    |
| `Timeout`         | `time.Duration`     | Maximum time to wait for the SSR server to respond. Default is 3 seconds when using defaults. |
| `Headers`         | `map[string]string` | Custom HTTP headers to send with the SSR request (useful for authentication or tracing).      |
| `CacheTTL`        | `time.Duration`     | Time-to-live for cached SSR results. Set to `0` to disable caching.                           |
| `CacheMaxEntries` | `int`               | Maximum number of SSR results to keep in the in-memory cache. Default is 256 when not set.    |
| `MaxRetries`      | `int`               | Maximum number of retries for SSR requests. Default is 1.                                     |
| `RetryDelay`      | `time.Duration`     | Delay between retries. Default is 10ms.                                                       |
| `RetryStatuses`   | `[]int`             | Optional list of HTTP statuses to retry. If empty, retries on 5xx.                            |
| `DisableRetries`  | `bool`              | Disable SSR retries even when defaults are applied.                                           |
| `SSRClient`       | `SSRClient`         | A custom implementation of the SSR HTTP client (must satisfy the `SSRClient` interface).      |

### 2. Update Root Template (`app.gohtml`)

The adapter provides a `.processSSR` variable in your template data. You must use it to render the head tags and the
pre-rendered body.

```html
<!DOCTYPE html>
<html>
<head>
    {{/* 1. Render meta tags and head content from SSR */}}
    {{ if .processSSR }}
    {{ range .processSSR.Head }}
    {{ raw . }}
    {{ end }}
    {{ end }}

    {{/* ... your scripts and styles ... */}}
</head>
<body>
{{/* 2. Render pre-rendered HTML body if available.
Note: The SSR body includes the root element (e.g.,
<div id="app">).
    If SSR fails or is disabled, fallback to regular CSR container. */}}
    {{ if .processSSR }}
    {{ raw .processSSR.Body }}
    {{ else }}
    <div id="app" data-page="{{ marshal .page }}"></div>
    {{ end }}
</body>
</html>
```

## How it Works

When SSR is enabled:

1. Before rendering the HTML template, `goinertia` makes a POST request to your Node.js SSR server with the page data (
   JSON).
2. The SSR server returns an object containing the `body` HTML and an array of `head` strings.
3. `goinertia` injects these into the `.processSSR` template variable.
4. If the SSR server is unreachable or returns an error, `goinertia` will return a 500 error (in production) to ensure
   consistency.

> **Tip:** In development, you can use `WithDevMode()` to enable hot-reloading features, but remember that the Node.js
> SSR server must be built and running for SSR to work.

## Retry Behavior

By default, SSR requests retry once on 5xx responses or network errors. To disable retries:

```go
goinertia.WithSSRConfig(goinertia.SSRConfig{
    URL:            "http://127.0.0.1:13714/render",
    DisableRetries: true,
})
```

Customize retry behavior:

```go
goinertia.WithSSRConfig(goinertia.SSRConfig{
    URL:           "http://127.0.0.1:13714/render",
    MaxRetries:    2,
    RetryDelay:    50 * time.Millisecond,
    RetryStatuses: []int{http.StatusTooManyRequests, http.StatusServiceUnavailable},
})
```

## Custom SSR Client

If you need a custom HTTP client (tracing, auth, custom transport), implement `SSRClient`:

```go
type mySSRClient struct{}

func (c *mySSRClient) Reset() {}

func (c *mySSRClient) Post(ctx context.Context, url string, body []byte, headers map[string]string) (int, []byte, error) {
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return 0, nil, err
    }
    for k, v := range headers {
        req.Header.Set(k, v)
    }
    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return 0, nil, err
    }
    defer resp.Body.Close()
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return 0, nil, err
    }
    return resp.StatusCode, respBody, nil
}

goinertia.WithSSRConfig(goinertia.SSRConfig{
    URL:       "http://127.0.0.1:13714/render",
    SSRClient: &mySSRClient{},
})
```

The default SSR transport is native net/http and returns owned body bytes.
Custom SSRClient implementations must support concurrent Post calls. Retries
do not call Reset. Cancellation, transport, status and decoding errors remain
errors; there is no automatic CSR fallback. Configure SSR before serving.

## Opt-in Inertia v3 rendering

Select `WithProtocolVersion(ProtocolV3)` on the root/Fiber adapter, or
`nethttp.WithCoreOptions(core.WithProtocolVersion(core.ProtocolV3))` on native
HTTP. The default remains v2. `SSRConfig` and `SsrDTO` keep their existing field
layouts.

Use the matching v3 client renderer. Its `/render` response contains the whole
bootstrap script and app mount point, including `data-server-rendered="true"`;
its head tags use `data-inertia`. Do not add a second script or mount point around
this body. Custom root templates can use the protocol-aware values:

```html
<head>{{ .inertiaHead }}</head>
<body>{{ .inertiaBody }}</body>
```

On CSR responses, `.inertiaBody` generates the v3
`<script data-page="app" type="application/json">` and an empty app div. The JSON
escapes slashes, HTML-sensitive characters, and Unicode line separators, so
values containing `</script>` or `<!--` remain data. On SSR responses,
`.inertiaHead` and `.inertiaBody` contain trusted renderer output. These values
also support v2 custom templates; the embedded v2 template retains its original
attribute-based bootstrap. `.pageJSON` and `.processSSR` remain available.

### Failures and observability

V3 reports failed SSR attempts after configured retries and falls back to CSR.
A network error, non-success status, invalid JSON, or missing rendered body
cannot masquerade as successful SSR. Configure either a logger or
`WithSSRFailureHandler`; if no logger is configured, v3 uses Go's default
`slog` logger. The callback runs once per failed render, after retries, and must
be concurrency-safe. Diagnostics stay on the server:

```go
goinertia.WithSSRFailureHandler(func(ctx context.Context, err error) {
    // Report to your error tracker. errors.As(err, &detail) can recover
    // *core.SSRResponseError: status, message, type, hint, browser API,
    // stack, and source location from a v3 renderer's error response.
})
```

Use `WithSSRErrorPolicy(SSRErrorPropagate)` to return the failure instead, such as
in end-to-end tests. `SSRErrorFallback` opts v2 into reported CSR fallback;
`SSRErrorDefault` preserves v2's historical propagation. Failure callbacks also
run when propagation is selected. Cancellation of the original request is
always returned rather than rendering a fallback for an abandoned request.
An SSR-specific timeout can still fall back while the original request is live.

### Disable SSR for one request

Use `manager.WithSSRDisabled(c, true)` in Fiber, or
`manager.WithSSRDisabled(nethttp.State(r), true)` in native HTTP. This changes
only that request, so another request can still render through SSR. It does not
reset a shared client or cache. Use `false` to re-enable SSR for that request.
The existing `EnableSSR` / `DisableSSR` methods configure startup state and must
not be toggled while serving concurrent requests.

### Vite development SSR

With v3, `WithDevMode()` and an enabled `SSRConfig`, a nonempty Vite hot file
selects `POST <hot-url>/__inertia_ssr`. The host must run a v3 Vite integration
that provides that route and exports its SSR render callback. Without a hot
file, the configured production URL (normally `/render`) is used. The hot URL
is reread per request; development rendering bypasses the production SSR cache.
Timeouts, custom headers, retries and error reporting apply to both routes.
`WithViteSSR(false)` keeps using the configured SSR URL even while Vite is hot;
`WithViteSSR(true)` also makes this routing available in v2 when the host supplies
a compatible endpoint. The library does not launch Vite or build frontend assets.

The transport tests exercise both endpoint contracts against a real loopback
HTTP server, custom injected clients, fallback, cancellation, and request-local
SSR suppression. They do not substitute for browser hydration tests.

References: [v3.8.0 SSR bootstrap](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/ssrUtils.ts),
[v3.8.0 SSR server](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/server.ts),
and the [pinned official HTTP gateway](https://github.com/inertiajs/inertia-laravel/blob/4da52b72da39396cad1d3ec8523b6c649009456e/src/Ssr/HttpGateway.php).

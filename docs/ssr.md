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

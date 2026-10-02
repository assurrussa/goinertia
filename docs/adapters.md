# Framework-neutral core and native adapters

The module still uses `github.com/assurrussa/goinertia`, with Go 1.26 as its
baseline. Existing root imports retain Fiber signatures and legacy callback
behavior. The module still requires Fiber; importing `core` or
`adapters/nethttp` does not put Fiber in the compiled dependency graph.

Dependencies flow from the root compatibility facade to `adapters/fiber` and
then `core`. Native adapters depend on core; core does not import either adapter.
There is no Fiber/HTTP bridge or universal framework context.

The chosen direction retains both native lifecycles. Protocol coverage is a
separate contract: consult the [v2 subset and v3 roadmap](protocol-coverage.md)
before choosing a client version or advanced prop feature.

## Native net/http

```go
import (
    "net/http"
    "github.com/assurrussa/goinertia/core"
    "github.com/assurrussa/goinertia/adapters/nethttp"
    "github.com/assurrussa/goinertia/views"
)

manager, err := nethttp.NewWithValidation("https://app.example",
    nethttp.WithCoreOptions(core.WithFS(views.Templates),
        core.WithAssetVersion("v1")),
)
if err != nil { panic(err) }

mux := http.NewServeMux()
mux.Handle("/", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
    manager.WithProp(nethttp.State(r), "message", "Hello")
    return manager.Render(w, r, "Home", nil)
}))
handler := manager.Middleware(mux)
```

`Middleware` installs a concrete `core.State`. Props, view data, lazy results
and page metadata belong to that request. `RequestMeta` contains fixed protocol
fields, without creating a complete request header map. State may be mutated
by the current handler; it is not safe for concurrent mutation or reuse across
requests. State/Render/redirect helpers bind the downstream request context,
including values and cancellation added with r.WithContext(r.Context()-derived
context). Native Request(ctx) returns that active request. Transparent writer
wrappers must expose Unwrap so error handling can observe commitment.

Fiber prop/flash helpers defer full metadata capture until rendering or explicit
State access. Reading props does not create an empty props map. A State already
requested by a native consumer is reused by helpers and synchronized to Locals.

Headers, 302/301 to 303 normalization, and redirect flash writes happen before
the first final `WriteHeader`. Flash hooks can set cookies at that point.
Arbitrary response bodies stream directly; only library-owned rendered pages
are materialized before commitment, so serialization/template/SSR errors can
be returned before a page is sent. Informational responses do not commit the
final response. Errors after commitment are logged without rewriting the body.

The writer preserves Flusher, Hijacker, Pusher and legacy CloseNotifier when
present, forwards ReaderFrom/StringWriter, and supports ResponseController
through `Unwrap`, including underlying deadlines and full duplex. A hijacked
connection belongs to its caller. Helpers must set flash before committing a
redirect; changing status or cookies afterward cannot work with net/http.

HTTP `SessionStore` owns its storage, cookie policy, authentication and rotation.
Its `Flash` and `GetFlash` methods receive the native writer and request. The
adapter supplies no credentials, session backend or CSRF token algorithm.
Token injection and write checking are independent options: a checker works
without a token provider. `WithCSRFPropName` selects the reserved prop name.
Hosts must validate untrusted redirect destinations before calling helpers;
external and scheme-relative URLs are intentionally supported.

## Context compatibility

Root `goinertia.New` and `NewWithValidation` preserve the historical behavior:
`LazyProp.Fn(context.Context)` receives `fiber.Ctx`. This includes lazy values
inside deferred/optional/once wrappers and shared props. Existing GoAdmin
breadcrumbs and CSRF callbacks therefore continue to work.

`adapters/fiber.New` and `NewWithValidation` use lifecycle context. A callback
can get the active Fiber helper with `fiberadapter.FromContext(ctx)`.
HTTP callbacks use the native request context; `nethttp.Request(ctx)` returns
the active request. These helpers are valid only while the request is active.
Fiber contexts are pooled and must not be retained. Hosts set Fiber's lifecycle
context with `c.SetContext` when cancellation is needed; SSR always uses that
lifecycle context, including through the legacy facade.
Repeated `State(c)` access and rendering refresh the state's lifecycle context
from `c.Context()`, so downstream `c.SetContext` values, deadlines and
cancellation take effect even when earlier middleware already obtained state.
The legacy facade still passes the actual `fiber.Ctx` to lazy callbacks.
Fiber convenience helpers mutate an already installed native `State`, preserving
props, view data and page metadata added through the embedded core API. Flash,
old input and validation errors added through that API also persist on redirects
and Inertia location conflicts, with the same consume-once session policy.

## Fiber session ownership

`NewFiberSessionAdapter` recognizes Fiber's raw `*session.Store` and releases
each acquired session after the operation, including save failures. A custom
raw store can transfer ownership explicitly with
`NewFiberSessionAdapterWithRelease(store, release)`.

Custom `SessionStore` implementations retain their own ownership. In particular,
GoAdmin 0.7's SessionBridge follows the response cookie after authentication
rotation and calls Release itself; it is passed directly to WithSessionStore.

When using `app.Use(session.New(...))`, use `MiddlewareSessionAdapter{}`. It
borrows the middleware-owned session. Fiber middleware saves and releases it
when the downstream handler finishes. Do not release middleware-owned sessions
or wrap their Store as a raw session source.

## SSR and configuration

SSR policy, cache and templates are shared in core. The default SSR transport
uses `http.NewRequestWithContext`, reads an owned response body before closing
it, and leaves the shared transport intact during retries. `SSRClient` remains
injectable. Custom clients must permit concurrent Post calls and return an
owned body. Reset is a configuration/shutdown hook, not a retry hook.

Timeouts and cancellation return errors. Transport failure, invalid SSR JSON
and error status retain the error policy; there is no automatic CSR fallback.
Cache hits return copies. Template publication uses sync.Once without an
unsynchronized pointer read before it. Dev mode reparses templates.

Configure engines/options and SSR enable/disable before serving requests.
Configuration mutation during serving is not supported; request state and the
SSR cache have their own lifetimes. A future multi-module split would change
module dependencies and imports and is outside this compatible stage.

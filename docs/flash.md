# Flash messages

## Compatible `props.flash`

The existing `WithFlash`, `WithFlashSuccess`, `WithFlashInfo`,
`WithFlashWarning`, `WithFlashError`, and `WithFlashMessages` helpers retain
exactly their ordinary `page.props.flash` behavior. They store string messages
by level. `WithFlashOld` and validation helpers likewise keep `props.old` and
`props.errors`.

```go
func ProcessAction(c fiber.Ctx, inertia *goinertia.Inertia) error {
    if err := save(); err != nil {
        inertia.WithFlashError(c, "Save failed")
        return inertia.RedirectBack(c)
    }
    inertia.WithFlashSuccess(c, "Saved")
    return inertia.RedirectBack(c)
}
```

Ordinary props can remain in browser history. These helpers do not trigger the
native Inertia flash event and are not automatically migrated.

## Opt-in native `page.flash`

`WithNativeFlash` adds a JSON-serializable value under an arbitrary key to the
separate, top-level `page.flash` object. It supports strings, objects, arrays,
booleans and null values. Values are data, not lazy/prop-wrapper callbacks.
Calling it again with the same key replaces that key; other keys are retained.
Both flash APIs may be used in the same response without overwriting each other.
Without native data, the top-level field is omitted, preserving the legacy
response shape.

Root and native Fiber APIs:

```go
inertia.WithNativeFlash(c, "toast", map[string]any{
    "type": "success",
    "message": "Profile saved",
})
inertia.WithFlashSuccess(c, "Compatible legacy message") // optional, independent
return inertia.Redirect(c, "/profile")
```

Native HTTP API:

```go
inertia.WithNativeFlash(nethttp.State(r), "message", "Profile saved")
return inertia.Redirect(w, r, "/profile")
```

Native flash is included independently of partial `only`/`except` prop
selection. Current-request keys override session keys. Default HTML bootstrap,
JSON responses and SSR receive the same native flash and scroll-reset metadata.

The pinned official **v2.3.18 and v2.3.28** clients understand `page.flash`,
provide `onFlash`/`flash` event handling and strip native flash from browser
history. They shallow-merge native flash on same-component partial visits;
automatic deferred requests retain the current flash without applying incoming
flash. This means a visible message can remain through a partial reload; the
server does not resend consumed session data. Full visits and Back/Forward are
subject to the client's own flash/history handling. See the pinned
[v2.3.18 page/history code](https://github.com/inertiajs/inertia/blob/v2.3.18/packages/core/src/page.ts),
[v2.3.28 page/history code](https://github.com/inertiajs/inertia/blob/v2.3.28/packages/core/src/page.ts),
and [v2.3.28 partial response handling](https://github.com/inertiajs/inertia/blob/v2.3.28/packages/core/src/response.ts).
This is a tested v2 subset, not a claim of full v3 compatibility.

## Session lifecycle

- A configured `WithSessionStore` is required to carry either API across a
  redirect. Direct rendering in the current request works without a store.
- The adapters persist native flash together with legacy flash, old input and
  validation errors in the existing session envelope. A native-only redirect
  also persists, even when no ordinary props were set.
- Writes occur only for redirect-like responses or `409 + X-Inertia-Location`.
  Ordinary renders do not put current-request flash back into the session.
- The session bridge's consume-once `GetFlash` runs when rendering a page.
  Redirect hops and early asset-version conflicts do not consume queued data.
  The next rendered page, including a partial response, receives it once.
- Precognition skips both flash consumption and persistence, including when
  the handler invokes flash helpers. Existing queued data remains available.
- Native HTTP session hooks run before response commitment, so cookie headers
  can still be written. Hosts own the session implementation and its errors.

Raw Fiber Store sessions are released by `NewFiberSessionAdapter`. Custom raw
stores can use `NewFiberSessionAdapterWithRelease`. For session middleware, use
`MiddlewareSessionAdapter{}`; its middleware owns save/release. Custom
`SessionStore` bridges keep their own ownership. See [adapters](adapters.md).

## Direct core consumers

`PageDTO` retains its existing field layout, including positional literals.
Native flash is request metadata, so neither `json.Marshal(page)` nor the older
`core.MarshalPage(page, resetHeader)` includes native flash. Serialize a built
page with the request state instead:

```go
engine.WithNativeFlash(state, "message", "Saved")
page, err := engine.BuildPage(state, "Profile", props)
if err != nil {
    return err
}
body, err := core.MarshalPageWithState(page, state)
```

`engine.RenderHTML(state, page)` uses this same wire view for the default
bootstrap and SSR request. Custom templates must serialize `.pageJSON`, not
`.page`, to include request metadata. The older `ProcessSSR(ctx, page)` accepts
only a DTO and cannot infer native flash from it. Core consumers own session
loading/persistence; the adapters handle these steps automatically.

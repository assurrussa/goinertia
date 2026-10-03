# Redirects and 409 Conflict

Inertia.js distinguishes **internal redirects** from **external redirects**:

- **Internal redirects** (same Inertia app) should be standard `302`/`303` responses.
- **External redirects** (force full reload) use **409 Conflict** with `X-Inertia-Location`.

In addition, a **409 Conflict** is used when the asset version changes (version mismatch).

## Session Configuration

To preserve information (like flash messages) across these redirects, you **must** configure a session store.

### Register the Session Store

Pass the adapter during initialization:

```go
store := session.New()
sessionAdapter := goinertia.NewFiberSessionAdapter[*session.Session](store)

inertiaAdapter := goinertia.New("http://localhost:8080",
    goinertia.WithSessionStore(sessionAdapter),
    // ... other options
)
```

## Usage Example: Internal Redirect with Error

When a conflict occurs (e.g., resource not found), set a flash message and redirect.

```go
func (c *Controller) HandleNotFound(ctx fiber.Ctx) error {
    // Set a flash message that will survive the redirect
    c.inertia.WithFlashError(ctx, "The requested resource was not found.")
    
    // inertia.Redirect will return 302/303 for Inertia requests
    return c.inertia.Redirect(ctx, "/")
}
```

## Usage Example: External Redirect

Use 409 when you need a full reload (external domain or leaving the app).

```go
func (c *Controller) Logout(ctx fiber.Ctx) error {
    return c.inertia.RedirectExternal(ctx, "https://example.com")
}
```

## Client-Side (Vue 3)

In your main layout, you can listen for these flash messages from the shared props.

```vue
<!-- Layout.vue -->
<template>
  <div>
    <div v-if="$page.props.flash.error" class="alert alert-error">
      {{ $page.props.flash.error }}
    </div>
    <slot />
  </div>
</template>

<script setup>
import { usePage } from '@inertiajs/vue3'
const page = usePage()
// flash is available via page.props.flash
</script>
```

When the user clicks a link that triggers the `HandleNotFound` logic:
1. The server responds with `302` (or `303` for non‑GET).
2. The Inertia client follows the redirect.
3. The target page is rendered with the flash error prop populated from the session.

## Opt-in v3 transport

With `WithProtocolVersion(ProtocolV3)`, asset-version conflicts additionally
return the current `X-Inertia-Version`. This lets the v3 client identify an asset
change on a background request. V2 retains its previous headers.

Inertia redirects whose target contains `#` become an empty `409` response with
`X-Inertia-Redirect: <target>`, so the client can issue a GET visit without losing
the fragment. Both native adapters apply this to their redirect helpers and to
ordinary HTTP redirect responses passing through middleware. Prefetch requests
(`Purpose`, `Sec-Purpose`, or `X-Moz` containing `prefetch`) retain normal redirect
semantics. Explicit external redirects continue to use `X-Inertia-Location`.
Flash data is persisted for both forms of 409 navigation.

Use `manager.WithPreserveFragment(c, true)` in Fiber or
`manager.WithPreserveFragment(nethttp.State(r), true)` in native HTTP to emit the
v3 `preserveFragment` page metadata. It preserves the incoming visit's fragment
when the response URL changes. No fragment is available in ordinary incoming
HTTP URLs; this is a client-side instruction. The option is request-local, has
no effect in v2, and leaves `PageDTO`'s field layout unchanged.

### Error statuses

Use `RenderWithStatus(c, status, component, props)` in Fiber, or
`RenderWithStatus(w, r, status, component, props)` in native HTTP for an Inertia
error page. These responses keep `X-Inertia: true`, regular shared props, and
the requested HTTP status, so v3 `httpException` handling can run. The host must
supply the corresponding frontend component.

The v3 default error handlers return a safe non-Inertia JSON response for an
ordinary failed Inertia request, preserving its error status. This includes a
failed non-rescued prop callback. They do not redirect repeatedly to the failed
page. Validation errors retain the existing redirect/session flow. V2 retains
its historical error behavior. Hosts can continue supplying custom error
handling and deciding when to render a component instead.

Reference: [pinned middleware](https://github.com/inertiajs/inertia-laravel/blob/4da52b72da39396cad1d3ec8523b6c649009456e/src/Middleware.php)
and [v3.8.0 response handling](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/response.ts).

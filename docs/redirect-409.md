# Redirects and 409 Conflict

Inertia.js distinguishes **internal redirects** from **external redirects**:

- **Internal redirects** (same Inertia app) should be standard `302`/`303` responses.
- **External redirects** (force full reload) use **409 Conflict** with `X-Inertia-Location`.

In addition, a **409 Conflict** is used when the asset version changes (version mismatch).

## Session Configuration

To preserve information (like flash messages) across these redirects, you **must** configure a session store.

### 1. Define a Session Adapter

First, implement the `SessionAdapter` interface to bridge Fiber's session middleware with `goinertia`.

```go
import (
    "github.com/gofiber/fiber/v3"
    "github.com/gofiber/fiber/v3/middleware/session"
    "github.com/assurrussa/goinertia"
)

// Wrapper to match goinertia.FiberSessionStore interface
type FiberSessionWrapper struct {
    *session.Session
}

func (w *FiberSessionWrapper) Set(key, val any) {
    if k, ok := key.(string); ok { w.Session.Set(k, val) }
}

func (w *FiberSessionWrapper) Get(key any) any {
    if k, ok := key.(string); ok { return w.Session.Get(k) }
    return nil
}

func (w *FiberSessionWrapper) Delete(key any) {
    if k, ok := key.(string); ok { w.Session.Delete(k) }
}

// Adapter to provide sessions to goinertia
type SessionProvider struct {
    store *session.Store
}

func (s *SessionProvider) Get(c fiber.Ctx) (*FiberSessionWrapper, error) {
    sess, err := s.store.Get(c)
    if err != nil { return nil, err }
    return &FiberSessionWrapper{Session: sess}, nil
}
```

### 2. Register the Session Store

Pass the adapter during initialization:

```go
store := session.New()
sessionProvider := &SessionProvider{store: store}
sessionAdapter := goinertia.NewFiberSessionAdapter(sessionProvider)

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

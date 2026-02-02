# Redirects and 409 Conflict

Inertia.js uses a special mechanism for redirects during non-GET requests (like POST, PUT, DELETE) or when a state conflict occurs. One common case is when a requested page is not found or a validation error occurs during a background request.

The server returns an HTTP **409 Conflict** status with an `X-Inertia-Location` header. The Inertia client-side library automatically intercepts this and performs a full page visit to the provided location.

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

## Usage Example: Redirecting with Error

When a conflict occurs (e.g., resource not found), set a flash message and redirect.

```go
func (c *Controller) HandleNotFound(ctx fiber.Ctx) error {
    // Set a flash message that will survive the redirect
    c.inertia.WithFlashError(ctx, "The requested resource was not found.")
    
    // inertia.Redirect will return 409 Conflict for Inertia requests
    return c.inertia.Redirect(ctx, "/")
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
1. The server responds with `409 Conflict`.
2. The Inertia client sees the 409 and fetches `/` (Home).
3. The Home page is rendered with the flash error prop populated from the session.

# goinertia

[![Go Reference](https://pkg.go.dev/badge/github.com/assurrussa/goinertia.svg)](https://pkg.go.dev/github.com/assurrussa/goinertia)
[![Go Report Card](https://goreportcard.com/badge/github.com/assurrussa/goinertia)](https://goreportcard.com/report/github.com/assurrussa/goinertia)
[![Go](https://github.com/assurrussa/goinertia/actions/workflows/go.yml/badge.svg)](https://github.com/assurrussa/goinertia/actions/workflows/go.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**The Fiber-first adapter for Inertia.js.**

`goinertia` allows you to build modern single-page apps using [Vue.js](https://vuejs.org/), [React](https://reactjs.org/), or [Svelte](https://svelte.dev/) while keeping routing and controllers in your [Go (Fiber)](https://gofiber.io/) backend. It strictly adheres to the [Inertia.js protocol](https://inertiajs.com/the-protocol).

Visit:
 - [inertiajs.com](https://inertiajs.com/) to learn more.
 - [fiber](https://gofiber.io/) to learn more.

## Features

- **⚡️ Fiber Native**: Built specifically for the Fiber web framework (v3).
- **🔄 Full Protocol Support**: Implements the complete Inertia.js spec.
  - **Asset Versioning**: Auto-reloads assets when versions change.
  - **Partial Reloads**: Only fetch the data you need.
  - **Lazy Evaluation**: Compute expensive props only when requested.
  - **Shared Data**: Global props (like "auth.user") available to all pages.
- **🛡️ Validation & Flash**: Built-in helpers for form validation errors and flash messages.
- **🚀 Server-Side Rendering (SSR)**: Native support for rendering initial HTML on the server.
- **🔒 CSRF Protection**: Hooks for CSRF token injection and verification.
- **🐛 Error Handling**: Customizable error pages and unified error handling middleware.

## Installation

```bash
go get github.com/assurrussa/goinertia
```

## Quick Start

Here is a minimal example showing how to set up the Inertia middleware and render a page.

```go
package main

import (
    "github.com/gofiber/fiber/v3"
    "github.com/assurrussa/goinertia"
)

func main() {
    // 1. Initialize Inertia
    inertiaManager := goinertia.New("http://localhost:3000",
        goinertia.WithAssetVersion("v1"),
        goinertia.WithSharedProps(map[string]any{
            "appName": "My App",
        }),
    )

    // 2. Register Middleware
  app := fiber.New(fiber.Config{ErrorHandler: inertiaManager.MiddlewareErrorListener()})
    // Handles Inertia requests, asset versioning, and validation redirects
    app.Use(inertiaManager.Middleware())

    // 3. Define Routes
    app.Get("/", func(c fiber.Ctx) error {
        // Render a Vue/React component named "Home"
        return inertiaManager.Render(c, "Home", map[string]any{
            "user": "John Doe",
        })
    })

    app.Listen(":3000")
}
```

## Core Concepts

### Rendering Pages
Use the `Render` method to return an Inertia response. If it's an XHR request, it returns JSON; otherwise, it renders the root HTML template.

```go
inertiaManager.Render(ctx, "Dashboard/Index", map[string]any{
    "stats": statsData,
})
```

### Flash Messages & Validation
`goinertia` provides helpers to seamlessly pass data to the client, especially after form submissions.

> **Note**: Requires `WithSessionStore` to be configured.

```go
// In your controller
if err := validate(form); err != nil {
    inertiaManager.WithFlashError(ctx, "Something went wrong.")
    inertiaManager.WithError(ctx, "email", "Email is already taken.")
    return inertiaManager.RedirectBack(ctx)
}

inertiaManager.WithFlashSuccess(ctx, "Profile updated!")
return inertiaManager.Redirect(ctx, "/profile")
```

### Lazy Evaluation
Optimize performance by wrapping expensive data in `WithLazyProp`. These are only executed if the client explicitly requests them via a partial reload.

```go
inertiaManager.WithLazyProp(ctx, "heavyData", func(c context.Context) (any, error) {
    return heavyDatabaseQuery(), nil
})
```

### Server-Side Rendering (SSR)
Enable SSR to improve SEO and initial load performance. `goinertia` communicates with a Node.js process (the Inertia SSR server) to render the page.

```go
inertiaManager := goinertia.New(url,
    goinertia.WithSSRConfig(goinertia.SSRConfig{
        URL:     "http://127.0.0.1:13714",
        Timeout: 3 * time.Second,
    }),
)
```

## Documentation

Comprehensive documentation is available in the `docs/` directory:

- [Basic Setup](docs/basic.md)
- [Flash Messages](docs/flash.md)
- [Validation & Redirects](docs/validation.md)
- [Handling 409 Conflicts](docs/redirect-409.md)
- [Lazy Properties](docs/lazy-props.md)
- [SSR Configuration](docs/ssr.md)

## Examples

Check the `examples/` directory for fully functional sample applications:
- **basic-app**: [A simple app](examples/basic-app/main.go) implementation showing routing, layout, and props.

## License

MIT
# Basic setup

```go
package main

import (
    "github.com/gofiber/fiber/v3"

    "github.com/assurrussa/goinertia"
)

func main() {
    inertiaAdapter := goinertia.New("http://localhost:3000",
        goinertia.WithAssetVersion("v1.0.0"),
        goinertia.WithSharedProps(map[string]any{
            "appName": "My App",
        }),
    )

	app := fiber.New(fiber.Config{ErrorHandler: inertiaAdapter.MiddlewareErrorListener()})

    app.Use(inertiaAdapter.Middleware())

    app.Get("/", func(c fiber.Ctx) error {
        return inertiaAdapter.Render(c, "Home", map[string]any{
            "title": "Hello",
        })
    })

    _ = app.Listen(":3000")
}
```

# Lazy props

```go
func Dashboard(c fiber.Ctx, inertia *goinertia.Inertia) error {
    inertia.WithLazyProp(c, "analytics", func(ctx context.Context) (any, error) {
        return calculateAnalytics(), nil
    })

    inertia.WithLazyProp(c, "reports", func(ctx context.Context) (any, error) {
        return generateReports(), nil
    })

    return inertia.Render(c, "Dashboard", map[string]any{
        "title": "Dashboard",
    })
}
```

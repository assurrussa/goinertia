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

## Deferred props

Deferred props are not included in the initial response. They are loaded later via a partial reload.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "heavy": goinertia.Defer(goinertia.LazyProp{
        Key: "heavy",
        Fn: func(ctx context.Context) (any, error) {
            return loadHeavyData(), nil
        },
    }),
})
```

## Optional props

Optional props are only included when explicitly requested.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "metrics": goinertia.Optional(goinertia.LazyProp{
        Key: "metrics",
        Fn: func(ctx context.Context) (any, error) {
            return loadMetrics(), nil
        },
    }),
})
```

## Once props

Once props are sent once and then skipped on subsequent requests unless explicitly refreshed.

```go
return inertia.Render(c, "Dashboard", map[string]any{
    "plans": goinertia.Once("basic", goinertia.WithOnceKey("plans_v1")),
})
```

## Merge props

Merge props are merged during partial reloads (append or prepend for lists).

```go
return inertia.Render(c, "Users", map[string]any{
    "users":   goinertia.Merge(users),
    "newest":  goinertia.Prepend(newUsers),
    "profile": goinertia.DeepMerge(profile),
})
```

## Scroll props

Scroll props add pagination metadata for infinite scroll.

```go
return inertia.Render(c, "Feed", map[string]any{
    "posts": goinertia.Scroll(posts, goinertia.ScrollPropConfig{
        PageName:    "page",
        NextPage:    3,
        CurrentPage: 2,
    }),
})
```

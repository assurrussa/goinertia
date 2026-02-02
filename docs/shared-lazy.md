# Shared lazy props

```go
inertiaAdapter := goinertia.New("http://localhost:3000",
    goinertia.WithSharedProps(map[string]any{
        "auth": goinertia.LazyProp{
            Key: "auth",
            Fn: func(ctx context.Context) (any, error) {
                return loadAuth(), nil
            },
        },
        "notifications": goinertia.LazyProp{
            Key: "notifications",
            Fn: func(ctx context.Context) (any, error) {
                return loadNotifications(), nil
            },
        },
    }),
)
```

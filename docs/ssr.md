# SSR

```go
inertiaAdapter, err := goinertia.NewWithValidation("http://localhost:3000",
    goinertia.WithSSRConfig(goinertia.SSRConfig{
        URL:             "http://127.0.0.1:13714",
        Timeout:         3 * time.Second,
        Headers:         map[string]string{"X-Trace-Id": "trace-123"},
        CacheTTL:        5 * time.Second,
        CacheMaxEntries: 256,
    }),
)
if err != nil {
    panic(err)
}
```

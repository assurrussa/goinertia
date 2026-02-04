# History Flags

Inertia v2 supports page-level history flags:

- `encryptHistory` — encrypt stored history state on the client.
- `clearHistory` — clear existing history state for the current visit.

## API

Use the helpers to set flags for the current response:

```go
func Dashboard(c fiber.Ctx, inertia *goinertia.Inertia) error {
    inertia.WithEncryptHistory(c)
    inertia.WithClearHistory(c)

    return inertia.Render(c, "Dashboard", map[string]any{
        "title": "Dashboard",
    })
}
```

The flags are emitted as `encryptHistory` and `clearHistory` in the page JSON.

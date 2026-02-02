# Flash messages

```go
func ProcessAction(c fiber.Ctx, inertia *goinertia.Inertia) error {
    action := c.FormValue("action")

    switch action {
    case "delete":
        if err := deleteItem(id); err != nil {
            inertia.WithFlashError(c, "Delete failed")
            return inertia.RedirectBack(c)
        }
        inertia.WithFlashSuccess(c, "Item deleted")
    case "archive":
        inertia.WithFlashInfo(c, "Item archived")
        inertia.WithFlashWarning(c, "Can restore within 30 days")
    }

    return inertia.RedirectBack(c)
}
```

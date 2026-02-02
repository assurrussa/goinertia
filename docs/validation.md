# Validation and redirects

```go
func CreateUser(c fiber.Ctx, inertia *goinertia.Inertia) error {
    type CreateUserRequest struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    var req CreateUserRequest
    if err := c.Bind().Body(&req); err != nil {
        inertia.WithFlashError(c, "Invalid form data")
        return inertia.RedirectBack(c)
    }

    if req.Name == "" {
        inertia.WithError(c, "name", "Name is required")
    }
    if req.Email == "" {
        inertia.WithError(c, "email", "Email is required")
    }

    return inertia.Render(c, "Users/Create", map[string]any{
        "old": req,
    })
}
```

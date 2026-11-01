# Validation and redirects

## JSON API errors (Fiber)

Non-Inertia requests that explicitly prefer `Accept: application/json` receive
errors as JSON with the original 4xx/5xx status, `message`, and `errors`
when validation errors exist. The configured error-detail visibility policy
still applies. These responses are not redirects and do not create flash data.
This includes failures from the CSRF checker before a route handler runs.

Use this Accept header for JSON `fetch` clients; do not add `X-Inertia` to a JSON
API request just to change redirect behavior. Clients must retain their draft
and handle 401/403/419 explicitly without automatically replaying a mutation.
Normal browser requests (including missing Accept or `*/*`), HTML-preferred
requests, Precognition, and Inertia visits keep their existing response policy.
JSON must be explicitly acceptable with a positive quality; an equally preferred
explicit HTML range wins. Redirect helpers remain navigation APIs, not JSON
error responses.

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


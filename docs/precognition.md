# Precognition

Precognition is a validation-only request flow. The client sends special headers, and the
server responds with validation errors without running full side effects.

## Headers

- `Precognition: true` — marks the request as precognition.
- `Precognition-Validate-Only: field1,field2` — optional list of fields to validate.
- `Precognition-Success: true` — returned on success (no errors).

## Responses

- **204 No Content** + `Precognition-Success: true` when there are no validation errors.
- **422 Unprocessable Entity** with JSON body:
  ```json
  {
    "errors": {
      "email": ["Invalid email"]
    }
  }
  ```

## Example

```go
func CreateUser(c fiber.Ctx, inertia *goinertia.Inertia) error {
    type CreateUserRequest struct {
        Name  string `json:"name"`
        Email string `json:"email"`
    }

    var req CreateUserRequest
    if err := c.Bind().Body(&req); err != nil {
        inertia.WithError(c, "form", "Invalid payload")
        return inertia.Render(c, "Users/Create", map[string]any{})
    }

    if req.Name == "" {
        inertia.WithError(c, "name", "Name is required")
    }
    if req.Email == "" {
        inertia.WithError(c, "email", "Email is required")
    }

    return inertia.Render(c, "Users/Create", map[string]any{})
}
```

If the request contains `Precognition` headers, the response will be validation-only.

## Helpers

You can detect Precognition requests early and skip side effects:

```go
if goinertia.IsPrecognition(c) {
    // skip side effects; just collect errors
}
```

By default, responses include `Vary: Precognition` (per protocol). You can disable it with:

```go
goinertia.WithPrecognitionVary(false)
```

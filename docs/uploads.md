# File uploads

```go
func UploadFiles(c fiber.Ctx, inertia *goinertia.Inertia) error {
    config := &goinertia.FileUploadConfig{
        MaxFileSize:       5 * 1024 * 1024,
        AllowedExtensions: []string{".jpg", ".jpeg", ".png", ".pdf"},
        UploadDir:         "uploads",
    }

    result, err := inertia.ProcessFileUploads(c, "files", config)
    if err != nil {
        inertia.WithFlashError(c, "Upload failed")
        return inertia.RedirectBack(c)
    }

    if len(result.Errors) > 0 {
        inertia.WithFileUploadErrors(c, result)
        return inertia.Render(c, "Upload/Form", map[string]any{
            "title": "Upload",
        })
    }

    inertia.WithFileUploadSuccess(c, result.Files)
    return inertia.Redirect(c, "/files")
}
```

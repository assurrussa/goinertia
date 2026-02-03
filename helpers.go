package goinertia

import (
	"context"
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
)

func Redirect(c fiber.Ctx, url string) error {
	if url == "" || url == "/" {
		url = c.BaseURL()
	}

	if c.Get(HeaderInertia) != "" {
		// For Inertia requests, use 409 Conflict with X-Inertia-Location header
		c.Set(HeaderLocation, url)
		c.Set(fiber.HeaderLocation, url)
		return c.SendStatus(fiber.StatusConflict)
	}

	// For regular requests, use standard redirect
	return c.Redirect().Status(fiber.StatusFound).To(url)
}

// Note: This is intentionally marked as safe for JSON data in templates.
func marshal(v any) (template.JS, error) {
	js, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("error marshaling template data: %w", err)
	}

	// #nosec G203 - This is intentionally used for JSON data in templates
	return template.JS(js), nil
}

func asset(path string) (string, error) {
	// If the hot file is missing, fall back to static assets.
	return "/public/dist/" + path, nil
}

// Note: This should only be used with trusted, pre-sanitized content.
func raw(v any) (template.HTML, error) {
	switch val := v.(type) {
	case []string:
		html := strings.Join(val, "\n")
		// #nosec G203 - This is intentionally used for trusted HTML content
		return template.HTML(html), nil
	case string:
		// #nosec G203 - This is intentionally used for trusted HTML content
		return template.HTML(val), nil
	default:
		return "", fmt.Errorf("unsupported type for raw template function: %T", v)
	}
}

func sleeper(c context.Context, sleep time.Duration) error {
	select {
	case <-c.Done():
		return c.Err()
	case <-time.After(sleep):
	}
	return nil
}

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
		// For Inertia requests, use standard redirect (internal visit).
		return c.Redirect().Status(fiber.StatusFound).To(url)
	}

	// For regular requests, use standard redirect
	return c.Redirect().Status(fiber.StatusFound).To(url)
}

// RedirectExternal forces a full page reload for Inertia requests.
func RedirectExternal(c fiber.Ctx, url string) error {
	if url == "" || url == "/" {
		url = c.BaseURL()
	}

	c.Set(HeaderLocation, url)
	c.Set(fiber.HeaderLocation, url)
	return c.SendStatus(fiber.StatusConflict)
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

func appendUnique(list []string, value string) []string {
	for _, item := range list {
		if item == value {
			return list
		}
	}
	return append(list, value)
}

func addVaryHeader(c fiber.Ctx, value string) {
	if value == "" {
		return
	}
	current := string(c.Response().Header.Peek("Vary"))
	if current == "" {
		c.Set("Vary", value)
		return
	}
	for _, item := range strings.Split(current, ",") {
		if strings.TrimSpace(item) == value {
			return
		}
	}
	c.Set("Vary", current+", "+value)
}

func IsPrecognition(c fiber.Ctx) bool {
	return strings.TrimSpace(c.Get(HeaderPrecognition)) != ""
}

func normalizeValidationErrors(value any) ValidationErrors {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case ValidationErrors:
		res := make(ValidationErrors, len(v))
		for field, msgs := range v {
			res[field] = append([]string{}, msgs...)
		}
		return res
	case map[string][]string:
		res := make(ValidationErrors, len(v))
		for field, msgs := range v {
			res[field] = append([]string{}, msgs...)
		}
		return res
	case map[string]string:
		res := make(ValidationErrors, len(v))
		for field, msg := range v {
			res[field] = []string{msg}
		}
		return res
	case map[string]map[string]string:
		res := make(ValidationErrors)
		for _, bag := range v {
			for field, msg := range bag {
				res[field] = []string{msg}
			}
		}
		if len(res) == 0 {
			return nil
		}
		return res
	case map[string]any:
		res := make(ValidationErrors)
		for field, val := range v {
			switch vv := val.(type) {
			case string:
				res[field] = []string{vv}
			case []string:
				res[field] = append([]string{}, vv...)
			case map[string]string:
				for nestedField, msg := range vv {
					res[nestedField] = []string{msg}
				}
			case map[string][]string:
				for nestedField, msgs := range vv {
					res[nestedField] = append([]string{}, msgs...)
				}
			}
		}
		if len(res) == 0 {
			return nil
		}
		return res
	default:
		return nil
	}
}

func flattenValidationErrors(value any) map[string]string {
	errors := normalizeValidationErrors(value)
	if len(errors) == 0 {
		return nil
	}
	flat := make(map[string]string, len(errors))
	for field, msgs := range errors {
		if len(msgs) > 0 {
			flat[field] = msgs[0]
		}
	}
	if len(flat) == 0 {
		return nil
	}
	return flat
}

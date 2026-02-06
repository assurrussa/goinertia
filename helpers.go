package goinertia

import (
	"context"
	"fmt"
	"html/template"
	"net/url"
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
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	current := string(c.Response().Header.Peek("Vary"))
	if current == "" {
		c.Set("Vary", value)
		return
	}
	for _, item := range strings.Split(current, ",") {
		if strings.EqualFold(strings.TrimSpace(item), value) {
			return
		}
	}
	c.Set("Vary", current+", "+value)
}

func IsPrecognition(c fiber.Ctx) bool {
	return strings.TrimSpace(c.Get(HeaderPrecognition)) != ""
}

func parseInertiaBaseURL(baseURL string) *url.URL {
	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil
	}

	return base
}

func buildInertiaLocation(base *url.URL, originalURL string) string {
	if base == nil {
		return originalURL
	}

	if originalURL == "" {
		return base.String()
	}

	ref, err := url.Parse(originalURL)
	if err != nil {
		return base.String()
	}

	// ResolveReference avoids malformed URLs (e.g. double slashes) and preserves query strings.
	return base.ResolveReference(ref).String()
}

func normalizeValidationErrors(value any) ValidationErrors {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case ValidationErrors:
		return cloneValidationErrors(v)
	case map[string][]string:
		return cloneValidationErrorsFromMapStringSlice(v)
	case map[string]string:
		return validationErrorsFromMapStringString(v)
	case map[string]map[string]string:
		return validationErrorsFromBags(v)
	case map[string]any:
		return validationErrorsFromMapAny(v)
	default:
		return nil
	}
}

func cloneValidationErrors(errors ValidationErrors) ValidationErrors {
	res := make(ValidationErrors, len(errors))
	for field, msgs := range errors {
		res[field] = append([]string{}, msgs...)
	}
	return res
}

func cloneValidationErrorsFromMapStringSlice(errors map[string][]string) ValidationErrors {
	res := make(ValidationErrors, len(errors))
	for field, msgs := range errors {
		res[field] = append([]string{}, msgs...)
	}
	return res
}

func validationErrorsFromMapStringString(errors map[string]string) ValidationErrors {
	res := make(ValidationErrors, len(errors))
	for field, msg := range errors {
		res[field] = []string{msg}
	}
	return res
}

func validationErrorsFromBags(bags map[string]map[string]string) ValidationErrors {
	res := make(ValidationErrors)
	for _, bag := range bags {
		for field, msg := range bag {
			res[field] = []string{msg}
		}
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

func validationErrorsFromMapAny(value map[string]any) ValidationErrors {
	res := make(ValidationErrors)
	for field, val := range value {
		mergeValidationErrors(res, field, val)
	}
	if len(res) == 0 {
		return nil
	}
	return res
}

func mergeValidationErrors(dst ValidationErrors, field string, value any) {
	switch v := value.(type) {
	case string:
		dst[field] = []string{v}
	case []string:
		dst[field] = append([]string{}, v...)
	case []any:
		if msgs := anySliceToStrings(v); len(msgs) > 0 {
			dst[field] = msgs
		}
	case map[string]string:
		for nestedField, nestedVal := range v {
			mergeValidationErrors(dst, nestedField, nestedVal)
		}
	case map[string][]string:
		for nestedField, nestedVal := range v {
			mergeValidationErrors(dst, nestedField, nestedVal)
		}
	case map[string]any:
		for nestedField, nestedVal := range v {
			mergeValidationErrors(dst, nestedField, nestedVal)
		}
	case ValidationErrors:
		for nestedField, nestedVal := range v {
			mergeValidationErrors(dst, nestedField, nestedVal)
		}
	}
}

func anySliceToStrings(items []any) []string {
	msgs := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			msgs = append(msgs, s)
		}
	}
	return msgs
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

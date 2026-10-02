package fiberadapter

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia/core"
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

func buildInertiaLocation(base *url.URL, path string) string { return core.Location(base, path) }

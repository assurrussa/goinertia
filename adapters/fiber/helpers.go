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
	current := c.Response().Header.Peek("Vary")
	if core.HasVaryToken(string(current), value) {
		return
	}
	if len(current) == 0 {
		c.Set("Vary", value)
		return
	}
	c.Set("Vary", string(current)+", "+value)
}

func (i *Inertia) applyVary(c fiber.Ctx) {
	if len(c.Response().Header.Peek("Vary")) == 0 {
		value := HeaderInertia
		if i.PrecognitionVary() {
			value = HeaderInertia + ", " + HeaderPrecognition
		}
		c.Set("Vary", value)
		return
	}
	addVaryHeader(c, HeaderInertia)
	if i.PrecognitionVary() {
		addVaryHeader(c, HeaderPrecognition)
	}
}

func IsPrecognition(c fiber.Ctx) bool {
	return strings.TrimSpace(c.Get(HeaderPrecognition)) != ""
}

func buildInertiaLocation(base *url.URL, path string) string { return core.Location(base, path) }

package goinertia_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia/core"
)

func TestRawRedirectFirstAccessClaimsBinding(t *testing.T) {
	outerStore, innerStore := &recordingFiberSession{}, &recordingFiberSession{}
	outer, inner := reviewManager(true, outerStore), reviewManager(false, innerStore)
	app := fiber.New()
	app.Use(outer.Middleware())
	app.Use(inner.Middleware())
	app.Post("/save", func(c fiber.Ctx) error {
		c.Locals(core.ContextKeyProps, map[string]any{
			"flash":  map[string]string{"success": "raw"},
			"errors": map[string]string{"email": "raw"},
			"old":    map[string]any{"email": "raw"},
		})
		return inner.RedirectBack(c)
	})
	requestStateRedirect(t, app)
	// The inner middleware accesses the fresh raw binding first while unwinding.
	// The documented first-access owner rule must exclude the outer manager.
	t.Logf("inner_writes=%d outer_writes=%d outer_saved=%v", innerStore.writes, outerStore.writes, outerStore.saved)
	if innerStore.writes != 1 || outerStore.writes != 0 {
		t.Error("raw redirect input was not claimed by the first accessing manager")
	}
}

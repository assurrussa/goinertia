package goinertia_test

import (
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
)

func TestFiberRawRedirectOwnership(t *testing.T) {
	for _, outerLegacy := range []bool{false, true} {
		for _, innerLegacy := range []bool{false, true} {
			for _, mode := range []string{"raw", "outer-state", "outer-helper", "inner-state", "inner-helper"} {
				name := map[bool]string{false: "native", true: "legacy"}[outerLegacy] + "-" +
					map[bool]string{false: "native", true: "legacy"}[innerLegacy] + "/" + mode
				t.Run(name, func(t *testing.T) {
					outerStore, innerStore := &recordingFiberSession{}, &recordingFiberSession{}
					outer, inner := reviewManager(outerLegacy, outerStore), reviewManager(innerLegacy, innerStore)
					app := fiber.New()
					app.Use(outer.Middleware())
					app.Use(inner.Middleware())
					app.Post("/save", func(c fiber.Ctx) error {
						c.Locals(core.ContextKeyProps, map[string]any{
							"flash":  map[string]string{"success": "raw"},
							"errors": map[string]string{"email": "raw"},
							"old":    map[string]any{"email": "raw"},
						})
						switch mode {
						case "outer-state":
							_ = outer.State(c)
						case "outer-helper":
							outer.WithProp(c, "owner", "outer")
						case "inner-state":
							_ = inner.State(c)
						case "inner-helper":
							inner.WithProp(c, "owner", "inner")
						}
						return inner.RedirectBack(c)
					})
					requestStateRedirect(t, app)
					owner, foreign := innerStore, outerStore
					if mode == "outer-state" || mode == "outer-helper" {
						owner, foreign = outerStore, innerStore
					}
					require.Equal(t, 1, owner.writes)
					require.Zero(t, foreign.writes)
					saved, ok := owner.saved.(map[string]any)
					require.True(t, ok)
					require.Equal(t, map[string]string{"success": "raw"}, saved["flash"])
					require.Equal(t, map[string]string{"email": "raw"}, saved["errors"])
					require.Equal(t, map[string]any{"email": "raw"}, saved["old"])
					require.NotContains(t, saved, "owner", "only redirect flash fields may persist")
				})
			}
		}
	}
}

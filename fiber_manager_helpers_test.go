package goinertia_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func TestFiberManagerHelperIsolation(t *testing.T) {
	t.Parallel()
	for _, firstLegacy := range []bool{false, true} {
		for _, secondLegacy := range []bool{false, true} {
			for _, sameOwner := range []bool{false, true} {
				name := map[bool]string{false: "native", true: "legacy"}[firstLegacy] + "-" +
					map[bool]string{false: "native", true: "legacy"}[secondLegacy] +
					map[bool]string{false: "/different-owner", true: "/same-owner"}[sameOwner]
				t.Run(name, func(t *testing.T) {
					t.Parallel()
					first, second := fiberLifecycleAdapter(firstLegacy), fiberLifecycleAdapter(secondLegacy)
					if sameOwner {
						second = first
					}
					calls := 0
					app := fiber.New()
					app.Get("/page", func(c fiber.Ctx) error {
						first.WithProp(c, "private", "first")
						first.WithViewData(c, "title", "first")
						first.WithEncryptHistory(c)
						first.WithLazyProp(c, "callback", func(ctx context.Context) (any, error) {
							calls++
							_, err := checkManagerCallback(ctx, c, firstLegacy)
							return "first", err
						})
						s := second.State(c)
						if !sameOwner && (s.Props["private"] != nil || s.ViewData["title"] != nil || s.LegacyPageMeta() != nil) {
							return errors.New("helper bindings leaked to another manager")
						}
						second.WithProp(c, "public", "second")
						return second.Render(c, "Page", nil)
					})
					page := readStatePage(t, app, "/page")
					require.Equal(t, "second", page.Props["public"])
					if sameOwner {
						require.Equal(t, 1, calls)
						require.Equal(t, "first", page.Props["callback"])
						require.True(t, page.EncryptHistory)
					} else {
						require.Zero(t, calls)
						require.NotContains(t, page.Props, "private")
						require.NotContains(t, page.Props, "callback")
						require.False(t, page.EncryptHistory)
					}
				})
			}
		}
	}
}

func TestFiberManagerHelperReturnToOwner(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "legacy"}[legacy], func(t *testing.T) {
			t.Parallel()
			first, second := fiberLifecycleAdapter(legacy), fiberLifecycleAdapter(!legacy)
			app := fiber.New()
			app.Get("/page", func(c fiber.Ctx) error {
				first.WithLazyProp(c, "callback", func(ctx context.Context) (any, error) { return checkManagerCallback(ctx, c, legacy) })
				first.WithProp(c, "private", "first")
				second.WithProp(c, "foreign", "second")
				second.WithViewData(c, "title", "second")
				second.WithClearHistory(c)
				first.WithProp(c, "resumed", "first")
				return first.Render(c, "Page", nil)
			})
			page := readStatePage(t, app, "/page")
			require.Equal(t, "first", page.Props["private"])
			require.Equal(t, "first", page.Props["resumed"])
			require.Equal(t, "second", page.Props["callback"])
			require.NotContains(t, page.Props, "foreign")
			require.False(t, page.ClearHistory)
		})
	}
}

func TestFiberManagerRedirectHelperIsolation(t *testing.T) {
	for _, boundary := range []string{"foreign-helper", "foreign-state", "same-owner", "raw-replacement"} {
		t.Run(boundary, func(t *testing.T) {
			store := &recordingFiberSession{}
			first, second := fiberLifecycleAdapter(true), fiberLifecycleAdapter(false)
			if boundary == "same-owner" {
				second = first
			}
			fiberadapter.WithSessionStore(store)(second)
			app := fiber.New()
			app.Use(second.Middleware())
			app.Post("/save", func(c fiber.Ctx) error {
				first.WithFlashSuccess(c, "first")
				first.WithError(c, "name", "first")
				first.WithFlashOld(c, map[string]any{"name": "first"})
				if boundary == "foreign-state" {
					_ = first.State(c)
				}
				if boundary == "raw-replacement" {
					c.Locals(fiberadapter.ContextKeyProps, map[string]any{
						"flash":  map[string]string{"success": "raw"},
						"errors": map[string]string{"name": "raw"},
						"old":    map[string]any{"name": "raw"},
					})
				}
				return second.RedirectBack(c)
			})
			requestStateRedirect(t, app)
			if boundary == "same-owner" || boundary == "raw-replacement" {
				require.Equal(t, 1, store.writes)
				saved, ok := store.saved.(map[string]any)
				require.True(t, ok)
				expected := "first"
				if boundary == "raw-replacement" {
					expected = "raw"
				}
				require.Equal(t, map[string]string{"success": expected}, saved["flash"])
				require.Equal(t, map[string]string{"name": expected}, saved["errors"])
				require.Equal(t, map[string]any{"name": expected}, saved["old"])
			} else {
				require.Zero(t, store.writes, "foreign flash, validation errors and old input must not persist")
			}
		})
	}
}

func TestFiberManagerRawLocalsClaim(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	first, second := fiberLifecycleAdapter(true), fiberLifecycleAdapter(false)
	app.Get("/page", func(c fiber.Ctx) error {
		c.Locals(core.ContextKeyProps, map[string]any{"raw": "first"})
		c.Locals(core.ContextKeyViewData, map[string]any{"title": "first"})
		first.WithEncryptHistory(c)
		s := first.State(c)
		if s.Props["raw"] != "first" || s.ViewData["title"] != "first" {
			return errors.New("raw Locals not adopted by first manager")
		}
		c.Locals(core.ContextKeyProps, map[string]any{"raw": "second"})
		c.Locals(core.ContextKeyViewData, map[string]any{"title": "second"})
		rawMeta := core.NewState(t.Context(), core.RequestMeta{})
		second.Inertia.WithClearHistory(rawMeta)
		c.Locals(core.ContextKeyPageMeta, rawMeta.LegacyPageMeta())
		other := second.State(c)
		if other.Props["raw"] != "second" || other.ViewData["title"] != "second" {
			return errors.New("raw replacement not adopted by second manager")
		}
		// Returning to the first manager must preserve its earlier claimed bindings.
		if first.State(c).Props["raw"] != "first" || s.ViewData["title"] != "first" {
			return errors.New("other manager replaced first ownership")
		}
		return second.Render(c, "Page", nil)
	})
	page := readStatePage(t, app, "/page")
	require.Equal(t, "second", page.Props["raw"])
	require.False(t, page.EncryptHistory)
	require.True(t, page.ClearHistory)
}

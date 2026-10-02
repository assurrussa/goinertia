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

func TestFiberManagerStateIsolation(t *testing.T) {
	t.Parallel()
	for _, firstLegacy := range []bool{false, true} {
		for _, secondLegacy := range []bool{false, true} {
			name := map[bool]string{false: "native", true: "legacy"}[firstLegacy] + "-" +
				map[bool]string{false: "native", true: "legacy"}[secondLegacy]
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				first, second := fiberLifecycleAdapter(firstLegacy), fiberLifecycleAdapter(secondLegacy)
				app := fiber.New()
				app.Get("/page", func(c fiber.Ctx) error {
					s := first.State(c)
					first.Inertia.WithProp(s, "private", "first")
					_, err := first.BuildPage(s, "Page", map[string]any{
						"value": core.LazyProp{Fn: func(context.Context) (any, error) { return "first", nil }},
					})
					if err != nil {
						return err
					}
					if second.State(c) == s {
						return errors.New("managers shared request State")
					}
					return second.Render(c, "Page", map[string]any{
						"value": core.LazyProp{Fn: func(context.Context) (any, error) { return "second", nil }},
						"contract": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
							return checkManagerCallback(ctx, c, secondLegacy)
						}},
					})
				})
				page := readStatePage(t, app, "/page")
				require.Equal(t, "second", page.Props["value"])
				require.Equal(t, "second", page.Props["contract"])
				require.NotContains(t, page.Props, "private")
			})
		}
	}
}

func checkManagerCallback(ctx context.Context, c fiber.Ctx, legacy bool) (any, error) {
	if legacy {
		value, ok := ctx.(fiber.Ctx)
		if !ok || value != c {
			return nil, errors.New("legacy manager lost Fiber callback contract")
		}
	} else {
		value, ok := fiberadapter.FromContext(ctx)
		if !ok || value != c {
			return nil, errors.New("native manager lost lifecycle callback contract")
		}
	}
	return "second", nil
}

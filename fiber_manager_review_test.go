package goinertia_test

import (
	"context"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func reviewManager(legacy bool, store *recordingFiberSession) *fiberadapter.Inertia {
	var options []fiberadapter.Option
	if store != nil {
		options = append(options, fiberadapter.WithSessionStore(store))
	}
	if legacy {
		return goinertia.New("https://app.example", options...)
	}
	return fiberadapter.New("https://app.example", options...)
}

func inspectManagerContext(t *testing.T, ctx context.Context, c fiber.Ctx, legacy bool) {
	t.Helper()
	if legacy {
		value, ok := ctx.(fiber.Ctx)
		if !ok || value != c {
			t.Error("legacy-owned callback invoked with another manager's native context")
		}
	} else {
		value, ok := fiberadapter.FromContext(ctx)
		if !ok || value != c {
			t.Error("native-owned callback invoked without its typed Fiber helper")
		}
	}
}

func TestHelperCreatedManagerIsolation(t *testing.T) {
	for _, pair := range []struct {
		name                            string
		firstLegacy, secondLegacy, same bool
	}{
		{"legacy-native", true, false, false},
		{"native-legacy", false, true, false},
		{"native-native", false, false, false},
		{"legacy-legacy", true, true, false},
		{"same-native", false, false, true},
		{"same-legacy", true, true, true},
	} {
		for _, mode := range []string{"helper-only", "source-state", "receiver-state", "core-published"} {
			t.Run(pair.name+"/"+mode, func(t *testing.T) {
				first, second := reviewManager(pair.firstLegacy, nil), reviewManager(pair.secondLegacy, nil)
				if pair.same {
					second = first
				}
				app := fiber.New()
				calls := 0
				app.Get("/page", func(c fiber.Ctx) error {
					fn := func(ctx context.Context) (any, error) {
						calls++
						inspectManagerContext(t, ctx, c, pair.firstLegacy)
						return "first", nil
					}
					switch mode {
					case "source-state":
						_ = first.State(c)
					case "receiver-state":
						_ = second.State(c)
					}
					if mode == "core-published" {
						s := first.State(c)
						first.Inertia.WithProp(s, "private", "first")
						first.Inertia.WithLazyProp(s, "privateLazy", fn)
						_ = first.State(c)
					} else {
						first.WithProp(c, "private", "first")
						first.WithLazyProp(c, "privateLazy", fn)
					}
					return second.Render(c, "Page", map[string]any{"own": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						inspectManagerContext(t, ctx, c, pair.secondLegacy)
						return "second", nil
					}}})
				})
				page := readStatePage(t, app, "/page")
				t.Logf("source_calls=%d private=%v privateLazy=%v own=%v",
					calls, page.Props["private"], page.Props["privateLazy"], page.Props["own"])
				if pair.same {
					if calls != 1 || page.Props["privateLazy"] != "first" {
						t.Error("same-owner helpers lost their props/callback")
					}
				} else if calls != 0 || page.Props["private"] != nil || page.Props["privateLazy"] != nil {
					t.Error("other manager imported helper-owned props and invoked its callback")
				}
			})
		}
	}
}

func TestExplicitRawLocalsCallbackCompatibility(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "legacy"}[legacy], func(t *testing.T) {
			i := reviewManager(legacy, nil)
			app := fiber.New()
			app.Get("/page", func(c fiber.Ctx) error {
				c.Locals(core.ContextKeyProps, map[string]any{
					"raw": "shared",
					"rawLazy": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						inspectManagerContext(t, ctx, c, legacy)
						return "raw", nil
					}},
				})
				return i.Render(c, "Page", nil)
			})
			page := readStatePage(t, app, "/page")
			if page.Props["raw"] != "shared" || page.Props["rawLazy"] != "raw" {
				t.Error("explicit raw Locals compatibility lost")
			}
		})
	}
}

func TestHelperCreatedManagerRedirectBoundary(t *testing.T) {
	for _, pair := range []struct {
		name                            string
		firstLegacy, secondLegacy, same bool
	}{
		{"legacy-native", true, false, false},
		{"native-legacy", false, true, false},
		{"native-native", false, false, false},
		{"same-native", false, false, true},
		{"same-legacy", true, true, true},
	} {
		t.Run(pair.name, func(t *testing.T) {
			firstStore, secondStore := &recordingFiberSession{}, &recordingFiberSession{}
			first, second := reviewManager(pair.firstLegacy, firstStore), reviewManager(pair.secondLegacy, secondStore)
			if pair.same {
				second = first
				secondStore = firstStore
			}
			app := fiber.New()
			app.Use(second.Middleware())
			app.Post("/save", func(c fiber.Ctx) error {
				first.WithFlashSuccess(c, "first")
				first.WithError(c, "source", "first")
				first.WithFlashOld(c, map[string]any{"source": "first"})
				return second.RedirectBack(c)
			})
			requestStateRedirect(t, app)
			want := 0
			if pair.same {
				want = 1
			}
			t.Logf("first_writes=%d second_writes=%d second_saved=%v", firstStore.writes, secondStore.writes, secondStore.saved)
			if secondStore.writes != want {
				t.Error("other manager persisted helper-owned flash/errors/old into its session")
			}
		})
	}
}

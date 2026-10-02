package goinertia_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func TestMixedFiberStateAndHelpers(t *testing.T) {
	for _, mode := range []string{
		"core-helper", "helper-core", "core-data-helper", "core-metadata-helper",
		"core-view-helper", "repeated-read", "clear-map", "delete-in-place",
	} {
		t.Run(mode, func(t *testing.T) {
			i := fiberadapter.New("https://app.example")
			app := fiber.New()
			app.Get("/page", func(c fiber.Ctx) error {
				applyMixedFiberStateCase(t, mode, i, c)
				return i.Render(c, "Page", nil)
			})
			page := readStatePage(t, app, "/page")
			t.Logf("props=%v clear=%v encrypt=%v", page.Props, page.ClearHistory, page.EncryptHistory)
			checkMixedFiberPage(t, mode, page)
		})
	}
}

type recordingFiberSession struct {
	saved  any
	writes int
}

//nolint:nilnil // A missing session value is valid.
func (*recordingFiberSession) Get(fiber.Ctx, string) (any, error) { return nil, nil }
func (*recordingFiberSession) Set(fiber.Ctx, string, any) error   { return nil }
func (*recordingFiberSession) Delete(fiber.Ctx, string) error     { return nil }
func (s *recordingFiberSession) Flash(_ fiber.Ctx, _ string, value any) error {
	s.saved = value
	s.writes++
	return nil
}

func (s *recordingFiberSession) GetFlash(fiber.Ctx, string) (any, error) {
	value := s.saved
	s.saved = nil
	return value, nil
}

func TestMixedFiberStateRedirectPersistence(t *testing.T) {
	for _, mode := range []string{"core-only", "core-helper", "helper-core", "clear-flash"} {
		t.Run(mode, func(t *testing.T) {
			store := &recordingFiberSession{}
			i := fiberadapter.New("https://app.example", fiberadapter.WithSessionStore(store))
			app := fiber.New()
			app.Use(i.Middleware())
			app.Post("/save", func(c fiber.Ctx) error {
				if mode == "helper-core" || mode == "clear-flash" {
					i.WithProp(c, "helper", "native")
				}
				s := i.State(c)
				i.Inertia.WithValidationErrors(s, core.ValidationErrors{"email": {"invalid"}})
				i.Inertia.WithFlashOld(s, map[string]any{"email": "old"})
				i.Inertia.WithFlashSuccess(s, "saved")
				if mode == "core-helper" {
					i.WithProp(c, "helper", "native")
				}
				if mode == "clear-flash" {
					i.Inertia.WithProp(s, core.ContextPropsFlash, nil)
				}
				return i.RedirectBack(c)
			})
			app.Get("/next", func(c fiber.Ctx) error { return i.Render(c, "Page", nil) })
			r := httptest.NewRequestWithContext(t.Context(), "POST", "https://app.example/save", nil)
			r.Header.Set(core.HeaderInertia, "true")
			r.Header.Set("Referer", "/next")
			resp, err := app.Test(r)
			if err != nil {
				t.Fatal(err)
			}
			_ = resp.Body.Close()
			t.Logf("redirect=%d flash_writes=%d stored=%v", resp.StatusCode, store.writes, store.saved)
			if resp.StatusCode != http.StatusSeeOther || store.writes != 1 {
				t.Error("core State flash/errors/old were not persisted on redirect")
			}
			page := readStatePage(t, app, "/next")
			checkMixedRedirectPage(t, mode, app, page)
		})
	}
}

func readStatePage(t *testing.T, app *fiber.App, path string) core.PageDTO {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), "GET", "https://app.example"+path, nil)
	r.Header.Set(core.HeaderInertia, "true")
	resp, err := app.Test(r)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("unexpected page status %d", resp.StatusCode)
	}
	var page core.PageDTO
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		t.Fatal(err)
	}
	return page
}

func applyMixedFiberStateCase(t *testing.T, mode string, i *fiberadapter.Inertia, c fiber.Ctx) {
	t.Helper()
	s := i.State(c)
	switch mode {
	case "core-helper":
		i.Inertia.WithProp(s, "auth", "core")
		i.WithProp(c, "helper", "native")
	case "helper-core":
		i.WithProp(c, "helper", "native")
		i.Inertia.WithProp(s, "auth", "core")
	case "core-data-helper":
		i.Inertia.WithValidationErrors(s, core.ValidationErrors{"email": {"invalid"}})
		i.Inertia.WithFlashOld(s, map[string]any{"email": "old"})
		i.Inertia.WithFlashSuccess(s, "saved")
		i.WithProp(c, "helper", "native")
	case "core-metadata-helper":
		i.Inertia.WithClearHistory(s)
		i.WithEncryptHistory(c)
	case "core-view-helper":
		i.Inertia.WithViewData(s, "title", "core")
		i.WithViewData(c, "other", "native")
		if i.State(c).ViewData["title"] != "core" {
			t.Error("core view data lost after helper")
		}
	case "repeated-read":
		i.Inertia.WithProp(s, "auth", "core")
		if i.State(c) != s || i.State(c).Props["auth"] != "core" {
			t.Error("repeated State read lost core prop")
		}
	case "clear-map":
		i.WithProp(c, "auth", "core")
		s.Props = nil
		if i.State(c).Props["auth"] != nil {
			t.Error("State reread resurrected cleared exported Props map")
		}
	case "delete-in-place":
		i.WithProp(c, "auth", "core")
		delete(s.Props, "auth")
	}
}

func checkMixedFiberPage(t *testing.T, mode string, page core.PageDTO) {
	t.Helper()
	switch mode {
	case "core-helper", "helper-core", "repeated-read":
		if page.Props["auth"] != "core" {
			t.Error("core auth prop lost")
		}
	case "core-data-helper":
		for _, key := range []string{"flash", "old"} {
			if page.Props[key] == nil {
				t.Errorf("core %s lost", key)
			}
		}
		if len(stateErrors(t, page)) == 0 {
			t.Error("core validation errors lost")
		}
	case "core-metadata-helper":
		if !page.ClearHistory || !page.EncryptHistory {
			t.Error("core clearHistory lost after helper metadata mutation")
		}
	case "clear-map", "delete-in-place":
		if page.Props["auth"] != nil {
			t.Error("cleared auth prop returned in Render")
		}
	}
}

func checkMixedRedirectPage(t *testing.T, mode string, app *fiber.App, page core.PageDTO) {
	t.Helper()
	if page.Props["old"] == nil || len(stateErrors(t, page)) == 0 {
		t.Error("redirect lost old input or validation errors")
	}
	if mode != "clear-flash" && page.Props["flash"] == nil {
		t.Error("redirect lost success flash")
	}
	if mode == "clear-flash" && page.Props["flash"] != nil {
		t.Error("cleared flash was persisted")
	}
	second := readStatePage(t, app, "/next")
	if second.Props["flash"] != nil || second.Props["old"] != nil || len(stateErrors(t, second)) != 0 {
		t.Error("flash was not consumed once")
	}
}

func stateErrors(t *testing.T, page core.PageDTO) map[string]any {
	t.Helper()
	errors, ok := page.Props["errors"].(map[string]any)
	if !ok {
		t.Fatalf("expected an errors object, got %T", page.Props["errors"])
	}
	return errors
}

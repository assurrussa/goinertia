package goinertia_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

//nolint:gocognit // Preserve the independent reviewer's exact case/boundary assertions.
func TestFiberStateReplacementOwnership(t *testing.T) {
	for _, mode := range []string{
		"props-nil", "props-replace", "view-nil", "view-replace", "meta-nil", "meta-replace", "raw-locals-control",
	} {
		for _, boundary := range []string{"read", "helper", "render"} {
			t.Run(mode+"/"+boundary, func(t *testing.T) {
				i := fiberadapter.New("https://app.example")
				app := fiber.New()
				app.Get("/page", func(c fiber.Ctx) error {
					i.WithProp(c, "auth", "stale")
					i.WithViewData(c, "auth", "stale")
					i.WithEncryptHistory(c)
					s := i.State(c)
					switch mode {
					case "props-nil":
						s.Props = nil
					case "props-replace":
						s.Props = map[string]any{"auth": "fresh"}
					case "view-nil":
						s.ViewData = nil
					case "view-replace":
						s.ViewData = map[string]any{"auth": "fresh"}
					case "meta-nil":
						s.SetLegacyPageMeta(nil)
					case "meta-replace":
						other := core.NewState(t.Context(), core.RequestMeta{})
						i.Inertia.WithClearHistory(other)
						s.SetLegacyPageMeta(other.LegacyPageMeta())
					case "raw-locals-control":
						c.Locals(fiberadapter.ContextKeyProps, map[string]any{"auth": "fresh"})
					}
					if boundary == "read" {
						_ = i.State(c)
					}
					if boundary == "helper" {
						i.WithProp(c, "helper", "retained")
						i.WithViewData(c, "helper", "retained")
						i.WithMatchPropsOn(c, "items.id")
					}
					err := i.Render(c, "Page", nil)
					if mode == "view-nil" && s.ViewData["auth"] != nil {
						t.Error("cleared view data resurrected")
					}
					if mode == "view-replace" && s.ViewData["auth"] != "fresh" {
						t.Error("replacement view data overwritten")
					}
					return err
				})
				page := readStatePage(t, app, "/page")
				switch mode {
				case "props-nil":
					if page.Props["auth"] != nil {
						t.Error("cleared auth prop resurrected")
					}
				case "props-replace", "raw-locals-control":
					if page.Props["auth"] != "fresh" {
						t.Error("replacement auth prop overwritten")
					}
				case "meta-nil":
					if page.EncryptHistory || page.ClearHistory {
						t.Error("cleared page metadata resurrected")
					}
				case "meta-replace":
					if page.EncryptHistory || !page.ClearHistory {
						t.Error("replacement page metadata overwritten")
					}
				}
			})
		}
	}
}

func TestFiberStateClearDoesNotPersistStaleFlash(t *testing.T) {
	store := &recordingFiberSession{}
	i := fiberadapter.New("https://app.example", fiberadapter.WithSessionStore(store))
	app := fiber.New()
	app.Use(i.Middleware())
	app.Post("/save", func(c fiber.Ctx) error {
		i.WithFlashSuccess(c, "stale")
		i.WithError(c, "email", "stale")
		i.WithFlashOld(c, map[string]any{"email": "stale"})
		s := i.State(c)
		s.Props = nil
		return i.RedirectBack(c)
	})
	// A read is unnecessary: redirect persistence itself must use current State.
	requestStateRedirect(t, app)
	if store.writes != 0 {
		t.Errorf("cleared State persisted stale session data: %v", store.saved)
	}
}

func requestStateRedirect(t *testing.T, app *fiber.App) {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), "POST", "https://app.example/save", nil)
	r.Header.Set(core.HeaderInertia, "true")
	r.Header.Set("Referer", "/next")
	response, err := app.Test(r)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("redirect status %d", response.StatusCode)
	}
}

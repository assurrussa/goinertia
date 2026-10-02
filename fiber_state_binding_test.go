package goinertia_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func TestFiberStateBindingOwnership(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		for _, mode := range []string{"state-replace", "state-clear", "locals-replace", "locals-clear", "both-rebind"} {
			t.Run(map[bool]string{false: "native/", true: "legacy/"}[legacy]+mode, func(t *testing.T) {
				t.Parallel()
				i := fiberLifecycleAdapter(legacy)
				app := fiber.New()
				app.Get("/page", func(c fiber.Ctx) error {
					i.WithProp(c, "obsolete", "old")
					s := i.State(c)
					bindFiberProps(i, c, s, mode)
					i.WithProp(c, "helper", "native")
					i.Inertia.WithProp(s, "late", "core")
					return i.Render(c, "Page", nil)
				})
				page := readStatePage(t, app, "/page")
				require.NotContains(t, page.Props, "obsolete")
				require.Equal(t, "native", page.Props["helper"])
				require.Equal(t, "core", page.Props["late"])
				switch mode {
				case "state-replace", "both-rebind":
					require.Equal(t, "state", page.Props["owner"])
				case "locals-replace":
					require.Equal(t, "locals", page.Props["owner"])
				default:
					require.NotContains(t, page.Props, "owner")
				}
			})
		}
	}
}

func bindFiberProps(i *fiberadapter.Inertia, c fiber.Ctx, s *core.State, mode string) {
	switch mode {
	case "state-replace":
		s.Props = map[string]any{"owner": "state"}
	case "state-clear":
		s.Props = nil
	case "locals-replace":
		c.Locals(core.ContextKeyProps, map[string]any{"owner": "locals"})
	case "locals-clear":
		c.Locals(core.ContextKeyProps, nil)
	case "both-rebind":
		s.Props = map[string]any{"owner": "state"}
		c.Locals(core.ContextKeyProps, map[string]any{"owner": "locals"})
	}
	_ = i.State(c)
}

func TestFiberStateViewAndMetadataClears(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		for _, mode := range []string{"state-clear", "state-replace", "locals-replace", "repair-invalid"} {
			t.Run(map[bool]string{false: "native/", true: "legacy/"}[legacy]+mode, func(t *testing.T) {
				t.Parallel()
				i := fiberLifecycleAdapter(legacy)
				fiberadapter.WithFS(fstest.MapFS{"app.gohtml": &fstest.MapFile{
					Data: []byte(`{{with .title}}{{.}}{{else}}empty{{end}}|{{with .obsolete}}obsolete{{end}}`),
				}})(i)
				app := fiber.New()
				app.Get("/view", func(c fiber.Ctx) error {
					i.WithViewData(c, "obsolete", "old")
					i.WithEncryptHistory(c)
					if mode == "repair-invalid" {
						c.Locals(core.ContextKeyViewData, []int{1})
					}
					s := i.State(c)
					bindFiberView(i, c, s, mode)
					s.SetLegacyPageMeta(nil)
					i.WithClearHistory(c)
					page, err := i.BuildPage(i.State(c), "Page", nil)
					if err != nil {
						return err
					}
					if page.EncryptHistory || !page.ClearHistory {
						return fiber.ErrInternalServerError
					}
					return i.Render(c, "Page", nil)
				})
				response, err := app.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/view", nil))
				require.NoError(t, err)
				defer response.Body.Close()
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.Equal(t, http.StatusOK, response.StatusCode)
				expected := map[string]string{
					"state-clear": "empty|", "state-replace": "state|", "locals-replace": "locals|", "repair-invalid": "repaired|",
				}
				require.Equal(t, expected[mode], string(body))
			})
		}
	}
}

func bindFiberView(i *fiberadapter.Inertia, c fiber.Ctx, s *core.State, mode string) {
	switch mode {
	case "state-clear":
		s.ViewData = nil
	case "state-replace":
		s.ViewData = map[string]any{"title": "state"}
	case "locals-replace":
		c.Locals(core.ContextKeyViewData, map[string]any{"title": "locals"})
	case "repair-invalid":
		i.Inertia.WithViewData(s, "title", "repaired")
	}
}

func TestFiberStateClearPreventsFlashPersistence(t *testing.T) {
	t.Parallel()
	store := &recordingFiberSession{}
	i := fiberadapter.New("https://app.example", fiberadapter.WithSessionStore(store))
	app := fiber.New()
	app.Use(i.Middleware())
	app.Post("/save", func(c fiber.Ctx) error {
		i.WithFlashSuccess(c, "obsolete")
		i.WithError(c, "obsolete", "old")
		i.WithFlashOld(c, map[string]any{"obsolete": "old"})
		i.State(c).Props = nil
		return i.Redirect(c, "/next")
	})
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save", nil)
	request.Header.Set(core.HeaderInertia, "true")
	response, err := app.Test(request)
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusSeeOther, response.StatusCode)
	require.Zero(t, store.writes)
}

func TestFiberLifecycleContextSnapshots(t *testing.T) {
	t.Parallel()
	i := fiberadapter.New("https://app.example")
	app := fiber.New()
	app.Get("/", func(c fiber.Ctx) error {
		first := context.WithValue(t.Context(), fiberAuthKey{}, "first")
		c.SetContext(first)
		before := i.State(c).Context
		c.SetContext(context.WithValue(first, fiberAuthKey{}, "second"))
		after := i.State(c).Context
		if before.Value(fiberAuthKey{}) != "first" || after.Value(fiberAuthKey{}) != "second" {
			return fiber.ErrInternalServerError
		}
		return c.SendStatus(http.StatusOK)
	})
	response, err := app.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
}

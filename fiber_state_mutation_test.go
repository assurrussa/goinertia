package goinertia_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func TestFiberNativeStateAndHelpersPreserveMutations(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "legacy"}[legacy], func(t *testing.T) {
			t.Parallel()
			i := fiberLifecycleAdapter(legacy)
			app := fiber.New()
			app.Get("/", func(c fiber.Ctx) error {
				s := i.State(c)
				i.Inertia.WithProp(s, "auth", "authenticated")
				i.Inertia.WithViewData(s, "title", "native title")
				i.Inertia.WithError(s, "native", "native error")
				i.Inertia.WithEncryptHistory(s)
				i.Inertia.WithMatchPropsOn(s, "items.id")
				i.WithProp(c, "role", "operator")
				i.WithViewData(c, "layout", "admin")
				i.WithError(c, "helper", "helper error")
				i.WithClearHistory(c)
				if i.State(c) != s || s.ViewData["title"] != "native title" || s.ViewData["layout"] != "admin" {
					return errors.New("Fiber helper replaced native State or view data")
				}
				return i.Render(c, "State", nil)
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			request.Header.Set(core.HeaderInertia, "true")
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusOK, response.StatusCode)
			var page core.PageDTO
			require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
			require.Equal(t, "authenticated", page.Props["auth"])
			require.Equal(t, "operator", page.Props["role"])
			require.Equal(t, map[string]any{"native": "native error", "helper": "helper error"}, page.Props["errors"])
			require.True(t, page.EncryptHistory)
			require.True(t, page.ClearHistory)
			require.Equal(t, []string{"items.id"}, page.MatchPropsOn)
		})
	}
}

func TestFiberNativeStateFlashPersistence(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		for _, conflict := range []bool{false, true} {
			name := map[bool]string{false: "native", true: "legacy"}[legacy] +
				map[bool]string{false: "/redirect", true: "/location-conflict"}[conflict]
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				store := session.NewStore()
				i := fiberLifecycleAdapter(legacy)
				fiberadapter.WithSessionStore(goinertia.NewFiberSessionAdapter(store))(i)
				app := fiber.New()
				app.Use(i.Middleware())
				app.Post("/save", func(c fiber.Ctx) error {
					s := i.State(c)
					i.Inertia.WithFlashSuccess(s, "saved")
					i.Inertia.WithError(s, "name", "required")
					i.Inertia.WithFlashOld(s, map[string]any{"name": "previous"})
					if conflict {
						c.Set(core.HeaderLocation, "/next")
						return c.SendStatus(http.StatusConflict)
					}
					return i.RedirectBack(c)
				})
				app.Get("/next", func(c fiber.Ctx) error { return i.Render(c, "State", nil) })
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save", nil)
				request.Header.Set(core.HeaderInertia, "true")
				request.Header.Set("Referer", "https://app.example/next")
				response, err := app.Test(request)
				require.NoError(t, err)
				if conflict {
					require.Equal(t, http.StatusConflict, response.StatusCode)
				} else {
					require.Equal(t, http.StatusSeeOther, response.StatusCode)
				}
				cookies := response.Cookies()
				require.NoError(t, response.Body.Close())
				require.NotEmpty(t, cookies)
				request = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/next", nil)
				request.Header.Set(core.HeaderInertia, "true")
				for _, cookie := range cookies {
					request.AddCookie(cookie)
				}
				for attempt := range 2 {
					response, err = app.Test(request)
					require.NoError(t, err)
					var page core.PageDTO
					require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
					require.NoError(t, response.Body.Close())
					if attempt == 0 {
						require.Equal(t, map[string]any{"success": "saved"}, page.Props["flash"])
						require.Equal(t, map[string]any{"name": "required"}, page.Props["errors"])
						require.Equal(t, map[string]any{"name": "previous"}, page.Props["old"])
					} else {
						require.NotContains(t, page.Props, "flash")
						require.Empty(t, page.Props["errors"])
						require.NotContains(t, page.Props, "old")
					}
				}
			})
		}
	}
}

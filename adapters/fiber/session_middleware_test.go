package fiberadapter_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/stretchr/testify/require"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

func TestMiddlewareSessionLifetime(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	app.Use(session.New())
	i := fiberadapter.New("https://app.example", fiberadapter.WithAssetVersion("v1"),
		fiberadapter.WithSessionStore(fiberadapter.MiddlewareSessionAdapter{}))
	app.Use(i.Middleware())
	app.Post("/save", func(c fiber.Ctx) error {
		managed := session.FromContext(c)
		managed.Set("auth", true)
		i.WithFlashSuccess(c, "saved")
		return i.Redirect(c, "/next")
	})
	app.Get("/next", func(c fiber.Ctx) error {
		return i.Render(c, "Page", map[string]any{"auth": session.FromContext(c).Get("auth")})
	})
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save", nil)
	req.Header.Set(core.HeaderInertia, "true")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusSeeOther, resp.StatusCode)
	cookies := resp.Cookies()
	_ = resp.Body.Close()
	require.NotEmpty(t, cookies)
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/next", nil)
	req.Header.Set(core.HeaderInertia, "true")
	req.Header.Set(core.HeaderVersion, "v1")
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	resp, err = app.Test(req)
	require.NoError(t, err)
	var page core.PageDTO
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	_ = resp.Body.Close()
	require.Equal(t, true, page.Props["auth"])
	require.Contains(t, page.Props, "flash")
	resp, err = app.Test(req)
	require.NoError(t, err)
	page = core.PageDTO{}
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
	_ = resp.Body.Close()
	require.Equal(t, true, page.Props["auth"])
	require.NotContains(t, page.Props, "flash")
}

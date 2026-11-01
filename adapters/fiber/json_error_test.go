package fiberadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/views"
)

func TestJSONErrorAcceptNegotiation(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		accept string
		json   bool
	}{
		{"application/json", true},
		{"Application/JSON; charset=utf-8", true},
		{"application/json, */*", true},
		{"application/json;q=0.8, text/html;q=0.2", true},
		{"application/json;q=1, text/*;q=0.5", true},
		{"", false},
		{"*/*", false},
		{"text/html", false},
		{"application/json;q=0", false},
		{"application/json;q=NaN", false},
		{"application/json;q=2", false},
		{"application/json;q=broken", false},
		{"application/json, text/html", false},
		{"application/json;q=0.5, text/html", false},
		{"application/json;q=0.5, */*", false},
	} {
		t.Run(test.accept, func(t *testing.T) {
			t.Parallel()
			manager := fiberadapter.New("https://app.example")
			app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
			app.Use(manager.Middleware())
			app.Put("/save", func(fiber.Ctx) error { return fiberadapter.NewError(419, "expired") })
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/save", nil)
			request.Header.Set("Accept", test.accept)
			request.Header.Set("Referer", "https://app.example/form")
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			if test.json {
				require.Equal(t, 419, response.StatusCode)
				require.Contains(t, response.Header.Get("Content-Type"), "application/json")
				require.Empty(t, response.Header.Get("Location"))
			} else {
				require.Equal(t, http.StatusFound, response.StatusCode)
				require.Equal(t, "https://app.example/form", response.Header.Get("Location"))
			}
		})
	}
}

func TestJSONHandlerErrorRetainsStatusAndSafeDetails(t *testing.T) {
	t.Parallel()
	for _, status := range []int{401, 403, 419, 422, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()
			manager := fiberadapter.New("https://app.example",
				fiberadapter.WithCanExposeDetails(func(context.Context, map[string][]string) bool { return false }),
				fiberadapter.WithCustomErrorDetailsHandler(func(_ *fiberadapter.Error, canExpose bool) string {
					if canExpose {
						return "private diagnostic"
					}
					return "safe message"
				}))
			app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
			app.Use(manager.Middleware())
			app.Patch("/save", func(c fiber.Ctx) error {
				c.Set(fiberadapter.HeaderInertia, "true")
				c.Set(fiberadapter.HeaderLocation, "/old-location")
				c.Set(fiberadapter.HeaderRedirect, "/old-redirect")
				c.Set("Location", "/old-location")
				return fiberadapter.NewValidationError(status, "private diagnostic", fiberadapter.ValidationErrors{"name": {"invalid"}})
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/save", nil)
			request.Header.Set("Accept", "application/json")
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, status, response.StatusCode)
			for _, header := range []string{"Location", fiberadapter.HeaderInertia, fiberadapter.HeaderLocation, fiberadapter.HeaderRedirect} {
				require.Empty(t, response.Header.Get(header), header)
			}
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Equal(t, "safe message", body["message"])
			require.Equal(t, map[string]any{"name": []any{"invalid"}}, body["errors"])
		})
	}
}

func TestJSONCSRFErrorDoesNotRedirectOrRunCommand(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		name := "native"
		if legacy {
			name = "legacy"
		}
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			store := &jsonErrorSessionStore{}
			options := []fiberadapter.Option{
				fiberadapter.WithFS(views.Templates),
				fiberadapter.WithSessionStore(store),
				fiberadapter.WithCSRFTokenCheckProvider(func(fiber.Ctx) error {
					return fiberadapter.NewError(419, "expired proof")
				}),
			}
			manager := fiberadapter.New("https://app.example", options...)
			if legacy {
				manager = fiberadapter.NewLegacy("https://app.example", options...)
			}
			app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
			app.Use(manager.Middleware())
			commands := 0
			app.Put("/auth/branding", func(c fiber.Ctx) error {
				commands++
				return c.SendStatus(http.StatusNoContent)
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/auth/branding", nil)
			request.Header.Set("Accept", "application/json")
			request.Header.Set("Referer", "https://app.example/auth/profile/settings")
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, 419, response.StatusCode)
			require.Zero(t, commands)
			require.Zero(t, store.writes, "a JSON CSRF error must not persist flash or validation data")
			require.Empty(t, response.Header.Get("Location"))
			require.Empty(t, response.Header.Get(fiberadapter.HeaderInertia))
			require.Empty(t, response.Header.Get(fiberadapter.HeaderLocation))
			require.Empty(t, response.Header.Get(fiberadapter.HeaderRedirect))
			require.Contains(t, response.Header.Get("Content-Type"), "application/json")
			require.Contains(t, response.Header.Get("Vary"), "Accept")
			require.Equal(t, "no-store", response.Header.Get("Cache-Control"))
			var body map[string]any
			require.NoError(t, json.NewDecoder(response.Body).Decode(&body))
			require.Contains(t, body, "message")
			require.NotContains(t, body, "flash")
		})
	}
}

type jsonErrorSessionStore struct {
	writes int
}

func (*jsonErrorSessionStore) Get(fiber.Ctx, string) (any, error) {
	return map[string]any{}, nil
}

func (s *jsonErrorSessionStore) Set(fiber.Ctx, string, any) error {
	s.writes++
	return nil
}

func (s *jsonErrorSessionStore) Delete(fiber.Ctx, string) error {
	s.writes++
	return nil
}

func (s *jsonErrorSessionStore) Flash(fiber.Ctx, string, any) error {
	s.writes++
	return nil
}

func (*jsonErrorSessionStore) GetFlash(fiber.Ctx, string) (any, error) {
	return map[string]any{}, nil
}

func TestJSONErrorDoesNotChangeBrowserOrInertiaRedirects(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, accept string
		inertia      bool
		status       int
	}{
		{"html", "text/html", false, http.StatusFound},
		{"default", "", false, http.StatusFound},
		{"wildcard", "*/*", false, http.StatusFound},
		{"inertia-v2", "application/json", true, http.StatusSeeOther},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			manager := fiberadapter.New("https://app.example", fiberadapter.WithFS(views.Templates),
				fiberadapter.WithCSRFTokenCheckProvider(func(fiber.Ctx) error { return fiberadapter.NewError(419, "expired") }))
			app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
			app.Use(manager.Middleware())
			app.Put("/save", func(c fiber.Ctx) error { return c.SendString("never") })
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPut, "/save", nil)
			request.Header.Set("Accept", test.accept)
			request.Header.Set("Referer", "https://app.example/form")
			if test.inertia {
				request.Header.Set(fiberadapter.HeaderInertia, "true")
			}
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, test.status, response.StatusCode)
			require.Equal(t, "https://app.example/form", response.Header.Get("Location"))
		})
	}
}

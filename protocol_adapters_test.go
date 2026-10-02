package goinertia_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

type protocolCase struct {
	name, method string
	headers      map[string]string
	action       string
	status       int
	check        func(*testing.T, *http.Response, map[string]any)
}

func protocolProps() map[string]any {
	return map[string]any{
		"plain": "value", "deferred": core.Defer("later"), "optional": core.Optional("extra"), "always": core.Always("keep"),
		"merged": core.Merge([]int{1}), "once": core.Once("first", core.WithOnceKey("once-key")),
		"lazy":   core.LazyProp{Key: "lazy", Fn: func(context.Context) (any, error) { return "computed", nil }},
		"scroll": core.Scroll([]int{2}, core.ScrollPropConfig{PageName: "page", CurrentPage: 1, NextPage: 2}),
	}
}

func TestProtocolAdapters(t *testing.T) {
	t.Parallel()

	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, fixture := range protocolFixtures() {
				t.Run(fixture.name, func(t *testing.T) {
					t.Parallel()
					runProtocolFixture(t, adapter, fixture)
				})
			}
		})
	}
}

func TestLegacyAndNeutralCallbackContexts(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{true, false} {
		t.Run(map[bool]string{true: "legacy", false: "neutral"}[legacy], func(t *testing.T) {
			t.Parallel()
			var i *fiberadapter.Inertia
			if legacy {
				i = goinertia.New("https://app.example")
			} else {
				i = fiberadapter.New("https://app.example")
			}
			app := fiber.New()
			app.Get("/deep/path", func(c fiber.Ctx) error {
				lifecycle, cancel := context.WithCancel(t.Context())
				cancel()
				c.SetContext(lifecycle)
				return i.Render(c,
					"Test",
					map[string]any{"breadcrumbs": core.LazyProp{
						Key: "breadcrumbs",
						Fn: func(ctx context.Context) (any,
							error,
						) {
							if legacy {
								value, ok := ctx.(fiber.Ctx)
								require.True(t, ok)
								return value.Path(), nil
							}
							require.ErrorIs(t, ctx.Err(), context.Canceled)
							value, ok := fiberadapter.FromContext(ctx)
							require.True(t, ok)
							return value.Path(), nil
						},
					}})
			})
			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/deep/path", nil)
			req.Header.Set(core.HeaderInertia, "true")
			resp, err := app.Test(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			var page core.PageDTO
			require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
			require.Equal(t, "/deep/path", page.Props["breadcrumbs"])
		})
	}
}

func protocolFixtures() []protocolCase {
	return []protocolCase{
		{
			name: "whitespace-precognition", method: "GET", status: http.StatusConflict,
			headers: map[string]string{core.HeaderPrecognition: " ", core.HeaderVersion: "old"},
		},

		{
			name: "precognition-without-inertia", method: "POST", action: "html", status: http.StatusNoContent,
			headers: map[string]string{core.HeaderPrecognition: "true", "Cache-Control": "no-cache"},
			check: func(t *testing.T, r *http.Response, _ map[string]any) {
				t.Helper()
				require.Contains(t, r.Header.Get("Vary"), core.HeaderPrecognition)
				require.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
			},
		},

		{name: "json", method: "GET", status: 200, check: func(t *testing.T, r *http.Response, p map[string]any) {
			t.Helper()
			require.Equal(t, "true", r.Header.Get(core.HeaderInertia))
			require.Equal(t, "computed", p["lazy"])
			require.NotContains(t, p, "deferred")
			require.NotContains(t, p, "optional")
			require.Contains(t, p, "errors")
		}},
		{name: "html", method: "GET", action: "html", status: 200, check: func(t *testing.T, r *http.Response, _ map[string]any) {
			t.Helper()
			require.Contains(t, r.Header.Get("Content-Type"), "text/html")
		}},
		{
			name:    "version-conflict",
			method:  "GET",
			headers: map[string]string{core.HeaderVersion: "old"},
			status:  409,
			check: func(t *testing.T,
				r *http.Response,
				_ map[string]any,
			) {
				t.Helper()
				require.Equal(t, "https://app.example/test?x=1", r.Header.Get(core.HeaderLocation))
			},
		},
		{
			name:   "partial",
			method: "GET",
			headers: map[string]string{
				core.HeaderPartialComponent: "Test",
				core.HeaderPartialOnly:      "optional,deferred",
			},
			status: 200,
			check: func(t *testing.T,
				_ *http.Response,
				p map[string]any,
			) {
				t.Helper()
				require.Equal(t, "extra", p["optional"])
				require.Equal(t, "later", p["deferred"])
				require.Equal(t, "keep", p["always"])
				require.NotContains(t, p, "plain")
			},
		},
		{
			name:   "except",
			method: "GET",
			headers: map[string]string{
				core.HeaderPartialComponent: "Test",
				core.HeaderPartialExcept:    "plain",
			},
			status: 200,
			check: func(t *testing.T,
				_ *http.Response,
				p map[string]any,
			) {
				t.Helper()
				require.NotContains(t, p, "plain")
				require.Contains(t, p, "always")
			},
		},
		{
			name:    "once-skip",
			method:  "GET",
			headers: map[string]string{core.HeaderExceptOnceProps: "once-key"},
			status:  200,
			check: func(t *testing.T,
				_ *http.Response,
				p map[string]any,
			) {
				t.Helper()
				require.NotContains(t,
					p,
					"once")
			},
		},

		{name: "post-redirect", method: "POST", action: "redirect", status: 303},
		{
			name:   "scheme-relative",
			method: "GET",
			action: "external",
			status: 409,
			check: func(t *testing.T,
				r *http.Response,
				_ map[string]any,
			) {
				t.Helper()
				require.Equal(t, "//other.example/path", r.Header.Get(core.HeaderLocation))
			},
		},
		{
			name:    "no-cache",
			method:  "GET",
			headers: map[string]string{"Cache-Control": "no-cache"},
			status:  200,
			check: func(t *testing.T,
				r *http.Response,
				_ map[string]any,
			) {
				t.Helper()
				require.Equal(t, "no-cache", r.Header.Get("Cache-Control"))
			},
		},
		{
			name:   "precognition",
			method: "POST",
			headers: map[string]string{
				core.HeaderPrecognition: "true",
				core.HeaderVersion:      "old",
			},
			status: 204,
			check: func(t *testing.T,
				r *http.Response,
				_ map[string]any,
			) {
				t.Helper()
				require.Equal(t, "true", r.Header.Get(core.HeaderPrecognitionSuccess))
			},
		},
		{
			name:   "precognition-errors",
			method: "POST",
			headers: map[string]string{
				core.HeaderPrecognition:             "true",
				core.HeaderPrecognitionValidateOnly: "email",
			},
			action: "validation",
			status: 422,
		},

		{
			name:    "error-bag",
			method:  "GET",
			headers: map[string]string{core.HeaderErrorBag: "form"},
			action:  "errors",
			status:  200,
			check: func(t *testing.T,
				_ *http.Response,
				p map[string]any,
			) {
				t.Helper()
				require.Equal(t, map[string]any{"form": map[string]any{"email": "invalid"}}, p["errors"])
			},
		},
	}
}

func runProtocolFixture(t *testing.T, adapter string, fixture protocolCase) {
	t.Helper()
	var test func(*http.Request) (*http.Response, error)
	if adapter == "http" {
		i := nethttp.New("https://app.example", nethttp.WithCoreOptions(core.WithFS(views.Templates), core.WithAssetVersion("v1")))
		h := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
			switch fixture.action {
			case "redirect":
				return i.Redirect(w, r, "/next")
			case "external":
				return i.Redirect(w, r, "//other.example/path")
			case "validation":
				i.WithErrors(nethttp.State(r), map[string]string{"email": "invalid", "name": "required"})
			case "errors":
				i.WithError(nethttp.State(r), "email", "invalid")
			}
			return i.Render(w, r, "Test", protocolProps())
		}))
		test = func(r *http.Request) (*http.Response, error) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w.Result(), nil
		}
	} else {
		var i *fiberadapter.Inertia
		if adapter == "legacy" {
			i = goinertia.New("https://app.example", goinertia.WithFS(views.Templates), goinertia.WithAssetVersion("v1"))
		} else {
			i = fiberadapter.New("https://app.example", fiberadapter.WithFS(views.Templates), fiberadapter.WithAssetVersion("v1"))
		}
		app := fiber.New(fiber.Config{ErrorHandler: i.MiddlewareErrorListener()})
		app.Use(i.Middleware())
		app.All("/test", func(c fiber.Ctx) error {
			switch fixture.action {
			case "redirect":
				return i.Redirect(c, "/next")
			case "external":
				return i.Redirect(c, "//other.example/path")
			case "validation":
				i.WithErrors(c, map[string]string{"email": "invalid", "name": "required"})
			case "errors":
				i.WithError(c, "email", "invalid")
			}
			return i.Render(c, "Test", protocolProps())
		})
		test = func(r *http.Request) (*http.Response, error) { return app.Test(r) }
	}
	req := httptest.NewRequestWithContext(t.Context(), fixture.method, "http://app.example/test?x=1", nil)
	if fixture.action != "html" {
		req.Header.Set(core.HeaderInertia, "true")
		req.Header.Set(core.HeaderVersion, "v1")
	}
	for key, value := range fixture.headers {
		req.Header.Set(key, value)
	}
	resp, err := test(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, fixture.status, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Vary"), core.HeaderInertia)
	props := map[string]any{}
	if strings.Contains(resp.Header.Get("Content-Type"), "application/json") && fixture.action != "validation" {
		var page core.PageDTO
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&page))
		props = page.Props
	}
	if fixture.check != nil {
		fixture.check(t, resp, props)
	}
}

func TestLegacyEmptyTemplateDefaults(t *testing.T) {
	t.Parallel()
	i, err := goinertia.NewWithValidation("https://app.example", goinertia.WithFS(views.Templates),
		goinertia.WithRootTemplate(""), goinertia.WithRootHotTemplate(""), goinertia.WithRootErrorTemplate(""))
	require.NoError(t, err)
	require.NotNil(t, i)
}

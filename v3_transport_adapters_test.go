package goinertia_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

type v3TransportFixture struct {
	action  string
	method  string
	status  int
	headers map[string]string
}

func runV3TransportFixture(
	t *testing.T, adapter string, version core.ProtocolVersion, fixture v3TransportFixture,
) *http.Response {
	t.Helper()
	r := httptest.NewRequestWithContext(t.Context(), fixture.method, "https://app.example/test", nil)
	r.Header.Set(core.HeaderInertia, "true")
	r.Header.Set(core.HeaderVersion, "assets-v3")
	for key, value := range fixture.headers {
		r.Header.Set(key, value)
	}
	props := map[string]any{"text": "</script><!-- \u2028 世界"}
	if adapter == "http" {
		i := nethttp.New("https://app.example", nethttp.WithCoreOptions(core.WithFS(views.Templates),
			core.WithProtocolVersion(version), core.WithAssetVersion("assets-v3")))
		h := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
			switch fixture.action {
			case "redirect":
				return i.Redirect(w, r, "/landing#section")
			case "raw-redirect":
				w.Header().Set("Content-Length", "14")
				http.Redirect(w, r, "/landing#section", fixture.status)
				return nil
			case "external":
				return i.Redirect(w, r, "https://external.example/#section")
			case "error":
				return errors.New("private backend detail")
			case "error-prop":
				return i.Render(w, r, "Test", map[string]any{"broken": core.LazyProp{Fn: func(context.Context) (any, error) {
					return nil, errors.New("private prop error")
				}}})
			case "error-page":
				return i.RenderWithStatus(w, r, http.StatusNotFound, "Error", props)
			default:
				i.WithPreserveFragment(nethttp.State(r), true)
				return i.Render(w, r, "Test", props)
			}
		}))
		recorder := httptest.NewRecorder()
		h.ServeHTTP(recorder, r)
		return recorder.Result()
	}
	options := []fiberadapter.Option{
		fiberadapter.WithFS(views.Templates), fiberadapter.WithProtocolVersion(version),
		fiberadapter.WithAssetVersion("assets-v3"),
	}
	i := fiberadapter.New("https://app.example", options...)
	if adapter == "root" {
		i = goinertia.New("https://app.example", options...)
	}
	app := fiber.New(fiber.Config{ErrorHandler: i.MiddlewareErrorListener()})
	app.Use(i.Middleware())
	app.All("/test", func(c fiber.Ctx) error {
		switch fixture.action {
		case "redirect":
			return i.Redirect(c, "/landing#section")
		case "raw-redirect":
			c.Set(fiber.HeaderContentLength, "14")
			return c.Redirect().Status(fixture.status).To("/landing#section")
		case "external":
			return i.Redirect(c, "https://external.example/#section")
		case "error":
			return errors.New("private backend detail")
		case "error-prop":
			return i.Render(c, "Test", map[string]any{"broken": core.LazyProp{Fn: func(context.Context) (any, error) {
				return nil, errors.New("private prop error")
			}}})
		case "error-page":
			return i.RenderWithStatus(c, http.StatusNotFound, "Error", props)
		default:
			i.WithPreserveFragment(c, true)
			return i.Render(c, "Test", props)
		}
	})
	resp, err := app.Test(r)
	require.NoError(t, err)
	return resp
}

func TestV3TransportAdapterMatrix(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"root", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, fixture := range []struct {
				name string
				v3TransportFixture
				want     int
				redirect string
			}{
				{
					name:               "fragment-helper-get",
					v3TransportFixture: v3TransportFixture{action: "redirect", method: "GET"},
					want:               409, redirect: "/landing#section",
				},
				{
					name:               "fragment-helper-write",
					v3TransportFixture: v3TransportFixture{action: "redirect", method: "POST"},
					want:               409, redirect: "/landing#section",
				},
				{
					name:               "fragment-raw",
					v3TransportFixture: v3TransportFixture{action: "raw-redirect", method: "PUT", status: 302},
					want:               409, redirect: "/landing#section",
				},
				{
					name:               "fragment-307",
					v3TransportFixture: v3TransportFixture{action: "raw-redirect", method: "GET", status: 307},
					want:               409, redirect: "/landing#section",
				},
				{
					name: "prefetch",
					v3TransportFixture: v3TransportFixture{
						action: "redirect", method: "GET",
						headers: map[string]string{"Purpose": "prefetch"},
					},
					want: 302,
				},
				{
					name: "secure-prefetch",
					v3TransportFixture: v3TransportFixture{
						action: "redirect", method: "GET",
						headers: map[string]string{"Sec-Purpose": "prefetch;prerender"},
					},
					want: 302,
				},
				{
					name: "plain-browser",
					v3TransportFixture: v3TransportFixture{
						action: "redirect", method: "GET",
						headers: map[string]string{core.HeaderInertia: ""},
					},
					want: 302,
				},
				{name: "external", v3TransportFixture: v3TransportFixture{action: "external", method: "GET"}, want: 409},
				{
					name:               "version-conflict",
					v3TransportFixture: v3TransportFixture{method: "GET", headers: map[string]string{core.HeaderVersion: "old"}},
					want:               409,
				},
				{name: "safe-error", v3TransportFixture: v3TransportFixture{action: "error", method: "GET"}, want: 500},
				{name: "safe-error-prop", v3TransportFixture: v3TransportFixture{action: "error-prop", method: "GET"}, want: 500},
				{name: "inertia-error-page", v3TransportFixture: v3TransportFixture{action: "error-page", method: "GET"}, want: 404},
				{name: "json-page", v3TransportFixture: v3TransportFixture{method: "GET"}, want: 200},
				{
					name:               "html-bootstrap",
					v3TransportFixture: v3TransportFixture{method: "GET", headers: map[string]string{core.HeaderInertia: ""}},
					want:               200,
				},
			} {
				t.Run(fixture.name, func(t *testing.T) {
					t.Parallel()
					response := runV3TransportFixture(t, adapter, core.ProtocolV3, fixture.v3TransportFixture)
					defer response.Body.Close()
					body, err := io.ReadAll(response.Body)
					require.NoError(t, err)
					require.Equal(t, fixture.want, response.StatusCode)
					require.Equal(t, fixture.redirect, response.Header.Get(core.HeaderRedirect))
					require.Contains(t, response.Header.Get("Vary"), core.HeaderInertia)
					if fixture.redirect != "" {
						require.Empty(t, body)
						require.Empty(t, response.Header.Get("Location"))
						require.Empty(t, response.Header.Get(core.HeaderInertia))
					}
					switch fixture.name {
					case "version-conflict":
						require.Equal(t, "assets-v3", response.Header.Get(core.HeaderVersion))
						require.Equal(t, "https://app.example/test", response.Header.Get(core.HeaderLocation))
					case "external":
						require.Equal(t, "https://external.example/#section", response.Header.Get(core.HeaderLocation))
					case "safe-error", "safe-error-prop":
						require.Empty(t, response.Header.Get(core.HeaderInertia))
						require.Empty(t, response.Header.Get("Location"))
						require.NotContains(t, string(body), "private")
					case "json-page", "inertia-error-page":
						require.Equal(t, "true", response.Header.Get(core.HeaderInertia))
						var page map[string]any
						require.NoError(t, json.Unmarshal(body, &page))
						if fixture.name == "json-page" {
							require.Equal(t, true, page["preserveFragment"])
						}
					case "html-bootstrap":
						require.Contains(t, string(body), `<script data-page="app" type="application/json">`)
						require.Contains(t, string(body), `\u003c\/script\u003e\u003c!--`)
						require.NotContains(t, string(body), `data-server-rendered`)
					}
				})
			}
		})
	}
}

func TestV2TransportDefaultsRetained(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"root", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, fixture := range []v3TransportFixture{
				{action: "redirect", method: "GET"},
				{action: "error", method: "GET"},
				{method: "GET", headers: map[string]string{core.HeaderVersion: "old"}},
				{method: "GET", headers: map[string]string{core.HeaderInertia: ""}},
			} {
				response := runV3TransportFixture(t, adapter, core.ProtocolV2, fixture)
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				require.Empty(t, response.Header.Get(core.HeaderRedirect))
				require.Empty(t, response.Header.Get(core.HeaderVersion))
				if fixture.action != "" {
					require.Equal(t, 302, response.StatusCode)
				}
				if fixture.headers[core.HeaderInertia] == "" && fixture.headers[core.HeaderVersion] != "old" && fixture.action == "" {
					require.Contains(t, string(body), `<div id="app" data-page=`)
				}
			}
		})
	}
}

func TestV3FragmentRedirectPersistsFlash(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"root", "fiber", "http"} {
		for _, raw := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/raw=%t", adapter, raw), func(t *testing.T) {
				t.Parallel()
				session := new(nativeFlashSession)
				request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "https://app.example/test", nil)
				request.Header.Set(core.HeaderInertia, "true")
				var response *http.Response
				if adapter == "http" {
					i := nethttp.New("https://app.example", nethttp.WithCoreOptions(core.WithProtocolVersion(core.ProtocolV3)),
						nethttp.WithSessionStore(&nativeFlashHTTPSession{session}))
					handler := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
						i.WithFlashSuccess(nethttp.State(r), "Saved")
						if raw {
							http.Redirect(w, r, "/target#saved", http.StatusFound)
							return nil
						}
						return i.Redirect(w, r, "/target#saved")
					}))
					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, request)
					response = recorder.Result()
					require.Contains(t, response.Header.Get("Set-Cookie"), "native-flash=saved")
				} else {
					opts := []fiberadapter.Option{
						fiberadapter.WithProtocolVersion(core.ProtocolV3),
						fiberadapter.WithSessionStore(&nativeFlashFiberSession{session}),
					}
					i := fiberadapter.New("https://app.example", opts...)
					if adapter == "root" {
						i = goinertia.New("https://app.example", opts...)
					}
					app := fiber.New()
					app.Use(i.Middleware())
					app.Post("/test", func(c fiber.Ctx) error {
						i.WithFlashSuccess(c, "Saved")
						if raw {
							return c.Redirect().Status(fiber.StatusFound).To("/target#saved")
						}
						return i.Redirect(c, "/target#saved")
					})
					var err error
					response, err = app.Test(request)
					require.NoError(t, err)
				}
				defer response.Body.Close()
				require.Equal(t, http.StatusConflict, response.StatusCode)
				require.Equal(t, "/target#saved", response.Header.Get(core.HeaderRedirect))
				_, writes := session.counts()
				require.Equal(t, 1, writes)
				data, ok := session.consume().(map[string]any)
				require.True(t, ok)
				require.Equal(t, map[string]string{"success": "Saved"}, data[core.ContextPropsFlash])
			})
		}
	}
}

type transportSSRClient struct{ requests atomic.Int32 }

func (*transportSSRClient) Reset() {}
func (c *transportSSRClient) Post(context.Context, string, []byte, map[string]string) (int, []byte, error) {
	c.requests.Add(1)
	return 500, []byte(`{"error":"private SSR failure"}`), nil
}

func TestV3SSRAdapterRequestControls(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"root", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := new(transportSSRClient)
			var reports atomic.Int32
			reporter := func(context.Context, error) { reports.Add(1) }
			config := core.SSRConfig{URL: "http://ssr.example/render", SSRClient: client, DisableRetries: true}
			var serve func(*http.Request) *http.Response
			if adapter == "http" {
				i := nethttp.New("https://app.example", nethttp.WithCoreOptions(core.WithProtocolVersion(core.ProtocolV3),
					core.WithFS(views.Templates), core.WithSSRConfig(config), core.WithSSRFailureHandler(reporter)))
				handler := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
					i.WithSSRDisabled(nethttp.State(r), r.URL.Query().Get("disabled") == "true")
					return i.Render(w, r, "Test", nil)
				}))
				serve = func(request *http.Request) *http.Response {
					recorder := httptest.NewRecorder()
					handler.ServeHTTP(recorder, request)
					return recorder.Result()
				}
			} else {
				opts := []fiberadapter.Option{
					fiberadapter.WithProtocolVersion(core.ProtocolV3), fiberadapter.WithFS(views.Templates),
					fiberadapter.WithSSRConfig(config), fiberadapter.WithSSRFailureHandler(reporter),
				}
				i := fiberadapter.New("https://app.example", opts...)
				if adapter == "root" {
					i = goinertia.New("https://app.example", opts...)
				}
				app := fiber.New()
				app.Use(i.Middleware())
				app.Get("/test", func(c fiber.Ctx) error {
					i.WithSSRDisabled(c, c.Query("disabled") == "true")
					return i.Render(c, "Test", nil)
				})
				serve = func(request *http.Request) *http.Response {
					response, err := app.Test(request)
					require.NoError(t, err)
					return response
				}
			}
			for _, fixture := range []struct {
				query string
				ajax  bool
				calls int32
			}{{"", false, 1}, {"?disabled=true", false, 1}, {"", true, 1}, {"", false, 2}} {
				request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://app.example/test"+fixture.query, nil)
				if fixture.ajax {
					request.Header.Set(core.HeaderInertia, "true")
				}
				response := serve(request)
				body, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				require.Equal(t, 200, response.StatusCode)
				require.NotContains(t, string(body), "private SSR failure")
				require.NotContains(t, string(body), "data-server-rendered")
				require.Equal(t, fixture.calls, client.requests.Load())
				require.Equal(t, fixture.calls, reports.Load())
			}
		})
	}
}

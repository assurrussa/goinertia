package goinertia_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

// The public origin uses HTTPS, but the backend sees plain HTTP without proxy
// trust. An explicit root must never acquire the backend's scheme.
func TestRedirectRootTargets(t *testing.T) {
	t.Parallel()

	for _, adapter := range []string{"legacy", "fiber", "http", "legacy-helper", "fiber-helper"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, protocol := range []struct {
				name    string
				version core.ProtocolVersion
			}{
				{"v2", core.ProtocolV2},
				{"v3", core.ProtocolV3},
			} {
				t.Run(protocol.name, func(t *testing.T) {
					runRootRedirectRequests(t, adapter, protocol.version)
				})
			}
		})
	}
}

func runRootRedirectRequests(t *testing.T, adapter string, protocol core.ProtocolVersion) {
	t.Helper()

	for _, inertia := range []bool{false, true} {
		requestKind := "ordinary"
		if inertia {
			requestKind = "inertia"
		}
		t.Run(requestKind, func(t *testing.T) {
			for _, method := range []string{
				http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
			} {
				t.Run(method, func(t *testing.T) {
					req := httptest.NewRequestWithContext(t.Context(), method, "http://app.example/login", nil)
					if inertia {
						req.Header.Set(core.HeaderInertia, "true")
						req.Header.Set(core.HeaderVersion, "v1")
					}
					runRootRedirectTargets(t, adapter, protocol, req)
				})
			}
		})
	}
}

func runRootRedirectTargets(t *testing.T, adapter string, protocol core.ProtocolVersion, req *http.Request) {
	t.Helper()

	for _, external := range []bool{false, true} {
		action := "redirect"
		if external {
			action = "external"
		}
		t.Run(action, func(t *testing.T) {
			for _, target := range []struct {
				name, value, location string
			}{
				{"root", "/", "/"},
				{"empty", "", "http://app.example"},
				{"absolute", "https://other.example/path?q=1", "https://other.example/path?q=1"},
			} {
				t.Run(target.name, func(t *testing.T) {
					resp := rootRedirectResponse(t, adapter, protocol, req, target.value, external)
					defer resp.Body.Close()

					inertia := req.Header.Get(core.HeaderInertia) != ""
					forced := external || (inertia && target.name != "root" &&
						adapter != "legacy-helper" && adapter != "fiber-helper")
					status := http.StatusFound
					if forced {
						status = http.StatusConflict
					} else if inertia && req.Method != http.MethodGet {
						status = http.StatusSeeOther
					}
					require.Equal(t, status, resp.StatusCode)
					require.Equal(t, target.location, resp.Header.Get("Location"))
					require.Empty(t, resp.Header.Get(core.HeaderRedirect))
					if forced {
						require.Equal(t, target.location, resp.Header.Get(core.HeaderLocation))
					} else {
						require.Empty(t, resp.Header.Get(core.HeaderLocation))
					}
					if target.name == "root" {
						requireBrowserRootOrigin(t, resp.Header.Get("Location"))
					}
				})
			}
		})
	}
}

func requireBrowserRootOrigin(t *testing.T, location string) {
	t.Helper()

	browserURL, err := url.Parse("https://app.example/login")
	require.NoError(t, err)
	target, err := url.Parse(location)
	require.NoError(t, err)
	require.Equal(t, "https://app.example/", browserURL.ResolveReference(target).String())
}

func rootRedirectResponse(
	t *testing.T,
	adapter string,
	protocol core.ProtocolVersion,
	req *http.Request,
	target string,
	external bool,
) *http.Response {
	t.Helper()

	if adapter == "http" {
		i := nethttp.New("https://app.example", nethttp.WithCoreOptions(
			core.WithAssetVersion("v1"), core.WithProtocolVersion(protocol),
		))
		handler := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
			if external {
				return i.RedirectExternal(w, r, target)
			}
			return i.Redirect(w, r, target)
		}))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w.Result()
	}

	var i *fiberadapter.Inertia
	if adapter == "legacy" || adapter == "legacy-helper" {
		i = goinertia.New("https://app.example",
			goinertia.WithAssetVersion("v1"), goinertia.WithProtocolVersion(protocol))
	} else {
		i = fiberadapter.New("https://app.example",
			fiberadapter.WithAssetVersion("v1"), fiberadapter.WithProtocolVersion(protocol))
	}
	app := fiber.New()
	app.Use(i.Middleware())
	app.All("/login", func(c fiber.Ctx) error {
		switch adapter {
		case "legacy-helper":
			if external {
				return goinertia.RedirectExternal(c, target)
			}
			return goinertia.Redirect(c, target)
		case "fiber-helper":
			if external {
				return fiberadapter.RedirectExternal(c, target)
			}
			return fiberadapter.Redirect(c, target)
		default:
			if external {
				return i.RedirectExternal(c, target)
			}
			return i.Redirect(c, target)
		}
	})
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp
}

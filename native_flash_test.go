package goinertia_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

type nativeFlashSession struct {
	mu            sync.Mutex
	data          any
	reads, writes int
}

func (s *nativeFlashSession) save(value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = value
	s.writes++
}

func (s *nativeFlashSession) consume() any {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reads++
	value := s.data
	s.data = nil
	return value
}

func (s *nativeFlashSession) counts() (reads, writes int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reads, s.writes
}

type nativeFlashFiberSession struct{ *nativeFlashSession }

func (s *nativeFlashFiberSession) Get(fiber.Ctx, string) (any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.data, nil
}
func (*nativeFlashFiberSession) Set(fiber.Ctx, string, any) error { return nil }
func (*nativeFlashFiberSession) Delete(fiber.Ctx, string) error   { return nil }
func (s *nativeFlashFiberSession) Flash(_ fiber.Ctx, _ string, v any) error {
	s.save(v)
	return nil
}

func (s *nativeFlashFiberSession) GetFlash(fiber.Ctx, string) (any, error) {
	return s.consume(), nil
}

type nativeFlashHTTPSession struct{ *nativeFlashSession }

func (s *nativeFlashHTTPSession) Flash(w http.ResponseWriter, _ *http.Request, _ string, v any) error {
	s.save(v)
	w.Header().Add("Set-Cookie", "native-flash=saved; Path=/")
	return nil
}

func (s *nativeFlashHTTPSession) GetFlash(http.ResponseWriter, *http.Request, string) (any, error) {
	return s.consume(), nil
}

type nativeFlashResult struct {
	StatusCode int
	Header     http.Header
}

type nativeFlashRequest func(method, path string, headers map[string]string) (nativeFlashResult, map[string]any)

func newNativeFlashFixture(t *testing.T, adapter string) (nativeFlashRequest, *nativeFlashSession) {
	t.Helper()
	store := new(nativeFlashSession)
	var test func(*http.Request) *http.Response
	if adapter == "http" {
		h := newNativeFlashHTTPHandler(store)
		test = func(r *http.Request) *http.Response {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			return w.Result()
		}
	} else {
		app := newNativeFlashFiberApp(adapter, store)
		test = func(r *http.Request) *http.Response {
			response, err := app.Test(r)
			require.NoError(t, err)
			return response
		}
	}
	return func(method, path string, headers map[string]string) (nativeFlashResult, map[string]any) {
		t.Helper()
		r := httptest.NewRequestWithContext(t.Context(), method, "http://app.example"+path, nil)
		r.Header.Set(core.HeaderInertia, "true")
		r.Header.Set(core.HeaderVersion, "v1")
		for key, value := range headers {
			r.Header.Set(key, value)
		}
		response := test(r)
		defer response.Body.Close()
		var page map[string]any
		if strings.Contains(response.Header.Get("Content-Type"), "application/json") {
			require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
		}
		return nativeFlashResult{StatusCode: response.StatusCode, Header: response.Header}, page
	}, store
}

func newNativeFlashHTTPHandler(store *nativeFlashSession) http.Handler {
	i := nethttp.New("https://app.example", nethttp.WithSessionStore(&nativeFlashHTTPSession{store}),
		nethttp.WithCoreOptions(core.WithAssetVersion("v1")))
	return i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
		s := nethttp.State(r)
		if r.URL.Path == "/legacy" {
			i.WithFlashSuccess(s, "legacy")
		}
		if r.URL.Path == "/save" || r.URL.Path == "/direct" || r.URL.Path == "/precognition" {
			i.WithNativeFlash(s, "message", "native")
			i.WithFlashSuccess(s, "legacy")
			i.WithError(s, "name", "required")
			i.WithFlashOld(s, map[string]any{"name": "old"})
		}
		if r.URL.Path == "/native-only" || r.URL.Path == "/external" {
			i.WithNativeFlash(s, "message", "native")
		}
		if r.URL.Path == "/override" {
			i.WithNativeFlash(s, "message", "current")
			i.WithNativeFlash(s, "extra", true)
		}
		if r.URL.Path == "/external" {
			return i.Redirect(w, r, "https://external.example/")
		}
		if r.URL.Path == "/save" || r.URL.Path == "/native-only" || r.URL.Path == "/redirect" {
			return i.Redirect(w, r, "/page")
		}
		return i.Render(w, r, "Page", map[string]any{"plain": true})
	}))
}

func newNativeFlashFiberApp(adapter string, store *nativeFlashSession) *fiber.App {
	var i *fiberadapter.Inertia
	if adapter == "legacy" {
		i = goinertia.New("https://app.example", goinertia.WithAssetVersion("v1"),
			goinertia.WithSessionStore(&nativeFlashFiberSession{store}))
	} else {
		i = fiberadapter.New("https://app.example", fiberadapter.WithAssetVersion("v1"),
			fiberadapter.WithSessionStore(&nativeFlashFiberSession{store}))
	}
	app := fiber.New(fiber.Config{ErrorHandler: i.MiddlewareErrorListener()})
	app.Use(i.Middleware())
	app.Use(func(c fiber.Ctx) error {
		if c.Path() == "/legacy" {
			i.WithFlashSuccess(c, "legacy")
		}
		if c.Path() == "/save" || c.Path() == "/direct" || c.Path() == "/precognition" {
			i.WithNativeFlash(c, "message", "native")
			i.WithFlashSuccess(c, "legacy")
			i.WithError(c, "name", "required")
			i.WithFlashOld(c, map[string]any{"name": "old"})
		}
		if c.Path() == "/native-only" || c.Path() == "/external" {
			i.WithNativeFlash(c, "message", "native")
		}
		if c.Path() == "/override" {
			i.WithNativeFlash(c, "message", "current")
			i.WithNativeFlash(c, "extra", true)
		}
		if c.Path() == "/external" {
			return i.Redirect(c, "https://external.example/")
		}
		if c.Path() == "/save" || c.Path() == "/native-only" || c.Path() == "/redirect" {
			return i.Redirect(c, "/page")
		}
		return i.Render(c, "Page", map[string]any{"plain": true})
	})
	return app
}

func TestNativeFlashAdapterLifecycle(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			send, store := newNativeFlashFixture(t, adapter)
			response, _ := send(http.MethodPost, "/save", nil)
			require.Equal(t, http.StatusSeeOther, response.StatusCode)
			if adapter == "http" {
				require.Contains(t, response.Header.Get("Set-Cookie"), "native-flash=saved")
			}
			reads, writes := store.counts()
			require.Zero(t, reads)
			require.Equal(t, 1, writes)
			response, _ = send(http.MethodGet, "/page", map[string]string{core.HeaderVersion: "old"})
			require.Equal(t, http.StatusConflict, response.StatusCode)
			response, _ = send(http.MethodGet, "/redirect", nil)
			require.Equal(t, http.StatusFound, response.StatusCode)
			response, _ = send(http.MethodPost, "/precognition", map[string]string{core.HeaderPrecognition: "true"})
			require.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
			response, _ = send(http.MethodPost, "/save", map[string]string{core.HeaderPrecognition: "true"})
			require.Equal(t, http.StatusSeeOther, response.StatusCode)
			reads, writes = store.counts()
			require.Zero(t, reads, "conflicts, redirect hops and Precognition must not consume flash")
			require.Equal(t, 1, writes, "Precognition must not persist flash")
			response, page := send(http.MethodGet, "/page", map[string]string{
				core.HeaderPartialComponent: "Page", core.HeaderPartialExcept: "flash,old,errors,plain",
			})
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, map[string]any{"message": "native"}, page["flash"])
			props := page["props"]
			require.Equal(t, map[string]any{
				"flash": map[string]any{"success": "legacy"}, "old": map[string]any{"name": "old"},
				"errors": map[string]any{"name": "required"},
			}, props)
			_, page = send(http.MethodGet, "/page", nil)
			require.NotContains(t, page, "flash")
			require.Equal(t, map[string]any{"plain": true, "errors": map[string]any{}}, page["props"])
			reads, writes = store.counts()
			require.Equal(t, 2, reads)
			require.Equal(t, 1, writes)
		})
	}
}

func TestNativeFlashAdapterDirectAndRedirect(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, path := range []string{"/direct", "/native-only", "/external"} {
				t.Run(path, func(t *testing.T) {
					t.Parallel()
					send, store := newNativeFlashFixture(t, adapter)
					response, page := send(http.MethodPost, path, nil)
					if path == "/direct" {
						require.Equal(t, http.StatusOK, response.StatusCode)
						_, writes := store.counts()
						require.Zero(t, writes, "direct renders do not persist native or legacy flash")
					} else {
						if path == "/external" {
							require.Equal(t, http.StatusConflict, response.StatusCode)
							require.Equal(t, "https://external.example/", response.Header.Get(core.HeaderLocation))
						} else {
							require.Equal(t, http.StatusSeeOther, response.StatusCode)
						}
						_, page = send(http.MethodGet, "/page", nil)
					}
					require.Equal(t, map[string]any{"message": "native"}, page["flash"])
					_, page = send(http.MethodGet, "/page", nil)
					require.NotContains(t, page, "flash")
				})
			}
		})
	}
}

func TestNativeFlashAdapterLegacyAndPrecedence(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			send, _ := newNativeFlashFixture(t, adapter)
			_, page := send(http.MethodGet, "/legacy", nil)
			require.NotContains(t, page, "flash")
			require.Equal(t, map[string]any{
				"plain": true, "errors": map[string]any{}, "flash": map[string]any{"success": "legacy"},
			}, page["props"])
			send(http.MethodPost, "/save", nil)
			_, page = send(http.MethodGet, "/override", map[string]string{
				core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "plain",
			})
			require.Equal(t, map[string]any{"message": "current", "extra": true}, page["flash"])
		})
	}
}

func TestNativeFlashFiberManagerIsolation(t *testing.T) {
	t.Parallel()
	for _, sameOwner := range []bool{false, true} {
		t.Run(map[bool]string{false: "foreign", true: "same"}[sameOwner], func(t *testing.T) {
			t.Parallel()
			store := new(nativeFlashSession)
			first := goinertia.New("https://app.example")
			second := fiberadapter.New("https://app.example")
			if sameOwner {
				second = first
			}
			fiberadapter.WithSessionStore(&nativeFlashFiberSession{store})(second)
			app := fiber.New()
			app.Use(second.Middleware())
			app.Post("/save", func(c fiber.Ctx) error {
				first.WithNativeFlash(c, "message", "private")
				return second.Redirect(c, "/page")
			})
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save", nil)
			request.Header.Set(core.HeaderInertia, "true")
			response, err := app.Test(request)
			require.NoError(t, err)
			defer response.Body.Close()
			require.Equal(t, http.StatusSeeOther, response.StatusCode)
			_, writes := store.counts()
			if sameOwner {
				require.Equal(t, 1, writes)
			} else {
				require.Zero(t, writes, "another manager must not persist private native flash metadata")
			}
		})
	}
}

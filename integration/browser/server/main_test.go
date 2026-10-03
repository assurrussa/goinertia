package main

import (
	"encoding/json"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
)

const fixtureURL = "http://127.0.0.1:18984"

type fixturePage struct {
	Component     string                         `json:"component"`
	Props         map[string]any                 `json:"props"`
	URL           string                         `json:"url"`
	Version       string                         `json:"version"`
	Flash         map[string]any                 `json:"flash"`
	DeferredProps map[string][]string            `json:"deferredProps"`
	MergeProps    []string                       `json:"mergeProps"`
	PrependProps  []string                       `json:"prependProps"`
	MatchPropsOn  []string                       `json:"matchPropsOn"`
	OnceProps     map[string]core.OncePropConfig `json:"onceProps"`
}

type fixtureResponse struct {
	StatusCode int
	Header     http.Header
	Body       string
	Cookies    []*http.Cookie
}

type fixtureClient struct {
	send    func(*http.Request) (fixtureResponse, error)
	cookies []*http.Cookie
}

func fixtureTransport(t *testing.T, cfg config) func(*http.Request) (fixtureResponse, error) {
	t.Helper()
	cfg.baseURL = fixtureURL
	if cfg.adapter == "fiber" {
		app, err := newFiber(cfg)
		require.NoError(t, err)
		return func(r *http.Request) (fixtureResponse, error) {
			//nolint:bodyclose // readResponse drains and closes the returned body.
			return readResponse(app.Test(r, fiber.TestConfig{}))
		}
	}
	handler, err := newHTTP(cfg)
	require.NoError(t, err)
	return func(r *http.Request) (fixtureResponse, error) {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, r)
		return readResponse(recorder.Result(), nil)
	}
}

func readResponse(response *http.Response, err error) (fixtureResponse, error) {
	if err != nil {
		return fixtureResponse{}, err
	}
	defer func() { _ = response.Body.Close() }()
	content, err := io.ReadAll(response.Body)
	return fixtureResponse{
		StatusCode: response.StatusCode, Header: response.Header,
		Body: string(content), Cookies: response.Cookies(),
	}, err
}

func (c *fixtureClient) request(t *testing.T, method, path, body string, headers map[string]string) (fixtureResponse, string) {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, fixtureURL+path, strings.NewReader(body))
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	response, err := c.send(req)
	require.NoError(t, err)
	for _, cookie := range response.Cookies {
		if cookie.Name == sessionCookie {
			c.cookies = []*http.Cookie{cookie}
		}
	}
	return response, response.Body
}

func (c *fixtureClient) page(t *testing.T, path string, extra map[string]string) fixturePage {
	t.Helper()
	headers := map[string]string{core.HeaderInertia: "true", core.HeaderVersion: assetVersion}
	for key, value := range extra {
		headers[key] = value
	}
	response, body := c.request(t, http.MethodGet, path, "", headers)
	require.Equal(t, http.StatusOK, response.StatusCode, body)
	require.Equal(t, "true", response.Header.Get(core.HeaderInertia))
	var page fixturePage
	require.NoError(t, json.Unmarshal([]byte(body), &page))
	return page
}

func (c *fixtureClient) submit(t *testing.T, body, contentType string) {
	t.Helper()
	response, data := c.request(t, http.MethodPost, "/submit", body, map[string]string{
		core.HeaderInertia: "true", core.HeaderVersion: assetVersion, "Content-Type": contentType,
	})
	require.Equal(t, http.StatusSeeOther, response.StatusCode, data)
	require.Equal(t, "/form", response.Header.Get("Location"))
}

func TestFixtureNavigationAndProps(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			assets := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(assets, "app.js"), []byte("window.fixtureLoaded = true;"), 0o600))
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: assets})}

			response, body := client.request(t, http.MethodGet, "/health", "", nil)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Equal(t, "ok", body)
			require.Empty(t, response.Cookies)
			response, body = client.request(t, http.MethodGet, "/assets/app.js", "", nil)
			require.Equal(t, http.StatusOK, response.StatusCode)
			require.Contains(t, body, "window.fixtureLoaded")
			response, body = client.request(t, http.MethodGet, "/", "", nil)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			require.Contains(t, body, `src="/assets/app.js"`)
			match := regexp.MustCompile(`data-page="([^"]+)"`).FindStringSubmatch(body)
			require.Len(t, match, 2)
			var initial fixturePage
			require.NoError(t, json.Unmarshal([]byte(html.UnescapeString(match[1])), &initial))
			require.Equal(t, "Home", initial.Component)
			require.Equal(t, adapter, initial.Props["adapter"])
			require.Equal(t, "Home", initial.Props["title"])
			require.Equal(t, assetVersion, initial.Version)
			require.Equal(t, []any{float64(1)}, initial.Props["items"])
			require.NotContains(t, initial.Props, "heavy")
			require.Equal(t, []string{"heavy"}, initial.DeferredProps["default"])
			require.Equal(t, []string{"items"}, initial.MergeProps)

			deferred := client.page(t, "/", map[string]string{
				core.HeaderPartialComponent: "Home", core.HeaderPartialOnly: "heavy",
			})
			require.Equal(t, "deferred-ready", deferred.Props["heavy"])
			require.NotContains(t, deferred.Props, "items")
			require.Empty(t, deferred.DeferredProps)
			merged := client.page(t, "/feed?page=2", map[string]string{
				core.HeaderPartialComponent: "Home", core.HeaderPartialOnly: "items",
			})
			require.Equal(t, "Home", merged.Component)
			require.Equal(t, "/feed?page=2", merged.URL)
			require.Equal(t, []any{float64(2)}, merged.Props["items"])
			require.Equal(t, []string{"items"}, merged.MergeProps)
			require.Empty(t, merged.DeferredProps)
			require.NotContains(t, merged.Props, "title")
			second := client.page(t, "/second", nil)
			require.Equal(t, "Second", second.Component)
			require.Equal(t, "Second", second.Props["title"])
		})
	}
}

func TestFixtureValidationAndFlash(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			send := fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})
			client, other := fixtureClient{send: send}, fixtureClient{send: send}
			client.submit(t, `{"name":""}`, "application/json")
			require.Empty(t, other.page(t, "/form", nil).Props["errors"])
			invalid := client.page(t, "/form", nil)
			require.Equal(t, "Form", invalid.Component)
			require.Equal(t, map[string]any{"name": "Required"}, invalid.Props["errors"])
			require.Empty(t, client.page(t, "/form", nil).Props["errors"])

			for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
				body := `{"name":"Browser"}`
				if contentType == "application/x-www-form-urlencoded" {
					body = "name=Browser"
				}
				client.submit(t, body, contentType)
				unrelated := other.page(t, "/form", nil)
				require.NotContains(t, unrelated.Props, "flash")
				require.Empty(t, unrelated.Flash)
				valid := client.page(t, "/form", map[string]string{
					core.HeaderPartialComponent: "Form", core.HeaderPartialOnly: "title",
				})
				require.Empty(t, valid.Props["errors"])
				require.Equal(t, map[string]any{"success": "Saved"}, valid.Props["flash"])
				require.Equal(t, map[string]any{"message": "Native saved"}, valid.Flash)
				next := client.page(t, "/form", nil)
				require.NotContains(t, next.Props, "flash")
				require.Empty(t, next.Flash)
			}
		})
	}
}

func TestFixtureConflictPreservesFlash(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})}
			client.submit(t, `{"name":"Browser"}`, "application/json")
			response, _ := client.request(t, http.MethodGet, "/form", "", map[string]string{
				core.HeaderInertia: "true", core.HeaderVersion: "obsolete",
			})
			require.Equal(t, http.StatusConflict, response.StatusCode)
			require.Equal(t, fixtureURL+"/form", response.Header.Get(core.HeaderLocation))
			page := client.page(t, "/form", nil)
			require.Equal(t, map[string]any{"success": "Saved"}, page.Props["flash"])
			require.Equal(t, map[string]any{"message": "Native saved"}, page.Flash)
		})
	}
}

func TestFixtureDirectNativeFlash(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})}
			page := client.page(t, "/native", nil)
			require.Equal(t, "Home", page.Component)
			require.Equal(t, map[string]any{"message": "Native saved"}, page.Flash)
			require.NotContains(t, page.Props, "flash")
			require.Empty(t, client.page(t, "/form", nil).Flash)
		})
	}
}

func TestFixtureConcurrentClients(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			send := fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})
			// Start Fiber's test server before concurrent requests to its routes.
			first := fixtureClient{send: send}
			first.page(t, "/form", nil)
			for clientIndex := range 8 {
				t.Run(strconv.Itoa(clientIndex), func(t *testing.T) {
					t.Parallel()
					client := fixtureClient{send: send}
					if clientIndex%2 == 0 {
						client.submit(t, `{"name":""}`, "application/json")
						page := client.page(t, "/form", nil)
						require.Equal(t, map[string]any{"name": "Required"}, page.Props["errors"])
						require.Empty(t, page.Flash)
					} else {
						client.submit(t, `{"name":"Browser"}`, "application/json")
						page := client.page(t, "/form", nil)
						require.Empty(t, page.Props["errors"])
						require.Equal(t, map[string]any{"message": "Native saved"}, page.Flash)
					}
					require.Empty(t, client.page(t, "/form", nil).Flash)
				})
			}
		})
	}
}

func TestFixtureSSRTransportAndTemplate(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			pages := make(chan fixturePage, 1)
			ssr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/render" || r.Method != http.MethodPost {
					http.Error(w, "unexpected SSR request", http.StatusBadRequest)
					return
				}
				var page fixturePage
				if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				pages <- page
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, `{"head":["<title>Rendered title</title>"],"body":"<div id=\"app\">Rendered body</div>"}`)
			}))
			t.Cleanup(ssr.Close)
			client := fixtureClient{send: fixtureTransport(t, config{
				adapter: adapter, assets: t.TempDir(), ssrURL: ssr.URL + "/render",
			})}
			response, body := client.request(t, http.MethodGet, "/native", "", nil)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			require.Contains(t, body, "<title>Rendered title</title>")
			require.Contains(t, body, `<div id="app">Rendered body</div>`)
			require.Contains(t, body, `src="/assets/app.js"`)
			require.NotContains(t, body, "data-page=")
			select {
			case page := <-pages:
				require.Equal(t, "Home", page.Component)
				require.Equal(t, adapter, page.Props["adapter"])
				require.Equal(t, map[string]any{"message": "Native saved"}, page.Flash)
			default:
				t.Fatal("SSR endpoint did not receive the page")
			}
		})
	}
}

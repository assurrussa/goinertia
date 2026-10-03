package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
)

func TestV3FixtureContract(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, protocol: 3, assets: t.TempDir()})}
			response, body := client.request(t, http.MethodGet, "/v3/unsafe", "", nil)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			matches := regexp.MustCompile(`<script[^>]*data-page="app"[^>]*>(.*?)</script>`).FindStringSubmatch(body)
			require.Len(t, matches, 2, body)
			var initial map[string]any
			require.NoError(t, json.Unmarshal([]byte(matches[1]), &initial))
			require.Equal(t, "SafeBootstrap", initial["component"])
			initialProps, ok := initial["props"].(map[string]any)
			require.True(t, ok)
			require.Contains(t, initialProps["unsafe"], "</script>")
			require.NotContains(t, body, "<script>window.__injected")
			require.NotContains(t, body, `<div id="app" data-page=`)

			headers := map[string]string{core.HeaderInertia: "true", core.HeaderVersion: assetVersion}
			response, body = client.request(t, http.MethodGet, "/v3/redirect", "", headers)
			require.Equal(t, http.StatusConflict, response.StatusCode, body)
			require.Equal(t, "/second#destination", response.Header.Get("X-Inertia-Redirect"))
			require.Empty(t, response.Header.Get(core.HeaderInertia))

			response, body = client.request(t, http.MethodGet, "/v3/fragment", "", headers)
			require.Equal(t, http.StatusFound, response.StatusCode, body)
			require.Equal(t, preservedFragmentPath, response.Header.Get("Location"))
			response, body = client.request(t, http.MethodGet, preservedFragmentPath, "", headers)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			var preserved map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &preserved))
			require.Equal(t, preservedFragmentPath, preserved["url"])
			require.Equal(t, true, preserved["preserveFragment"])

			for _, status := range []string{"404", "500"} {
				response, body = client.request(t, http.MethodGet, "/v3/status?code="+status, "", headers)
				require.Equal(t, status, strconv.Itoa(response.StatusCode))
				var page fixturePage
				require.NoError(t, json.Unmarshal([]byte(body), &page))
				require.Equal(t, "Exception page", page.Props["title"])
				require.Equal(t, "shared-value", page.Props["sharedMarker"])
			}

			response, body = client.request(t, http.MethodGet, "/v3/bigint", "", headers)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			require.Contains(t, body, `"preserveBigIntegers":true`)
			require.Contains(t, body, `"$bigint":"9007199254740993"`)
		})
	}
}

func TestV3FixtureRecursiveRescueAndPartial(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, protocol: 3, assets: t.TempDir()})}
			page := client.page(t, "/v3/recursive", nil)
			require.Equal(t, []string{"panel.heavy"}, page.DeferredProps["nested"])
			require.Equal(t, []string{"panel.rescued"}, page.DeferredProps["rescue"])
			require.Equal(t, "panel.cached", page.OnceProps["nested-cache"].Prop)
			headers := map[string]string{
				core.HeaderInertia: "true", core.HeaderVersion: assetVersion,
				core.HeaderPartialComponent: "Recursive", core.HeaderPartialOnly: "panel.rescued",
			}
			response, body := client.request(t, http.MethodGet, "/v3/recursive", "", headers)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			var rescued map[string]any
			require.NoError(t, json.Unmarshal([]byte(body), &rescued))
			require.Equal(t, []any{"panel.rescued"}, rescued["rescuedProps"])
			headers[core.HeaderPartialOnly] = "panel.cached,panel.label"
			headers[core.HeaderPartialExcept] = "panel.label"
			response, body = client.request(t, http.MethodGet, "/v3/recursive?step=3", "", headers)
			require.Equal(t, http.StatusOK, response.StatusCode, body)
			require.NoError(t, json.Unmarshal([]byte(body), &page))
			require.Equal(t, map[string]any{"cached": "cache-3"}, page.Props["panel"])
			// A callback-returned container follows the reference adapter: its
			// provider owns selection, unlike a directly nested map.
			provider := client.page(t, "/v3/provider", map[string]string{
				core.HeaderPartialComponent: "Provider", core.HeaderPartialOnly: "panel.provided",
			})
			require.Equal(t, map[string]any{"provided": "provider-value", "extra": "provider-child"}, provider.Props["panel"])
		})
	}
}

func TestV3FixtureOnceRefreshMetadata(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, protocol: 3, assets: t.TempDir()})}
			initial := client.page(t, "/v3/once-refresh", nil)
			require.Equal(t, "OnceMetadata", initial.Component)
			require.Equal(t, map[string]any{
				"title": "Once metadata", "adapter": adapter, "step": float64(1),
				"sharedMarker": "shared-value", "errors": map[string]any{},
				"tree": map[string]any{"name": "name-1", "sibling": "sibling-1"},
			}, initial.Props)
			require.Len(t, initial.OnceProps, 1)
			initialOnce := initial.OnceProps["container-cache"]
			require.Equal(t, "tree", initialOnce.Prop)
			require.NotNil(t, initialOnce.ExpiresAt)
			for _, visit := range []struct {
				step int
				path string
				only string
			}{
				{2, "tree", "tree.name"},
				{3, "tree", ""},
				{4, "renamed", "renamed.name"},
				{5, "renamed", ""},
			} {
				headers := map[string]string{core.HeaderExceptOnceProps: "container-cache"}
				if visit.only != "" {
					headers[core.HeaderPartialComponent] = "OnceMetadata"
					headers[core.HeaderPartialOnly] = visit.only
				}
				page := client.page(t, "/v3/once-refresh?step="+strconv.Itoa(visit.step), headers)
				require.Equal(t, "OnceMetadata", page.Component)
				require.Len(t, page.OnceProps, 1)
				once := page.OnceProps["container-cache"]
				require.Equal(t, visit.path, once.Prop)
				require.NotNil(t, once.ExpiresAt)
				require.Greater(t, *once.ExpiresAt, *initialOnce.ExpiresAt+(30*time.Minute).Milliseconds())
				if visit.only != "" {
					require.Equal(t, map[string]any{
						"errors": map[string]any{},
						visit.path: map[string]any{
							"name": "name-" + strconv.Itoa(visit.step), "sibling": "sibling-" + strconv.Itoa(visit.step),
						},
					}, page.Props)
				} else {
					require.Equal(t, map[string]any{
						"title": "Once metadata", "adapter": adapter, "step": float64(visit.step),
						"sharedMarker": "shared-value", "errors": map[string]any{},
					}, page.Props)
				}
			}
		})
	}
}

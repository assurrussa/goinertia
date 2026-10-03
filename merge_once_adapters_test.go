package goinertia_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

func TestNestedMergeAndOnceAdapters(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			for _, fixture := range []struct {
				name                                                   string
				headers                                                map[string]string
				fresh, refreshOnPartial, deferred                      bool
				wantValue, wantOnce, wantDeferred, wantData, wantOlder bool
			}{
				{
					name: "full mixed strategies", wantValue: true, wantOnce: true, wantData: true, wantOlder: true,
				},
				{
					name: "nested partial only", wantValue: true, wantOnce: true, wantData: true,
					headers: map[string]string{core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "feed.data"},
				},
				{
					name: "nested partial except", wantValue: true, wantOnce: true, wantOlder: true,
					headers: map[string]string{core.HeaderPartialComponent: "Page", core.HeaderPartialExcept: "feed.data"},
				},
				{
					name: "root reset suppresses merge and match", wantValue: true, wantOnce: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "feed", core.HeaderReset: "feed",
					},
				},
				{
					name: "remembered omission", wantOnce: true,
					headers: map[string]string{core.HeaderExceptOnceProps: "feed-key"},
				},
				{
					name: "explicit refresh", wantValue: true, wantOnce: true, wantData: true, wantOlder: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "feed", core.HeaderExceptOnceProps: "feed-key",
					},
				},
				{
					name: "except legacy remembers", wantOnce: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialExcept: "other", core.HeaderExceptOnceProps: "feed-key",
					},
				},
				{
					name: "except opt in refreshes", refreshOnPartial: true,
					wantValue: true, wantOnce: true, wantData: true, wantOlder: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialExcept: "other", core.HeaderExceptOnceProps: "feed-key",
					},
				},
				{
					name: "fresh remembered", fresh: true, wantValue: true, wantOnce: true, wantData: true, wantOlder: true,
					headers: map[string]string{core.HeaderExceptOnceProps: "feed-key"},
				},
				{
					name: "fresh excluded", fresh: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "other", core.HeaderExceptOnceProps: "feed-key",
					},
				},
				{
					name: "deferred initial", deferred: true, wantOnce: true, wantDeferred: true,
				},
				{
					name: "remembered deferred not scheduled", deferred: true, wantOnce: true,
					headers: map[string]string{core.HeaderExceptOnceProps: "feed-key"},
				},
				{
					name: "deferred nested partial", deferred: true, wantValue: true, wantOnce: true, wantData: true,
					headers: map[string]string{
						core.HeaderPartialComponent: "Page", core.HeaderPartialOnly: "feed.data", core.HeaderExceptOnceProps: "feed-key",
					},
				},
			} {
				t.Run(fixture.name, func(t *testing.T) {
					t.Parallel()
					calls := 0
					value := core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						_, isFiber := ctx.(fiber.Ctx)
						require.Equal(t, adapter == "legacy", isFiber)
						calls++
						return map[string]any{"data": []int{2}, "older": []int{1}}, nil
					}}
					mergeAt, appendAt, prependAt := core.MergeAt, core.AppendAt, core.PrependAt
					once, fresh, refresh := core.Once, core.WithOnceFresh, core.WithOnceRefreshOnPartial
					switch adapter {
					case "legacy":
						mergeAt, appendAt, prependAt = goinertia.MergeAt, goinertia.AppendAt, goinertia.PrependAt
						once, fresh, refresh = goinertia.Once, goinertia.WithOnceFresh, goinertia.WithOnceRefreshOnPartial
					case "fiber":
						mergeAt, appendAt, prependAt = fiberadapter.MergeAt, fiberadapter.AppendAt, fiberadapter.PrependAt
						once, fresh, refresh = fiberadapter.Once, fiberadapter.WithOnceFresh, fiberadapter.WithOnceRefreshOnPartial
					}
					options := []core.OnceOption{core.WithOnceKey("feed-key"), fresh(fixture.fresh)}
					if fixture.refreshOnPartial {
						options = append(options, refresh())
					}
					var prop any = mergeAt(once(value, options...), appendAt("data", "id"), prependAt("older", "id"))
					if fixture.deferred {
						prop = core.Defer(prop, "history")
					}
					page := stage3AdapterPage(t, adapter, map[string]any{"feed": prop}, fixture.headers)
					_, present := page.Props["feed"]
					require.Equal(t, fixture.wantValue, present)
					require.Equal(t, map[bool]int{false: 0, true: 1}[fixture.wantValue], calls)
					stage3AssertAdapterMetadata(t, page, fixture.wantOnce, fixture.wantDeferred, fixture.wantData, fixture.wantOlder)
				})
			}
		})
	}
}

func stage3AdapterPage(t *testing.T, adapter string, props map[string]any, headers map[string]string) core.PageDTO {
	t.Helper()
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://app.example/page", nil)
	request.Header.Set(core.HeaderInertia, "true")
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	var response *http.Response
	if adapter == "http" {
		inertia := nethttp.New("https://app.example")
		handler := inertia.Middleware(inertia.Handler(func(w http.ResponseWriter, r *http.Request) error {
			return inertia.Render(w, r, "Page", props)
		}))
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		response = recorder.Result()
	} else {
		inertia := fiberadapter.New("https://app.example")
		if adapter == "legacy" {
			inertia = goinertia.New("https://app.example")
		}
		app := fiber.New(fiber.Config{ErrorHandler: inertia.MiddlewareErrorListener()})
		app.Use(inertia.Middleware())
		app.Get("/page", func(c fiber.Ctx) error { return inertia.Render(c, "Page", props) })
		var err error
		response, err = app.Test(request)
		require.NoError(t, err)
	}
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.Contains(t, response.Header.Get("Vary"), core.HeaderInertia)
	var page core.PageDTO
	require.NoError(t, json.NewDecoder(response.Body).Decode(&page))
	return page
}

func stage3AssertAdapterMetadata(t *testing.T, page core.PageDTO, wantOnce, wantDeferred, wantData, wantOlder bool) {
	t.Helper()
	if wantOnce {
		require.Equal(t, map[string]core.OncePropConfig{"feed-key": {Prop: "feed"}}, page.OnceProps)
	} else {
		require.Empty(t, page.OnceProps)
	}
	if wantDeferred {
		require.Equal(t, map[string][]string{"history": {"feed"}}, page.DeferredProps)
	} else {
		require.Empty(t, page.DeferredProps)
	}
	var appendPaths, prependPaths, matchPaths []string
	if wantData {
		appendPaths = []string{"feed.data"}
		matchPaths = append(matchPaths, "feed.data.id")
	}
	if wantOlder {
		prependPaths = []string{"feed.older"}
		matchPaths = append(matchPaths, "feed.older.id")
	}
	require.ElementsMatch(t, appendPaths, page.MergeProps)
	require.ElementsMatch(t, prependPaths, page.PrependProps)
	require.ElementsMatch(t, matchPaths, page.MatchPropsOn)
}

package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
)

func TestFixtureOnceLifecycle(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})}
			initial := client.page(t, "/props", nil)
			require.Equal(t, "cached-1", initial.Props["cached"])
			require.Equal(t, "catalog", initial.OnceProps["shared-catalog"].Prop)
			require.NotNil(t, initial.OnceProps["expiring"].ExpiresAt)
			require.ElementsMatch(t, []string{"deferredOnce", "onceDeferred"}, initial.DeferredProps["composed"])
			for _, prop := range []string{"deferredOnce", "onceDeferred", "optionalOnce", "onceOptional"} {
				require.NotContains(t, initial.Props, prop)
				require.Equal(t, core.OncePropConfig{Prop: prop}, initial.OnceProps[prop])
			}

			const remembered = "stable-cache,shared-catalog,expiring,refreshable,deferredOnce,onceDeferred,optionalOnce,onceOptional"
			header := map[string]string{core.HeaderExceptOnceProps: remembered}
			reused := client.page(t, "/props?step=2", header)
			require.Empty(t, reused.DeferredProps)
			for _, prop := range []string{
				"cached", "catalog", "expiring", "refreshable", "deferredOnce", "onceDeferred", "optionalOnce", "onceOptional",
			} {
				require.NotContains(t, reused.Props, prop)
			}
			require.Equal(t, "cached", reused.OnceProps["stable-cache"].Prop)
			renamed := client.page(t, "/props-renamed?step=2", header)
			require.Equal(t, "PropsRenamed", renamed.Component)
			require.Equal(t, "renamedCatalog", renamed.OnceProps["shared-catalog"].Prop)
			require.NotContains(t, renamed.Props, "renamedCatalog")
			fresh := client.page(t, "/props?step=2&fresh=1", header)
			require.Equal(t, "cached-2", fresh.Props["cached"])
			require.NotContains(t, fresh.Props, "catalog")

			header[core.HeaderPartialComponent] = "Props"
			header[core.HeaderPartialOnly] = "cached"
			explicit := client.page(t, "/props?step=2", header)
			require.Equal(t, "cached-2", explicit.Props["cached"])
			delete(header, core.HeaderPartialOnly)
			header[core.HeaderPartialExcept] = "catalog"
			except := client.page(t, "/props?step=2", header)
			require.Equal(t, "refreshable-2", except.Props["refreshable"])
			require.NotContains(t, except.Props, "cached")
		})
	}
}

func TestFixtureComposedOnceSelection(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})}
			deferred := client.page(t, "/props", map[string]string{
				core.HeaderPartialComponent: "Props", core.HeaderPartialOnly: "deferredOnce,onceDeferred",
			})
			require.Equal(t, "deferredOnce-1", deferred.Props["deferredOnce"])
			require.Equal(t, "onceDeferred-1", deferred.Props["onceDeferred"])
			require.NotContains(t, deferred.Props, "cached")
			require.Empty(t, deferred.DeferredProps)
			optional := client.page(t, "/props", map[string]string{
				core.HeaderPartialComponent: "Props", core.HeaderPartialOnly: "optionalOnce,onceOptional",
			})
			require.Equal(t, "optionalOnce-1", optional.Props["optionalOnce"])
			require.Equal(t, "onceOptional-1", optional.Props["onceOptional"])
			require.NotContains(t, optional.Props, "deferredOnce")
			require.Empty(t, optional.DeferredProps)
		})
	}
}

func TestFixtureNestedMergeAndSelection(t *testing.T) {
	t.Parallel()
	for _, adapter := range []string{"fiber", "nethttp"} {
		t.Run(adapter, func(t *testing.T) {
			t.Parallel()
			client := fixtureClient{send: fixtureTransport(t, config{adapter: adapter, assets: t.TempDir()})}
			header := map[string]string{core.HeaderPartialComponent: "Nested", core.HeaderPartialOnly: "feed"}
			merged := client.page(t, "/nested?step=2", header)
			require.Equal(t, []string{"feed.data"}, merged.MergeProps)
			require.Equal(t, []string{"feed.older"}, merged.PrependProps)
			require.ElementsMatch(t, []string{"feed.data.id", "feed.older.id"}, merged.MatchPropsOn)
			require.NotContains(t, merged.Props, "selection")
			header[core.HeaderReset] = "feed"
			reset := client.page(t, "/nested?step=3", header)
			require.Empty(t, reset.MergeProps)
			require.Empty(t, reset.PrependProps)
			require.Empty(t, reset.MatchPropsOn)
			delete(header, core.HeaderReset)
			header[core.HeaderPartialOnly] = "selection.profile.name"
			selected := client.page(t, "/nested?step=2", header)
			require.Equal(t, map[string]any{"profile": map[string]any{"name": "name-2"}}, selected.Props["selection"])
			require.NotContains(t, selected.Props, "feed")
			delete(header, core.HeaderPartialOnly)
			header[core.HeaderPartialExcept] = "selection.profile.email"
			excluded := client.page(t, "/nested?step=3", header)
			require.Equal(t, map[string]any{
				"profile": map[string]any{"name": "name-3"}, "settings": map[string]any{"theme": "theme-3"},
			}, excluded.Props["selection"])
		})
	}
}

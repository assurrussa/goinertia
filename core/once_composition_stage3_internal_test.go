package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOnceRootWrapperCompositionBothOrders(t *testing.T) {
	t.Parallel()
	for _, wrapper := range []struct {
		name string
		wrap func(any) any
	}{
		{name: "deferred", wrap: func(value any) any { return Defer(value, "history") }},
		{name: "optional", wrap: func(value any) any { return Optional(value) }},
		{name: "merge", wrap: func(value any) any { return Merge(value) }},
		{name: "prepend", wrap: func(value any) any { return Prepend(value) }},
		{name: "deep merge", wrap: func(value any) any { return DeepMerge(value) }},
		{
			name: "nested merge",
			wrap: func(value any) any { return MergeAt(value, AppendAt("data", "id"), PrependAt("older", "id")) },
		},
	} {
		for _, onceOutside := range []bool{false, true} {
			for _, fixture := range []struct {
				name                       string
				meta                       RequestMeta
				options                    []OnceOption
				selected, refresh, partial bool
			}{
				{name: "initial", selected: true, refresh: true},
				{name: "remembered", meta: RequestMeta{ExceptOnceProps: "feed-key"}, selected: true},
				{
					name:     "only refresh",
					meta:     RequestMeta{PartialComponent: "Page", PartialOnly: "feed", ExceptOnceProps: "feed-key"},
					selected: true,
					refresh:  true,
					partial:  true,
				},
				{
					name:    "only excludes",
					meta:    RequestMeta{PartialComponent: "Page", PartialOnly: "other", ExceptOnceProps: "feed-key"},
					partial: true,
				},
				{
					name:     "except remembers",
					meta:     RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-key"},
					selected: true,
					partial:  true,
				},
				{
					name:     "except opt in",
					meta:     RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-key"},
					options:  []OnceOption{WithOnceRefreshOnPartial()},
					selected: true,
					refresh:  true,
					partial:  true,
				},
				{
					name:    "except excluded opt in",
					meta:    RequestMeta{PartialComponent: "Page", PartialExcept: "feed", ExceptOnceProps: "feed-key"},
					options: []OnceOption{WithOnceRefreshOnPartial()},
					partial: true,
				},
				{
					name:     "fresh initial",
					meta:     RequestMeta{ExceptOnceProps: "feed-key"},
					options:  []OnceOption{WithOnceFresh()},
					selected: true,
					refresh:  true,
				},
				{
					name:     "fresh selected partial",
					meta:     RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-key"},
					options:  []OnceOption{WithOnceFresh()},
					selected: true,
					refresh:  true,
					partial:  true,
				},
				{
					name:    "fresh excluded partial",
					meta:    RequestMeta{PartialComponent: "Page", PartialOnly: "other", ExceptOnceProps: "feed-key"},
					options: []OnceOption{WithOnceFresh()},
					partial: true,
				},
				{
					name:     "component mismatch",
					meta:     RequestMeta{PartialComponent: "Other", PartialOnly: "feed", ExceptOnceProps: "feed-key"},
					options:  []OnceOption{WithOnceRefreshOnPartial()},
					selected: true,
				},
			} {
				name := wrapper.name + "/" + map[bool]string{false: "wrapper outside", true: "once outside"}[onceOutside]
				t.Run(name+"/"+fixture.name, func(t *testing.T) {
					t.Parallel()
					calls := 0
					value := LazyProp{Fn: func(context.Context) (any, error) { calls++; return stage3MergeValue(), nil }}
					options := append([]OnceOption{WithOnceKey("feed-key")}, fixture.options...)
					prop := wrapper.wrap(Once(value, options...))
					if onceOutside {
						prop = Once(wrapper.wrap(value), options...)
					}
					page, err := New("https://app.example").BuildPage(NewState(t.Context(), fixture.meta), "Page", map[string]any{"feed": prop})
					require.NoError(t, err)
					initialValue := wrapper.name != "deferred" && wrapper.name != "optional"
					wantValue := fixture.selected && fixture.refresh && (fixture.partial || initialValue)
					_, present := page.Props["feed"]
					require.Equal(t, wantValue, present)
					require.Equal(t, map[bool]int{false: 0, true: 1}[wantValue], calls)
					if fixture.selected {
						require.Equal(t, map[string]OncePropConfig{"feed-key": {Prop: "feed"}}, page.OnceProps)
					} else {
						require.Empty(t, page.OnceProps)
					}
					if wrapper.name == "deferred" && fixture.selected && fixture.refresh && !fixture.partial {
						require.Equal(t, map[string][]string{"history": {"feed"}}, page.DeferredProps)
					} else {
						require.Empty(t, page.DeferredProps)
					}
					stage3AssertOnceMergeMetadata(t, page, wrapper.name, wantValue)
				})
			}
		}
	}
}

func TestOnceDeferredNestedMergeCombined(t *testing.T) {
	t.Parallel()
	for _, remembered := range []bool{false, true} {
		calls := 0
		value := LazyProp{Fn: func(context.Context) (any, error) { calls++; return stage3MergeValue(), nil }}
		prop := Defer(MergeAt(Once(value, WithOnceKey("feed-key")), AppendAt("data", "id"), PrependAt("older", "id")), "history")
		meta := RequestMeta{}
		if remembered {
			meta.ExceptOnceProps = "feed-key"
		}
		page, err := New("https://app.example").BuildPage(NewState(t.Context(), meta), "Page", map[string]any{"feed": prop})
		require.NoError(t, err)
		require.Zero(t, calls)
		require.NotContains(t, page.Props, "feed")
		require.Equal(t, map[string]OncePropConfig{"feed-key": {Prop: "feed"}}, page.OnceProps)
		require.Empty(t, page.MergeProps)
		require.Empty(t, page.PrependProps)
		require.Empty(t, page.MatchPropsOn)
		if remembered {
			require.Empty(t, page.DeferredProps)
		} else {
			require.Equal(t, map[string][]string{"history": {"feed"}}, page.DeferredProps)
		}
		meta.PartialComponent, meta.PartialOnly = "Page", "feed.data"
		page, err = New("https://app.example").BuildPage(NewState(t.Context(), meta), "Page", map[string]any{"feed": prop})
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Equal(t, map[string]any{"data": stage3MergeValue()["data"]}, page.Props["feed"])
		require.Equal(t, []string{"feed.data"}, page.MergeProps)
		require.Equal(t, []string{"feed.data.id"}, page.MatchPropsOn)
		require.Empty(t, page.PrependProps)
		require.Empty(t, page.DeferredProps)
	}
}

func stage3AssertOnceMergeMetadata(t *testing.T, page *PageDTO, wrapper string, wantValue bool) {
	t.Helper()
	var appendPaths, prependPaths, deepPaths, matchPaths []string
	if wantValue {
		require.Equal(t, stage3MergeValue(), page.Props["feed"])
		switch wrapper {
		case "merge":
			appendPaths = []string{"feed"}
		case "prepend":
			prependPaths = []string{"feed"}
		case "deep merge":
			deepPaths = []string{"feed"}
		case "nested merge":
			appendPaths = []string{"feed.data"}
			prependPaths = []string{"feed.older"}
			matchPaths = []string{"feed.data.id", "feed.older.id"}
		}
	}
	require.ElementsMatch(t, appendPaths, page.MergeProps)
	require.ElementsMatch(t, prependPaths, page.PrependProps)
	require.ElementsMatch(t, deepPaths, page.DeepMergeProps)
	require.ElementsMatch(t, matchPaths, page.MatchPropsOn)
}

package core

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func stage3MergeValue() map[string]any {
	return map[string]any{
		"data":  []any{map[string]any{"id": 2, "name": "new"}},
		"older": []any{map[string]any{"id": 1, "name": "old"}},
		"meta":  map[string]any{"page": 2},
	}
}

func TestNestedMergeSelectedTargetsAndReset(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		name                                  string
		meta                                  RequestMeta
		wantValue                             bool
		appendPaths, prependPaths, matchPaths []string
	}{
		{
			name:         "full",
			wantValue:    true,
			appendPaths:  []string{"feed.data"},
			prependPaths: []string{"feed.older"},
			matchPaths:   []string{"feed.data.id", "feed.older.id"},
		},
		{
			name:         "partial root",
			meta:         RequestMeta{PartialComponent: "Page", PartialOnly: "feed"},
			wantValue:    true,
			appendPaths:  []string{"feed.data"},
			prependPaths: []string{"feed.older"},
			matchPaths:   []string{"feed.data.id", "feed.older.id"},
		},
		{
			name:        "only append target",
			meta:        RequestMeta{PartialComponent: "Page", PartialOnly: "feed.data"},
			wantValue:   true,
			appendPaths: []string{"feed.data"},
			matchPaths:  []string{"feed.data.id"},
		},
		{
			name:         "only prepend target",
			meta:         RequestMeta{PartialComponent: "Page", PartialOnly: "feed.older"},
			wantValue:    true,
			prependPaths: []string{"feed.older"},
			matchPaths:   []string{"feed.older.id"},
		},
		{name: "only nonmerge leaf", meta: RequestMeta{PartialComponent: "Page", PartialOnly: "feed.meta.page"}, wantValue: true},
		{name: "only absent leaf", meta: RequestMeta{PartialComponent: "Page", PartialOnly: "feed.absent"}},
		{name: "only unrelated", meta: RequestMeta{PartialComponent: "Page", PartialOnly: "other"}},
		{
			name:         "except append target",
			meta:         RequestMeta{PartialComponent: "Page", PartialExcept: "feed.data"},
			wantValue:    true,
			prependPaths: []string{"feed.older"},
			matchPaths:   []string{"feed.older.id"},
		},
		{name: "except root", meta: RequestMeta{PartialComponent: "Page", PartialExcept: "feed"}},
		{name: "full root reset", meta: RequestMeta{Reset: "feed"}, wantValue: true},
		{
			name:      "partial root reset",
			meta:      RequestMeta{PartialComponent: "Page", PartialOnly: "feed.data", Reset: " feed "},
			wantValue: true,
		},
		{
			name:         "mismatched component",
			meta:         RequestMeta{PartialComponent: "Other", PartialOnly: "other"},
			wantValue:    true,
			appendPaths:  []string{"feed.data"},
			prependPaths: []string{"feed.older"},
			matchPaths:   []string{"feed.data.id", "feed.older.id"},
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			page, err := New("https://app.example").BuildPage(NewState(t.Context(), fixture.meta), "Page", map[string]any{
				"feed": MergeAt(stage3MergeValue(), AppendAt("data", "id"), PrependAt("older", "id"), AppendAt("absent", "id")),
			})
			require.NoError(t, err)
			_, present := page.Props["feed"]
			require.Equal(t, fixture.wantValue, present)
			require.ElementsMatch(t, fixture.appendPaths, page.MergeProps)
			require.ElementsMatch(t, fixture.prependPaths, page.PrependProps)
			require.ElementsMatch(t, fixture.matchPaths, page.MatchPropsOn)
			require.Empty(t, page.DeepMergeProps)
			if fixture.name == "only append target" {
				require.Equal(t, map[string]any{"data": stage3MergeValue()["data"]}, page.Props["feed"])
			}
		})
	}
}

func TestNestedMergeTargetsOwnTheirSlice(t *testing.T) {
	t.Parallel()
	targets := []MergeTarget{AppendAt("data", "id"), PrependAt("older", "id")}
	prop := MergeAt(stage3MergeValue(), targets...)
	targets[0] = PrependAt("incorrect", "wrong")
	targets[1].MatchOn = "wrong"
	page, err := New("https://app.example").BuildPage(NewState(t.Context(), RequestMeta{}), "Page", map[string]any{"feed": prop})
	require.NoError(t, err)
	require.Equal(t, []string{"feed.data"}, page.MergeProps)
	require.Equal(t, []string{"feed.older"}, page.PrependProps)
	require.Equal(t, []string{"feed.data.id", "feed.older.id"}, page.MatchPropsOn)
	require.Equal(t, []MergeTarget{AppendAt("data", "id"), PrependAt("older", "id")}, prop.Targets)
}

func TestNestedMergeRootNestedAndDuplicateTargets(t *testing.T) {
	t.Parallel()
	page, err := New("https://app.example").BuildPage(NewState(t.Context(), RequestMeta{}), "Page", map[string]any{
		"list": MergeAt([]int{1}, AppendAt("", "id"), AppendAt("", "id")),
		"feed": MergeAt(map[string]any{"history": map[string]any{"data": []int{2}}},
			PrependAt("history.data", "user.id"), PrependAt("history.data", "user.id")),
	})
	require.NoError(t, err)
	require.Equal(t, []string{"list"}, page.MergeProps)
	require.Equal(t, []string{"feed.history.data"}, page.PrependProps)
	require.ElementsMatch(t, []string{"list.id", "feed.history.data.user.id"}, page.MatchPropsOn)
}

func TestNestedMergeCallbacksAndPrunedFailures(t *testing.T) {
	t.Parallel()
	for _, rootFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "child failure", true: "root failure"}[rootFailure], func(t *testing.T) {
			t.Parallel()
			calls := 0
			broken := LazyProp{Fn: func(context.Context) (any, error) { calls++; return nil, errors.New("fixture failure") }}
			var value any = map[string]any{"data": broken, "older": []int{1}}
			if rootFailure {
				value = broken
			}
			page, err := New("https://app.example").BuildPage(NewState(t.Context(), RequestMeta{}), "Page", map[string]any{
				"feed": MergeAt(value, AppendAt("data", "id"), PrependAt("older", "id")),
			})
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.NotContains(t, page.Props, "feed")
			require.Empty(t, page.MergeProps)
			require.Empty(t, page.PrependProps)
			require.Empty(t, page.MatchPropsOn)
		})
	}
	calls := 0
	input := map[string]any{
		"data": LazyProp{Fn: func(context.Context) (any, error) { calls++; return []int{2}, nil }},
		"older": LazyProp{Fn: func(context.Context) (any, error) {
			t.Error("excluded callback ran")
			return nil, errors.New("excluded")
		}},
	}
	state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed.data"})
	inertia := New("https://app.example")
	for range 2 {
		page, err := inertia.BuildPage(state, "Page", map[string]any{
			"feed": MergeAt(input, AppendAt("data", "id"), PrependAt("older", "id")),
		})
		require.NoError(t, err)
		require.Equal(t, map[string]any{"data": []int{2}}, page.Props["feed"])
		require.Equal(t, []string{"feed.data"}, page.MergeProps)
		require.Equal(t, []string{"feed.data.id"}, page.MatchPropsOn)
		require.Empty(t, page.PrependProps)
	}
	require.Equal(t, 1, calls)
	require.IsType(t, LazyProp{}, input["data"])
	require.IsType(t, LazyProp{}, input["older"])
}

func TestNestedMergeDeferredBothOrders(t *testing.T) {
	t.Parallel()
	for _, deferredOutside := range []bool{false, true} {
		for _, partial := range []bool{false, true} {
			name := map[bool]string{false: "merge outside", true: "deferred outside"}[deferredOutside]
			name += "/" + map[bool]string{false: "initial", true: "partial"}[partial]
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				calls := 0
				value := LazyProp{Fn: func(context.Context) (any, error) { calls++; return stage3MergeValue(), nil }}
				var prop any = MergeAt(Defer(value, "history"), AppendAt("data", "id"), PrependAt("older", "id"))
				if deferredOutside {
					prop = Defer(MergeAt(value, AppendAt("data", "id"), PrependAt("older", "id")), "history")
				}
				meta := RequestMeta{}
				if partial {
					meta.PartialComponent = "Page"
					meta.PartialOnly = "feed"
				}
				page, err := New("https://app.example").BuildPage(NewState(t.Context(), meta), "Page", map[string]any{"feed": prop})
				require.NoError(t, err)
				if partial {
					require.Equal(t, 1, calls)
					require.Equal(t, stage3MergeValue(), page.Props["feed"])
					require.Equal(t, []string{"feed.data"}, page.MergeProps)
					require.Equal(t, []string{"feed.older"}, page.PrependProps)
					require.Equal(t, []string{"feed.data.id", "feed.older.id"}, page.MatchPropsOn)
					require.Empty(t, page.DeferredProps)
				} else {
					require.Zero(t, calls)
					require.NotContains(t, page.Props, "feed")
					require.Equal(t, map[string][]string{"history": {"feed"}}, page.DeferredProps)
					require.Empty(t, page.MergeProps)
					require.Empty(t, page.PrependProps)
					require.Empty(t, page.MatchPropsOn)
				}
			})
		}
	}
}

func TestNestedMergeOverrideDoesNotRetainStaleMetadata(t *testing.T) {
	t.Parallel()
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing target", true: "failed target"}[failure], func(t *testing.T) {
			t.Parallel()
			inertia := New("https://app.example")
			state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed"})
			inertia.WithProp(state, "feed", MergeAt(stage3MergeValue(), AppendAt("data", "id"), PrependAt("older", "id")))
			var replacement any = map[string]any{"meta": "replacement"}
			if failure {
				replacement = LazyProp{Fn: func(context.Context) (any, error) { return nil, errors.New("replacement failed") }}
			}
			page, err := inertia.BuildPage(state, "Page", map[string]any{
				"feed": MergeAt(replacement, AppendAt("data", "id"), PrependAt("older", "id")),
			})
			require.NoError(t, err)
			require.Empty(t, page.MergeProps)
			require.Empty(t, page.PrependProps)
			require.Empty(t, page.MatchPropsOn)
			if failure {
				require.NotContains(t, page.Props, "feed")
			} else {
				require.Equal(t, replacement, page.Props["feed"])
			}
		})
	}
}

func TestOnceFreshAndPartialPolicy(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		name                    string
		meta                    RequestMeta
		options                 []OnceOption
		wantValue, wantMetadata bool
	}{
		{name: "first visit", wantValue: true, wantMetadata: true},
		{name: "remembered full", meta: RequestMeta{ExceptOnceProps: "feed-v1"}, wantMetadata: true},
		{
			name:         "explicit only refreshes legacy",
			meta:         RequestMeta{PartialComponent: "Page", PartialOnly: "feed", ExceptOnceProps: "feed-v1"},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:         "nested only refreshes legacy",
			meta:         RequestMeta{PartialComponent: "Page", PartialOnly: "feed.data", ExceptOnceProps: "feed-v1"},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:         "except keeps legacy remembered",
			meta:         RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-v1"},
			wantMetadata: true,
		},
		{
			name:         "except opt in refreshes",
			meta:         RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceRefreshOnPartial()},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:    "except opt in does not override exclusion",
			meta:    RequestMeta{PartialComponent: "Page", PartialExcept: "feed", ExceptOnceProps: "feed-v1"},
			options: []OnceOption{WithOnceRefreshOnPartial()},
		},
		{
			name:         "component mismatch does not refresh",
			meta:         RequestMeta{PartialComponent: "Other", PartialOnly: "feed", ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceRefreshOnPartial()},
			wantMetadata: true,
		},
		{
			name:         "fresh implicit",
			meta:         RequestMeta{ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceFresh()},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:         "fresh explicit true",
			meta:         RequestMeta{ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceFresh(true)},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:         "fresh disabled",
			meta:         RequestMeta{ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceFresh(false)},
			wantMetadata: true,
		},
		{
			name:         "fresh last option wins",
			meta:         RequestMeta{ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceFresh(), WithOnceFresh(false)},
			wantMetadata: true,
		},
		{
			name:    "fresh excluded by only",
			meta:    RequestMeta{PartialComponent: "Page", PartialOnly: "other", ExceptOnceProps: "feed-v1"},
			options: []OnceOption{WithOnceFresh()},
		},
		{
			name:    "fresh excluded by except",
			meta:    RequestMeta{PartialComponent: "Page", PartialExcept: "feed", ExceptOnceProps: "feed-v1"},
			options: []OnceOption{WithOnceFresh()},
		},
		{
			name:         "option order preserves partial policy",
			meta:         RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceRefreshOnPartial(), WithOnceFresh(false)},
			wantValue:    true,
			wantMetadata: true,
		},
		{
			name:         "reverse option order preserves partial policy",
			meta:         RequestMeta{PartialComponent: "Page", PartialExcept: "other", ExceptOnceProps: "feed-v1"},
			options:      []OnceOption{WithOnceFresh(false), WithOnceRefreshOnPartial()},
			wantValue:    true,
			wantMetadata: true,
		},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			value := LazyProp{Fn: func(context.Context) (any, error) { calls++; return stage3MergeValue(), nil }}
			expires := time.Unix(2_000_000_000, 123_000_000)
			options := append([]OnceOption{WithOnceKey("feed-v1"), WithOnceExpiresAt(expires)}, fixture.options...)
			page, err := New("https://app.example").BuildPage(NewState(t.Context(), fixture.meta), "Page", map[string]any{
				"feed": Once(value, options...),
			})
			require.NoError(t, err)
			_, present := page.Props["feed"]
			require.Equal(t, fixture.wantValue, present)
			require.Equal(t, map[bool]int{false: 0, true: 1}[fixture.wantValue], calls)
			if fixture.wantMetadata {
				require.Equal(t, "feed", page.OnceProps["feed-v1"].Prop)
				require.NotNil(t, page.OnceProps["feed-v1"].ExpiresAt)
				require.Equal(t, expires.UnixMilli(), *page.OnceProps["feed-v1"].ExpiresAt)
			} else {
				require.Empty(t, page.OnceProps)
			}
		})
	}
}

func TestOnceLegacyPositionalLayouts(t *testing.T) {
	t.Parallel()
	// Deliberately unkeyed: downstream users must keep the original layouts.
	merge := MergeProp{[]int{1}, false, false}
	prepend := MergeProp{[]int{2}, true, false}
	once := OnceProp{"cache", nil, "value"}
	page, err := New("https://app.example").BuildPage(NewState(t.Context(), RequestMeta{}), "Page", map[string]any{
		"merged": merge, "prepend": prepend, "once": once,
	})
	require.NoError(t, err)
	require.Equal(t, []string{"merged"}, page.MergeProps)
	require.Equal(t, []string{"prepend"}, page.PrependProps)
	require.Equal(t, "value", page.Props["once"])
	require.Equal(t, OncePropConfig{Prop: "once"}, page.OnceProps["cache"])
}

func TestOnceMissingNestedSelectionDoesNotPublishMetadata(t *testing.T) {
	t.Parallel()
	for _, options := range [][]OnceOption{nil, {WithOnceFresh()}, {WithOnceRefreshOnPartial()}} {
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed.missing.child"})
		page, err := New("https://app.example").BuildPage(state, "Page", map[string]any{
			"feed": Once(stage3MergeValue(), options...),
		})
		require.NoError(t, err)
		require.NotContains(t, page.Props, "feed")
		require.Empty(t, page.OnceProps)
	}
}

func TestNestedMergeOverrideRetainsOnlyReplacementTargets(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	state := NewState(t.Context(), RequestMeta{})
	inertia.WithProp(state, "feed", MergeAt(stage3MergeValue(), AppendAt("data", "id"), PrependAt("older", "id")))
	page, err := inertia.BuildPage(state, "Page", map[string]any{
		"feed": MergeAt(map[string]any{"fresh": []int{3}}, AppendAt("fresh", "key")),
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"fresh": []int{3}}, page.Props["feed"])
	require.Equal(t, []string{"feed.fresh"}, page.MergeProps)
	require.Equal(t, []string{"feed.fresh.key"}, page.MatchPropsOn)
	require.Empty(t, page.PrependProps)
}

func TestNestedSelectionDoesNotEraseLiteralDottedMergeMetadata(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	for range 200 {
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed.items"})
		page, err := inertia.BuildPage(state, "Page", map[string]any{
			"feed": map[string]any{}, "feed.items": Merge([]int{1}),
		})
		require.NoError(t, err)
		require.NotContains(t, page.Props, "feed")
		require.Equal(t, []int{1}, page.Props["feed.items"])
		require.Equal(t, []string{"feed.items"}, page.MergeProps)
	}
}

func TestMissingLiteralSelectionDoesNotEraseNestedMergeMetadata(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	for range 200 {
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed.items.name"})
		page, err := inertia.BuildPage(state, "Page", map[string]any{
			"feed":       MergeAt(map[string]any{"items": map[string]any{"name": []int{1}}}, AppendAt("items.name", "id")),
			"feed.items": map[string]any{"other": "literal"},
		})
		require.NoError(t, err)
		require.NotContains(t, page.Props, "feed.items")
		require.Equal(t, map[string]any{"items": map[string]any{"name": []int{1}}}, page.Props["feed"])
		require.Equal(t, []string{"feed.items.name"}, page.MergeProps)
		require.Equal(t, []string{"feed.items.name.id"}, page.MatchPropsOn)
	}
}

func TestNestedMergeOverrideClearsPreviousWrapperMetadata(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		name     string
		previous any
	}{
		{name: "once", previous: Once("old", WithOnceKey("old-key"))},
		{name: "deferred", previous: Defer("old", "old-group")},
		{name: "once deferred", previous: Once(Defer("old", "old-group"), WithOnceKey("old-key"))},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			inertia := New("https://app.example")
			state := NewState(t.Context(), RequestMeta{})
			inertia.WithProp(state, "feed", fixture.previous)
			page, err := inertia.BuildPage(state, "Page", map[string]any{
				"feed": MergeAt(map[string]any{"data": []int{2}}, AppendAt("data", "id")),
			})
			require.NoError(t, err)
			require.Equal(t, map[string]any{"data": []int{2}}, page.Props["feed"])
			require.Equal(t, []string{"feed.data"}, page.MergeProps)
			require.Equal(t, []string{"feed.data.id"}, page.MatchPropsOn)
			require.Empty(t, page.OnceProps)
			require.Empty(t, page.DeferredProps)
		})
	}
}

func TestRememberedNestedMergeOverrideDropsPreviousValue(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	state := NewState(t.Context(), RequestMeta{ExceptOnceProps: "new-key"})
	inertia.WithProp(state, "feed", "stale-context")
	page, err := inertia.BuildPage(state, "Page", map[string]any{
		"feed": MergeAt(Once(stage3MergeValue(), WithOnceKey("new-key")), AppendAt("data", "id")),
	})
	require.NoError(t, err)
	require.NotContains(t, page.Props, "feed")
	require.Empty(t, page.MergeProps)
	require.Empty(t, page.MatchPropsOn)
	require.Equal(t, map[string]OncePropConfig{"new-key": {Prop: "feed"}}, page.OnceProps)
}

func TestNestedMergePlainLiteralDoesNotOwnRemovedMetadata(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	for range 200 {
		state := NewState(t.Context(), RequestMeta{})
		inertia.WithProp(state, "feed", MergeAt(map[string]any{"items": []int{1}}, AppendAt("items", "id")))
		page, err := inertia.BuildPage(state, "Page", map[string]any{
			"feed":       MergeAt(map[string]any{"other": "replacement"}),
			"feed.items": []int{2},
		})
		require.NoError(t, err)
		require.Equal(t, map[string]any{"other": "replacement"}, page.Props["feed"])
		require.Equal(t, []int{2}, page.Props["feed.items"])
		require.Empty(t, page.MergeProps)
		require.Empty(t, page.PrependProps)
		require.Empty(t, page.MatchPropsOn)
	}
}

func TestNestedMergeOverrideWithPlainValueClearsMetadata(t *testing.T) {
	t.Parallel()
	inertia := New("https://app.example")
	state := NewState(t.Context(), RequestMeta{})
	inertia.WithProp(state, "feed", MergeAt(stage3MergeValue(), AppendAt("data", "id"), PrependAt("older", "id")))
	page, err := inertia.BuildPage(state, "Page", map[string]any{
		"feed": map[string]any{"data": []int{3}, "older": []int{4}},
	})
	require.NoError(t, err)
	require.Equal(t, map[string]any{"data": []int{3}, "older": []int{4}}, page.Props["feed"])
	require.Empty(t, page.MergeProps)
	require.Empty(t, page.PrependProps)
	require.Empty(t, page.MatchPropsOn)
}

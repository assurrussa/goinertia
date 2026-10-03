package core

import (
	"context"
	"errors"
	"math"
	"net/http"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
)

func v3Page(t *testing.T, meta RequestMeta, props map[string]any) (*PageDTO, map[string]any) {
	t.Helper()
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	state := NewState(t.Context(), meta)
	page, err := engine.BuildPage(state, "Page", props)
	require.NoError(t, err)
	data, err := MarshalPageWithState(page, state)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(data, &wire))
	return page, wire
}

func TestV3NestedWrappersAndMetadata(t *testing.T) {
	t.Parallel()
	calls := 0
	lazy := func(context.Context) (any, error) { calls++; return "resolved", nil }
	props := map[string]any{"tree": map[string]any{
		"deferred": Rescue(Defer(lazy, "nested")),
		"optional": Optional(lazy), "once": Once("cached"),
		"merge": Merge([]int{1}), "prepend": Prepend([]int{2}),
		"deep":   DeepMerge(map[string]any{"value": true}),
		"always": Always("here"),
	}}
	page, _ := v3Page(t, RequestMeta{}, props)
	require.Zero(t, calls)
	require.Equal(t, map[string][]string{"nested": {"tree.deferred"}}, page.DeferredProps)
	require.Equal(t, "tree.once", page.OnceProps["tree.once"].Prop)
	require.Equal(t, []string{"tree.merge"}, page.MergeProps)
	require.Equal(t, []string{"tree.prepend"}, page.PrependProps)
	require.Equal(t, []string{"tree.deep"}, page.DeepMergeProps)
	tree, ok := page.Props["tree"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, tree, "deferred")
	require.NotContains(t, tree, "optional")
	page, _ = v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "tree.deferred,tree.optional"}, props)
	require.Equal(t, 2, calls)
	require.Equal(t, map[string]any{"deferred": "resolved", "optional": "resolved", "always": "here"}, page.Props["tree"])
	require.Empty(t, page.DeferredProps)
}

func TestV3PartialPolicy(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		meta RequestMeta
		want map[string]any
	}{
		{
			"intersection",
			RequestMeta{PartialComponent: "Page", PartialOnly: "tree.name", PartialExcept: "tree.secret"},
			map[string]any{"name": "Ada"},
		},
		{
			"except wins overlap",
			RequestMeta{PartialComponent: "Page", PartialOnly: "tree", PartialExcept: "tree.name"},
			map[string]any{"secret": "private"},
		},
		{
			"component mismatch",
			RequestMeta{PartialComponent: "Other", PartialOnly: "tree.name"},
			map[string]any{"name": "Ada", "secret": "private"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			page, _ := v3Page(t, test.meta, map[string]any{"tree": map[string]any{"name": "Ada", "secret": "private"}})
			require.Equal(t, test.want, page.Props["tree"])
		})
	}
}

func TestV3ResolvedContainerAndDynamicWrappers(t *testing.T) {
	t.Parallel()
	calls := 0
	page, _ := v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "tree.name"}, map[string]any{
		"tree": func(context.Context) (any, error) {
			calls++
			return map[string]any{"name": "Ada", "sibling": Once("included")}, nil
		},
		"skipped": func(context.Context) (any, error) { t.Fatal("excluded callback ran"); return "unreachable", nil },
	})
	require.Equal(t, 1, calls)
	require.Equal(t, map[string]any{"name": "Ada", "sibling": "included"}, page.Props["tree"])
	page, _ = v3Page(t, RequestMeta{}, map[string]any{
		"dynamic": func(context.Context) (any, error) { return Defer("later", "dynamic"), nil },
	})
	require.NotContains(t, page.Props, "dynamic")
	require.Equal(t, []string{"dynamic"}, page.DeferredProps["dynamic"])
}

func TestV3OncePartialRefreshAndOrdering(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name string
		meta RequestMeta
		want bool
	}{
		{"initial", RequestMeta{}, true},
		{"remembered inertia", RequestMeta{Inertia: "true", ExceptOnceProps: "cache"}, false},
		{"non inertia ignores loaded", RequestMeta{ExceptOnceProps: "cache"}, true},
		{"except partial refreshes", RequestMeta{
			Inertia: "true", ExceptOnceProps: "cache", PartialComponent: "Page", PartialExcept: "other",
		}, true},
		{"component-only partial refreshes", RequestMeta{Inertia: "true", ExceptOnceProps: "cache", PartialComponent: "Page"}, true},
		{"excluded partial", RequestMeta{
			Inertia: "true", ExceptOnceProps: "cache", PartialComponent: "Page", PartialOnly: "other",
		}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			calls := 0
			page, _ := v3Page(t, test.meta, map[string]any{"tree": map[string]any{
				"value": Once(func(context.Context) (any, error) { calls++; return "new", nil }, WithOnceKey("cache")),
			}})
			require.Equal(t, map[bool]int{true: 1, false: 0}[test.want], calls)
			if test.want {
				require.Equal(t, map[string]any{"value": "new"}, page.Props["tree"])
			}
		})
	}
	for _, prop := range []any{Once(Defer("value")), Defer(Once("value")), Rescue(Defer(Once("value")))} {
		page, _ := v3Page(t, RequestMeta{}, map[string]any{"value": prop})
		require.NotContains(t, page.Props, "value")
		require.Equal(t, []string{"value"}, page.DeferredProps["default"])
		require.Equal(t, "value", page.OnceProps["value"].Prop)
		page, _ = v3Page(t, RequestMeta{Inertia: "true", ExceptOnceProps: "value"}, map[string]any{"value": prop})
		require.Empty(t, page.DeferredProps)
	}
}

func TestV3RescueAndCancellation(t *testing.T) {
	t.Parallel()
	errExpected := errors.New("private provider failed")
	failing := func(context.Context) (any, error) { return nil, errExpected }
	meta := RequestMeta{PartialComponent: "Page", PartialOnly: "tree.value"}
	page, wire := v3Page(t, meta, map[string]any{"tree": map[string]any{"value": Rescue(Defer(failing))}})
	require.Equal(t, map[string]any{}, page.Props["tree"])
	require.Equal(t, []any{"tree.value"}, wire["rescuedProps"])
	require.NotContains(t, wire, "private provider failed")
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	_, err := engine.BuildPage(NewState(t.Context(), meta), "Page", map[string]any{"tree": map[string]any{"value": Defer(failing)}})
	require.ErrorIs(t, err, errExpected)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = engine.BuildPage(NewState(ctx, meta), "Page", map[string]any{"tree": Rescue(failing)})
	require.ErrorIs(t, err, context.Canceled)
}

func TestV3ContainersAndSelectionSafety(t *testing.T) {
	t.Parallel()
	type named map[string]any
	props := map[string]any{
		"array": []map[string]any{{"deferred": Defer("later"), "visible": 1}},
		"typed": named{"nested": Once("typed")},
		"bytes": []byte("abc"),
	}
	page, wire := v3Page(t, RequestMeta{}, props)
	require.Equal(t, []string{"array.0.deferred"}, page.DeferredProps["default"])
	require.Equal(t, "typed.nested", page.OnceProps["typed.nested"].Prop)
	values, ok := wire["props"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "YWJj", values["bytes"])
	original, ok := props["array"].([]map[string]any)
	require.True(t, ok)
	require.IsType(t, DeferredProp{}, original[0]["deferred"])
	cycle := map[string]any{}
	cycle["cycle"] = cycle
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	_, err := engine.BuildPage(NewState(t.Context(), RequestMeta{}), "Page", cycle)
	require.ErrorContains(t, err, "exceeds nested resolution depth")
}

func TestV3MergeScrollAndReset(t *testing.T) {
	t.Parallel()
	props := map[string]any{"tree": map[string]any{
		"feed":  ScrollAt(map[string]any{"items": []int{1}}, ScrollPropConfig{NextPage: 2, CurrentPage: 1}, "items"),
		"merge": MergeAt(map[string]any{"data": []int{1}, "older": []int{0}}, AppendAt("data", "id"), PrependAt("older")),
	}}
	page, wire := v3Page(t, RequestMeta{ScrollMergeIntent: "prepend"}, props)
	require.Equal(t, []string{"tree.merge.data"}, page.MergeProps)
	require.ElementsMatch(t, []string{"tree.feed.items", "tree.merge.older"}, page.PrependProps)
	require.Equal(t, []string{"tree.merge.data.id"}, page.MatchPropsOn)
	scroll, ok := wire["scrollProps"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, map[string]any{
		"pageName": "page", "previousPage": nil, "nextPage": float64(2), "currentPage": float64(1), "reset": false,
	}, scroll["tree.feed"])
	page, wire = v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "tree", Reset: "tree"}, props)
	require.Empty(t, page.MergeProps)
	require.Empty(t, page.PrependProps)
	require.Empty(t, page.MatchPropsOn)
	scroll, ok = wire["scrollProps"].(map[string]any)
	require.True(t, ok)
	feed, ok := scroll["tree.feed"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, feed["reset"])
}

func TestV3SharedPrecedenceAndWireBigIntegers(t *testing.T) {
	t.Parallel()
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3), WithPreserveBigIntegers(true),
		WithSharedProps(map[string]any{
			"shared": "shared", "overridden": func(context.Context) (any, error) {
				t.Fatal("overridden callback ran")
				return "unreachable", nil
			},
		}))
	state := NewState(t.Context(), RequestMeta{Method: http.MethodGet})
	engine.WithProp(state, "overridden", "context")
	engine.WithNativeFlash(state, "large", uint64(math.MaxUint64))
	page, err := engine.BuildPage(state, "Page", map[string]any{
		"overridden": "request", "large": int64(math.MaxInt64), "negative": int64(math.MinInt64),
		"safe": int64(9007199254740991), "float": 1.5,
		"struct": struct {
			Identifier uint64 `json:"identifier"`
		}{math.MaxUint64},
	})
	require.NoError(t, err)
	data, err := MarshalPageWithState(page, state)
	require.NoError(t, err)
	var wire map[string]any
	require.NoError(t, json.Unmarshal(data, &wire))
	require.Equal(t, []any{"overridden", "shared"}, wire["sharedProps"])
	require.Equal(t, true, wire["preserveBigIntegers"])
	values, ok := wire["props"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "request", values["overridden"])
	require.Equal(t, map[string]any{"$bigint": "9223372036854775807"}, values["large"])
	require.Equal(t, map[string]any{"$bigint": "-9223372036854775808"}, values["negative"])
	require.InDelta(t, float64(9007199254740991), values["safe"], 0)
	require.InDelta(t, 1.5, values["float"], 0)
	require.Contains(t, string(data), `"$bigint":"18446744073709551615"`)
	require.Equal(t, int64(math.MaxInt64), page.Props["large"])
}

func TestV3WireFailureAndV2Isolation(t *testing.T) {
	t.Parallel()
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3), WithPreserveBigIntegers(true))
	state := NewState(t.Context(), RequestMeta{})
	page, err := engine.BuildPage(state, "Page", map[string]any{"bad": make(chan int)})
	require.NoError(t, err)
	_, err = MarshalPageWithState(page, state)
	require.Error(t, err)
	v2 := New("https://app.example", WithPreserveBigIntegers(true))
	require.Equal(t, ProtocolV2, v2.ProtocolVersion())
	state = NewState(t.Context(), RequestMeta{})
	page, err = v2.BuildPage(state, "Page", map[string]any{"large": int64(math.MaxInt64)})
	require.NoError(t, err)
	data, err := MarshalPageWithState(page, state)
	require.NoError(t, err)
	require.NotContains(t, string(data), "$bigint")
	require.NotContains(t, string(data), "preserveBigIntegers")
}

func TestV3ValidationArraysBagAndSession(t *testing.T) {
	t.Parallel()
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	state := NewState(t.Context(), RequestMeta{ErrorBag: "profile", PartialComponent: "Page", PartialExcept: "errors"})
	values := ValidationErrors{"name": {"required", "too short"}}
	engine.WithAllValidationErrors(state, values)
	engine.WithError(state, "email", "invalid")
	values["name"][0] = "caller mutated"
	saved := state.FlashToPersist()
	state = NewState(t.Context(), state.Meta)
	state.FlashData = saved
	page, err := engine.BuildPage(state, "Page", nil)
	require.NoError(t, err)
	data, err := MarshalPageWithState(page, state)
	require.NoError(t, err)
	require.Contains(t, string(data), `"name":["required","too short"]`)
	require.Contains(t, string(data), `"email":"invalid"`)
	require.Contains(t, string(data), `"errors":{"profile":`)
	require.NotContains(t, string(data), "caller mutated")
	legacy := New("https://app.example")
	state = NewState(t.Context(), RequestMeta{})
	legacy.WithAllValidationErrors(state, ValidationErrors{"name": {"first", "second"}})
	page, err = legacy.BuildPage(state, "Page", nil)
	require.NoError(t, err)
	require.Equal(t, map[string]string{"name": "first"}, page.Props["errors"])
}

func TestV3DirectArrayOmissionsStayAbsent(t *testing.T) {
	t.Parallel()
	props := map[string]any{"items": []any{Defer("later"), "visible", Optional("optional")}}
	page, wire := v3Page(t, RequestMeta{}, props)
	require.Equal(t, map[string]any{"1": "visible"}, page.Props["items"])
	require.Equal(t, []string{"items.0"}, page.DeferredProps["default"])
	data, err := json.Marshal(wire)
	require.NoError(t, err)
	require.NotContains(t, string(data), `"items":[null`)
	page, _ = v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "items.0"}, props)
	require.Equal(t, map[string]any{"0": "later"}, page.Props["items"])
	page, _ = v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "items"}, props)
	require.Equal(t, []any{"later", "visible", "optional"}, page.Props["items"])
	props["items"] = []any{Rescue(Defer(func(context.Context) (any, error) { return nil, errors.New("unavailable") })), "visible"}
	page, wire = v3Page(t, RequestMeta{PartialComponent: "Page", PartialOnly: "items.0"}, props)
	require.Equal(t, map[string]any{}, page.Props["items"])
	require.Equal(t, []any{"items.0"}, wire["rescuedProps"])
}

func TestV3WrappedContainerReferenceParity(t *testing.T) {
	t.Parallel()
	for _, wrapper := range []struct {
		name string
		wrap func(any) any
	}{
		{"once", func(value any) any { return Once(value) }},
		{"merge", func(value any) any { return Merge(value) }},
		{"optional", func(value any) any { return Optional(value) }},
		{"deferred", func(value any) any { return Defer(value) }},
		{"scroll", func(value any) any { return Scroll(value, ScrollPropConfig{}) }},
	} {
		t.Run(wrapper.name, func(t *testing.T) {
			t.Parallel()
			value := map[string]any{"name": "Ada", "sibling": "present"}
			provider := func(context.Context) (any, error) { return value, nil }
			meta := RequestMeta{PartialComponent: "Page", PartialOnly: "tree.name"}
			plain, _ := v3Page(t, meta, map[string]any{"tree": wrapper.wrap(value)})
			lazy, _ := v3Page(t, meta, map[string]any{"tree": wrapper.wrap(provider)})
			require.Equal(t, value, plain.Props["tree"])
			require.Equal(t, plain.Props, lazy.Props)
		})
	}
}

func TestV3RescueDoesNotSwallowIndependentChildFailure(t *testing.T) {
	t.Parallel()
	errChild := errors.New("independent child failed")
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "tree"})
	page, err := engine.BuildPage(state, "Page", map[string]any{"tree": Rescue(Defer(map[string]any{
		"a": Once("first"),
		"z": func(context.Context) (any, error) { return nil, errChild },
	}))})
	require.ErrorIs(t, err, errChild)
	require.Nil(t, page)
	require.Empty(t, state.v3.rescuedProps)
}

func TestV3ValidationHelpersComposeForPrecognition(t *testing.T) {
	t.Parallel()
	engine := New("https://app.example", WithProtocolVersion(ProtocolV3))
	for _, allFirst := range []bool{false, true} {
		state := NewState(t.Context(), RequestMeta{Precognition: "true", ValidateOnly: "name,email"})
		all := func() { engine.WithAllValidationErrors(state, ValidationErrors{"name": {"required", "short"}}) }
		single := func() { engine.WithValidationErrors(state, ValidationErrors{"email": {"invalid", "ignored"}}) }
		if allFirst {
			all()
			single()
		} else {
			single()
			all()
		}
		require.Equal(t, ValidationErrors{"name": {"required", "short"}, "email": {"invalid"}}, engine.PrecognitionErrors(state))
	}
}

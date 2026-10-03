package core

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
)

func TestNestedPartialSelection(t *testing.T) {
	t.Parallel()
	profile := map[string]any{"name": "Alice", "email": "alice@example.test", "nil": nil}
	for _, f := range []struct {
		name, only, except, component string
		want                          map[string]any
	}{
		{
			name: "only leaf",
			only: "auth.profile.name",
			want: map[string]any{"auth": map[string]any{"profile": map[string]any{"name": "Alice"}}},
		},
		{
			name: "only siblings",
			only: " auth.profile.name, auth.active, auth.profile.name ",
			want: map[string]any{"auth": map[string]any{"profile": map[string]any{"name": "Alice"}, "active": true}},
		},
		{
			name: "only absent",
			only: "auth.profile.absent,missing.child",
			want: map[string]any{},
		},
		{
			name: "only nil leaf",
			only: "auth.profile.nil",
			want: map[string]any{"auth": map[string]any{"profile": map[string]any{"nil": nil}}},
		},
		{
			name: "only nil child",
			only: "nil.child,auth.profile.nil.child",
			want: map[string]any{},
		},
		{
			name: "only parent overlap",
			only: "auth.profile.name,auth.profile",
			want: map[string]any{"auth": map[string]any{"profile": profile}},
		},
		{
			name:   "except leaf",
			except: "auth.profile.email",
			want: map[string]any{
				"auth":  map[string]any{"profile": map[string]any{"name": "Alice", "nil": nil}, "active": true},
				"other": "kept", "nil": nil,
			},
		},
		{
			name:   "except parent overlap",
			except: "auth.profile.name,auth.profile",
			want:   map[string]any{"auth": map[string]any{"active": true}, "other": "kept", "nil": nil},
		},
		{
			name:   "except wins",
			only:   "auth.profile.name",
			except: "auth.profile.email",
			want: map[string]any{
				"auth":  map[string]any{"profile": map[string]any{"name": "Alice", "nil": nil}, "active": true},
				"other": "kept", "nil": nil,
			},
		},
		{
			name:   "except absent",
			except: "auth.missing,other.child,nil.child",
			want:   map[string]any{"auth": map[string]any{"profile": profile, "active": true}, "other": "kept", "nil": nil},
		},
		{
			name:      "component mismatch",
			only:      "auth.profile.name",
			except:    "auth.profile.email",
			component: "Other",
			want:      map[string]any{"auth": map[string]any{"profile": profile, "active": true}, "other": "kept", "nil": nil},
		},
	} {
		t.Run(f.name, func(t *testing.T) {
			t.Parallel()
			component := f.component
			if component == "" {
				component = "Page"
			}
			state := NewState(t.Context(), RequestMeta{PartialComponent: component, PartialOnly: f.only, PartialExcept: f.except})
			page, err := New("https://app.example").BuildPage(state, "Page", map[string]any{
				"auth": map[string]any{"profile": profile, "active": true}, "other": "kept", "nil": nil,
			})
			if err != nil {
				t.Fatal(err)
			}
			delete(page.Props, ContextPropsErrors)
			if !reflect.DeepEqual(page.Props, f.want) {
				t.Fatalf("got %#v, want %#v", page.Props, f.want)
			}
		})
	}
	if len(profile) != 3 || profile["email"] != "alice@example.test" {
		t.Fatal("selection mutated caller map")
	}
}

func TestNestedPartialLiteralDottedKeys(t *testing.T) {
	t.Parallel()
	for _, except := range []bool{false, true} {
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "auth.user.name"})
		if except {
			state.Meta.PartialOnly = ""
			state.Meta.PartialExcept = "auth.user.name"
		}
		calls := make(map[string]int)
		lazy := func(key string) LazyProp {
			return LazyProp{Fn: func(context.Context) (any, error) { calls[key]++; return key, nil }}
		}
		page, err := New("https://app.example").BuildPage(state, "Page", map[string]any{
			"auth.user.name": lazy("root literal"),
			"auth.user":      map[string]any{"name": lazy("parent literal"), "other": "kept"},
			"auth": map[string]any{
				"user.name": lazy("nested literal"),
				"user":      map[string]any{"name": lazy("nested"), "other": "kept"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		delete(page.Props, ContextPropsErrors)
		want := map[string]any{
			"auth.user.name": "root literal",
			"auth.user":      map[string]any{"name": "parent literal"},
			"auth":           map[string]any{"user.name": "nested literal", "user": map[string]any{"name": "nested"}},
		}
		if except {
			want = map[string]any{
				"auth.user": map[string]any{"other": "kept"},
				"auth":      map[string]any{"user": map[string]any{"other": "kept"}},
			}
		}
		if !reflect.DeepEqual(page.Props, want) {
			t.Fatalf("except=%v: got %#v, want %#v", except, page.Props, want)
		}
		if except && len(calls) != 0 {
			t.Fatalf("excluded literal/nested callbacks ran: %v", calls)
		}
		wantCalls := map[string]int{"root literal": 1, "parent literal": 1, "nested literal": 1, "nested": 1}
		if !except && !reflect.DeepEqual(calls, wantCalls) {
			t.Fatalf("literal/nested cache paths must remain distinct: %v", calls)
		}
	}
}

func TestNestedPartialCallbacksAreFilteredBeforeResolution(t *testing.T) {
	t.Parallel()
	for _, only := range []bool{true, false} {
		calls := make(map[string]int)
		lazy := func(key string, value any) LazyProp {
			return LazyProp{Fn: func(context.Context) (any, error) { calls[key]++; return value, nil }}
		}
		broken := LazyProp{Fn: func(context.Context) (any, error) { calls["excluded"]++; return nil, errors.New("must not execute") }}
		parent := map[string]any{"name": lazy("name", "Alice"), "secret": broken}
		props := map[string]any{"auth": lazy("root", map[string]any{"user": lazy("user", parent), "ignored": broken})}
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "auth.user.name"})
		if !only {
			state.Meta.PartialOnly = ""
			state.Meta.PartialExcept = "auth.user.secret,auth.ignored"
		}
		inertia := New("https://app.example")
		for range 2 {
			page, err := inertia.BuildPage(state, "Page", props)
			if err != nil {
				t.Fatal(err)
			}
			want := map[string]any{"user": map[string]any{"name": "Alice"}}
			if !reflect.DeepEqual(page.Props["auth"], want) {
				t.Fatalf("only=%v: got %#v", only, page.Props)
			}
		}
		if !reflect.DeepEqual(calls, map[string]int{"root": 1, "user": 1, "name": 1}) {
			t.Fatalf("only=%v: skipped, duplicated or over-evaluated callbacks: %v", only, calls)
		}
		if _, ok := parent["name"].(LazyProp); !ok || len(parent) != 2 {
			t.Fatal("selection/resolution mutated caller/cache value")
		}
	}
}

func TestNestedPartialPlainCallbacksAndOpaqueValues(t *testing.T) {
	t.Parallel()
	type record struct{ Name string }
	calls := 0
	props := map[string]any{
		"plain": func(context.Context) (any, error) {
			calls++
			return map[string]any{
				"name": "Alice",
				"ignored": func(context.Context) (any, error) {
					t.Fatal("unselected callback ran")
					return nil, errors.New("unexpected callback")
				},
			}, nil
		},
		"typed": map[string]string{"name": "Alice"}, "record": record{Name: "Alice"},
		"array": []any{LazyProp{Fn: func(context.Context) (any, error) {
			t.Fatal("numeric array selection must not evaluate children")
			return nil, errors.New("unexpected callback")
		}}},
		"nilmap": map[string]any(nil),
	}
	state := NewState(t.Context(), RequestMeta{
		PartialComponent: "Page", PartialOnly: "plain.name,typed.name,record.Name,array.0,nilmap.name",
	})
	page, err := New("https://app.example").BuildPage(state, "Page", props)
	if err != nil {
		t.Fatal(err)
	}
	delete(page.Props, ContextPropsErrors)
	if !reflect.DeepEqual(page.Props, map[string]any{"plain": map[string]any{"name": "Alice"}}) || calls != 1 {
		t.Fatalf("opaque selection or callback result incorrect: %#v; calls=%d", page.Props, calls)
	}
	state.Meta.PartialOnly = ""
	state.Meta.PartialExcept = "typed.name,record.Name,nilmap.name"
	page, err = New("https://app.example").BuildPage(state, "Page", map[string]any{
		"typed": props["typed"], "record": props["record"], "nilmap": props["nilmap"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(page.Props["typed"], props["typed"]) ||
		!reflect.DeepEqual(page.Props["record"], props["record"]) ||
		!reflect.DeepEqual(page.Props["nilmap"], props["nilmap"]) {
		t.Fatal("except paths into unsupported values must leave them unchanged")
	}
}

func TestNestedPartialReservedAndAlwaysProps(t *testing.T) {
	t.Parallel()
	for _, except := range []string{"", "errors.name,flash.name,old.name,token.name,always.name"} {
		inertia := New("https://app.example")
		inertia.csrfEnabled, inertia.csrfPropName = true, "token"
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "other.name", PartialExcept: except})
		value := map[string]any{"name": "kept", "sibling": "kept"}
		page, err := inertia.BuildPage(state, "Page", map[string]any{
			"errors": value, "flash": value, "old": value, "token": value, "always": Always(value),
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"errors", "flash", "old", "token", "always"} {
			if !reflect.DeepEqual(page.Props[key], value) {
				t.Errorf("reserved/always prop %q was filtered: %#v", key, page.Props[key])
			}
		}
	}
}

func TestNestedPartialSharedContextRequestPrecedence(t *testing.T) {
	t.Parallel()
	calls := make(map[string]int)
	makeValue := func(source string) map[string]any {
		return map[string]any{
			"name": LazyProp{Fn: func(context.Context) (any, error) { calls[source]++; return source, nil }},
			"ignored": LazyProp{Fn: func(context.Context) (any, error) {
				t.Fatal("unselected sibling ran")
				return nil, errors.New("unexpected callback")
			}},
		}
	}
	inertia := New("https://app.example", WithSharedProps(map[string]any{
		"auth": makeValue("shared"), "shared": makeValue("unshadowed"),
	}))
	state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "auth.name,shared.name"})
	inertia.WithProp(state, "auth", makeValue("context"))
	props := map[string]any{"auth": makeValue("request")}
	for range 2 {
		page, err := inertia.BuildPage(state, "Page", props)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(page.Props["auth"], map[string]any{"name": "request"}) ||
			!reflect.DeepEqual(page.Props["shared"], map[string]any{"name": "unshadowed"}) {
			t.Fatalf("selected source precedence incorrect: %#v", page.Props)
		}
	}
	if !reflect.DeepEqual(calls, map[string]int{"context": 1, "request": 1, "unshadowed": 1}) {
		t.Fatalf("selected callbacks changed source ownership/cache behavior: %v", calls)
	}
}

func TestNestedPartialConcurrentMapOwnership(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	input := map[string]any{
		"selected": LazyProp{Fn: func(context.Context) (any, error) { calls.Add(1); return "value", nil }},
		"ignored":  "untouched",
	}
	inertia := New("https://app.example", WithSharedProps(map[string]any{"data": input}))
	var workers sync.WaitGroup
	for range 16 {
		workers.Go(func() {
			state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "data.selected"})
			page, err := inertia.BuildPage(state, "Page", nil)
			if err != nil {
				t.Error(err)
				return
			}
			data := nestedTestMap(t, page.Props["data"])
			data["selected"] = "local mutation"
		})
	}
	workers.Wait()
	if calls.Load() != 16 || input["ignored"] != "untouched" || len(input) != 2 {
		t.Fatalf("shared map ownership/callback count changed: %#v, calls=%d", input, calls.Load())
	}
	if _, ok := input["selected"].(LazyProp); !ok {
		t.Fatal("caller map mutated")
	}
}

func TestNestedPartialMissingRequestPathOverridesContext(t *testing.T) {
	t.Parallel()
	for _, request := range []any{nil, map[string]any{"other": "request"}, "request scalar"} {
		inertia := New("https://app.example", WithSharedProps(map[string]any{"auth": map[string]any{"name": "shared"}}))
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "auth.name"})
		inertia.WithProp(state, "auth", map[string]any{"name": "context"})
		page, err := inertia.BuildPage(state, "Page", map[string]any{"auth": request})
		if err != nil {
			t.Fatal(err)
		}
		if _, exists := page.Props["auth"]; exists {
			t.Fatalf("missing selected path in request must not restore context/shared value: %#v", page.Props)
		}
	}
}

func TestNestedPartialCachedMapsRemainUnfiltered(t *testing.T) {
	t.Parallel()
	calls := make(map[string]int)
	lazy := func(key string, value any) LazyProp {
		return LazyProp{Fn: func(context.Context) (any, error) { calls[key]++; return value, nil }}
	}
	input := map[string]any{"first": lazy("first", "one"), "second": lazy("second", "two")}
	props := map[string]any{"data": lazy("root", input)}
	state := NewState(t.Context(), RequestMeta{PartialComponent: "Page"})
	inertia := New("https://app.example")
	for _, key := range []string{"first", "second", "first"} {
		state.Meta.PartialOnly = "data." + key
		page, err := inertia.BuildPage(state, "Page", props)
		if err != nil {
			t.Fatal(err)
		}
		data := nestedTestMap(t, page.Props["data"])
		if len(data) != 1 || data[key] == nil {
			t.Fatalf("cached callback result was filtered/mutated: %#v", page.Props)
		}
	}
	if !reflect.DeepEqual(calls, map[string]int{"root": 1, "first": 1, "second": 1}) {
		t.Fatalf("bad cache counts across filtered builds: %v", calls)
	}
}

func TestNestedPartialMissingRequestPathRemovesContextMetadata(t *testing.T) {
	t.Parallel()
	value := map[string]any{"missing": []any{map[string]any{"id": 1}}}
	for _, wrapped := range []any{
		Merge(value), Prepend(value), DeepMerge(value), Scroll(value, ScrollPropConfig{}), Once(value),
	} {
		inertia := New("https://app.example")
		state := NewState(t.Context(), RequestMeta{PartialComponent: "Page", PartialOnly: "feed.missing"})
		inertia.WithProp(state, "feed", wrapped)
		page, err := inertia.BuildPage(state, "Page", map[string]any{"feed": map[string]any{"other": "request"}})
		if err != nil {
			t.Fatal(err)
		}
		if len(page.MergeProps)+len(page.PrependProps)+len(page.DeepMergeProps)+len(page.ScrollProps)+len(page.OnceProps) != 0 {
			t.Fatalf("missing request descendant retained lower-layer metadata: %+v", page)
		}
	}
}

func TestOmitUnselectedPropPaths(t *testing.T) {
	t.Parallel()
	page := &PageDTO{
		Props:          map[string]any{"feed": "value", "other": "kept"},
		MergeProps:     []string{"feed", "feed.items", "feedback.items"},
		PrependProps:   []string{"feed.items", "other.items"},
		DeepMergeProps: []string{"feed", "other"},
		MatchPropsOn:   []string{"feed.items.id", "feedback.items.id"},
		ScrollProps:    map[string]ScrollPropConfig{"feed": {}, "other": {}},
		OnceProps:      map[string]OncePropConfig{"feed-cache": {Prop: "feed"}, "other-cache": {Prop: "other"}},
	}
	state := NewState(t.Context(), RequestMeta{})
	appendPropMetadata(state, page, "feed", "feed", appendMetadata)
	appendPropMetadata(state, page, "feed", "feed.items", appendMetadata)
	appendPropMetadata(state, page, "feed", "feed.items", prependMetadata)
	appendPropMetadata(state, page, "feed", "feed", deepMetadata)
	appendPropMetadata(state, page, "feed", "feed.items.id", matchMetadata)
	omitUnselectedProp(state, page, "feed")
	if len(page.Props) != 1 || len(page.ScrollProps) != 1 || len(page.OnceProps) != 1 ||
		!reflect.DeepEqual(page.MergeProps, []string{"feedback.items"}) ||
		!reflect.DeepEqual(page.PrependProps, []string{"other.items"}) ||
		!reflect.DeepEqual(page.DeepMergeProps, []string{"other"}) ||
		!reflect.DeepEqual(page.MatchPropsOn, []string{"feedback.items.id"}) {
		t.Fatalf("missing descendant cleanup removed unrelated paths or kept target metadata: %+v", page)
	}
}

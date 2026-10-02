package core

import (
	"context"
	"testing"
)

func TestNestedLazyPathsDoNotAliasLiteralKeys(t *testing.T) {
	i := New("https://app.example")
	c := NewState(context.Background(), RequestMeta{URL: "/page"})
	calls := 0
	lazy := func(value string) LazyProp {
		return LazyProp{Fn: func(context.Context) (any, error) {
			calls++
			return value, nil
		}}
	}
	p, err := i.BuildPage(c, "Page", map[string]any{
		"auth":      map[string]any{"user": lazy("nested")},
		"auth.user": lazy("literal"),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("props=%v calls=%d", p.Props, calls)
	if nestedTestMap(t, p.Props["auth"])["user"] != "nested" || p.Props["auth.user"] != "literal" || calls != 2 {
		t.Fatal("distinct nested and literal JSON keys share the request lazy cache")
	}
}

func TestNestedLazyRequestOverridesContext(t *testing.T) {
	i := New("https://app.example")
	c := NewState(context.Background(), RequestMeta{URL: "/page"})
	i.WithProp(c, "auth", map[string]any{"user": LazyProp{Fn: func(context.Context) (any, error) {
		return "context-user", nil
	}}})
	p, err := i.BuildPage(c, "Page", map[string]any{
		"auth": map[string]any{"user": LazyProp{Fn: func(context.Context) (any, error) {
			return "request-user", nil
		}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("props=%v", p.Props)
	if nestedTestMap(t, p.Props["auth"])["user"] != "request-user" {
		t.Fatal("nested context lazy result incorrectly shadows request override")
	}
}

func TestNestedLazyCacheEncodingAndLayers(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	s := NewState(t.Context(), RequestMeta{URL: "/page"})
	calls := make(map[string]int)
	lazy := func(value string) LazyProp {
		return LazyProp{Fn: func(context.Context) (any, error) {
			calls[value]++
			return value, nil
		}}
	}
	s.FlashData = map[string]any{ContextPropsOld: map[string]any{"user": lazy("flash")}}
	i.WithProp(s, ContextPropsOld, map[string]any{"user": lazy("context")})
	props := map[string]any{
		ContextPropsOld: map[string]any{"user": lazy("request")},
		"auth": map[string]any{
			"user.name": lazy("literal nested"),
			"user":      map[string]any{"name": lazy("deep")},
			"":          lazy("empty"),
			"m0:":       lazy("separator"),
		},
		"items": []any{lazy("index"), map[string]any{"0": lazy("numeric key")}},
	}
	for range 2 {
		page, err := i.BuildPage(s, "Page", props)
		if err != nil {
			t.Fatal(err)
		}
		if nestedTestMap(t, page.Props[ContextPropsOld])["user"] != "request" {
			t.Fatal("request must override context and flash")
		}
		auth := nestedTestMap(t, page.Props["auth"])
		if auth["user.name"] != "literal nested" || nestedTestMap(t, auth["user"])["name"] != "deep" ||
			auth[""] != "empty" || auth["m0:"] != "separator" {
			t.Fatalf("encoded paths aliased: %v", auth)
		}
		items := nestedTestSlice(t, page.Props["items"])
		if items[0] != "index" || nestedTestMap(t, items[1])["0"] != "numeric key" {
			t.Fatal("array index and map key must stay distinct")
		}
	}
	if len(calls) != 9 {
		t.Fatalf("missing callbacks: %v", calls)
	}
	for key, count := range calls {
		if count != 1 {
			t.Errorf("%q evaluated %d times", key, count)
		}
	}
	if _, ok := nestedTestMap(t, props["auth"])["user.name"].(LazyProp); !ok {
		t.Fatal("caller container mutated")
	}
}

func nestedTestMap(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", value)
	}
	return result
}

func nestedTestSlice(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("expected slice, got %T", value)
	}
	return result
}

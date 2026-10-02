package core

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

func TestVaryTokenPolicy(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct{ input, want string }{
		{"", "X-Inertia"},
		{"Accept-Encoding", "Accept-Encoding, X-Inertia"},
		{"Accept-Encoding, x-INERTIA ", "Accept-Encoding, x-INERTIA "},
		{"*", "*"},
		{"Accept-Encoding, *", "Accept-Encoding, *"},
	} {
		if got := VaryValue(fixture.input, HeaderInertia); got != fixture.want {
			t.Errorf("VaryValue(%q): got %q, want %q", fixture.input, got, fixture.want)
		}
	}
}

func TestNestedLazyOwnershipAndRequestCache(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	var calls atomic.Int64
	value := LazyProp{Fn: func(context.Context) (any, error) { calls.Add(1); return "Alice", nil }}
	shared := map[string]any{"user": value, "items": []any{map[string]any{"owner": value}}}
	const requests = 32
	var workers sync.WaitGroup
	for range requests {
		workers.Go(func() {
			s := NewState(t.Context(), RequestMeta{Method: "GET", URL: "/"})
			for range 2 {
				page, err := i.BuildPage(s, "Page", map[string]any{"auth": shared})
				if err != nil {
					t.Error(err)
					return
				}
				auth, ok := page.Props["auth"].(map[string]any)
				if !ok || auth["user"] != "Alice" {
					t.Error("nested callback did not resolve")
				}
			}
		})
	}
	workers.Wait()
	if calls.Load() != 2*requests {
		t.Errorf("nested lazy calls=%d, expected %d", calls.Load(), 2*requests)
	}
	if _, ok := shared["user"].(LazyProp); !ok {
		t.Fatal("shared input was mutated")
	}
	if _, ok := shared["items"].([]any)[0].(map[string]any)["owner"].(LazyProp); !ok {
		t.Fatal("nested shared slice was mutated")
	}
}

func TestFailedWrappedPropsDoNotPublishMergeMetadata(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	failed := LazyProp{Fn: func(context.Context) (any, error) { return nil, errors.New("fixture failure") }}
	page, err := i.BuildPage(NewState(t.Context(), RequestMeta{Method: "GET", URL: "/"}), "Test", map[string]any{
		"merge": Merge(failed), "prepend": Prepend(failed), "deep": DeepMerge(failed),
		"scroll": Scroll(failed, ScrollPropConfig{PageName: "page"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.MergeProps)+len(page.PrependProps)+len(page.DeepMergeProps)+len(page.ScrollProps) != 0 {
		t.Fatalf("metadata for unresolved props: %+v", page)
	}
}

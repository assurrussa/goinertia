package core

import (
	"context"
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/assurrussa/goinertia/views"
)

type scrollResetSSR struct{ payload []byte }

func (*scrollResetSSR) Reset() {}
func (c *scrollResetSSR) Post(_ context.Context, _ string, payload []byte, _ map[string]string) (int, []byte, error) {
	c.payload = append([]byte(nil), payload...)
	return 200, []byte(`{"body":"<p>SSR</p>","head":[]}`), nil
}

func TestScrollResetHTMLAndSSROwnership(t *testing.T) {
	t.Parallel()
	client := new(scrollResetSSR)
	i := New("https://app.example", WithFS(views.Templates), WithSSRConfig(SSRConfig{
		URL: "http://fixture.invalid/render", SSRClient: client, DisableRetries: true,
	}))
	s := NewState(t.Context(), RequestMeta{Reset: "posts", URL: "/page"})
	page, err := i.BuildPage(s, "Page", map[string]any{
		"posts": Scroll(map[string]any{"data": []int{2}}, ScrollPropConfig{"page", nil, 3, 2}),
	})
	if err != nil {
		t.Fatal(err)
	}
	body, err := i.RenderHTML(s, page)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html.UnescapeString(string(body)), `"reset":true`) ||
		!strings.Contains(string(client.payload), `"reset":true`) {
		t.Fatal("HTML bootstrap and SSR must receive request-only reset metadata")
	}
	plain, err := json.Marshal(page)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), `"reset"`) {
		t.Fatal("wire serialization mutated the public DTO")
	}
}

func TestScrollResetCoreWireSerialization(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	s := NewState(t.Context(), RequestMeta{Reset: "posts"})
	page, err := i.BuildPage(s, "Page", map[string]any{
		"posts": Scroll([]int{2}, ScrollPropConfig{"page", nil, 2, 1}),
	})
	if err != nil {
		t.Fatal(err)
	}
	wire, err := MarshalPage(page, s.Meta.Reset)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(wire), `"reset":true`) {
		t.Fatal("core wire serializer omitted reset")
	}
}

package core

import (
	"encoding/json"
	"html"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/views"
)

func TestNativeFlashWireSerialization(t *testing.T) {
	t.Parallel()
	for _, reset := range []string{"", "posts", "other"} {
		t.Run(reset, func(t *testing.T) {
			t.Parallel()
			i := New("https://app.example")
			s := NewState(t.Context(), RequestMeta{Reset: reset, PartialComponent: "Page", PartialOnly: "posts"})
			s.FlashData = map[string]any{nativeFlashSessionKey: map[string]any{"message": "stored", "retained": true}}
			i.WithNativeFlash(s, "message", "first")
			i.WithNativeFlash(s, "message", "current")
			i.WithNativeFlash(s, "null", nil)
			i.WithNativeFlash(s, "nested", map[string]any{"list": []any{1, "two"}})
			i.WithFlashSuccess(s, "legacy")
			page, err := i.BuildPage(s, "Page", map[string]any{
				"posts": Scroll([]int{2}, ScrollPropConfig{"page", nil, 2, 1}), "excluded": true,
			})
			require.NoError(t, err)
			wire, err := MarshalPageWithState(page, s)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(wire, &got))
			require.Equal(t, map[string]any{
				"message": "current", "retained": true, "null": nil,
				"nested": map[string]any{"list": []any{float64(1), "two"}},
			}, got["flash"])
			require.NotContains(t, page.Props, nativeFlashSessionKey)
			require.NotContains(t, page.Props, "excluded")
			require.Equal(t, map[string]string{"success": "legacy"}, page.Props["flash"])
			require.Equal(t, map[string]any{"message": "stored", "retained": true}, s.FlashData[nativeFlashSessionKey])
			require.Equal(t, reset == "posts", strings.Contains(string(wire), `"reset":true`))
			plain, err := json.Marshal(page)
			require.NoError(t, err)
			require.NotContains(t, string(plain), `"message"`)
			require.NotContains(t, string(plain), `"reset"`)
			legacyWire, err := MarshalPage(page, reset)
			require.NoError(t, err)
			require.NotContains(t, string(legacyWire), `"message"`)
		})
	}
}

func TestNativeFlashHTMLAndSSR(t *testing.T) {
	t.Parallel()
	client := new(scrollResetSSR)
	i := New("https://app.example", WithFS(views.Templates), WithSSRConfig(SSRConfig{
		URL: "http://fixture.invalid/render", SSRClient: client, DisableRetries: true,
	}))
	s := NewState(t.Context(), RequestMeta{Reset: "posts", URL: "/page"})
	i.WithNativeFlash(s, "message", `Saved <script>alert("test")</script>`)
	i.WithFlashSuccess(s, "legacy")
	page, err := i.BuildPage(s, "Page", map[string]any{"posts": Scroll([]int{2}, ScrollPropConfig{"page", nil, 3, 2})})
	require.NoError(t, err)
	body, err := i.RenderHTML(s, page)
	require.NoError(t, err)
	require.NotContains(t, string(body), `<script>alert("test")</script>`)
	var ssr map[string]any
	require.NoError(t, json.Unmarshal(client.payload, &ssr))
	require.Equal(t, map[string]any{"message": `Saved <script>alert("test")</script>`}, ssr["flash"])
	wire, err := MarshalPageWithState(page, s)
	require.NoError(t, err)
	require.JSONEq(t, string(wire), string(client.payload))
	require.Contains(t, html.UnescapeString(string(body)), string(wire))
}

func TestNativeFlashAbsentAndPrecognition(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	s := NewState(t.Context(), RequestMeta{})
	page, err := i.BuildPage(s, "Page", nil)
	require.NoError(t, err)
	for _, state := range []*State{nil, s} {
		wire, marshalErr := MarshalPageWithState(page, state)
		require.NoError(t, marshalErr)
		require.NotContains(t, string(wire), `"flash"`)
	}
	i.WithNativeFlash(s, "message", "not for validation")
	s.Meta.Precognition = "true"
	wire, err := MarshalPageWithState(page, s)
	require.NoError(t, err)
	require.NotContains(t, string(wire), `"flash"`)
	wire, err = MarshalPageWithState(nil, s)
	require.NoError(t, err)
	require.Equal(t, "null", string(wire))
}

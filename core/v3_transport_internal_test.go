package core

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/views"
)

func TestV3BootstrapEscapesScriptPayload(t *testing.T) {
	t.Parallel()
	value := "</script><script>alert('x')</script><!-- & > \u2028\u2029 Привет 世界"
	i := New("https://app.example", WithProtocolVersion(ProtocolV3), WithFS(views.Templates))
	s := NewState(t.Context(), RequestMeta{URL: "/hello"})
	page, err := i.BuildPage(s, "Test", map[string]any{"value": value})
	require.NoError(t, err)
	body, err := i.RenderHTML(s, page)
	require.NoError(t, err)
	markup := string(body)
	require.Contains(t, markup, `<script data-page="app" type="application/json">`)
	require.Contains(t, markup, `<div id="app"></div>`)
	require.NotContains(t, markup, `<div id="app" data-page=`)
	require.NotContains(t, markup, `data-server-rendered="true"`)
	_, rest, ok := strings.Cut(markup, `<script data-page="app" type="application/json">`)
	require.True(t, ok)
	payload, _, ok := strings.Cut(rest, "</script>")
	require.True(t, ok)
	require.NotContains(t, payload, "<")
	require.NotContains(t, payload, "\u2028")
	require.NotContains(t, payload, "\u2029")
	require.Contains(t, payload, `\u003c\/script\u003e`)
	require.Contains(t, payload, `\u003c!--`)
	var decoded PageDTO
	require.NoError(t, json.Unmarshal([]byte(payload), &decoded))
	require.Equal(t, value, decoded.Props["value"])
}

func TestBootstrapV2AndMarshalError(t *testing.T) {
	t.Parallel()
	body, err := Bootstrap(map[string]any{"text": `" onmouseover="bad<`}, ProtocolV2)
	require.NoError(t, err)
	require.Contains(t, string(body), `data-page="{&#34;text&#34;:`)
	require.NotContains(t, string(body), `<script`)
	require.NotContains(t, string(body), ` onmouseover="bad`)
	_, err = Bootstrap(map[string]any{"invalid": make(chan int)}, ProtocolV3)
	require.ErrorContains(t, err, "error marshaling bootstrap page")
}

func TestV3PreserveFragmentRequestLocal(t *testing.T) {
	t.Parallel()
	for _, version := range []ProtocolVersion{ProtocolV2, ProtocolV3} {
		i := New("https://app.example", WithProtocolVersion(version))
		for _, preserve := range []bool{true, false} {
			s := NewState(t.Context(), RequestMeta{})
			i.WithPreserveFragment(s, preserve)
			page, err := i.BuildPage(s, "Test", nil)
			require.NoError(t, err)
			wire, err := MarshalPageWithState(page, s)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(wire, &got))
			if version == ProtocolV3 && preserve {
				require.Equal(t, true, got["preserveFragment"])
			} else {
				require.NotContains(t, got, "preserveFragment")
			}
		}
	}
}

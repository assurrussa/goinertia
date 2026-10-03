package core

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/views"
)

type v3SSRClient func(context.Context, string, []byte, map[string]string) (int, []byte, error)

func (v3SSRClient) Reset() {}
func (f v3SSRClient) Post(ctx context.Context, url string, body []byte, headers map[string]string) (int, []byte, error) {
	return f(ctx, url, body, headers)
}

func TestV3SSRFailurePolicyAndReporting(t *testing.T) {
	t.Parallel()
	fixtures := []struct {
		name   string
		status int
		body   string
		err    error
	}{
		{name: "network", err: errors.New("connection refused")},
		{name: "http", status: 500, body: `{"error":"secret SSR diagnostic","type":"browser-api","hint":"use onMounted",` +
			`"browserApi":"window","stack":"private stack","sourceLocation":{"file":"private.ts","line":3}}`},
		{name: "redirect", status: 302, body: `{"head":[],"body":"unexpected"}`},
		{name: "malformed", status: 200, body: `not JSON`},
		{name: "null", status: 200, body: `null`},
		{name: "missing-body", status: 200, body: `{}`},
	}
	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			t.Parallel()
			var reported error
			calls := 0
			client := v3SSRClient(func(context.Context, string, []byte, map[string]string) (int, []byte, error) {
				return fixture.status, []byte(fixture.body), fixture.err
			})
			i := New("https://app.example", WithFS(views.Templates), WithProtocolVersion(ProtocolV3),
				WithSSRConfig(SSRConfig{URL: "http://ssr.example/render", SSRClient: client, DisableRetries: true}),
				WithSSRFailureHandler(func(_ context.Context, err error) { calls++; reported = err }))
			s := NewState(t.Context(), RequestMeta{})
			page, err := i.BuildPage(s, "Test", map[string]any{"safe": "hello"})
			require.NoError(t, err)
			body, err := i.RenderHTML(s, page)
			require.NoError(t, err)
			require.Equal(t, 1, calls)
			require.Error(t, reported)
			require.Contains(t, string(body), `<script data-page="app" type="application/json">`)
			require.NotContains(t, string(body), `data-server-rendered`)
			require.NotContains(t, string(body), "secret SSR diagnostic")
			if fixture.name == "http" {
				var detail *SSRResponseError
				require.ErrorAs(t, reported, &detail)
				require.ErrorIs(t, detail, ErrBadSsrStatusCode)
				require.Equal(t, 500, detail.Status)
				require.Equal(t, "browser-api", detail.Type)
				require.Equal(t, "window", detail.BrowserAPI)
				require.Contains(t, string(detail.SourceLocation), "private.ts")
			}
		})
	}
}

func TestSSRPolicyCompatibilityAndRetryReportOnce(t *testing.T) {
	t.Parallel()
	for _, fixture := range []struct {
		version ProtocolVersion
		policy  SSRErrorPolicy
		fails   bool
	}{
		{ProtocolV2, SSRErrorDefault, true},
		{ProtocolV2, SSRErrorFallback, false},
		{ProtocolV3, SSRErrorDefault, false},
		{ProtocolV3, SSRErrorPropagate, true},
	} {
		var requests, reports int
		client := v3SSRClient(func(context.Context, string, []byte, map[string]string) (int, []byte, error) {
			requests++
			return 503, nil, nil
		})
		i := New("https://app.example", WithFS(views.Templates), WithProtocolVersion(fixture.version),
			WithSSRErrorPolicy(fixture.policy),
			WithSSRConfig(SSRConfig{URL: "http://ssr.example/render", SSRClient: client, MaxRetries: 2, RetryDelay: time.Nanosecond}),
			WithSSRFailureHandler(func(context.Context, error) { reports++ }))
		s := NewState(t.Context(), RequestMeta{})
		page, err := i.BuildPage(s, "Test", nil)
		require.NoError(t, err)
		_, err = i.RenderHTML(s, page)
		if fixture.fails {
			require.ErrorIs(t, err, ErrBadSsrStatusCode)
		} else {
			require.NoError(t, err)
		}
		require.Equal(t, 3, requests)
		require.Equal(t, 1, reports)
	}
}

func TestV3SSRDisableIsRequestLocal(t *testing.T) {
	t.Parallel()
	var requests atomic.Int32
	ssrBody := `<script data-page="app" type="application/json">{}</script>` +
		`<div data-server-rendered="true" id="app"><p>Rendered</p></div>`
	client := v3SSRClient(func(context.Context, string, []byte, map[string]string) (int, []byte, error) {
		requests.Add(1)
		body, err := json.Marshal(SsrDTO{Head: []string{`<title data-inertia="">SSR title</title>`}, Body: ssrBody})
		return 200, body, err
	})
	i := New("https://app.example", WithFS(views.Templates), WithProtocolVersion(ProtocolV3),
		WithSSRConfig(SSRConfig{URL: "http://ssr.example/render", SSRClient: client, DisableRetries: true}))
	for n := range 12 {
		t.Run(string(rune('a'+n)), func(t *testing.T) {
			t.Parallel()
			s := NewState(t.Context(), RequestMeta{})
			disabled := n%2 == 0
			i.WithSSRDisabled(s, disabled)
			page, err := i.BuildPage(s, "Test", nil)
			require.NoError(t, err)
			body, err := i.RenderHTML(s, page)
			require.NoError(t, err)
			require.Equal(t, !disabled, strings.Contains(string(body), `data-server-rendered="true"`))
			if !disabled {
				require.Contains(t, string(body), `<title data-inertia="">SSR title</title>`)
				require.Equal(t, 1, strings.Count(string(body), `<script data-page="app"`))
			}
		})
	}
	t.Cleanup(func() { require.EqualValues(t, 6, requests.Load()) })
}

func TestV3ViteSSRUsesHotEndpointAndBypassesCache(t *testing.T) {
	t.Parallel()
	var development, production atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/__inertia_ssr":
			development.Add(1)
		case "/render":
			production.Add(1)
		default:
			http.NotFound(w, r)
			return
		}
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "fixture", r.Header.Get("X-SSR-Test"))
		payload, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Contains(t, string(payload), `"component":"Test"`)
		_, err = w.Write([]byte(`{"head":["<title data-inertia=\"\">SSR</title>"],` +
			`"body":"<div data-server-rendered=\"true\" id=\"app\">SSR</div>"}`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	for _, enabled := range []bool{true, false} {
		i := New("https://app.example", WithFS(views.Templates), WithProtocolVersion(ProtocolV3), WithDevMode(),
			WithPublicFS(fstest.MapFS{"hot": {Data: []byte(server.URL + "/\n")}}), WithViteSSR(enabled),
			WithSSRConfig(SSRConfig{
				URL: server.URL + "/render", Timeout: time.Second, CacheTTL: time.Minute,
				Headers: map[string]string{"X-SSR-Test": "fixture"},
			}))
		for range 2 {
			s := NewState(t.Context(), RequestMeta{})
			page, err := i.BuildPage(s, "Test", nil)
			require.NoError(t, err)
			body, err := i.RenderHTML(s, page)
			require.NoError(t, err)
			require.Contains(t, string(body), `data-server-rendered="true"`)
		}
	}
	require.EqualValues(t, 2, development.Load())
	require.EqualValues(t, 1, production.Load())
}

func TestV3SSRCanceledRequestDoesNotFallback(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	client := v3SSRClient(func(ctx context.Context, _ string, _ []byte, _ map[string]string) (int, []byte, error) {
		cancel()
		return 0, nil, ctx.Err()
	})
	reports := 0
	i := New("https://app.example", WithFS(views.Templates), WithProtocolVersion(ProtocolV3),
		WithSSRConfig(SSRConfig{URL: "http://ssr.example/render", SSRClient: client, DisableRetries: true}),
		WithSSRFailureHandler(func(context.Context, error) { reports++ }))
	state := NewState(ctx, RequestMeta{})
	page, err := i.BuildPage(state, "Test", nil)
	require.NoError(t, err)
	body, err := i.RenderHTML(state, page)
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, body)
	require.Equal(t, 1, reports)
}

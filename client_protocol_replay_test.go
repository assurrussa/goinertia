package goinertia_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

type replayFixture struct {
	Adapter  string                     `json:"adapter"`
	Initial  json.RawMessage            `json:"initial"`
	Deferred map[string]json.RawMessage `json:"deferred"`
	Excluded json.RawMessage            `json:"excluded"`
	Append   json.RawMessage            `json:"append"`
	Prepend  json.RawMessage            `json:"prepend"`
	Reset    json.RawMessage            `json:"reset"`
}

func TestPinnedClientProtocolReplay(t *testing.T) {
	if os.Getenv("GOINERTIA_CLIENT_REPLAY") != "1" {
		t.Skip("explicit client replay gate; requires Node 24")
	}
	node, err := exec.LookPath("node")
	require.NoError(t, err)
	fixtures := make([]replayFixture, 0, 3)
	for _, adapter := range []string{"legacy", "fiber", "http"} {
		fixture := replayFixture{Adapter: adapter, Deferred: make(map[string]json.RawMessage)}
		fixture.Initial = renderReplayPage(t, adapter, nil, 1)
		for _, prop := range []string{"a", "b"} {
			fixture.Deferred[prop] = renderReplayPage(t, adapter, map[string]string{core.HeaderPartialOnly: prop}, 1)
		}
		fixture.Excluded = renderReplayPage(t, adapter, map[string]string{core.HeaderPartialOnly: "title"}, 2)
		fixture.Append = renderReplayPage(t, adapter, map[string]string{core.HeaderPartialOnly: "posts"}, 2)
		fixture.Prepend = renderReplayPage(t, adapter, map[string]string{
			core.HeaderPartialOnly: "posts", core.HeaderInfiniteScrollMergeIntent: "prepend",
		}, 2)
		fixture.Reset = renderReplayPage(t, adapter, map[string]string{
			core.HeaderPartialOnly: "posts", core.HeaderReset: "posts",
		}, 2)
		fixtures = append(fixtures, fixture)
	}
	data, err := json.Marshal(fixtures)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "pages.json")
	require.NoError(t, os.WriteFile(path, data, 0o600))
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, node, "integration/protocol-replay/replay.ts", path).CombinedOutput()
	require.NoError(t, err, "%s", output)
	t.Logf("%s", output)
}

func renderReplayPage(t *testing.T, adapter string, headers map[string]string, page int) json.RawMessage {
	t.Helper()
	props := map[string]any{
		"title": "new", "profile": core.DeepMerge(map[string]any{"name": "Alice"}),
		"a": core.Defer("A", "first"), "b": core.Defer("B", "second"),
		"posts": core.Scroll(map[string]any{"data": []int{page}}, core.ScrollPropConfig{
			PageName: "page", CurrentPage: page, NextPage: page + 1,
		}),
	}
	r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "https://app.example/page", nil)
	r.Header.Set(core.HeaderInertia, "true")
	for key, value := range headers {
		r.Header.Set(key, value)
	}
	if len(headers) > 0 {
		r.Header.Set(core.HeaderPartialComponent, "Page")
	}
	var response *http.Response
	if adapter == "http" {
		i := nethttp.New("https://app.example")
		h := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
			return i.Render(w, r, "Page", props)
		}))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		response = w.Result()
	} else {
		var i *fiberadapter.Inertia
		if adapter == "legacy" {
			i = goinertia.New("https://app.example")
		} else {
			i = fiberadapter.New("https://app.example")
		}
		app := fiber.New()
		app.Use(i.Middleware())
		app.Get("/page", func(c fiber.Ctx) error { return i.Render(c, "Page", props) })
		var err error
		response, err = app.Test(r)
		require.NoError(t, err)
	}
	defer response.Body.Close()
	require.Equal(t, http.StatusOK, response.StatusCode)
	data, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.True(t, json.Valid(data))
	return json.RawMessage(data)
}

package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

func TestConcurrentNativeRequests(t *testing.T) {
	handler, err := New()
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(handler)
	defer server.Close()
	const requests = 32
	failed := make(chan error, requests)
	var workers sync.WaitGroup
	for index := range requests {
		workers.Go(func() {
			path := fmt.Sprintf("/request/%d", index)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+path, nil)
			if err != nil {
				failed <- err
				return
			}
			req.Header.Set(core.HeaderInertia, "true")
			req.Header.Set(core.HeaderVersion, "v1")
			response, err := server.Client().Do(req)
			if err != nil {
				failed <- err
				return
			}
			defer response.Body.Close()
			var page core.PageDTO
			if err := json.NewDecoder(response.Body).Decode(&page); err != nil {
				failed <- err
				return
			}
			if response.StatusCode != http.StatusOK || page.Props["native"] != path || page.Props["path"] != path || page.URL != path {
				failed <- fmt.Errorf("request %s leaked state: status=%d page=%+v", path, response.StatusCode, page)
			}
		})
	}
	workers.Wait()
	close(failed)
	for err := range failed {
		t.Error(err)
	}
}

func TestClientCancellationReachesSSR(t *testing.T) {
	started := make(chan struct{})
	stopped := make(chan struct{})
	ssr := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		close(started)
		<-r.Context().Done()
		close(stopped)
	}))
	defer ssr.Close()
	i := nethttp.New("https://app.example", nethttp.WithCoreOptions(core.WithFS(views.Templates),
		core.WithSSRConfig(core.SSRConfig{URL: ssr.URL, DisableRetries: true})))
	result := make(chan error, 1)
	inspected := make(chan bool, 1)
	app := httptest.NewServer(i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(context.WithValue(r.Context(), authenticatedKey{}, "authenticated"))
		result <- i.Render(w, r, "Consumer", map[string]any{
			"auth": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
				request, ok := nethttp.Request(ctx)
				inspected <- ok && request == r && ctx.Value(authenticatedKey{}) == "authenticated"
				return "authenticated", nil
			}},
		})
	})))
	defer app.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, app.URL+"/cancel", nil)
	if err != nil {
		t.Fatal(err)
	}
	clientResult := make(chan error, 1)
	go func() {
		response, err := app.Client().Do(request)
		if response != nil {
			_ = response.Body.Close()
		}
		clientResult <- err
	}()
	awaitSignal(t, started)
	if !<-inspected {
		t.Error("downstream context/helper lost during actual HTTP request")
	}
	cancel()
	for _, completion := range []<-chan error{clientResult, result} {
		select {
		case err := <-completion:
			if !errors.Is(err, context.Canceled) {
				t.Errorf("expected cancellation, got %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("cancellation did not finish")
		}
	}
	awaitSignal(t, stopped)
}

type authenticatedKey struct{}

func awaitSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(5 * time.Second):
		t.Fatal("local HTTP lifecycle did not finish")
	}
}

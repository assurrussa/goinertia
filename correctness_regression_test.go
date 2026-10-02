package goinertia

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/assurrussa/goinertia/inertiat/fibert"
	"github.com/assurrussa/goinertia/views"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"
)

func TestSSRDefaultOwnedBodyAndCancellation(t *testing.T) {
	t.Parallel()
	var sequence atomic.Int64
	started := make(chan struct{}, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/cancel" {
			started <- struct{}{}
			<-r.Context().Done()
			return
		}
		_, _ = fmt.Fprintf(w, `{"body":"%d","head":[]}`, sequence.Add(1))
	}))
	defer server.Close()
	client := &defaultSSRClient{client: &http.Client{}}
	_, owned, err := client.Post(t.Context(), server.URL, nil, nil)
	require.NoError(t, err)
	saved := string(owned)
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { _, _, _ = client.Post(t.Context(), server.URL, nil, nil) })
	}
	wg.Wait()
	require.Equal(t, saved, string(owned))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { _, _, err := client.Post(ctx, server.URL+"/cancel", nil, nil); done <- err }()
	<-started
	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("SSR cancellation did not interrupt transport")
	}
}

func TestSSRLocalRetryParallel(t *testing.T) {
	t.Parallel()
	var attempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1)%2 == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, `{"body":"owned","head":[]}`)
	}))
	defer server.Close()
	inr := New("http://localhost", WithSSRConfig(SSRConfig{URL: server.URL, MaxRetries: 20, RetryDelay: time.Microsecond}))
	var wg sync.WaitGroup
	failures := make(chan error, 16)
	for range 16 {
		wg.Go(func() {
			c := fibert.Default()
			_, err := inr.processSSR(c, &PageDTO{Component: "Test"})
			failures <- err
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
}

func TestColdTemplatesConcurrent(t *testing.T) {
	t.Parallel()
	inr := New("http://localhost", WithFS(views.Templates))
	var wg sync.WaitGroup
	failures := make(chan error, 64)
	for range 32 {
		wg.Go(func() {
			_, err := inr.createRootTemplate()
			failures <- err
			_, err = inr.createRootErrorTemplate()
			failures <- err
		})
	}
	wg.Wait()
	close(failures)
	for err := range failures {
		require.NoError(t, err)
	}
}

func TestCheckerOnlyCSRFAndVary(t *testing.T) {
	t.Parallel()
	blocked := errors.New("blocked")
	var checked atomic.Int64
	inr := New("http://localhost", WithAssetVersion("v1"), WithCSRFTokenCheckProvider(func(_ fiber.Ctx) error { checked.Add(1); return blocked }))
	app := fiber.New()
	app.Use(inr.Middleware())
	app.Post("/", func(c fiber.Ctx) error { return c.SendString("should not run") })
	app.Get("/", func(c fiber.Ctx) error { return c.SendString("ok") })
	resp, err := app.Test(httptest.NewRequest(http.MethodPost, "/", nil))
	require.NoError(t, err)
	require.Equal(t, int64(1), checked.Load())
	_ = resp.Body.Close()
	for _, version := range []string{"", "v1"} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		if version != "" {
			req.Header.Set(HeaderInertia, "true")
			req.Header.Set(HeaderVersion, "old")
		}
		resp, err = app.Test(req)
		require.NoError(t, err)
		require.Contains(t, resp.Header.Get("Vary"), HeaderInertia)
		_ = resp.Body.Close()
	}
}

func TestSchemeRelativeRedirect(t *testing.T) {
	t.Parallel()
	inr := New("https://internal.example")
	require.True(t, inr.isExternalRedirect("//external.example/path"))
	require.False(t, inr.isExternalRedirect("//internal.example/path"))
	require.False(t, inr.isExternalRedirect("/path"))
}

type ownedSession struct {
	released int
	saveErr  error
	value    any
}

func (s *ownedSession) Get(any) any  { return s.value }
func (s *ownedSession) Set(_, v any) { s.value = v }
func (s *ownedSession) Delete(any)   { s.value = nil }
func (s *ownedSession) Save() error  { return s.saveErr }

type ownedStore struct{ session *ownedSession }

func (s ownedStore) Get(fiber.Ctx) (*ownedSession, error) { return s.session, nil }
func TestExplicitSessionOwnership(t *testing.T) {
	t.Parallel()
	sess := &ownedSession{saveErr: errors.New("save")}
	adapter := NewFiberSessionAdapterWithRelease(ownedStore{sess}, func(s *ownedSession) { s.released++ })
	c := fibert.Default()
	_, err := adapter.Get(c, "x")
	require.NoError(t, err)
	require.Error(t, adapter.Set(c, "x", 1))
	require.Error(t, adapter.Delete(c, "x"))
	sess.value = 1
	_, err = adapter.GetFlash(c, "x")
	require.Error(t, err)
	require.Equal(t, 4, sess.released)
	borrowed := NewFiberSessionAdapter(ownedStore{sess})
	_, err = borrowed.Get(c, "x")
	require.NoError(t, err)
	require.Equal(t, 4, sess.released)
}

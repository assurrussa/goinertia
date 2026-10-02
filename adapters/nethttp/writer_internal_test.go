package nethttp

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

type cookieSession struct {
	data  map[string]any
	saves int
}

func (s *cookieSession) Flash(w http.ResponseWriter, _ *http.Request, key string, value any) error {
	s.data[key] = value
	s.saves++
	http.SetCookie(w,
		&http.Cookie{
			Name:     "session",
			Value:    "rotated",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		})
	return nil
}

func (s *cookieSession) GetFlash(w http.ResponseWriter, _ *http.Request, key string) (any, error) {
	value := s.data[key]
	delete(s.data, key)
	if value != nil {
		http.SetCookie(w,
			&http.Cookie{
				Name:     "session",
				Value:    "rotated",
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			})
	}
	return value, nil
}

func TestHTTPPrecommitFlashAndStreaming(t *testing.T) {
	t.Parallel()
	store := &cookieSession{data: make(map[string]any)}
	i := New("https://app.example", WithSessionStore(store))
	handler := i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
		i.WithFlashSuccess(State(r), "saved")
		return i.Redirect(w, r, "/next")
	}))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/save", nil)
	req.Header.Set(core.HeaderInertia, "true")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Equal(t, http.StatusSeeOther, w.Code)
	require.Empty(t, w.Body.String())
	require.Equal(t, 1, store.saves)
	require.Contains(t, w.Header().Get("Set-Cookie"), "session=rotated")
	handler = i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error { return i.Render(w, r, "Page", nil) }))
	req = httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/next", nil)
	req.Header.Set(core.HeaderInertia, "true")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "saved")
	require.Empty(t, store.data)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	require.NotContains(t, w.Body.String(), "saved")
	streamRecorder := httptest.NewRecorder()
	handler = i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, err := io.WriteString(w, "first")
		assert.NoError(t, err)
		assert.Equal(t, "first", streamRecorder.Body.String())
		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)
		flusher.Flush()
		_, err = io.Copy(w, strings.NewReader("second"))
		assert.NoError(t, err)
	}))
	w = streamRecorder
	handler.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/stream", nil))
	require.Equal(t, "firstsecond", w.Body.String())
	require.True(t, w.Flushed)
}

type capabilityWriter struct {
	*httptest.ResponseRecorder
	hijacked bool
	pushes   int
}

func (w *capabilityWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return nil, nil, nil
}
func (w *capabilityWriter) Push(string, *http.PushOptions) error { w.pushes++; return nil }
func TestHTTPWriterCapabilitiesAndInformationals(t *testing.T) {
	t.Parallel()
	raw := &capabilityWriter{ResponseRecorder: httptest.NewRecorder()}
	base := &responseWriter{ResponseWriter: raw, before: func(code int) int { return code }}
	wrapped := preserveCapabilities(base)
	require.Implements(t, (*http.Flusher)(nil), wrapped)
	require.Implements(t, (*http.Hijacker)(nil), wrapped)
	require.Implements(t, (*http.Pusher)(nil), wrapped)
	require.Implements(t, (*io.ReaderFrom)(nil), wrapped)
	pusher, ok := wrapped.(http.Pusher)
	require.True(t, ok)
	require.NoError(t, pusher.Push("/asset", nil))
	require.Equal(t, 1, raw.pushes)
	require.NoError(t, http.NewResponseController(wrapped).Flush())
	require.True(t, base.wrote)
	hijacker, ok := wrapped.(http.Hijacker)
	require.True(t, ok)
	_, _, err := hijacker.Hijack()
	require.NoError(t, err)
	require.True(t, raw.hijacked)
	plain := &responseWriter{ResponseWriter: plainWriter{httptest.NewRecorder()}, before: func(code int) int { return code }}
	require.NotImplements(t, (*http.Flusher)(nil), preserveCapabilities(plain))
	require.ErrorIs(t, http.NewResponseController(plain).Flush(), http.ErrNotSupported)
	informative := &responseWriter{ResponseWriter: plainWriter{httptest.NewRecorder()}, before: func(code int) int { return code }}
	informative.WriteHeader(http.StatusEarlyHints)
	require.False(t, informative.wrote)
	informative.WriteHeader(http.StatusOK)
	require.True(t, informative.wrote)
}

type plainWriter struct{ w http.ResponseWriter }

func (w plainWriter) Header() http.Header         { return w.w.Header() }
func (w plainWriter) Write(b []byte) (int, error) { return w.w.Write(b) }
func (w plainWriter) WriteHeader(code int)        { w.w.WriteHeader(code) }

type transparentWriter struct{ http.ResponseWriter }

func (w transparentWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func TestHTTPErrorThroughTransparentWriter(t *testing.T) {
	t.Parallel()
	i := New("https://app.example", WithCoreOptions(core.WithFS(views.Templates)))
	h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wrapped := transparentWriter{transparentWriter{w}}
		i.Handler(func(w http.ResponseWriter, _ *http.Request) error {
			_, err := io.WriteString(w, "already")
			assert.NoError(t, err)
			return errors.New("stream failed")
		}).ServeHTTP(wrapped, r)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "already", w.Body.String())
}

type derivedContextKey struct{}

type inspectSSRContext struct{ inspect func(context.Context) }

func (*inspectSSRContext) Reset() {}
func (s *inspectSSRContext) Post(ctx context.Context, _ string, _ []byte, _ map[string]string) (int, []byte, error) {
	s.inspect(ctx)
	return 0, nil, ctx.Err()
}

func TestHTTPDownstreamContextReachesCallbacksAndSSR(t *testing.T) {
	t.Parallel()
	for _, inertiaRequest := range []bool{true, false} {
		t.Run(map[bool]string{true: "lazy", false: "ssr"}[inertiaRequest], func(t *testing.T) {
			t.Parallel()
			called := false
			var active *http.Request
			inspect := func(ctx context.Context) {
				called = true
				assert.Equal(t, "authenticated", ctx.Value(derivedContextKey{}))
				require.ErrorIs(t, ctx.Err(), context.Canceled)
				request, ok := Request(ctx)
				assert.True(t, ok)
				assert.Same(t, active, request)
			}
			i := New("https://app.example", WithCoreOptions(core.WithFS(views.Templates),
				core.WithSSRConfig(core.SSRConfig{
					URL: "http://fixture.invalid", DisableRetries: true,
					SSRClient: &inspectSSRContext{inspect: inspect},
				})))
			h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx, cancel := context.WithCancel(context.WithValue(r.Context(), derivedContextKey{}, "authenticated"))
				defer cancel()
				cancel()
				active = r.WithContext(ctx)
				err := i.Render(w, active, "Page", map[string]any{
					"user": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						if inertiaRequest {
							inspect(ctx)
						}
						return "Alice", nil
					}},
				})
				if inertiaRequest {
					assert.NoError(t, err)
				} else {
					assert.ErrorIs(t, err, context.Canceled)
				}
			}))
			r := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
			if inertiaRequest {
				r.Header.Set(core.HeaderInertia, "true")
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.True(t, called)
			if !inertiaRequest {
				require.Empty(t, w.Body.String())
			}
		})
	}
}

func TestHTTPVaryPreservesMultipleFieldLines(t *testing.T) {
	t.Parallel()
	for _, values := range [][]string{
		{"Accept-Encoding", "Accept-Language"},
		{"Accept-Encoding", "x-inertia, Precognition"},
		{"Accept-Encoding", "*"},
	} {
		i := New("https://app.example")
		h := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			for _, value := range values {
				w.Header().Add("Vary", value)
			}
			w.WriteHeader(http.StatusNoContent)
		}))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
		got := w.Result()
		_ = got.Body.Close()
		require.Equal(t, values, got.Header.Values("Vary")[:len(values)])
		combined := strings.Join(got.Header.Values("Vary"), ",")
		require.True(t, core.HasVaryToken(combined, core.HeaderInertia))
		require.True(t, core.HasVaryToken(combined, core.HeaderPrecognition))
		if values[1] == "*" || values[1] == "x-inertia, Precognition" {
			require.Equal(t, values, got.Header.Values("Vary"))
		}
	}
}

func TestHTTPSSRServerCancellationAndErrors(t *testing.T) {
	t.Parallel()
	started := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter,
		r *http.Request,
	) {
		_,
			_ = io.Copy(io.Discard,
			r.Body)
		started <- struct{}{}
		<-r.Context().Done()
	}))
	defer srv.Close()
	i := New("https://app.example",
		WithCoreOptions(core.WithFS(views.Templates),
			core.WithSSRConfig(core.SSRConfig{
				URL:            srv.URL,
				DisableRetries: true,
			})))
	ctx, cancel := context.WithCancel(t.Context())
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)
	failed := make(chan error, 1)
	handler := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { failed <- i.Render(w, r, "Page", nil) }))
	done := make(chan struct{})
	go func() { handler.ServeHTTP(httptest.NewRecorder(), req); close(done) }()
	<-started
	cancel()
	select {
	case err := <-failed:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("SSR did not cancel")
	}
	<-done
}

func TestHTTPCheckerOnlyAndPrecognitionFlash(t *testing.T) {
	t.Parallel()
	var called atomic.Int64
	i := New("https://app.example",
		WithCSRFTokenCheckProvider(func(*http.Request) error {
			called.Add(1)
			return core.NewError(http.StatusForbidden,
				"denied")
		}))
	h := i.Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Error("checker did not block handler") }))
	r := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", nil)
	r.Header.Set(core.HeaderPrecognition, "true")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Equal(t, int64(1), called.Load())
	require.Equal(t, http.StatusForbidden, w.Code)
	store := &cookieSession{data: make(map[string]any)}
	i = New("https://app.example", WithSessionStore(store))
	h = i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
		i.WithFlashSuccess(State(r), "skip")
		return i.Redirect(w, r, "/next")
	}))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	require.Zero(t, store.saves)
}

func TestHTTPErrorAfterCommitAndInvalidSSR(t *testing.T) {
	t.Parallel()
	i := New("https://app.example")
	h := i.Middleware(i.Handler(func(w http.ResponseWriter, _ *http.Request) error {
		_, err := w.Write([]byte("already"))
		assert.NoError(t, err)
		return errors.New("stream failed")
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Equal(t, "already", w.Body.String())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,
		_ *http.Request,
	) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	i = New("https://app.example",
		WithCoreOptions(core.WithFS(views.Templates),
			core.WithSSRConfig(core.SSRConfig{
				URL:            srv.URL,
				DisableRetries: true,
			})))
	h = i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.ErrorIs(t, i.Render(w, r, "Page", nil), core.ErrBadSsrStatusCode)
	}))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	require.Empty(t, w.Body.String())
}

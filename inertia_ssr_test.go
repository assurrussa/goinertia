package goinertia_test

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/inertiat"
	"github.com/assurrussa/goinertia/inertiat/fibert"
)

func TestInertia_SSR_Retry_WithServer(t *testing.T) {
	t.Parallel()

	t.Run("SuccessAfterRetry", func(t *testing.T) {
		t.Parallel()

		var attempts int32
		client := &mockSSRClient{
			onPost: func(_ context.Context) (int, []byte, error) {
				val := atomic.AddInt32(&attempts, 1)
				if val <= 1 {
					return http.StatusServiceUnavailable, nil, nil
				}
				return http.StatusOK, []byte(`{"body":"<h1>SSR Success</h1>","head":[]}`), nil
			},
		}

		tmpDir := createSSRTemplates(t)

		ta := inertiat.NewTestAppWithoutMiddleware(t,
			goinertia.WithSSRConfig(goinertia.SSRConfig{
				URL:       "http://ssr.local",
				SSRClient: client,
			}),
			goinertia.WithFS(os.DirFS(tmpDir)),
			goinertia.WithRootTemplate("ssr.gohtml"),
		)
		require.NoError(t, ta.Inrt.ParseTemplates())

		c := fibert.Default()
		err := ta.Inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
		assert.Contains(t, string(c.Response().Body()), "<h1>SSR Success</h1>")
	})

	t.Run("FailAfterMaxRetries", func(t *testing.T) {
		t.Parallel()

		var attempts int32
		client := &mockSSRClient{
			onPost: func(_ context.Context) (int, []byte, error) {
				atomic.AddInt32(&attempts, 1)
				return http.StatusServiceUnavailable, nil, nil
			},
		}

		ta := inertiat.NewTestAppWithoutMiddleware(t,
			goinertia.WithSSRConfig(goinertia.SSRConfig{
				URL:       "http://ssr.local",
				SSRClient: client,
			}),
		)
		require.NoError(t, ta.Inrt.ParseTemplates())

		c := fibert.Default()
		err := ta.Inrt.Render(c, "Home", nil)
		require.Error(t, err)
		require.ErrorIs(t, err, goinertia.ErrBadSsrStatusCode)
		// Should retry at least once
		assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
	})

	t.Run("DisableRetries", func(t *testing.T) {
		t.Parallel()

		var attempts int32
		client := &mockSSRClient{
			onPost: func(_ context.Context) (int, []byte, error) {
				atomic.AddInt32(&attempts, 1)
				return http.StatusServiceUnavailable, nil, nil
			},
		}

		ta := inertiat.NewTestAppWithoutMiddleware(t,
			goinertia.WithSSRConfig(goinertia.SSRConfig{
				URL:            "http://ssr.local",
				SSRClient:      client,
				DisableRetries: true,
			}),
		)
		require.NoError(t, ta.Inrt.ParseTemplates())

		c := fibert.Default()
		err := ta.Inrt.Render(c, "Home", nil)
		require.Error(t, err)
		assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
	})

	t.Run("RetryOnCustomStatus", func(t *testing.T) {
		t.Parallel()

		var attempts int32
		client := &mockSSRClient{
			onPost: func(_ context.Context) (int, []byte, error) {
				val := atomic.AddInt32(&attempts, 1)
				if val <= 1 {
					return http.StatusTooManyRequests, nil, nil
				}
				return http.StatusOK, []byte(`{"body":"<h1>SSR Success</h1>","head":[]}`), nil
			},
		}

		tmpDir := createSSRTemplates(t)
		ta := inertiat.NewTestAppWithoutMiddleware(t,
			goinertia.WithSSRConfig(goinertia.SSRConfig{
				URL:           "http://ssr.local",
				SSRClient:     client,
				RetryStatuses: []int{http.StatusTooManyRequests},
			}),
			goinertia.WithFS(os.DirFS(tmpDir)),
			goinertia.WithRootTemplate("ssr.gohtml"),
		)
		require.NoError(t, ta.Inrt.ParseTemplates())

		c := fibert.Default()
		err := ta.Inrt.Render(c, "Home", nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, atomic.LoadInt32(&attempts), int32(2))
	})
}

func TestInertia_SSR_Timeout(t *testing.T) {
	t.Parallel()

	var attempts int32
	client := &mockSSRClient{
		onPost: func(ctx context.Context) (int, []byte, error) {
			atomic.AddInt32(&attempts, 1)
			<-ctx.Done()
			return 0, nil, ctx.Err()
		},
	}

	ta := inertiat.NewTestAppWithoutMiddleware(t,
		goinertia.WithSSRConfig(goinertia.SSRConfig{
			URL:            "http://ssr.local",
			SSRClient:      client,
			Timeout:        10 * time.Millisecond,
			DisableRetries: true,
		}),
	)
	require.NoError(t, ta.Inrt.ParseTemplates())

	c := fibert.Default()
	err := ta.Inrt.Render(c, "Home", nil)
	require.Error(t, err)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	assert.Equal(t, int32(1), atomic.LoadInt32(&attempts))
}

func createSSRTemplates(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	tmplPath := filepath.Join(tmpDir, "ssr.gohtml")
	err := os.WriteFile(tmplPath, []byte(`{{ if .processSSR }}{{ raw .processSSR.Body }}{{ end }}`), 0o600)
	require.NoError(t, err)

	errTmplPath := filepath.Join(tmpDir, "error.gohtml")
	err = os.WriteFile(errTmplPath, []byte("Error"), 0o600)
	require.NoError(t, err)

	return tmpDir
}

type mockSSRClient struct {
	onPost func(ctx context.Context) (int, []byte, error)
}

func (m *mockSSRClient) Reset() {}

func (m *mockSSRClient) Post(ctx context.Context, _ string, _ []byte, _ map[string]string) (int, []byte, error) {
	if m.onPost == nil {
		return http.StatusOK, nil, nil
	}
	return m.onPost(ctx)
}

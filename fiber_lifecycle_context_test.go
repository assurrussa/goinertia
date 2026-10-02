package goinertia_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/require"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

type fiberAuthKey struct{}

// State can be obtained before authentication middleware derives a new context.
// Exercise both explicit State access and Render's implicit state access, with
// separate simultaneous requests so pooled helpers cannot conceal state leaks.
func TestFiberReusedStateContext(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "legacy"}[legacy], func(t *testing.T) {
			t.Parallel()
			i := fiberLifecycleAdapter(legacy)
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				s := i.State(c)
				i.Inertia.WithProp(s, "early", c.Path())
				return c.Next()
			})
			app.Get("/*", func(c fiber.Ctx) error {
				ctx := context.WithValue(t.Context(), fiberAuthKey{}, c.Path())
				ctx, cancel := context.WithTimeout(ctx, time.Minute)
				defer cancel()
				c.SetContext(ctx)
				if strings.HasPrefix(c.Path(), "/state/") {
					if i.State(c).Context.Value(fiberAuthKey{}) != c.Path() {
						return errors.New("State retained the context from before authentication")
					}
				}
				return i.Render(c, "Context", map[string]any{
					"auth": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						if err := checkFiberLifecycleContext(ctx, c, legacy); err != nil {
							return nil, err
						}
						return c.Path(), nil
					}},
				})
			})
			const requests = 32
			failed := make(chan error, requests)
			var workers sync.WaitGroup
			for index := range requests {
				workers.Go(func() {
					path := fmt.Sprintf("/%s/%d", []string{"render", "state"}[index%2], index)
					if err := requestFiberLifecycle(t.Context(), app, path); err != nil {
						failed <- err
					}
				})
			}
			workers.Wait()
			close(failed)
			for err := range failed {
				t.Error(err)
			}
		})
	}
}

func TestFiberReusedStateCancellationReachesSSR(t *testing.T) {
	t.Parallel()
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "native", true: "legacy"}[legacy], func(t *testing.T) {
			t.Parallel()
			started, stopped := make(chan struct{}), make(chan struct{})
			ssr := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				close(started)
				<-r.Context().Done()
				close(stopped)
			}))
			defer ssr.Close()
			i := fiberLifecycleAdapter(legacy)
			i.EnableSSR(core.SSRConfig{URL: ssr.URL, Timeout: time.Second, DisableRetries: true})
			app := fiber.New()
			app.Use(func(c fiber.Ctx) error {
				_ = i.State(c)
				return c.Next()
			})
			ctx := context.WithValue(t.Context(), fiberAuthKey{}, "/cancel")
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			result, inspected := make(chan error, 1), make(chan error, 1)
			app.Get("/cancel", func(c fiber.Ctx) error {
				c.SetContext(ctx)
				err := i.Render(c, "Context", map[string]any{ //nolint:contextcheck // Context comes from c.SetContext.
					"auth": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
						// Keep going after inspection so a stale lazy context and a
						// stale SSR context are independently observable.
						inspected <- checkFiberLifecycleContext(ctx, c, legacy)
						return "authenticated", nil
					}},
				})
				result <- err
				return err
			})
			completed := make(chan error, 1)
			go func() {
				resp, err := app.Test(httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/cancel", nil),
					fiber.TestConfig{Timeout: 3 * time.Second})
				if resp != nil {
					_ = resp.Body.Close()
				}
				completed <- err
			}()
			awaitFiberSignal(t, started)
			if err := <-inspected; err != nil {
				t.Error(err)
			}
			cancel()
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.Canceled)
			case <-time.After(2 * time.Second):
				t.Fatal("updated Fiber context did not cancel SSR")
			}
			awaitFiberSignal(t, stopped)
			require.NoError(t, <-completed)
		})
	}
}

func fiberLifecycleAdapter(legacy bool) *fiberadapter.Inertia {
	if legacy {
		return goinertia.New("https://app.example", goinertia.WithFS(views.Templates))
	}
	return fiberadapter.New("https://app.example", fiberadapter.WithFS(views.Templates))
}

//nolint:contextcheck // Legacy lifecycle is exposed by fiber.Ctx.Context.
func checkFiberLifecycleContext(ctx context.Context, c fiber.Ctx, legacy bool) error {
	if legacy {
		value, ok := ctx.(fiber.Ctx)
		if !ok || value != c {
			return errors.New("legacy callback lost the actual Fiber context")
		}
		ctx = value.Context()
	} else if value, ok := fiberadapter.FromContext(ctx); !ok || value != c {
		return errors.New("native callback lost the typed Fiber helper")
	}
	if ctx.Value(fiberAuthKey{}) != c.Path() {
		return errors.New("callback lost downstream authentication value")
	}
	expected, expectedOK := c.Context().Deadline()
	actual, actualOK := ctx.Deadline()
	if !expectedOK || !actualOK || !actual.Equal(expected) {
		return errors.New("callback lost downstream deadline")
	}
	return nil
}

func requestFiberLifecycle(ctx context.Context, app *fiber.App, path string) error {
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	req.Header.Set(core.HeaderInertia, "true")
	resp, err := app.Test(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var page core.PageDTO
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		return fmt.Errorf("%s: status=%d: %w", path, resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK || page.Props["auth"] != path || page.Props["early"] != path {
		return fmt.Errorf("%s: status=%d page=%+v", path, resp.StatusCode, page)
	}
	return nil
}

func awaitFiberSignal(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(3 * time.Second):
		t.Fatal("local Fiber/SSR lifecycle did not finish")
	}
}

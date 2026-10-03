package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	nethttp "github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

const (
	preservedFragmentPath = "/v3/preserved"
	noSSRPath             = "/v3/no-ssr"
	slowPath              = "/v3/slow"
	encryptedPath         = "/v3/encrypted"
	clearHistoryPath      = "/v3/clear-history"
	validationName        = "name"
)

// Reports are test-only observations of the configured SSR failure callback.
// They contain no request data or production credentials.
type failureReports struct {
	mu    sync.Mutex
	count int
}

func (f *failureReports) report(_ context.Context, _ error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.count++
}

func (f *failureReports) total() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.count
}

func recursiveProps(adapter, rawStep string, fail bool) map[string]any {
	step := fixtureStep(rawStep)
	props := pageProps(adapter, "Recursive")
	props["panel"] = map[string]any{
		"label":    fmt.Sprintf("panel-%d", step),
		"cached":   core.Once(fmt.Sprintf("cache-%d", step), core.WithOnceKey("nested-cache")),
		"optional": core.Optional(fmt.Sprintf("optional-%d", step)),
		"heavy":    core.Defer(fmt.Sprintf("heavy-%d", step), "nested"),
		"items":    core.Merge([]int{step}),
		"rescued": core.Rescue(core.Defer(core.LazyProp{Fn: func(context.Context) (any, error) {
			if fail {
				return nil, errors.New("synthetic rescued dependency failure")
			}
			return "recovered", nil
		}}, "rescue")),
	}
	return props
}

func v3Props(cfg config, path, rawStep string, fail bool) (string, map[string]any) {
	switch path {
	case "/v3/recursive":
		return "Recursive", recursiveProps(cfg.adapter, rawStep, fail)
	case "/v3/array":
		props := pageProps(cfg.adapter, "Array wrappers")
		props["arrayFixture"] = true
		props["array"] = []any{
			core.Defer("deferred-array-value", "array"),
			fmt.Sprintf("visible-%d", fixtureStep(rawStep)),
			core.Optional("optional-array-value"),
			core.Rescue(core.Defer(core.LazyProp{Fn: func(context.Context) (any, error) {
				return nil, errors.New("synthetic rescued array failure")
			}}, "array-rescue")),
		}
		return "Array", props
	case "/v3/scroll":
		step := fixtureStep(rawStep)
		props := pageProps(cfg.adapter, "Scroll")
		props["results"] = core.ScrollAt(map[string]any{"records": []int{step}}, core.ScrollPropConfig{
			PageName: "step", PreviousPage: step - 1, NextPage: step + 1, CurrentPage: step,
		}, "records")
		return "Scroll", props
	case "/v3/provider":
		props := pageProps(cfg.adapter, "Provider")
		props["panel"] = core.LazyProp{Fn: func(context.Context) (any, error) {
			return map[string]any{"provided": "provider-value", "extra": core.Optional("provider-child")}, nil
		}}
		return "Provider", props
	case "/v3/bigint":
		props := pageProps(cfg.adapter, "Big integers")
		props["large"] = map[string]any{"id": int64(9007199254740993)}
		return "BigInteger", props
	case "/v3/unsafe":
		props := pageProps(cfg.adapter, "Safe bootstrap")
		props["unsafe"] = "</script><script>window.__injected = true</script>&\"雪\u2028\u2029"
		return "SafeBootstrap", props
	case "/v3/ssr-failure":
		return "SSRFailure", pageProps(cfg.adapter, "SSR fallback")
	case noSSRPath:
		return "NoSSR", pageProps(cfg.adapter, "SSR disabled")
	case slowPath:
		return "Slow", pageProps(cfg.adapter, "Slow")
	case encryptedPath:
		return "Encrypted", pageProps(cfg.adapter, "Encrypted secret")
	case clearHistoryPath:
		return "Cleared", pageProps(cfg.adapter, "History cleared")
	default:
		return "Status", pageProps(cfg.adapter, "Exception page")
	}
}

func waitSlow(ctx context.Context) error {
	timer := time.NewTimer(500 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func registerFiberV3(app *fiber.App, manager *fiberadapter.Inertia, cfg config, failures *failureReports) {
	if cfg.protocol != 3 {
		return
	}
	app.Get("/v3/ssr-reports", func(c fiber.Ctx) error { return c.JSON(map[string]int{"count": failures.total()}) })
	app.Get("/v3/fragment", func(c fiber.Ctx) error {
		return c.Redirect().Status(fiber.StatusFound).To(preservedFragmentPath)
	})
	app.Get("/v3/redirect", func(c fiber.Ctx) error { return manager.Redirect(c, "/second#destination") })
	app.Post("/v3/array-errors", func(c fiber.Ctx) error {
		manager.WithAllValidationErrors(c, core.ValidationErrors{validationName: {"Required", "Must be unique"}})
		return manager.Redirect(c, "/form")
	})
	app.Get("/v3/fatal-prop", func(c fiber.Ctx) error {
		return manager.Render(c, "Failure", map[string]any{"broken": core.LazyProp{Fn: func(context.Context) (any, error) {
			return nil, errors.New("synthetic non-rescued failure")
		}}})
	})
	app.Get("/v3/:feature", func(c fiber.Ctx) error {
		path := c.Path()
		if path == preservedFragmentPath {
			manager.WithPreserveFragment(c, true)
		}
		if path == slowPath {
			if err := waitSlow(c.Context()); err != nil {
				return err
			}
		}
		if path == noSSRPath {
			manager.WithSSRDisabled(c, true)
		}
		if path == encryptedPath {
			manager.WithEncryptHistory(c)
		}
		if path == clearHistoryPath {
			manager.WithClearHistory(c)
		}
		component, props := v3Props(cfg, path, c.Query("step"), c.Query("recover") != "1")
		if path == "/v3/status" {
			status, _ := strconv.Atoi(c.Query("code", "500"))
			if status != 404 {
				status = 500
			}
			props["status"] = status
			return manager.RenderWithStatus(c, status, component, props)
		}
		return manager.Render(c, component, props)
	})
}

func registerHTTPV3(pages *http.ServeMux, manager *nethttp.Inertia, cfg config, failures *failureReports) {
	if cfg.protocol != 3 {
		return
	}
	pages.HandleFunc("GET /v3/ssr-reports", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"count":%d}`, failures.total())
	})
	pages.Handle("GET /v3/fragment", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		http.Redirect(w, r, preservedFragmentPath, http.StatusFound)
		return nil
	}))
	pages.Handle("GET /v3/redirect", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Redirect(w, r, "/second#destination")
	}))
	pages.Handle("POST /v3/array-errors", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		manager.WithAllValidationErrors(nethttp.State(r), core.ValidationErrors{validationName: {"Required", "Must be unique"}})
		return manager.Redirect(w, r, "/form")
	}))
	pages.Handle("GET /v3/fatal-prop", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Failure", map[string]any{"broken": core.LazyProp{Fn: func(context.Context) (any, error) {
			return nil, errors.New("synthetic non-rescued failure")
		}}})
	}))
	pages.Handle("GET /v3/{feature}", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		path := r.URL.Path
		if path == slowPath {
			if err := waitSlow(r.Context()); err != nil {
				return err
			}
		}
		state := nethttp.State(r)
		if path == preservedFragmentPath {
			manager.WithPreserveFragment(state, true)
		}
		if path == noSSRPath {
			manager.WithSSRDisabled(state, true)
		}
		if path == encryptedPath {
			manager.WithEncryptHistory(state)
		}
		if path == clearHistoryPath {
			manager.WithClearHistory(state)
		}
		component, props := v3Props(cfg, path, r.URL.Query().Get("step"), r.URL.Query().Get("recover") != "1")
		if path == "/v3/status" {
			status, _ := strconv.Atoi(r.URL.Query().Get("code"))
			if status != 404 {
				status = 500
			}
			props["status"] = status
			return manager.RenderWithStatus(w, r, status, component, props)
		}
		return manager.Render(w, r, component, props)
	}))
}

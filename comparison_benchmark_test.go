package goinertia_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/views"
)

type benchmarkCase struct {
	mode        string
	props, lazy int
	parallel    bool
}

func benchmarkCases() []benchmarkCase {
	cases := make([]benchmarkCase, 0, 32)
	for _, mode := range []string{"JSON", "HTML"} {
		for _, count := range []int{0, 10, 100} {
			for _, lazy := range []int{0, 1, 10} {
				if lazy > count {
					continue
				}
				for _, parallel := range []bool{false, true} {
					cases = append(cases, benchmarkCase{mode: mode, props: count, lazy: lazy, parallel: parallel})
				}
			}
		}
	}
	for _, mode := range []string{"flash", "SSR-cache", "SSR-retry", "SSR-cancel"} {
		cases = append(cases, benchmarkCase{mode: mode, props: 10})
	}
	return cases
}

func (c benchmarkCase) name() string {
	name := fmt.Sprintf("%s/props=%d/lazy=%d", c.mode, c.props, c.lazy)
	if c.parallel {
		name += "/parallel"
	}
	return name
}

func benchmarkProps(c benchmarkCase) map[string]any {
	props := make(map[string]any, c.props)
	for n := range c.props {
		key := fmt.Sprintf("prop%d", n)
		if n < c.lazy {
			props[key] = goinertia.LazyProp{Key: key, Fn: func(context.Context) (any, error) { return "computed", nil }}
		} else {
			props[key] = "value"
		}
	}
	return props
}

type benchmarkSSRClient struct {
	mode  string
	calls atomic.Int64
}

func (*benchmarkSSRClient) Reset() {}
func (c *benchmarkSSRClient) Post(ctx context.Context, _ string, _ []byte, _ map[string]string) (int, []byte, error) {
	if c.mode == "SSR-cancel" {
		return 0, nil, ctx.Err()
	}
	if c.mode == "SSR-retry" && c.calls.Add(1)%2 == 1 {
		return http.StatusServiceUnavailable, nil, nil
	}
	return http.StatusOK, []byte(`{"body":"<p>SSR</p>","head":[]}`), nil
}

func benchmarkOptions(c benchmarkCase) []goinertia.Option {
	opts := []goinertia.Option{
		goinertia.WithFS(views.Templates),
		goinertia.WithAssetVersion("v1"),
		goinertia.WithLogger(benchmarkLogger{}),
	}
	if c.mode == "flash" {
		opts = append(opts, goinertia.WithSessionStore(benchmarkFiberSession{}))
	}
	if len(c.mode) > 4 && c.mode[:4] == "SSR-" {
		cfg := goinertia.SSRConfig{
			URL: "http://fixture.invalid/render", SSRClient: &benchmarkSSRClient{mode: c.mode},
			RetryDelay: time.Nanosecond,
		}
		if c.mode == "SSR-cache" {
			cfg.CacheTTL = time.Hour
		}
		if c.mode == "SSR-cancel" {
			cfg.DisableRetries = true
		}
		opts = append(opts, goinertia.WithSSRConfig(cfg))
	}
	return opts
}

type benchmarkFiber interface {
	Render(ctx fiber.Ctx, component string, props map[string]any) error
	Middleware() fiber.Handler
	ParseTemplates() error
	WithFlashSuccess(ctx fiber.Ctx, message string)
	Redirect(ctx fiber.Ctx, target string) error
}
type benchmarkFiberSession struct{}

func (benchmarkFiberSession) Get(fiber.Ctx, string) (any, error) { return map[string]any{}, nil }

func (benchmarkFiberSession) GetFlash(fiber.Ctx, string) (any, error) { return map[string]any{}, nil }

func (benchmarkFiberSession) Set(c fiber.Ctx, _ string, _ any) error {
	c.Cookie(&fiber.Cookie{Name: "session", Value: "fixture", Path: "/", HTTPOnly: true, Secure: true, SameSite: "Lax"})
	return nil
}
func (s benchmarkFiberSession) Flash(c fiber.Ctx, k string, v any) error { return s.Set(c, k, v) }
func (benchmarkFiberSession) Delete(fiber.Ctx, string) error             { return nil }
func benchmarkFiberCase(b *testing.B, c benchmarkCase, i benchmarkFiber) {
	b.Helper()
	if err := i.ParseTemplates(); err != nil {
		b.Fatal(err)
	}
	app := fiber.New()
	app.Use(i.Middleware())
	props := benchmarkProps(c)
	var lazyCalls atomic.Int64
	for key, value := range props {
		if lazy, ok := value.(goinertia.LazyProp); ok {
			lazy.Fn = func(context.Context) (any, error) { lazyCalls.Add(1); return "computed", nil }
			props[key] = lazy
		}
	}
	app.All("/test", func(ctx fiber.Ctx) error {
		if c.mode == "flash" {
			i.WithFlashSuccess(ctx, "saved")
			return i.Redirect(ctx, "/next")
		}
		if c.mode == "SSR-cancel" {
			lifecycle, cancel := context.WithCancel(context.Background())
			cancel()
			ctx.SetContext(lifecycle)
		}
		err := i.Render(ctx, "Page", props)
		if c.mode == "SSR-cancel" {
			if err == nil {
				return errors.New("expected cancellation")
			}
			return ctx.Status(http.StatusServiceUnavailable).Send(nil)
		}
		return err
	})
	handler := app.Handler()
	prepare := func() *fasthttp.RequestCtx {
		ctx := new(fasthttp.RequestCtx)
		ctx.Request.SetRequestURI("http://app.example/test")
		ctx.Request.Header.SetMethod(http.MethodGet)
		if c.mode == "JSON" || c.mode == "flash" {
			ctx.Request.Header.Set(goinertia.HeaderInertia, "true")
			ctx.Request.Header.Set(goinertia.HeaderVersion, "v1")
		}
		if c.mode == "flash" {
			ctx.Request.Header.SetMethod(http.MethodPost)
		}
		return ctx
	}
	ctx := prepare()
	handler(ctx)
	expected := http.StatusOK
	if c.mode == "flash" {
		expected = http.StatusSeeOther
	}
	if c.mode == "SSR-cancel" {
		expected = http.StatusServiceUnavailable
	}
	if ctx.Response.StatusCode() != expected {
		b.Fatalf("got %d: %s", ctx.Response.StatusCode(), ctx.Response.Body())
	}
	if c.mode != "flash" && lazyCalls.Load() != int64(c.lazy) {
		b.Fatalf("lazy count: got %d, want %d", lazyCalls.Load(), c.lazy)
	}
	b.ReportAllocs()
	b.ResetTimer()
	if c.parallel {
		b.RunParallel(func(pb *testing.PB) {
			worker := prepare()
			for pb.Next() {
				worker.Response.Reset()
				worker.ResetUserValues()
				handler(worker)
			}
		})
		b.StopTimer()
		checkBenchmarkLazy(b, c, lazyCalls.Load())
		return
	}
	for b.Loop() {
		ctx.Response.Reset()
		ctx.ResetUserValues()
		handler(ctx)
	}
	checkBenchmarkLazy(b, c, lazyCalls.Load())
}

// BenchmarkCorrectedFiber uses the same harness in the corrected pre-extraction checkout.
func BenchmarkCorrectedFiber(b *testing.B) {
	for _, c := range benchmarkCases() {
		b.Run(c.name(), func(b *testing.B) {
			benchmarkFiberCase(b, c, goinertia.New("https://app.example", benchmarkOptions(c)...))
		})
	}
}

// Keep imports and helpers common to both adapters without transport buffering.
type benchmarkHTTPWriter struct {
	headers http.Header
	body    bytes.Buffer
	status  int
}

func (w *benchmarkHTTPWriter) Header() http.Header    { return w.headers }
func (w *benchmarkHTTPWriter) WriteHeader(status int) { w.status = status }
func (w *benchmarkHTTPWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.Write(p)
}
func (w *benchmarkHTTPWriter) reset() { clear(w.headers); w.body.Reset(); w.status = 0 }
func (w *benchmarkHTTPWriter) ReadFrom(r io.Reader) (int64, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	return w.body.ReadFrom(r)
}

type benchmarkLogger struct{}

func (benchmarkLogger) DebugContext(context.Context, string, ...any) {}
func (benchmarkLogger) InfoContext(context.Context, string, ...any)  {}
func (benchmarkLogger) WarnContext(context.Context, string, ...any)  {}
func (benchmarkLogger) ErrorContext(context.Context, string, ...any) {}

func checkBenchmarkLazy(b *testing.B, c benchmarkCase, calls int64) {
	b.Helper()
	if c.mode != "flash" && calls != int64(b.N+1)*int64(c.lazy) {
		b.Fatalf("per-request lazy calls: got %d, want %d", calls, int64(b.N+1)*int64(c.lazy))
	}
}

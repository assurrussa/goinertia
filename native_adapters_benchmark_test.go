package goinertia_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"

	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

func BenchmarkNativeFiber(b *testing.B) {
	for _, c := range benchmarkCases() {
		b.Run(c.name(), func(b *testing.B) {
			benchmarkFiberCase(b, c, fiberadapter.New("https://app.example", benchmarkOptions(c)...))
		})
	}
}

type benchmarkHTTPSession struct{}

func (benchmarkHTTPSession) Flash(w http.ResponseWriter, _ *http.Request, _ string, _ any) error {
	http.SetCookie(w, &http.Cookie{
		Name: "session", Value: "fixture", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
	})
	return nil
}

func (benchmarkHTTPSession) GetFlash(http.ResponseWriter, *http.Request, string) (any, error) {
	return map[string]any{}, nil
}

func benchmarkHTTPCase(b *testing.B, c benchmarkCase) {
	b.Helper()
	opts := []nethttp.Option{
		nethttp.WithCoreOptions(core.WithFS(views.Templates),
			core.WithAssetVersion("v1")),
		nethttp.WithLogger(benchmarkLogger{}),
	}
	if c.mode == "flash" {
		opts = append(opts, nethttp.WithSessionStore(benchmarkHTTPSession{}))
	}
	if len(c.mode) > 4 && c.mode[:4] == "SSR-" {
		cfg := core.SSRConfig{
			URL: "http://fixture.invalid/render", SSRClient: &benchmarkSSRClient{mode: c.mode},
			RetryDelay: time.Nanosecond,
		}
		if c.mode == "SSR-cache" {
			cfg.CacheTTL = time.Hour
		}
		if c.mode == "SSR-cancel" {
			cfg.DisableRetries = true
		}
		opts = append(opts, nethttp.WithCoreOptions(core.WithSSRConfig(cfg)))
	}
	i, err := nethttp.NewWithValidation("https://app.example", opts...)
	if err != nil {
		b.Fatal(err)
	}
	props := benchmarkProps(c)
	var lazyCalls atomic.Int64
	for key, value := range props {
		if lazy, ok := value.(goinertia.LazyProp); ok {
			lazy.Fn = func(context.Context) (any, error) { lazyCalls.Add(1); return "computed", nil }
			props[key] = lazy
		}
	}
	handler := i.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c.mode == "flash" {
			i.WithFlashSuccess(nethttp.State(r), "saved")
			_ = i.Redirect(w, r, "/next")
			return
		}
		err := i.Render(w, r, "Page", props)
		if c.mode == "SSR-cancel" {
			if err == nil {
				b.Error("expected cancellation")
			}
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if err != nil {
			b.Error(err)
		}
	}))

	req, w := prepareBenchmarkHTTP(c)
	handler.ServeHTTP(w, req)
	expected := http.StatusOK
	if c.mode == "flash" {
		expected = http.StatusSeeOther
	}
	if c.mode == "SSR-cancel" {
		expected = http.StatusServiceUnavailable
	}
	if w.status != expected {
		b.Fatalf("got %d: %s", w.status, w.body.String())
	}
	if c.mode != "flash" && lazyCalls.Load() != int64(c.lazy) {
		b.Fatalf("lazy count: got %d, want %d", lazyCalls.Load(), c.lazy)
	}
	b.ReportAllocs()
	b.ResetTimer()
	if c.parallel {
		b.RunParallel(func(pb *testing.PB) {
			request, writer := prepareBenchmarkHTTP(c)
			for pb.Next() {
				writer.reset()
				handler.ServeHTTP(writer, request)
			}
		})
		b.StopTimer()
		checkBenchmarkLazy(b, c, lazyCalls.Load())
		return
	}
	for b.Loop() {
		w.reset()
		handler.ServeHTTP(w, req)
	}
	checkBenchmarkLazy(b, c, lazyCalls.Load())
}

func BenchmarkNativeHTTP(b *testing.B) {
	for _, c := range benchmarkCases() {
		b.Run(c.name(), func(b *testing.B) { benchmarkHTTPCase(b, c) })
	}
}

// BenchmarkFiberMetadataOwnership measures the required pooled-string copies
// separately; it is not an adapter-only cost.
var benchmarkMetaSink core.RequestMeta

func BenchmarkFiberMetadataOwnership(b *testing.B) {
	app := fiber.New()
	raw := new(fasthttp.RequestCtx)
	raw.Request.SetRequestURI("http://app.example/page?query=value")
	raw.Request.Header.SetMethod(http.MethodGet)
	for _, key := range []string{
		core.HeaderInertia,
		core.HeaderVersion,
		core.HeaderPartialComponent,
		core.HeaderPartialOnly,
		core.HeaderPartialExcept,
		core.HeaderErrorBag,
		core.HeaderReset,
		core.HeaderExceptOnceProps,
		core.HeaderInfiniteScrollMergeIntent,
		core.HeaderPrecognition,
		core.HeaderPrecognitionValidateOnly,
	} {
		raw.Request.Header.Set(key, "fixture")
	}
	ctx := app.AcquireCtx(raw)
	defer app.ReleaseCtx(ctx)
	b.ReportAllocs()
	for b.Loop() {
		benchmarkMetaSink = fiberadapter.RequestMeta(ctx)
	}
}

func prepareBenchmarkHTTP(c benchmarkCase) (*http.Request, *benchmarkHTTPWriter) {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://app.example/test", nil)
	if c.mode == "JSON" || c.mode == "flash" {
		req.Header.Set(core.HeaderInertia, "true")
		req.Header.Set(core.HeaderVersion, "v1")
	}
	if c.mode == "flash" {
		req.Method = http.MethodPost
	}
	if c.mode == "SSR-cancel" {
		ctx, cancel := context.WithCancel(req.Context())
		cancel()
		req = req.WithContext(ctx)
	}
	return req, &benchmarkHTTPWriter{headers: make(http.Header)}
}

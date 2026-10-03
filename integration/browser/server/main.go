// Command server runs the real native adapters for the browser integration suite.
package main

import (
	"context"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	nethttp "github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

const assetVersion = "browser-v1"

//go:embed templates/*.gohtml
var templates embed.FS

type config struct {
	adapter  string
	baseURL  string
	assets   string
	ssrURL   string
	protocol int
}

type submission struct {
	Name string `json:"name" form:"name"`
}

func main() {
	adapter := flag.String("adapter", "fiber", "native adapter: fiber or nethttp")
	port := flag.Int("port", 18984, "HTTP port on the loopback interface")
	assets := flag.String("assets", "integration/browser/dist", "directory containing app.js")
	ssrURL := flag.String("ssr-url", "", "optional Inertia Node SSR /render endpoint")
	protocol := flag.Int("protocol", 2, "Inertia wire protocol: 2 or 3")
	flag.Parse()
	if *protocol != 2 && *protocol != 3 {
		log.Fatal("protocol must be 2 or 3")
	}
	if *port < 1 || *port > 65535 {
		log.Fatal("port must be between 1 and 65535")
	}
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(*port))
	cfg := config{adapter: *adapter, baseURL: "http://" + address, assets: *assets, ssrURL: *ssrURL, protocol: *protocol}
	if err := serve(cfg, address); err != nil {
		log.Fatal(err)
	}
}

func serve(cfg config, address string) error {
	switch cfg.adapter {
	case "fiber":
		app, err := newFiber(cfg)
		if err != nil {
			return err
		}
		return app.Listen(address)
	case "nethttp":
		handler, err := newHTTP(cfg)
		if err != nil {
			return err
		}
		server := &http.Server{
			Addr: address, Handler: handler,
			ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second,
			WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second,
		}
		return server.ListenAndServe()
	default:
		return fmt.Errorf("unknown adapter %q (want fiber or nethttp)", cfg.adapter)
	}
}

func pageProps(adapter, title string) map[string]any {
	return map[string]any{"title": title, "adapter": adapter}
}

func homeProps(adapter, rawPage string, deferred bool) map[string]any {
	page, err := strconv.Atoi(rawPage)
	if err != nil || page < 1 {
		page = 1
	}
	props := pageProps(adapter, "Home")
	props["items"] = core.Merge([]int{page})
	if deferred {
		props["heavy"] = core.Defer(core.LazyProp{Fn: func(context.Context) (any, error) {
			return "deferred-ready", nil
		}})
	}
	return props
}

func newFiber(cfg config) (*fiber.App, error) {
	sessions := fiberSessions{newSessions()}
	ssrFailures := &failureReports{}
	opts := []fiberadapter.Option{
		fiberadapter.WithFS(templates),
		fiberadapter.WithRootTemplate("templates/app.gohtml"),
		fiberadapter.WithRootErrorTemplate("templates/error.gohtml"),
		fiberadapter.WithAssetVersion(assetVersion),
		fiberadapter.WithSessionStore(sessions),
	}
	if cfg.protocol == 3 {
		opts = append(opts, fiberadapter.WithProtocolVersion(core.ProtocolV3),
			fiberadapter.WithSSRFailureHandler(ssrFailures.report),
			fiberadapter.WithPreserveBigIntegers(true),
			fiberadapter.WithSharedProps(map[string]any{"sharedMarker": "shared-value"}))
	}
	if cfg.ssrURL != "" {
		opts = append(opts, fiberadapter.WithSSRConfig(core.SSRConfig{URL: cfg.ssrURL, DisableRetries: true}))
	}
	manager, err := fiberadapter.NewWithValidation(cfg.baseURL, opts...)
	if err != nil {
		return nil, err
	}
	app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
	app.Get("/health", func(c fiber.Ctx) error { return c.SendString("ok") })
	app.Get("/assets/*", static.New(cfg.assets))
	app.Use(sessions.middleware, manager.Middleware())
	app.Get("/", func(c fiber.Ctx) error {
		return manager.Render(c, "Home", homeProps(cfg.adapter, c.Query("page"), true))
	})
	app.Get("/feed", func(c fiber.Ctx) error {
		return manager.Render(c, "Home", homeProps(cfg.adapter, c.Query("page"), false))
	})
	app.Get("/props", func(c fiber.Ctx) error {
		return manager.Render(c, "Props", onceProps(cfg.adapter, c.Query("step"), false, c.Query("fresh") == "1"))
	})
	app.Get("/props-renamed", func(c fiber.Ctx) error {
		return manager.Render(c, "PropsRenamed", onceProps(cfg.adapter, c.Query("step"), true, false))
	})
	app.Get("/nested", func(c fiber.Ctx) error {
		return manager.Render(c, "Nested", nestedProps(cfg.adapter, c.Query("step")))
	})
	app.Get("/second", func(c fiber.Ctx) error {
		return manager.Render(c, "Second", pageProps(cfg.adapter, "Second"))
	})
	app.Get("/form", func(c fiber.Ctx) error {
		return manager.Render(c, "Form", pageProps(cfg.adapter, "Form"))
	})
	app.Get("/native", func(c fiber.Ctx) error {
		manager.WithNativeFlash(c, "message", "Native saved")
		return manager.Render(c, "Home", homeProps(cfg.adapter, c.Query("page"), false))
	})
	registerFiberV3(app, manager, cfg, ssrFailures)
	app.Post("/submit", func(c fiber.Ctx) error {
		var form submission
		if err := c.Bind().Body(&form); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		if strings.TrimSpace(form.Name) == "" {
			manager.WithError(c, "name", "Required")
		} else {
			manager.WithFlashSuccess(c, "Saved")
			manager.WithNativeFlash(c, "message", "Native saved")
		}
		return manager.Redirect(c, "/form")
	})
	return app, nil
}

func newHTTP(cfg config) (http.Handler, error) {
	sessions := httpSessions{newSessions()}
	ssrFailures := &failureReports{}
	opts := []core.Option{
		core.WithFS(templates), core.WithRootTemplate("templates/app.gohtml"),
		core.WithRootErrorTemplate("templates/error.gohtml"), core.WithAssetVersion(assetVersion),
	}
	if cfg.protocol == 3 {
		opts = append(opts, core.WithProtocolVersion(core.ProtocolV3), core.WithSSRFailureHandler(ssrFailures.report),
			core.WithPreserveBigIntegers(true), core.WithSharedProps(map[string]any{"sharedMarker": "shared-value"}))
	}
	if cfg.ssrURL != "" {
		opts = append(opts, core.WithSSRConfig(core.SSRConfig{URL: cfg.ssrURL, DisableRetries: true}))
	}
	manager, err := nethttp.NewWithValidation(cfg.baseURL,
		nethttp.WithCoreOptions(opts...), nethttp.WithSessionStore(sessions))
	if err != nil {
		return nil, err
	}
	pages := http.NewServeMux()
	pages.Handle("GET /{$}", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Home", homeProps(cfg.adapter, r.URL.Query().Get("page"), true))
	}))
	pages.Handle("GET /feed", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Home", homeProps(cfg.adapter, r.URL.Query().Get("page"), false))
	}))
	pages.Handle("GET /props", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		props := onceProps(cfg.adapter, r.URL.Query().Get("step"), false, r.URL.Query().Get("fresh") == "1")
		return manager.Render(w, r, "Props", props)
	}))
	pages.Handle("GET /props-renamed", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "PropsRenamed", onceProps(cfg.adapter, r.URL.Query().Get("step"), true, false))
	}))
	pages.Handle("GET /nested", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Nested", nestedProps(cfg.adapter, r.URL.Query().Get("step")))
	}))
	pages.Handle("GET /second", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Second", pageProps(cfg.adapter, "Second"))
	}))
	pages.Handle("GET /form", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		return manager.Render(w, r, "Form", pageProps(cfg.adapter, "Form"))
	}))
	pages.Handle("GET /native", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		manager.WithNativeFlash(nethttp.State(r), "message", "Native saved")
		return manager.Render(w, r, "Home", homeProps(cfg.adapter, r.URL.Query().Get("page"), false))
	}))
	registerHTTPV3(pages, manager, cfg, ssrFailures)
	pages.Handle("POST /submit", manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		form, err := parseSubmission(w, r)
		if err != nil {
			http.Error(w, "invalid form", http.StatusBadRequest)
			return nil //nolint:nilerr // The 400 response is complete; skip adapter error redirects.
		}
		state := nethttp.State(r)
		if strings.TrimSpace(form.Name) == "" {
			manager.WithError(state, "name", "Required")
		} else {
			manager.WithFlashSuccess(state, "Saved")
			manager.WithNativeFlash(state, "message", "Native saved")
		}
		return manager.Redirect(w, r, "/form")
	}))
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(cfg.assets))))
	mux.Handle("/", sessions.middleware(manager.Middleware(pages)))
	return mux, nil
}

func parseSubmission(w http.ResponseWriter, r *http.Request) (submission, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var form submission
	if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		err := json.NewDecoder(r.Body).Decode(&form)
		return form, err
	}
	if err := r.ParseForm(); err != nil {
		return form, err
	}
	form.Name = r.Form.Get("name")
	return form, nil
}

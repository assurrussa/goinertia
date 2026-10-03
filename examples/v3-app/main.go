// Command v3-app demonstrates both native Go adapters with either official v3
// client. Select the framework at build time and the adapter at startup.
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	nethttp "github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
)

//go:embed templates/*.gohtml
var templates embed.FS

func props(adapter, path string) map[string]any {
	title := "Welcome to Inertia v3"
	if path == "/about" {
		title = "A real Inertia navigation"
	}
	if path == "/without-ssr" {
		title = "SSR disabled for this request"
	}
	return map[string]any{
		"title": title, "adapter": adapter,
		"details": map[string]any{
			"message": core.Defer(core.LazyProp{Fn: func(context.Context) (any, error) {
				return "Nested deferred props are loaded by the official v3 client.", nil
			}}),
			"catalog": core.Once([]string{"Go", "Vue", "React"}, core.WithOnceKey("example-catalog")),
		},
	}
}

func reportSSR(_ context.Context, err error) { log.Printf("SSR failed; falling back to CSR: %v", err) }

func main() {
	adapter := flag.String("adapter", "fiber", "fiber or nethttp")
	port := flag.Int("port", 8383, "loopback HTTP port")
	ssrURL := flag.String("ssr-url", "", "optional private SSR endpoint, e.g. http://127.0.0.1:13714/render")
	flag.Parse()
	address := fmt.Sprintf("127.0.0.1:%d", *port)
	baseURL := "http://" + address
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot locate example assets")
	}
	assets := filepath.Join(filepath.Dir(filename), "dist")
	if *adapter == "fiber" {
		opts := []fiberadapter.Option{
			fiberadapter.WithProtocolVersion(core.ProtocolV3), fiberadapter.WithFS(templates),
			fiberadapter.WithRootTemplate("templates/app.gohtml"),
			fiberadapter.WithRootErrorTemplate("templates/error.gohtml"), fiberadapter.WithAssetVersion("example-v3"),
			fiberadapter.WithSSRFailureHandler(reportSSR),
		}
		if *ssrURL != "" {
			opts = append(opts, fiberadapter.WithSSRConfig(core.SSRConfig{URL: *ssrURL}))
		}
		manager, err := fiberadapter.NewWithValidation(baseURL, opts...)
		if err != nil {
			log.Fatal(err)
		}
		app := fiber.New(fiber.Config{ErrorHandler: manager.MiddlewareErrorListener()})
		app.Get("/assets/*", static.New(assets))
		app.Use(manager.Middleware())
		app.Get("/*", func(c fiber.Ctx) error {
			if c.Path() == "/without-ssr" {
				manager.WithSSRDisabled(c, true)
			}
			return manager.Render(c, "Page", props("Fiber", c.Path()))
		})
		log.Fatal(app.Listen(address))
	}
	if *adapter != "nethttp" {
		log.Fatal("adapter must be fiber or nethttp")
	}
	opts := []core.Option{
		core.WithProtocolVersion(core.ProtocolV3), core.WithFS(templates),
		core.WithRootTemplate("templates/app.gohtml"),
		core.WithRootErrorTemplate("templates/error.gohtml"), core.WithAssetVersion("example-v3"),
		core.WithSSRFailureHandler(reportSSR),
	}
	if *ssrURL != "" {
		opts = append(opts, core.WithSSRConfig(core.SSRConfig{URL: *ssrURL}))
	}
	manager, err := nethttp.NewWithValidation(baseURL, nethttp.WithCoreOptions(opts...))
	if err != nil {
		log.Fatal(err)
	}
	pages := manager.Middleware(manager.Handler(func(w http.ResponseWriter, r *http.Request) error {
		if r.URL.Path == "/without-ssr" {
			manager.WithSSRDisabled(nethttp.State(r), true)
		}
		return manager.Render(w, r, "Page", props("net/http", r.URL.Path))
	}))
	mux := http.NewServeMux()
	mux.Handle("GET /assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir(assets))))
	mux.Handle("GET /", pages)
	server := &http.Server{Addr: address, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("Listening on %s", baseURL)
	log.Fatal(server.ListenAndServe())
}

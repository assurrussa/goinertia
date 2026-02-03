package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/assurrussa/goinertia"
)

type MenuItem struct {
	Label string `json:"label"`
	Href  string `json:"href"`
}

type BaseController struct {
	inertia *goinertia.Inertia
}

func NewBaseController(inertia *goinertia.Inertia) *BaseController {
	return &BaseController{inertia: inertia}
}

func (c *BaseController) Home(ctx fiber.Ctx) error {
	return c.inertia.Render(ctx, "Home", map[string]any{
		"title": "Home",
	})
}

func (c *BaseController) Users(ctx fiber.Ctx) error {
	return c.inertia.Render(ctx, "Users", map[string]any{
		"title": "Users",
	})
}

func (c *BaseController) Settings(ctx fiber.Ctx) error {
	return c.inertia.Render(ctx, "Settings", map[string]any{
		"title": "Settings",
	})
}

func (c *BaseController) Conflict(ctx fiber.Ctx) error {
	// Flash message
	c.inertia.WithFlashError(ctx, "You have been redirected because the page was not found (409 -> Redirect).")
	// Redirect or RedirectBack
	// return c.inertia.Redirect(ctx, "/")
	return c.inertia.RedirectBack(ctx)
}

// FiberSessionWrapper wraps fiber session to match FiberSessionStore interface.
type FiberSessionWrapper struct {
	*session.Session
}

func (w *FiberSessionWrapper) Set(key, val any) {
	if k, ok := key.(string); ok {
		w.Session.Set(k, val)
	}
}

func (w *FiberSessionWrapper) Get(key any) any {
	if k, ok := key.(string); ok {
		return w.Session.Get(k)
	}
	return nil
}

func (w *FiberSessionWrapper) Delete(key any) {
	if k, ok := key.(string); ok {
		w.Session.Delete(k)
	}
}

type SessionProvider struct {
	store *session.Store
}

func (s *SessionProvider) Get(c fiber.Ctx) (*FiberSessionWrapper, error) {
	sess, err := s.store.Get(c)
	if err != nil {
		return nil, err
	}
	return &FiberSessionWrapper{Session: sess}, nil
}

func main() {
	port := flag.String("port", "8383", "Port to listen on")
	flag.Parse()

	menu := []MenuItem{
		{Label: "Home", Href: "/"},
		{Label: "Users", Href: "/users"},
		{Label: "Settings", Href: "/settings"},
		{Label: "Conflict (409)", Href: "/not-found"},
	}

	// Initialize Session Store
	_, store := session.NewWithStore()
	sessionProvider := &SessionProvider{store: store}
	sessionAdapter := goinertia.NewFiberSessionAdapter(sessionProvider)

	viewFS := os.DirFS(examplePath("views"))
	folderPublicPath := examplePath("public")
	inertiaAdapter := goinertia.Must(goinertia.NewWithValidation("http://localhost:"+*port,
		goinertia.WithFS(viewFS),
		goinertia.WithRootTemplate("app.gohtml"),
		goinertia.WithRootErrorTemplate("error.gohtml"),
		goinertia.WithSessionStore(sessionAdapter),
		// global props
		goinertia.WithSharedProps(map[string]any{
			"menu": menu,
		}),
		// for example shared func
		goinertia.WithSetSharedFuncMap(map[string]any{
			"asset": asset(folderPublicPath),
		}),
	))

	app := fiber.New(fiber.Config{ErrorHandler: inertiaAdapter.MiddlewareErrorListener()})
	app.Get("/assets/*", static.New(folderPublicPath))
	app.Use(inertiaAdapter.Middleware())

	controller := NewBaseController(inertiaAdapter)
	app.Get("/", controller.Home)
	app.Get("/users", controller.Users)
	app.Get("/settings", controller.Settings)
	app.Get("/not-found", controller.Conflict)

	log.Fatal(app.Listen(":" + *port))
}

func asset(dir string) func(path string) (string, error) {
	return func(path string) (string, error) {
		// for example shared func
		filePath := filepath.Join(dir, path)
		fs, err := os.Stat(filePath)
		if err != nil {
			return "", err
		}
		unixTime := strconv.FormatInt(fs.ModTime().Unix(), 10)
		return "/assets/" + path + "?" + unixTime, nil
	}
}

func examplePath(path string) string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return path
	}

	return filepath.Join(filepath.Dir(filename), path)
}

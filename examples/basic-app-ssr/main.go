package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

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

func main() {
	port := flag.String("port", "8383", "Port to listen on")
	flag.Parse()

	menu := []MenuItem{
		{Label: "Home", Href: "/"},
		{Label: "Users", Href: "/users"},
		{Label: "Settings", Href: "/settings"},
		{Label: "Conflict (409)", Href: "/not-found"},
		{Label: "Not Found", Href: "/not-found-404"},
	}

	// Initialize Session Store
	_, store := session.NewWithStore()
	sessionAdapter := goinertia.NewFiberSessionAdapter[*session.Session](store)

	viewFS := os.DirFS(examplePath("views"))
	folderPublicPath := examplePath("public")
	inertiaAdapter := goinertia.Must(goinertia.NewWithValidation("http://localhost:"+*port,
		goinertia.WithFS(viewFS),
		goinertia.WithRootTemplate("app.gohtml"),
		goinertia.WithRootErrorTemplate("error.gohtml"),
		goinertia.WithCanExposeDetails(func(_ context.Context, _ map[string][]string) bool {
			// If you need to display errors as they are. true/false
			return true
		}),
		goinertia.WithSessionStore(sessionAdapter),
		// global props
		goinertia.WithSharedProps(map[string]any{
			"menu": menu,
		}),
		// Enable SSR with custom client and cache
		goinertia.WithSSRConfig(goinertia.SSRConfig{
			URL:             goinertia.DefaultSSRURL,
			Timeout:         goinertia.DefaultSSRTimeout,
			CacheTTL:        1 * time.Minute,
			CacheMaxEntries: 1024,
			// SSRClient:       createClient...,
		}),
		// for example shared func
		goinertia.WithSetSharedFuncMap(map[string]any{
			"asset": asset(folderPublicPath),
		}),
	))

	// inertiaAdapter.EnableSSRWithDefault() // or used default settings

	app := fiber.New(fiber.Config{ErrorHandler: inertiaAdapter.MiddlewareErrorListener()})
	app.Get("/assets/*", static.New(folderPublicPath))
	app.Use(inertiaAdapter.Middleware())

	controller := NewBaseController(inertiaAdapter)
	app.Get("/", controller.Home)
	app.Get("/users", controller.Users)
	app.Get("/settings", controller.Settings)
	app.Get("/not-found", controller.Conflict)

	// Catch-all route for 404
	app.Use(func(c fiber.Ctx) error {
		c.Status(fiber.StatusNotFound)
		return inertiaAdapter.Render(c, "Error", map[string]any{
			"status":  404,
			"message": "Page not found",
		})
	})

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

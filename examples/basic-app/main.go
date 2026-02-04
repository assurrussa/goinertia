package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

type User struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
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
		"plan":  goinertia.Once("Pro", goinertia.WithOnceKey("plan_v1")),
		"heavy": goinertia.Defer(goinertia.LazyProp{
			Key: "heavy",
			Fn: func(_ context.Context) (any, error) {
				return []int64{123, 234, 345, 456, 789}, nil
			},
		}),
	})
}

func (c *BaseController) Users(ctx fiber.Ctx) error {
	sortBy := fiber.Query[string](ctx, "sort", "name")
	page := fiber.Query[int](ctx, "page", 1)
	const pageSize = 3

	users := sortUsers(allUsers(), sortBy)
	pageUsers, totalPages, prevPage, nextPage, page := paginateUsers(users, page, pageSize)
	c.inertia.WithMatchPropsOn(ctx, "sort")

	return c.inertia.Render(ctx, "Users", map[string]any{
		"title":      "Users",
		"sort":       sortBy,
		"page":       page,
		"pageSize":   pageSize,
		"total":      len(users),
		"totalPages": totalPages,
		"prevPage":   prevPage,
		"nextPage":   nextPage,
		"users": goinertia.Scroll(pageUsers, goinertia.ScrollPropConfig{
			PageName:     "page",
			PreviousPage: prevPage,
			NextPage:     nextPage,
			CurrentPage:  page,
		}),
	})
}

func (c *BaseController) Settings(ctx fiber.Ctx) error {
	return c.inertia.Render(ctx, "Settings", map[string]any{
		"title": "Settings",
		"diagnostics": goinertia.Optional(goinertia.LazyProp{
			Key: "diagnostics",
			Fn: func(_ context.Context) (any, error) {
				return map[string]any{
					"server_time": time.Now().UTC().Format(time.RFC3339),
					"version":     runtime.Version(),
				}, nil
			},
		}),
	})
}

// CreateUser demonstrates validation-only (Precognition) behavior.
// When the client sends Precognition headers, the response will be 204/422 with errors only.
func (c *BaseController) CreateUser(ctx fiber.Ctx) error {
	name := ctx.FormValue("name")
	email := ctx.FormValue("email")
	isPrecognition := goinertia.IsPrecognition(ctx)

	hasErrors := false
	if name == "" {
		hasErrors = true
		c.inertia.WithError(ctx, "name", "Name is required")
	}
	if email == "" {
		hasErrors = true
		c.inertia.WithError(ctx, "email", "Email is required")
	}

	if isPrecognition {
		return c.inertia.Render(ctx, "Settings", map[string]any{
			"title": "Settings",
		})
	}

	if !hasErrors {
		c.inertia.WithFlashSuccess(ctx, "User created")
		return c.inertia.RedirectBack(ctx)
	}

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
		goinertia.WithSessionStore(sessionAdapter),
		// goinertia.WithPrecognitionVary(false),
		goinertia.WithCanExposeDetails(func(_ context.Context, _ map[string][]string) bool {
			// If you need to display errors as they are. true/false
			return false
		}),
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
	app.Post("/users/create", controller.CreateUser)
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

func allUsers() []User {
	return []User{
		{ID: 1, Name: "Alice", Role: "Admin"},
		{ID: 2, Name: "Bob", Role: "Editor"},
		{ID: 3, Name: "Carla", Role: "Viewer"},
		{ID: 4, Name: "Dmitry", Role: "Admin"},
		{ID: 5, Name: "Elena", Role: "Editor"},
		{ID: 6, Name: "Farid", Role: "Viewer"},
		{ID: 7, Name: "Gita", Role: "Editor"},
		{ID: 8, Name: "Hector", Role: "Viewer"},
		{ID: 9, Name: "Ira", Role: "Admin"},
	}
}

func sortUsers(users []User, sortBy string) []User {
	result := make([]User, len(users))
	copy(result, users)

	switch sortBy {
	case "name_desc":
		sort.Slice(result, func(i, j int) bool { return result[i].Name > result[j].Name })
	case "id_desc":
		sort.Slice(result, func(i, j int) bool { return result[i].ID > result[j].ID })
	case "role":
		sort.Slice(result, func(i, j int) bool { return result[i].Role < result[j].Role })
	default:
		sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	}

	return result
}

func paginateUsers(users []User, page int, pageSize int) (data []User, totalPages int, prevPage int, nextPage int, curPage int) {
	if page < 1 {
		page = 1
	}

	totalPages = (len(users) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		page = totalPages
	}

	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(users) {
		start = len(users)
	}
	if end > len(users) {
		end = len(users)
	}

	if page > 1 {
		prevPage = page - 1
	}
	if page < totalPages {
		nextPage = page + 1
	}

	return users[start:end], totalPages, prevPage, nextPage, page
}

package goinertia_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/inertiat"
)

func Benchmark_Middleware(b *testing.B) {
	inrt := inertiat.NewForTest("http://localhost:3000")

	app := fiber.New()
	app.Use(inrt.Middleware())

	app.Get("/test", func(c fiber.Ctx) error {
		return c.SendString("success")
	})

	req := inertiat.NewRequest(http.MethodGet, "/test", nil, nil)
	req.Header.Set(goinertia.HeaderInertia, "true")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

func Benchmark_FiberErrorHandler(b *testing.B) {
	inrt := inertiat.NewForTest("http://localhost:3000")

	app := fiber.New(fiber.Config{
		ErrorHandler: inrt.MiddlewareErrorListener(),
	})

	app.Get("/test", func(_ fiber.Ctx) error {
		return fiber.NewError(404, "Not found")
	})

	req := inertiat.NewRequest(http.MethodGet, "/test", nil, nil)
	req.Header.Set(goinertia.HeaderInertia, "true")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

func Benchmark_AppErrorHandler(b *testing.B) {
	inrt := inertiat.NewForTest("http://localhost:3000")

	app := fiber.New(fiber.Config{
		ErrorHandler: inrt.MiddlewareErrorListener(),
	})

	app.Get("/test", func(_ fiber.Ctx) error {
		return goinertia.NewError(404, "Not found", errors.New("not found page"))
	})

	req := inertiat.NewRequest(http.MethodGet, "/test", nil, nil)
	req.Header.Set(goinertia.HeaderInertia, "true")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

func Benchmark_AppValidationErrorHandler(b *testing.B) {
	inrt := inertiat.NewForTest("http://localhost:3000")

	app := fiber.New(fiber.Config{
		ErrorHandler: inrt.MiddlewareErrorListener(),
	})

	app.Get("/test", func(_ fiber.Ctx) error {
		return goinertia.NewValidationError(400, "Bad request", goinertia.ValidationErrors{
			"password": {"Не валидный пароль"},
			"name":     {"Не валидное имя"},
		})
	})

	req := inertiat.NewRequest(http.MethodGet, "/test", nil, nil)
	req.Header.Set(goinertia.HeaderInertia, "true")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

func Benchmark_AppServerValidationErrorHandler(b *testing.B) {
	inrt := inertiat.NewForTest("http://localhost:3000")

	app := fiber.New(fiber.Config{
		ErrorHandler: inrt.MiddlewareErrorListener(),
	})

	app.Get("/test", func(_ fiber.Ctx) error {
		return goinertia.NewValidationError(400, "Bad request", goinertia.ValidationErrors{
			"password": {"Не валидный пароль"},
			"name":     {"Не валидное имя"},
		})
	})

	req := inertiat.NewRequest(http.MethodGet, "/test", nil, nil)
	req.Header.Set(goinertia.HeaderInertia, "true")

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp, _ := app.Test(req)
		_ = resp.Body.Close()
	}
}

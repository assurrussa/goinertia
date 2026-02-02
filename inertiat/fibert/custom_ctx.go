package fibert

import (
	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

// Default returns a real Fiber test context.
// Use this in tests instead of attempting to implement fiber.Ctx.
func Default(apps ...*fiber.App) fiber.Ctx {
	var app *fiber.App
	if len(apps) == 0 {
		app = fiber.New()
	} else {
		app = apps[0]
	}
	// Create a bare fasthttp context so Fiber can operate safely on it.
	fctx := &fasthttp.RequestCtx{}
	return app.AcquireCtx(fctx)
}

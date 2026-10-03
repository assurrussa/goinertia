package fiberadapter

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

// MiddlewareSessionAdapter borrows the session from Fiber's session middleware.
// That middleware owns saving and releasing it after the downstream handler.
type MiddlewareSessionAdapter struct{}

func middlewareSession(c fiber.Ctx) (*session.Middleware, error) {
	s := session.FromContext(c)
	if s == nil {
		return nil, errors.New("inertia: Fiber session middleware is required")
	}
	return s, nil
}

// Get reads a middleware-owned session value.
func (MiddlewareSessionAdapter) Get(c fiber.Ctx, key string) (any, error) {
	s, err := middlewareSession(c)
	if err != nil {
		return nil, err
	}
	return s.Get(key), nil
}

// Set updates a middleware-owned session value.
func (MiddlewareSessionAdapter) Set(c fiber.Ctx, key string, value any) error {
	s, err := middlewareSession(c)
	if err != nil {
		return err
	}
	s.Set(key, value)
	return nil
}

// Delete removes a middleware-owned session value.
func (MiddlewareSessionAdapter) Delete(c fiber.Ctx, key string) error {
	s, err := middlewareSession(c)
	if err != nil {
		return err
	}
	s.Delete(key)
	return nil
}

// Flash stores a value until the next GetFlash.
func (s MiddlewareSessionAdapter) Flash(c fiber.Ctx, key string, value any) error {
	return s.Set(c, key, value)
}

// GetFlash removes and returns a middleware-owned value once.
func (MiddlewareSessionAdapter) GetFlash(c fiber.Ctx, key string) (any, error) {
	s, err := middlewareSession(c)
	if err != nil {
		return nil, err
	}
	v := s.Get(key)
	s.Delete(key)
	return v, nil
}

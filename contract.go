package goinertia

import (
	"context"

	"github.com/gofiber/fiber/v3"
)

//go:generate mockgen -source=$GOFILE -destination=mocks/inertia_mock.gen.go -package=${GOPACKAGE}mocks

type (
	CSRFTokenProvider      func(c fiber.Ctx) (string, error)
	CSRFTokenCheckProvider func(c fiber.Ctx) error
)

type SessionStore interface {
	Get(c fiber.Ctx, key string) (any, error)
	Set(c fiber.Ctx, key string, value any) error
	Delete(c fiber.Ctx, key string) error
	Flash(c fiber.Ctx, key string, value any) error
	GetFlash(c fiber.Ctx, key string) (any, error)
}

type SessionAdapter[T FiberSessionStore] interface {
	Get(c fiber.Ctx) (T, error)
}

type FiberSessionStore interface {
	Set(key, val any)
	Get(key any) any
	Delete(key any)
	Save() error
}

type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

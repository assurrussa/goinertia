package goinertia

import (
	"encoding/gob"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

//nolint:gochecknoinits // need for gob registration
func init() {
	// Register types for gob encoder.
	gob.Register([]any{})
	gob.Register(map[string]any{})
	gob.Register(map[string]string{})
}

type FiberSessionAdapter[T FiberSessionStore] struct {
	store SessionAdapter[T]
}

func NewFiberSessionAdapter[T FiberSessionStore](store SessionAdapter[T]) *FiberSessionAdapter[T] {
	return &FiberSessionAdapter[T]{
		store: store,
	}
}

func (f *FiberSessionAdapter[T]) Get(c fiber.Ctx, key string) (any, error) {
	sess, err := f.store.Get(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return sess.Get(key), nil
}

func (f *FiberSessionAdapter[T]) Set(c fiber.Ctx, key string, value any) error {
	sess, err := f.store.Get(c)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	sess.Set(key, value)

	if err := sess.Save(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (f *FiberSessionAdapter[T]) Delete(c fiber.Ctx, key string) error {
	sess, err := f.store.Get(c)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	sess.Delete(key)

	if err := sess.Save(); err != nil {
		return fmt.Errorf("failed to save session: %w", err)
	}

	return nil
}

func (f *FiberSessionAdapter[T]) Flash(c fiber.Ctx, key string, value any) error {
	return f.Set(c, key, value)
}

func (f *FiberSessionAdapter[T]) GetFlash(c fiber.Ctx, key string) (any, error) {
	sess, err := f.store.Get(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	value := sess.Get(key)
	if value != nil {
		sess.Delete(key)
		if err := sess.Save(); err != nil {
			return nil, fmt.Errorf("failed to save session after flash get: %w", err)
		}
	}

	return value, nil
}

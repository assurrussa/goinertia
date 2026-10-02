package goinertia

import (
	"encoding/gob"
	"fmt"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/session"
)

//nolint:gochecknoinits // need for gob registration
func init() {
	// Register types for gob encoder.
	gob.Register([]any{})
	gob.Register(map[string]any{})
	gob.Register(map[string]string{})
}

type FiberSessionAdapter[T FiberSessionStore] struct {
	store   SessionAdapter[T]
	release func(T)
}

func NewFiberSessionAdapter[T FiberSessionStore](store SessionAdapter[T]) *FiberSessionAdapter[T] {
	adapter := &FiberSessionAdapter[T]{store: store}
	// Only Fiber's raw Store transfers ownership. Custom and middleware stores
	// keep their existing lifetime unless the caller explicitly supplies release.
	if _, raw := any(store).(*session.Store); raw {
		adapter.release = func(sess T) {
			if r, ok := any(sess).(interface{ Release() }); ok {
				r.Release()
			}
		}
	}
	return adapter
}

// NewFiberSessionAdapterWithRelease takes ownership of each acquired session.
// Use it for custom raw stores; middleware-managed sessions must not be released.
func NewFiberSessionAdapterWithRelease[T FiberSessionStore](store SessionAdapter[T], release func(T)) *FiberSessionAdapter[T] {
	return &FiberSessionAdapter[T]{store: store, release: release}
}

func (f *FiberSessionAdapter[T]) releaseSession(sess T) {
	if f.release != nil {
		f.release(sess)
	}
}

func (f *FiberSessionAdapter[T]) Get(c fiber.Ctx, key string) (any, error) {
	sess, err := f.store.Get(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	defer f.releaseSession(sess)
	return sess.Get(key), nil
}

func (f *FiberSessionAdapter[T]) Set(c fiber.Ctx, key string, value any) error {
	sess, err := f.store.Get(c)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	defer f.releaseSession(sess)
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

	defer f.releaseSession(sess)
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

	defer f.releaseSession(sess)
	value := sess.Get(key)
	if value != nil {
		sess.Delete(key)
		if err := sess.Save(); err != nil {
			return nil, fmt.Errorf("failed to save session after flash get: %w", err)
		}
	}

	return value, nil
}

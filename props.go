package goinertia

import (
	"context"
	"time"
)

// LazyProp represents a prop that is evaluated lazily.
type LazyProp struct {
	Key string
	Fn  func(ctx context.Context) (any, error)
}

// DeferredProp marks a prop as deferred (loaded via a follow-up partial reload).
type DeferredProp struct {
	Group string
	Value any
}

// OptionalProp marks a prop as optional (only included when explicitly requested).
type OptionalProp struct {
	Value any
}

// AlwaysProp marks a prop as always included, even on partial reloads.
type AlwaysProp struct {
	Value any
}

// MergeProp marks a prop as mergeable during partial reloads.
type MergeProp struct {
	Value   any
	Prepend bool
	Deep    bool
}

// ScrollProp marks a prop as an infinite-scroll prop and adds scroll metadata.
type ScrollProp struct {
	Value  any
	Config ScrollPropConfig
}

// OnceProp marks a prop as once and optionally sets expiration.
type OnceProp struct {
	Key       string
	ExpiresAt *int64
	Value     any
}

// Defer wraps a value as a deferred prop. If group is empty, "default" is used.
func Defer(value any, group ...string) DeferredProp {
	g := "default"
	if len(group) > 0 && group[0] != "" {
		g = group[0]
	}
	return DeferredProp{Group: g, Value: value}
}

// Optional wraps a value as an optional prop.
func Optional(value any) OptionalProp {
	return OptionalProp{Value: value}
}

// Always wraps a value as an always prop.
func Always(value any) AlwaysProp {
	return AlwaysProp{Value: value}
}

// Merge wraps a value as a merge prop.
func Merge(value any) MergeProp {
	return MergeProp{Value: value}
}

// Prepend wraps a value as a prepend merge prop.
func Prepend(value any) MergeProp {
	return MergeProp{Value: value, Prepend: true}
}

// DeepMerge wraps a value as a deep merge prop.
func DeepMerge(value any) MergeProp {
	return MergeProp{Value: value, Deep: true}
}

// Scroll wraps a value as a scroll prop with metadata.
func Scroll(value any, cfg ScrollPropConfig) ScrollProp {
	return ScrollProp{Value: value, Config: cfg}
}

// Once wraps a value as a once prop.
func Once(value any, opts ...OnceOption) OnceProp {
	op := OnceProp{Value: value}
	for _, opt := range opts {
		opt(&op)
	}
	return op
}

// OnceOption configures a OnceProp.
type OnceOption func(*OnceProp)

// WithOnceKey sets a custom key for a once prop.
func WithOnceKey(key string) OnceOption {
	return func(op *OnceProp) {
		op.Key = key
	}
}

// WithOnceExpiresAt sets an expiration time for a once prop.
func WithOnceExpiresAt(t time.Time) OnceOption {
	return func(op *OnceProp) {
		ms := t.UnixMilli()
		op.ExpiresAt = &ms
	}
}

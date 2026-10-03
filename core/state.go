// Package core contains framework-neutral Inertia page and rendering policy.
package core

import "context"

// RequestMeta holds only the protocol inputs needed by the engine.
// Adapters must own strings that can outlive a pooled request buffer.
type RequestMeta struct {
	Method, URL, BaseURL, Referer                                  string
	Inertia, Version, PartialComponent, PartialOnly, PartialExcept string
	Reset, ErrorBag, ExceptOnceProps, ScrollMergeIntent            string
	Precognition, ValidateOnly, CacheControl                       string
}

// State belongs to one request and is not safe for concurrent mutation.
// Context is the request lifecycle context; never retain framework helpers
// beyond that lifecycle.
type State struct {
	Context                    context.Context //nolint:containedctx // State explicitly owns the request lifecycle.
	Meta                       RequestMeta
	Props, ViewData, FlashData map[string]any
	InvalidViewData            bool
	pageMeta                   *pageMeta
	lazyCache                  map[lazyCacheKey]any
	propSource                 propSource
	propMetadata               map[propMetadataLabel][]string
	nestedMergeRoots           map[string]bool
	legacyContext              context.Context //nolint:containedctx // Compatibility callback context is request-local.
}

type propSource uint8

const (
	flashSource propSource = iota
	sharedSource
	contextSource
	requestSource
)

// Source and nesting are separate from the key, so literal JSON keys cannot
// alias nested paths or callbacks from a different precedence layer.
type lazyCacheKey struct {
	path   string
	source propSource
	nested bool
}

// NewState constructs a neutral request state.
func NewState(ctx context.Context, meta RequestMeta) *State { return &State{Context: ctx, Meta: meta} }

// NewLegacyState preserves the historical callback context of a compatibility
// facade. Transport and SSR always use the lifecycle context instead.
func NewLegacyState(ctx, callbacks context.Context, meta RequestMeta) *State {
	s := NewState(ctx, meta)
	s.legacyContext = callbacks
	return s
}

func (s *State) propContext() context.Context {
	if s.legacyContext != nil {
		return s.legacyContext
	}
	return s.Context
}

// FlashToPersist selects values that survive redirects.
func (s *State) FlashToPersist() map[string]any {
	data := make(map[string]any)
	for _, key := range []string{ContextPropsFlash, ContextPropsErrors} {
		if v, ok := s.Props[key].(map[string]string); ok && len(v) > 0 {
			data[key] = v
		}
	}
	if v, ok := s.Props[ContextPropsOld].(map[string]any); ok && len(v) > 0 {
		data[ContextPropsOld] = v
	}
	if s.pageMeta != nil && len(s.pageMeta.nativeFlash) > 0 {
		data[nativeFlashSessionKey] = s.pageMeta.nativeFlash
	}
	return data
}

// LegacyPageMeta exposes the opaque compatibility facade metadata value.
// Native consumers should use engine metadata helpers on State.
func (s *State) LegacyPageMeta() any {
	if s.pageMeta == nil {
		return nil
	}
	return s.pageMeta
}

// SetLegacyPageMeta restores a metadata value previously exposed by the facade.
func (s *State) SetLegacyPageMeta(value any) { s.pageMeta, _ = value.(*pageMeta) }

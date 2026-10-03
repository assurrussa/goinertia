// Package goinertia preserves the historical Fiber API. New applications can
// import core and a native adapter directly.
package goinertia

import (
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

type (
	Inertia                                  = fiberadapter.Inertia
	Option                                   = fiberadapter.Option
	CSRFTokenProvider                        = fiberadapter.CSRFTokenProvider
	CSRFTokenCheckProvider                   = fiberadapter.CSRFTokenCheckProvider
	SessionStore                             = fiberadapter.SessionStore
	FiberSessionStore                        = fiberadapter.FiberSessionStore
	SessionAdapter[T FiberSessionStore]      = fiberadapter.SessionAdapter[T]
	FiberSessionAdapter[T FiberSessionStore] = fiberadapter.FiberSessionAdapter[T]
)

const (
	ContextKeyProps                 = core.ContextKeyProps
	ContextKeyViewData              = core.ContextKeyViewData
	ContextKeyPageMeta              = core.ContextKeyPageMeta
	HeaderInertia                   = core.HeaderInertia
	HeaderLocation                  = core.HeaderLocation
	HeaderVersion                   = core.HeaderVersion
	HeaderPartialComponent          = core.HeaderPartialComponent
	HeaderPartialOnly               = core.HeaderPartialOnly
	HeaderPartialExcept             = core.HeaderPartialExcept
	HeaderReset                     = core.HeaderReset
	HeaderErrorBag                  = core.HeaderErrorBag
	HeaderExceptOnceProps           = core.HeaderExceptOnceProps
	HeaderInfiniteScrollMergeIntent = core.HeaderInfiniteScrollMergeIntent
	HeaderPrecognition              = core.HeaderPrecognition
	HeaderPrecognitionValidateOnly  = core.HeaderPrecognitionValidateOnly
	HeaderPrecognitionSuccess       = core.HeaderPrecognitionSuccess
	ContextPropsErrors              = core.ContextPropsErrors
	ContextPropsOld                 = core.ContextPropsOld
	ContextPropsFlash               = core.ContextPropsFlash
	ContextPropsCSRFToken           = core.ContextPropsCSRFToken
)

type (
	PageDTO          = core.PageDTO
	SsrDTO           = core.SsrDTO
	ScrollPropConfig = core.ScrollPropConfig
	OncePropConfig   = core.OncePropConfig
	LazyProp         = core.LazyProp
	DeferredProp     = core.DeferredProp
	OptionalProp     = core.OptionalProp
	AlwaysProp       = core.AlwaysProp
	MergeProp        = core.MergeProp
	MergeTarget      = core.MergeTarget
	NestedMergeProp  = core.NestedMergeProp
	ScrollProp       = core.ScrollProp
	OnceProp         = core.OnceProp
	OnceOption       = core.OnceOption
)

var (
	Defer                    = core.Defer
	Optional                 = core.Optional
	Always                   = core.Always
	MergeAt                  = core.MergeAt
	AppendAt                 = core.AppendAt
	PrependAt                = core.PrependAt
	Merge                    = core.Merge
	Prepend                  = core.Prepend
	DeepMerge                = core.DeepMerge
	Scroll                   = core.Scroll
	Once                     = core.Once
	WithOnceFresh            = core.WithOnceFresh
	WithOnceRefreshOnPartial = core.WithOnceRefreshOnPartial
	WithOnceKey              = core.WithOnceKey
	WithOnceExpiresAt        = core.WithOnceExpiresAt
)

type (
	ValidationErrors = core.ValidationErrors
	Error            = core.Error
	FlashLevel       = core.FlashLevel
	FlashError       = core.FlashError
	ValidationError  = core.ValidationError
)

var (
	ErrInvalidContextViewData = core.ErrInvalidContextViewData
	ErrBadSsrStatusCode       = core.ErrBadSsrStatusCode
	ErrBaseURLEmpty           = core.ErrBaseURLEmpty
	ErrNillable               = core.ErrNillable
	ErrInternal               = core.ErrInternal
	NewError                  = core.NewError
	NewFlashError             = core.NewFlashError
	NewValidationError        = core.NewValidationError
)

type (
	Logger    = core.Logger
	SSRClient = core.SSRClient
	SSRConfig = core.SSRConfig
)

var NewLoggerAdapter = core.NewLoggerAdapter

type LoggerAdapter = core.LoggerAdapter

const (
	DefaultSSRURL          = core.DefaultSSRURL
	DefaultSSRTimeout      = core.DefaultSSRTimeout
	DefaultCacheTTL        = core.DefaultCacheTTL
	DefaultCacheMaxEntries = core.DefaultCacheMaxEntries
	DefaultSSRMaxRetries   = core.DefaultSSRMaxRetries
	DefaultSSRRetryDelay   = core.DefaultSSRRetryDelay
)

var (
	DefaultCanExpose              = core.DefaultCanExpose
	DefaultCustomGettingError     = core.DefaultCustomGettingError
	DefaultCustomErrorDetails     = core.DefaultCustomErrorDetails
	WithFS                        = fiberadapter.WithFS
	WithPublicFS                  = fiberadapter.WithPublicFS
	WithRootTemplate              = fiberadapter.WithRootTemplate
	WithRootHotTemplate           = fiberadapter.WithRootHotTemplate
	WithRootErrorTemplate         = fiberadapter.WithRootErrorTemplate
	WithAssetVersion              = fiberadapter.WithAssetVersion
	WithSessionStore              = fiberadapter.WithSessionStore
	WithLogger                    = fiberadapter.WithLogger
	WithSetSharedFuncMap          = fiberadapter.WithSetSharedFuncMap
	WithSharedViewData            = fiberadapter.WithSharedViewData
	WithSharedProps               = fiberadapter.WithSharedProps
	WithCanExposeDetails          = fiberadapter.WithCanExposeDetails
	WithCustomErrorGettingHandler = fiberadapter.WithCustomErrorGettingHandler
	WithCustomErrorDetailsHandler = fiberadapter.WithCustomErrorDetailsHandler
	WithCSRFTokenProvider         = fiberadapter.WithCSRFTokenProvider
	WithCSRFTokenCheckProvider    = fiberadapter.WithCSRFTokenCheckProvider
	WithCSRFPropName              = fiberadapter.WithCSRFPropName
	WithSSRConfig                 = fiberadapter.WithSSRConfig
	WithDevMode                   = fiberadapter.WithDevMode
	WithPrecognitionVary          = fiberadapter.WithPrecognitionVary
	Redirect                      = fiberadapter.Redirect
	RedirectExternal              = fiberadapter.RedirectExternal
	IsPrecognition                = fiberadapter.IsPrecognition
	Must                          = fiberadapter.Must
)

// New preserves legacy lazy callbacks receiving fiber.Ctx.
func New(baseURL string, opts ...Option) *Inertia {
	return fiberadapter.NewLegacy(baseURL, legacyDefaults(opts)...)
}

// NewWithValidation preserves legacy callbacks and validates templates.
func NewWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	return fiberadapter.NewLegacyWithValidation(baseURL, legacyDefaults(opts)...)
}

func legacyDefaults(opts []Option) []Option {
	result := make([]Option, 0, 3+len(opts))
	result = append(result,
		WithCanExposeDetails(DefaultCanExpose),
		WithCustomErrorGettingHandler(DefaultCustomGettingError),
		WithCustomErrorDetailsHandler(DefaultCustomErrorDetails),
	)
	return append(result, opts...)
}

// NewFiberSessionAdapter releases Fiber's raw Store sessions; custom stores keep their ownership.
func NewFiberSessionAdapter[T FiberSessionStore](store SessionAdapter[T]) *FiberSessionAdapter[T] {
	return fiberadapter.NewFiberSessionAdapter(store)
}

// NewFiberSessionAdapterWithRelease explicitly owns custom raw sessions.
func NewFiberSessionAdapterWithRelease[T FiberSessionStore](store SessionAdapter[T], release func(T)) *FiberSessionAdapter[T] {
	return fiberadapter.NewFiberSessionAdapterWithRelease(store, release)
}

const FlashLevelSuccess = core.FlashLevelSuccess

const FlashLevelInfo = core.FlashLevelInfo

const FlashLevelWarning = core.FlashLevelWarning

const FlashLevelError = core.FlashLevelError

// MiddlewareSessionAdapter borrows Fiber middleware-owned sessions.
type MiddlewareSessionAdapter = fiberadapter.MiddlewareSessionAdapter

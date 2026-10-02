package fiberadapter

import "github.com/assurrussa/goinertia/core"

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
	ScrollProp       = core.ScrollProp
	OnceProp         = core.OnceProp
	OnceOption       = core.OnceOption
)

var (
	Defer             = core.Defer
	Optional          = core.Optional
	Always            = core.Always
	Merge             = core.Merge
	Prepend           = core.Prepend
	DeepMerge         = core.DeepMerge
	Scroll            = core.Scroll
	Once              = core.Once
	WithOnceKey       = core.WithOnceKey
	WithOnceExpiresAt = core.WithOnceExpiresAt
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
	DefaultCanExpose          = core.DefaultCanExpose
	DefaultCustomGettingError = core.DefaultCustomGettingError
	DefaultCustomErrorDetails = core.DefaultCustomErrorDetails
)

const FlashLevelSuccess = core.FlashLevelSuccess

const FlashLevelInfo = core.FlashLevelInfo

const FlashLevelWarning = core.FlashLevelWarning

const FlashLevelError = core.FlashLevelError

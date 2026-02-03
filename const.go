package goinertia

type contextKey string

// Context.
const (
	// ContextKeyProps key.
	ContextKeyProps = contextKey("props")
	// ContextKeyViewData key.
	ContextKeyViewData = contextKey("viewData")
	// ContextKeyPageMeta key.
	ContextKeyPageMeta = contextKey("pageMeta")
)

// Header.
const (
	// HeaderInertia header.
	HeaderInertia = "X-Inertia"
	// HeaderLocation header.
	HeaderLocation = "X-Inertia-Location"
	// HeaderVersion header.
	HeaderVersion = "X-Inertia-Version"
	// HeaderPartialComponent header.
	HeaderPartialComponent = "X-Inertia-Partial-Component"
	// HeaderPartialOnly header.
	HeaderPartialOnly = "X-Inertia-Partial-Data"
	// HeaderPartialExcept header.
	HeaderPartialExcept = "X-Inertia-Partial-Except"
	// HeaderReset header.
	HeaderReset = "X-Inertia-Reset"
	// HeaderErrorBag header.
	HeaderErrorBag = "X-Inertia-Error-Bag"
	// HeaderExceptOnceProps header.
	HeaderExceptOnceProps = "X-Inertia-Except-Once-Props"
	// HeaderInfiniteScrollMergeIntent header.
	HeaderInfiniteScrollMergeIntent = "X-Inertia-Infinite-Scroll-Merge-Intent"
)

// Keys.
const (
	ContextPropsErrors    = "errors"
	ContextPropsOld       = "old"
	ContextPropsFlash     = "flash"
	ContextPropsCSRFToken = "csrf_token"
)

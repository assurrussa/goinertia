package goinertia

type contextKey string

// Context.
const (
	// ContextKeyProps key.
	ContextKeyProps = contextKey("props")
	// ContextKeyViewData key.
	ContextKeyViewData = contextKey("viewData")
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
)

// Keys.
const (
	ContextPropsErrors    = "errors"
	ContextPropsOld       = "old"
	ContextPropsFlash     = "flash"
	ContextPropsCSRFToken = "csrf_token"
)

package core

import (
	"context"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// IsWriteMethod identifies methods whose redirects must use 303.
func IsWriteMethod(method string) bool {
	return method == http.MethodPost || method == http.MethodPut || method == http.MethodPatch || method == http.MethodDelete
}

// IsFlashResponse identifies responses that persist request flash.
func IsFlashResponse(status int, location string) bool {
	return status == 301 || status == 302 || status == 303 || status == 307 || status == 308 || (status == 409 && location != "")
}

// NormalizeRedirect applies Inertia write redirect policy.
func NormalizeRedirect(meta RequestMeta, status int) int {
	if meta.Inertia != "" && IsWriteMethod(meta.Method) && (status == 301 || status == 302) {
		return 303
	}
	return status
}

// VaryValue appends a token once, preserving existing values and wildcard.
func VaryValue(current, value string) string {
	if HasVaryToken(current, value) {
		return current
	}
	if current == "" {
		return value
	}
	return current + ", " + value
}

// HasVaryToken checks tokens without allocating a split slice.
func HasVaryToken(current, value string) bool {
	for {
		token, rest, more := strings.Cut(current, ",")
		token = strings.TrimSpace(token)
		if token == "*" || strings.EqualFold(token, value) {
			return true
		}
		if !more {
			return false
		}
		current = rest
	}
}

// Location constructs an asset conflict location.
func Location(base *url.URL, path string) string { return buildInertiaLocation(base, path) }

// AssetVersion returns the configured asset version.
func (i *Inertia) AssetVersion() string { return i.assetVersion }

// ConflictLocation returns the canonical URL for a version conflict.
func (i *Inertia) ConflictLocation(path string) string {
	return buildInertiaLocation(i.baseURLParsed, path)
}

// PrecognitionVary reports the configured Vary policy.
func (i *Inertia) PrecognitionVary() bool { return i.precognitionVary }

// PrecognitionErrors returns filtered request validation errors.
func (i *Inertia) PrecognitionErrors(s *State) ValidationErrors {
	return filterValidationErrors(normalizeValidationErrors(s.Props[ContextPropsErrors]), parseHeaderList(s.Meta.ValidateOnly))
}

// ApplyErrorBag normalizes validation errors for the request's named bag.
func (i *Inertia) ApplyErrorBag(s *State, p *PageDTO) { i.applyErrorBag(s, p) }

// NewSSRClient returns an injectable, concurrency-safe HTTP transport with owned bodies.
func NewSSRClient(client *http.Client) SSRClient {
	if client == nil {
		client = &http.Client{}
	}
	return &defaultSSRClient{client: client}
}

// Marshal exposes the default JSON template function.
func Marshal(v any) (template.JS, error) { return marshal(v) }

// Raw exposes the trusted HTML template function.
func Raw(v any) (template.HTML, error) { return raw(v) }

// Asset exposes the default asset template function.
func Asset(v string) (string, error) { return asset(v) }

// AppendUnique returns a slice containing the given value once.
func AppendUnique(v []string, s string) []string { return appendUnique(v, s) }

// NormalizeValidationErrors accepts the supported validation payloads.
func NormalizeValidationErrors(v any) ValidationErrors { return normalizeValidationErrors(v) }

// FlattenValidationErrors keeps the first error for each field.
func FlattenValidationErrors(v any) map[string]string { return flattenValidationErrors(v) }

// ParseBaseURL validates a base URL.
func ParseBaseURL(v string) *url.URL { return parseInertiaBaseURL(v) }

// SleepContext waits for retry delay or cancellation.
func SleepContext(ctx context.Context, d time.Duration) error { return sleeper(ctx, d) }

// BaseURL returns the normalized configured origin.
func (i *Inertia) BaseURL() string { return i.baseURL }

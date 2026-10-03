package fiberadapter

import (
	"fmt"

	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia/core"
)

const errorMessageKey = "message"

// WithProtocolVersion selects the wire contract; v2 remains the default.
func WithProtocolVersion(version core.ProtocolVersion) Option {
	return func(i *Inertia) { core.WithProtocolVersion(version)(i.Inertia) }
}

// WithSSRErrorPolicy overrides protocol-specific SSR failure behavior.
func WithSSRErrorPolicy(policy core.SSRErrorPolicy) Option {
	return func(i *Inertia) { core.WithSSRErrorPolicy(policy)(i.Inertia) }
}

// WithSSRFailureHandler reports failed SSR attempts after retries.
func WithSSRFailureHandler(handler core.SSRFailureHandler) Option {
	return func(i *Inertia) { core.WithSSRFailureHandler(handler)(i.Inertia) }
}

// WithViteSSR enables or disables hot-file-based Vite SSR routing.
func WithViteSSR(enabled bool) Option {
	return func(i *Inertia) { core.WithViteSSR(enabled)(i.Inertia) }
}

// WithPreserveFragment preserves the client visit's fragment in v3 responses.
func (i *Inertia) WithPreserveFragment(c fiber.Ctx, preserve bool) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithPreserveFragment(s, preserve)
	i.syncState(c, s)
}

// WithSSRDisabled suppresses SSR only for the current request.
func (i *Inertia) WithSSRDisabled(c fiber.Ctx, disabled bool) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithSSRDisabled(s, disabled)
	i.syncState(c, s)
}

// RenderWithStatus renders a page with an explicit HTTP status, including v3
// error pages consumed by the client's httpException flow.
func (i *Inertia) RenderWithStatus(c fiber.Ctx, status int, component string, props map[string]any) error {
	if status < 200 || status > 599 {
		return fmt.Errorf("invalid page status: %d", status)
	}
	c.Status(status)
	return i.Render(c, component, props)
}

func isPrefetch(c fiber.Ctx) bool {
	return core.IsPrefetch(c.Get("Purpose"), c.Get("Sec-Purpose"), c.Get("X-Moz"))
}

// WithPreserveBigIntegers enables v3 string encoding with bigint path metadata.
func WithPreserveBigIntegers(enabled bool) Option {
	return func(i *Inertia) { core.WithPreserveBigIntegers(enabled)(i.Inertia) }
}

// WithAllValidationErrors preserves all field messages in v3 page props and
// flash sessions. The compatible WithValidationErrors keeps first-message behavior.
func (i *Inertia) WithAllValidationErrors(c fiber.Ctx, validation core.ValidationErrors) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithAllValidationErrors(s, validation)
	i.syncState(c, s)
}

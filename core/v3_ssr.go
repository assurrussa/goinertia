package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// SSRErrorPolicy controls rendering after an SSR failure. The protocol default
// propagates errors in v2 and reports them with client-side fallback in v3.
type SSRErrorPolicy uint8

const (
	SSRErrorDefault SSRErrorPolicy = iota
	SSRErrorPropagate
	SSRErrorFallback
)

// SSRFailureHandler receives server-side diagnostics after an SSR attempt fails.
// It runs once per failed render, after retries, and must support concurrent calls.
// The error is not sent to the browser.
type SSRFailureHandler func(context.Context, error)

// WithSSRErrorPolicy overrides the protocol's default SSR error behavior without
// changing the source-compatible SSRConfig layout. Configure before serving.
func WithSSRErrorPolicy(policy SSRErrorPolicy) Option {
	return func(i *Inertia) { i.ssrErrorPolicy = policy }
}

// WithSSRFailureHandler installs an optional error reporter alongside logging.
func WithSSRFailureHandler(handler SSRFailureHandler) Option {
	return func(i *Inertia) { i.ssrFailureHandler = handler }
}

// WithViteSSR selects the Vite /__inertia_ssr endpoint when development mode has
// a hot file. V3 enables this by default; SSR still requires WithSSRConfig.
// False keeps the configured production endpoint even while Vite is running.
func WithViteSSR(enabled bool) Option {
	return func(i *Inertia) { i.viteSSR = &enabled }
}

// WithSSRDisabled disables SSR only for this request. It never mutates the
// shared SSR client, cache or startup configuration.
func (i *Inertia) WithSSRDisabled(state *State, disabled bool) {
	if state == nil {
		return
	}
	if state.pageMeta == nil {
		state.pageMeta = &pageMeta{}
	}
	state.pageMeta.ssrDisabled = disabled
}

func (i *Inertia) renderSSR(state *State, page any) (*SsrDTO, error) {
	if !i.IsSSREnabled() || (state.pageMeta != nil && state.pageMeta.ssrDisabled) {
		return nil, nil //nolint:nilnil // Nil explicitly selects client-side rendering.
	}
	ssr, err := i.processSSR(state.Context, page)
	if err == nil {
		return ssr, nil
	}
	// Always report the final failure, including cancellation or retry delay errors.
	if logger, ok := i.logger.(*LoggerAdapter); i.IsProtocolV3() && (i.logger == nil || (ok && logger.logger == nil)) {
		slog.ErrorContext(state.Context, "SSR rendering failed", "error", err)
	} else if i.logger != nil {
		i.logger.ErrorContext(state.Context, "SSR rendering failed", "error", err)
	}
	if i.ssrFailureHandler != nil {
		i.ssrFailureHandler(state.Context, err)
	}
	if contextErr := state.Context.Err(); contextErr != nil {
		return nil, contextErr
	}
	fallback := i.ssrErrorPolicy == SSRErrorFallback || (i.ssrErrorPolicy == SSRErrorDefault && i.IsProtocolV3())
	if !fallback {
		return nil, err
	}
	return nil, nil //nolint:nilnil // Reported failure intentionally falls back to client-side rendering.
}

func (i *Inertia) ssrEndpoint() (endpoint string, development bool) {
	useVite := i.IsProtocolV3()
	if i.viteSSR != nil {
		useVite = *i.viteSSR
	}
	if i.isDev && useVite {
		if hot := i.HotServerURL(); hot != "" {
			return strings.TrimRight(hot, "/") + "/__inertia_ssr", true
		}
	}
	return i.ssrConfig.URL, false
}

func validateSSRResponse(ssr *SsrDTO) error {
	if ssr == nil || strings.TrimSpace(ssr.Body) == "" {
		return errors.New("invalid SSR response: missing rendered body")
	}
	return nil
}

// SSRResponseError contains diagnostics returned by a v3 production or Vite SSR
// server. They are available to server-side reporters and are never page props.
type SSRResponseError struct {
	Status         int             `json:"-"`
	Message        string          `json:"error"`
	Type           string          `json:"type"`
	Hint           string          `json:"hint"`
	BrowserAPI     string          `json:"browserApi"`
	Stack          string          `json:"stack"`
	SourceLocation json.RawMessage `json:"sourceLocation"`
}

func (e *SSRResponseError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("SSR returned status %d: %s", e.Status, e.Message)
	}
	return fmt.Sprintf("SSR returned status %d", e.Status)
}

// Unwrap preserves errors.Is compatibility with the existing status sentinel.
func (e *SSRResponseError) Unwrap() error { return ErrBadSsrStatusCode }

func ssrResponseError(status int, body []byte) error {
	details := &SSRResponseError{}
	// Malformed error diagnostics do not mask the original HTTP failure.
	_ = json.Unmarshal(body, details)
	details.Status = status
	return details
}

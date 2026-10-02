package fiberadapter

import (
	"context"
	"html/template"
	"io/fs"

	"github.com/assurrussa/goinertia/core"
)

// Option configures the native Fiber adapter.
type Option func(*Inertia)

func WithFS(fs fs.FS) Option { return func(i *Inertia) { core.WithFS(fs)(i.Inertia) } }
func WithPublicFS(fs fs.ReadFileFS) Option {
	return func(i *Inertia) { core.WithPublicFS(fs)(i.Inertia) }
}

func WithRootTemplate(rootTemplate string) Option {
	return func(i *Inertia) { core.WithRootTemplate(rootTemplate)(i.Inertia) }
}

func WithRootHotTemplate(rootHotTemplate string) Option {
	return func(i *Inertia) { core.WithRootHotTemplate(rootHotTemplate)(i.Inertia) }
}

func WithRootErrorTemplate(rootErrorTemplate string) Option {
	return func(i *Inertia) { core.WithRootErrorTemplate(rootErrorTemplate)(i.Inertia) }
}

func WithAssetVersion(assetVersion string) Option {
	return func(i *Inertia) { core.WithAssetVersion(assetVersion)(i.Inertia) }
}

func WithSessionStore(sessionStore SessionStore) Option {
	return func(i *Inertia) {
		i.sessionStore = sessionStore
	}
}

func WithLogger(logger Logger) Option {
	return func(i *Inertia) { core.WithLogger(logger)(i.Inertia); i.logger = logger }
}

func WithSetSharedFuncMap(data template.FuncMap) Option {
	return func(i *Inertia) { core.WithSetSharedFuncMap(data)(i.Inertia) }
}

func WithSharedViewData(data map[string]any) Option {
	return func(i *Inertia) { core.WithSharedViewData(data)(i.Inertia) }
}

func WithSharedProps(data map[string]any) Option {
	return func(i *Inertia) { core.WithSharedProps(data)(i.Inertia) }
}

func WithCanExposeDetails(fn func(ctx context.Context, headers map[string][]string) bool) Option {
	return func(i *Inertia) {
		i.canExposeDetails = fn
	}
}

// WithCustomErrorGettingHandler sets function callback for custom getting errors.
func WithCustomErrorGettingHandler(fn func(err error) *Error) Option {
	return func(i *Inertia) {
		i.customErrorGettingHandler = fn
	}
}

// WithCustomErrorDetailsHandler sets a callback to handler error details.
func WithCustomErrorDetailsHandler(fn func(errReturn *Error, isCanDetails bool) string) Option {
	return func(i *Inertia) {
		i.customErrorDetailsHandler = fn
	}
}

// WithCSRFTokenProvider registers a resolver that injects CSRF token into every rendered page.
func WithCSRFTokenProvider(provider CSRFTokenProvider) Option {
	return func(i *Inertia) {
		i.csrfTokenProvider = provider
	}
}

// WithCSRFTokenCheckProvider registers a resolver that check CSRF token into every rendered page.
func WithCSRFTokenCheckProvider(provider CSRFTokenCheckProvider) Option {
	return func(i *Inertia) {
		i.csrfTokenCheckProvider = provider
	}
}

// WithCSRFPropName overrides the prop key used when injecting CSRF token.
func WithCSRFPropName(prop string) Option {
	return func(i *Inertia) {
		core.WithCSRFPropName(prop)(i.Inertia)
		if prop != "" {
			i.csrfPropName = prop
		}
	}
}

func WithSSRConfig(cfg SSRConfig) Option {
	return func(i *Inertia) { core.WithSSRConfig(cfg)(i.Inertia) }
}
func WithDevMode() Option { return func(i *Inertia) { core.WithDevMode()(i.Inertia) } }
func WithPrecognitionVary(enabled bool) Option {
	return func(i *Inertia) { core.WithPrecognitionVary(enabled)(i.Inertia) }
}

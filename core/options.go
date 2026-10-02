package core

import (
	"context"
	"html/template"
	"io/fs"
	"net/http"
)

var (
	DefaultCanExpose = func(_ context.Context, _ map[string][]string) bool {
		return false
	}
	DefaultCustomGettingError = func(_ error) *Error {
		return nil
	}
	DefaultCustomErrorDetails = func(appErr *Error, isCanDetails bool) string {
		details := "Something went wrong. Try again later"
		switch {
		case isCanDetails:
			details = appErr.Error()
			if errCause := appErr.Unwrap(); errCause != nil {
				if addDetailErr := errCause.Error(); addDetailErr != details {
					details += ": " + addDetailErr
				}
			}
		default:
			code := appErr.Code
			switch {
			case code == http.StatusBadRequest:
				details = "Bad request"
			case code == http.StatusNotFound:
				details = "Page not found"
			case code == http.StatusForbidden:
				details = "Permission denied"
			case code == http.StatusUnauthorized:
				details = "Unauthorized"
			case code == 419:
				details = "The page expired, please try again"
			case code == http.StatusTooManyRequests:
				details = "Too many request"
			case code >= http.StatusInternalServerError:
				details = "Something went wrong. Try again later"
			}
		}

		return details
	}
)

type Option func(*Inertia)

func WithFS(fs fs.FS) Option {
	return func(i *Inertia) {
		i.templateFS = fs
	}
}

func WithPublicFS(fs fs.ReadFileFS) Option {
	return func(i *Inertia) {
		i.publicFS = fs
	}
}

func WithRootTemplate(rootTemplate string) Option {
	return func(i *Inertia) {
		i.rootTemplate = rootTemplate
	}
}

func WithRootHotTemplate(rootHotTemplate string) Option {
	return func(i *Inertia) {
		i.rootHotTemplate = rootHotTemplate
	}
}

func WithRootErrorTemplate(rootErrorTemplate string) Option {
	return func(i *Inertia) {
		i.rootErrorTemplate = rootErrorTemplate
	}
}

func WithAssetVersion(assetVersion string) Option {
	return func(i *Inertia) {
		i.assetVersion = assetVersion
	}
}

func WithLogger(logger Logger) Option {
	return func(i *Inertia) {
		i.logger = logger
	}
}

func WithSetSharedFuncMap(data template.FuncMap) Option {
	return func(i *Inertia) {
		for k, v := range data {
			i.sharedFuncMap[k] = v
		}
	}
}

func WithSharedViewData(data map[string]any) Option {
	return func(i *Inertia) {
		for k, v := range data {
			i.sharedViewData[k] = v
		}
	}
}

func WithSharedProps(data map[string]any) Option {
	return func(i *Inertia) {
		for k, v := range data {
			i.sharedProps[k] = v
		}
	}
}

// WithCanExposeDetails sets a callback to decide if current request may see error details in production.
// Useful to allow main admins to see full error messages.
func WithCSRFPropName(prop string) Option {
	return func(i *Inertia) {
		if prop == "" {
			return
		}

		old := i.csrfPropName
		i.csrfPropName = prop
		if old != "" && old != prop {
			delete(i.sharedProps, old)
		}
	}
}

// WithSSRConfig enables SSR with the provided config.
func WithSSRConfig(cfg SSRConfig) Option {
	return func(i *Inertia) {
		i.EnableSSR(cfg)
	}
}

// WithDevMode enables development mode.
// In this mode, the Vite hot file is checked on every request, allowing for dynamic starts/restarts of the Vite server.
func WithDevMode() Option {
	return func(i *Inertia) {
		i.isDev = true
	}
}

// WithPrecognitionVary controls whether "Vary: Precognition" is added to Inertia responses.
// Defaults to true to match the protocol recommendation.
func WithPrecognitionVary(enabled bool) Option {
	return func(i *Inertia) {
		i.precognitionVary = enabled
	}
}

// WithCSRFEnabled reserves the CSRF prop during partial reloads; adapters supply tokens.
func WithCSRFEnabled(enabled bool) Option { return func(i *Inertia) { i.csrfEnabled = enabled } }

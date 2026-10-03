// Package nethttp implements Inertia using the native net/http request lifecycle.
package nethttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/goccy/go-json"

	"github.com/assurrussa/goinertia/core"
)

const errorMessageKey = "message"

// SessionStore owns its session lifecycle. Mutations may set cookies on w and
// must complete before response commitment. Hosts own storage and rotation.
type SessionStore interface {
	Flash(w http.ResponseWriter, r *http.Request, key string, value any) error
	GetFlash(w http.ResponseWriter, r *http.Request, key string) (any, error)
}

// CSRFTokenProvider resolves a host-owned token during page rendering.
type CSRFTokenProvider func(http.ResponseWriter, *http.Request) (string, error)

// CSRFTokenCheckProvider verifies write requests before invoking the handler.
type CSRFTokenCheckProvider func(*http.Request) error

// Option configures the native HTTP adapter.
type Option func(*Inertia)

// Inertia combines a neutral engine with native HTTP session/CSRF hooks.
type Inertia struct {
	*core.Inertia
	sessionStore SessionStore
	csrfProvider CSRFTokenProvider
	csrfChecker  CSRFTokenCheckProvider
	csrfPropName string
	logger       core.Logger
}

// WithCoreOptions configures neutral rendering policy.
func WithCoreOptions(opts ...core.Option) Option {
	return func(i *Inertia) {
		for _, opt := range opts {
			opt(i.Inertia)
		}
	}
}

// WithSessionStore attaches a host-owned session implementation.
func WithSessionStore(s SessionStore) Option { return func(i *Inertia) { i.sessionStore = s } }

// WithCSRFTokenProvider enables token injection, independently of checking.
func WithCSRFTokenProvider(fn CSRFTokenProvider) Option {
	return func(i *Inertia) { i.csrfProvider = fn; core.WithCSRFEnabled(fn != nil)(i.Inertia) }
}

// WithCSRFTokenCheckProvider enables token checking, independently of injection.
func WithCSRFTokenCheckProvider(fn CSRFTokenCheckProvider) Option {
	return func(i *Inertia) { i.csrfChecker = fn }
}

// WithCSRFPropName configures the injected token's prop name.
func WithCSRFPropName(name string) Option {
	return func(i *Inertia) {
		if name != "" {
			i.csrfPropName = name
			core.WithCSRFPropName(name)(i.Inertia)
		}
	}
}

// WithLogger configures adapter and core logging.
func WithLogger(log core.Logger) Option {
	return func(i *Inertia) { i.logger = log; core.WithLogger(log)(i.Inertia) }
}

// New constructs an HTTP adapter with delayed template parsing.
func New(baseURL string, opts ...Option) *Inertia {
	i := &Inertia{Inertia: core.New(baseURL), csrfPropName: core.ContextPropsCSRFToken, logger: core.NewLoggerAdapter(nil)}
	for _, opt := range opts {
		opt(i)
	}
	i.NormalizeConfig()
	return i
}

// NewWithValidation validates the origin and parses templates at startup.
func NewWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	i := New(baseURL, opts...)
	if core.ParseBaseURL(i.BaseURL()) == nil {
		return nil, core.ErrBaseURLEmpty
	}
	if err := i.ParseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return i, nil
}

type (
	stateKey   struct{}
	requestKey struct{}
	requestRef struct{ request *http.Request }
)

// Request returns the native request during the active callback lifecycle.
func Request(ctx context.Context) (*http.Request, bool) {
	ref, ok := ctx.Value(requestKey{}).(*requestRef)
	if !ok {
		return nil, false
	}
	return ref.request, true
}

// RequestMeta reads only the protocol inputs used by the engine.
func RequestMeta(r *http.Request) core.RequestMeta {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return core.RequestMeta{
		Method:            r.Method,
		URL:               r.URL.RequestURI(),
		BaseURL:           scheme + "://" + r.Host,
		Referer:           r.Referer(),
		Inertia:           r.Header.Get(core.HeaderInertia),
		Version:           r.Header.Get(core.HeaderVersion),
		PartialComponent:  r.Header.Get(core.HeaderPartialComponent),
		PartialOnly:       r.Header.Get(core.HeaderPartialOnly),
		PartialExcept:     r.Header.Get(core.HeaderPartialExcept),
		Reset:             r.Header.Get(core.HeaderReset),
		ErrorBag:          r.Header.Get(core.HeaderErrorBag),
		ExceptOnceProps:   r.Header.Get(core.HeaderExceptOnceProps),
		ScrollMergeIntent: r.Header.Get(core.HeaderInfiniteScrollMergeIntent),
		Precognition:      strings.TrimSpace(r.Header.Get(core.HeaderPrecognition)),
		ValidateOnly:      r.Header.Get(core.HeaderPrecognitionValidateOnly),
		CacheControl:      r.Header.Get("Cache-Control"),
	}
}

// State returns middleware-installed concrete request state. It returns nil
// outside this adapter's Middleware, so hosts can detect missing wiring.
func State(r *http.Request) *core.State {
	s, _ := r.Context().Value(stateKey{}).(*core.State)
	if s != nil {
		// Downstream middleware may derive authentication/deadline context.
		// Bind callbacks and SSR to the request actually passed by the handler.
		s.Context = r.Context()
		if ref, ok := r.Context().Value(requestKey{}).(*requestRef); ok {
			ref.request = r
		}
	}
	return s
}

// Middleware installs request state and normalizes headers, status and session
// cookies immediately before commitment. Arbitrary bodies stream directly.
func (i *Inertia) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ref := &requestRef{}
		ctx := context.WithValue(r.Context(), requestKey{}, ref)
		s := core.NewState(ctx, RequestMeta(r))
		r = r.WithContext(context.WithValue(ctx, stateKey{}, s))
		ref.request = r
		s.Context = r.Context()
		writer := &responseWriter{
			ResponseWriter: w,
			before: func(status int) int {
				return i.beforeResponse(w, ref.request, s, status)
			},
		}
		wrapped := preserveCapabilities(writer)
		if i.csrfChecker != nil && core.IsWriteMethod(r.Method) {
			if err := i.csrfChecker(r); err != nil {
				i.HandleError(wrapped, r, err)
				writer.finish()
				return
			}
		}
		if s.Meta.Inertia != "" && r.Method == http.MethodGet && s.Meta.Version != i.AssetVersion() && s.Meta.Precognition == "" {
			wrapped.Header().Set(core.HeaderLocation, i.ConflictLocation(s.Meta.URL))
			if i.IsProtocolV3() {
				wrapped.Header().Set(core.HeaderVersion, i.AssetVersion())
			}
			wrapped.WriteHeader(http.StatusConflict)
			return
		}
		next.ServeHTTP(wrapped, r)
		writer.finish()
	})
}

func (i *Inertia) beforeResponse(w http.ResponseWriter, r *http.Request, state *core.State, status int) int {
	h := w.Header()
	appendVary(h, core.HeaderInertia)
	if i.PrecognitionVary() {
		appendVary(h, core.HeaderPrecognition)
	}
	meta := state.Meta
	if (meta.Inertia != "" || meta.Precognition != "") && strings.Contains(strings.ToLower(meta.CacheControl), "no-cache") {
		h.Set("Cache-Control", "no-cache")
	}
	flashResponse := core.IsFlashResponse(status, h.Get(core.HeaderLocation)) ||
		(status == http.StatusConflict && h.Get(core.HeaderRedirect) != "")
	if i.sessionStore != nil && meta.Precognition == "" && flashResponse {
		if data := state.FlashToPersist(); len(data) > 0 {
			if err := i.sessionStore.Flash(w, r, string(core.ContextKeyProps), data); err != nil {
				i.logger.ErrorContext(r.Context(), "could not set flash session props", "error", err)
			}
		}
	}
	if i.IsFragmentRedirect(meta.Inertia, isPrefetch(r), status, h.Get("Location")) {
		h.Set(core.HeaderRedirect, h.Get("Location"))
		h.Del("Location")
		h.Del(core.HeaderInertia)
		return http.StatusConflict
	}
	return core.NormalizeRedirect(meta, status)
}

// Handler adapts an error-returning native HTTP handler without a framework bridge.
func (i *Inertia) Handler(fn func(http.ResponseWriter, *http.Request) error) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			i.HandleError(w, r, err)
		}
	})
}

// Render creates JSON or HTML using native request state.
func (i *Inertia) Render(w http.ResponseWriter, r *http.Request, component string, props map[string]any) error {
	return i.renderWithStatus(w, r, 0, component, props)
}

// RenderWithStatus renders an Inertia page with an explicit HTTP status. The
// page is fully serialized before status, headers or body are committed.
func (i *Inertia) RenderWithStatus(
	w http.ResponseWriter, r *http.Request, status int, component string, props map[string]any,
) error {
	if status < 200 || status > 599 {
		return fmt.Errorf("invalid page status: %d", status)
	}
	return i.renderWithStatus(w, r, status, component, props)
}

func (i *Inertia) renderWithStatus(
	w http.ResponseWriter, r *http.Request, status int, component string, props map[string]any,
) error {
	s := State(r)
	if s == nil {
		return errors.New("inertia: net/http Middleware is required")
	}
	if s.Meta.Precognition != "" {
		return i.renderPrecognition(w, i.PrecognitionErrors(s))
	}
	if i.sessionStore != nil {
		value, err := i.sessionStore.GetFlash(w, r, string(core.ContextKeyProps))
		if err == nil {
			s.FlashData, _ = value.(map[string]any)
		}
	}
	if i.csrfProvider != nil {
		token, err := i.csrfProvider(w, r)
		if err != nil {
			i.logger.WarnContext(r.Context(), "failed to evaluate prop", "key", i.csrfPropName, "error", err)
		} else {
			i.WithProp(s, i.csrfPropName, token)
		}
	}
	page, err := i.BuildPage(s, component, props) //nolint:contextcheck // Concrete state carries the lifecycle context.
	if err != nil {
		return fmt.Errorf("could not build page: %w", err)
	}
	if s.Meta.Inertia != "" {
		data, err := core.MarshalPageWithState(page, s)
		if err != nil {
			return fmt.Errorf("error marshaling page: %w", err)
		}
		w.Header().Set(core.HeaderInertia, "true")
		w.Header().Set("Content-Type", "application/json")
		if status != 0 {
			w.WriteHeader(status)
		}
		_, err = w.Write(data) //nolint:gosec // Serialized JSON has application/json content type.
		return err
	}
	data, err := i.RenderHTML(s, page)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, err = w.Write(data) //nolint:gosec // html/template escapes page data; SSR HTML is trusted host output.
	return err
}

func (i *Inertia) renderPrecognition(w http.ResponseWriter, errs core.ValidationErrors) error {
	appendVary(w.Header(), core.HeaderPrecognition)
	w.Header().Set(core.HeaderPrecognition, "true")
	if len(errs) == 0 {
		w.Header().Set(core.HeaderPrecognitionSuccess, "true")
		w.WriteHeader(http.StatusNoContent)
		return nil
	}
	data, err := json.Marshal(map[string]any{"errors": errs})
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_, err = w.Write(data)
	return err
}

// Redirect applies native Inertia external/internal redirect policy.
func (i *Inertia) Redirect(w http.ResponseWriter, r *http.Request, target string) error {
	State(r)
	if target == "" || target == "/" {
		target = RequestMeta(r).BaseURL
	}
	if r.Header.Get(core.HeaderInertia) != "" && i.IsExternalRedirect(target) {
		return i.RedirectExternal(w, r, target)
	}
	if i.IsFragmentRedirect(r.Header.Get(core.HeaderInertia), isPrefetch(r), http.StatusFound, target) {
		w.Header().Set(core.HeaderRedirect, target)
		w.WriteHeader(http.StatusConflict)
		return nil
	}
	http.Redirect(w, r, target, http.StatusFound) //nolint:gosec // Hosts validate redirect targets.
	return nil
}

// RedirectExternal forces an Inertia location visit.
func (i *Inertia) RedirectExternal(w http.ResponseWriter, r *http.Request, target string) error {
	State(r)
	if target == "" || target == "/" {
		target = RequestMeta(r).BaseURL
	}
	w.Header().Set(core.HeaderLocation, target)
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusConflict)
	return nil
}

// RedirectBack uses the referrer or the original request URI.
func (i *Inertia) RedirectBack(w http.ResponseWriter, r *http.Request) error {
	target := r.Referer()
	if target == "" {
		target = r.URL.RequestURI()
	}
	return i.Redirect(w, r, target)
}

// HandleError renders a safe error response before commitment. Already committed
// streaming responses are left intact and the error is logged.
func (i *Inertia) HandleError(w http.ResponseWriter, r *http.Request, err error) {
	if committed(w) {
		i.logger.ErrorContext(r.Context(), "error after response commitment", "error", err)
		return
	}
	appErr := core.ErrInternal
	var value *core.Error
	if errors.As(err, &value) {
		appErr = value
	}
	var validation *core.ValidationError
	if errors.As(err, &validation) {
		appErr = core.NewError(validation.StatusCode(), validation.Error()).CloneValidationError(validation)
	}
	s := State(r)
	if s != nil && s.Meta.Precognition != "" {
		errs := appErr.ValidationErrors()
		i.WithValidationErrors(s, errs)
		errs = i.PrecognitionErrors(s)
		if len(errs) > 0 {
			_ = i.renderPrecognition(w, errs)
			return
		}
		appendVary(w.Header(), core.HeaderPrecognition)
		w.Header().Set(core.HeaderPrecognition, "true")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(appErr.Code)
		_ = json.NewEncoder(w).Encode(map[string]any{errorMessageKey: appErr.Message})
		return
	}
	if i.IsProtocolV3() && s != nil && s.Meta.Inertia != "" && len(appErr.ValidationErrors()) == 0 {
		status := appErr.Code
		if status < http.StatusBadRequest || status > 599 {
			status = http.StatusInternalServerError
		}
		w.Header().Del(core.HeaderInertia)
		w.Header().Del(core.HeaderLocation)
		w.Header().Del(core.HeaderRedirect)
		w.Header().Del("Location")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{errorMessageKey: core.DefaultCustomErrorDetails(appErr, false)})
		return
	}
	if s != nil && (s.Meta.Inertia != "" || r.Method != http.MethodGet) {
		i.WithValidationErrors(s, appErr.ValidationErrors())
		i.WithFlashMessages(s, appErr.FlashErrors()...)
		if len(appErr.ValidationErrors()) == 0 {
			i.WithFlashError(s, core.DefaultCustomErrorDetails(appErr, false))
		}
		_ = i.RedirectBack(w, r)
		return
	}
	tmpl, templateErr := i.ErrorTemplate()
	if templateErr != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	data,
		templateErr := core.ExecuteTemplate(tmpl,
		map[string]any{
			"code":          appErr.Code,
			errorMessageKey: appErr.Message,
			"details": core.DefaultCustomErrorDetails(appErr,
				false),
		})
	if templateErr != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(appErr.Code)
	_, _ = w.Write(data)
}

func isPrefetch(r *http.Request) bool {
	return core.IsPrefetch(r.Header.Get("Purpose"), r.Header.Get("Sec-Purpose"), r.Header.Get("X-Moz"))
}

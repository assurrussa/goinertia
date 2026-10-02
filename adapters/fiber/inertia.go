// Package fiberadapter implements the native Fiber lifecycle for Inertia.
package fiberadapter

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia/core"
)

// Inertia owns Fiber lifecycle, session and CSRF hooks; the embedded engine
// contains framework-neutral rendering policy.
type Inertia struct {
	*core.Inertia
	sessionStore              SessionStore
	logger                    Logger
	canExposeDetails          func(context.Context, map[string][]string) bool
	customErrorDetailsHandler func(*Error, bool) string
	customErrorGettingHandler func(error) *Error
	csrfTokenCheckProvider    CSRFTokenCheckProvider
	csrfTokenProvider         CSRFTokenProvider
	csrfPropName              string
	legacy                    bool
}

// Must panics on constructor failure.
func Must(inr *Inertia, err error) *Inertia {
	if err != nil {
		panic(err)
	}
	return inr
}

// New constructs a native adapter whose callbacks receive lifecycle context.
func New(baseURL string, opts ...Option) *Inertia { return newInertia(baseURL, false, opts...) }

// NewLegacy constructs the root facade's historical callback behavior.
func NewLegacy(baseURL string, opts ...Option) *Inertia { return newInertia(baseURL, true, opts...) }

func newInertia(baseURL string, legacy bool, opts ...Option) *Inertia {
	inr := &Inertia{
		Inertia:                   core.New(baseURL),
		logger:                    core.NewLoggerAdapter(nil),
		canExposeDetails:          DefaultCanExpose,
		customErrorGettingHandler: DefaultCustomGettingError,
		customErrorDetailsHandler: DefaultCustomErrorDetails,
		csrfPropName:              ContextPropsCSRFToken,
		legacy:                    legacy,
	}
	for _, opt := range opts {
		opt(inr)
	}
	inr.NormalizeConfig()
	inr.registerCSRFSharedProp()
	return inr
}

// NewWithValidation constructs a native adapter and parses templates at startup.
func NewWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	return validate(New(baseURL, opts...))
}

// NewLegacyWithValidation validates the compatibility facade.
func NewLegacyWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	return validate(NewLegacy(baseURL, opts...))
}

func validate(i *Inertia) (*Inertia, error) {
	if core.ParseBaseURL(i.BaseURL()) == nil {
		return nil, ErrBaseURLEmpty
	}
	if err := i.ParseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}
	return i, nil
}

type (
	fiberContextKey struct{}
	stateKey        struct{ owner *Inertia }
	helperKey       struct{ owner *Inertia }
	propsOwnerKey   struct{}
	viewOwnerKey    struct{}
	metaOwnerKey    struct{}
)

// Retain the last synchronized map bindings, not copies of their contents.
// Rebinding State (including assigning nil) takes precedence over stale Locals;
// a changed Locals binding is adopted when State's binding is unchanged.
type requestState struct {
	core.State
	initialContext lifecycleContext
	requestValues
}

// Each binding record belongs to one manager-scoped State/helper key. Its
// identity marks published Locals ownership without storing another owner.
// Helper-only requests retain just bindings. Full request
// metadata and lifecycle context are still captured only by State/Render.
type requestValues struct {
	props, view map[string]any
	invalidView bool
	pageMeta    any
}

type lifecycleContext struct {
	context.Context //nolint:containedctx // Immutable adapter-owned request lifecycle snapshot.
	fiber           fiber.Ctx
}

func (c *lifecycleContext) Value(key any) any {
	if _, ok := key.(fiberContextKey); ok {
		return c.fiber
	}
	return c.Context.Value(key)
}

// Context returns the lifecycle context with a typed Fiber helper. The helper
// may only be used during this request; Fiber contexts are pooled.
func Context(c fiber.Ctx) context.Context {
	return context.WithValue(c.Context(), fiberContextKey{}, c)
}

// FromContext returns the Fiber helper during the active request lifecycle.
func FromContext(ctx context.Context) (fiber.Ctx, bool) {
	c, ok := ctx.Value(fiberContextKey{}).(fiber.Ctx)
	return c, ok
}

// RequestMeta copies only protocol inputs from the Fiber request.
func RequestMeta(c fiber.Ctx) core.RequestMeta {
	own := strings.Clone
	return core.RequestMeta{
		Method:            own(c.Method()),
		URL:               own(c.OriginalURL()),
		BaseURL:           own(c.BaseURL()),
		Referer:           own(c.Get(fiber.HeaderReferer)),
		Inertia:           own(c.Get(HeaderInertia)),
		Version:           own(c.Get(HeaderVersion)),
		PartialComponent:  own(c.Get(HeaderPartialComponent)),
		PartialOnly:       own(c.Get(HeaderPartialOnly)),
		PartialExcept:     own(c.Get(HeaderPartialExcept)),
		Reset:             own(c.Get(HeaderReset)),
		ErrorBag:          own(c.Get(HeaderErrorBag)),
		ExceptOnceProps:   own(c.Get(HeaderExceptOnceProps)),
		ScrollMergeIntent: own(c.Get(HeaderInfiniteScrollMergeIntent)),
		Precognition:      own(strings.TrimSpace(c.Get(HeaderPrecognition))),
		ValidateOnly:      own(c.Get(HeaderPrecognitionValidateOnly)),
		CacheControl:      own(c.Get(fiber.HeaderCacheControl)),
	}
}

func (i *Inertia) state(c fiber.Ctx) *core.State {
	if s, ok := c.Locals(stateKey{owner: i}).(*requestState); ok {
		i.refreshContext(c, &s.State)
		i.reconcileState(c, s)
		return &s.State
	}
	s := &requestState{}
	if i.legacy {
		s.State = *core.NewLegacyState(c.Context(), c, RequestMeta(c))
	} else {
		s.initialContext = lifecycleContext{Context: c.Context(), fiber: c}
		s.State = *core.NewState(&s.initialContext, RequestMeta(c))
	}
	if helpers, ok := c.Locals(helperKey{owner: i}).(*requestValues); ok {
		helpers.load(&s.State)
		i.reconcileBindings(c, &s.State, helpers)
	} else {
		i.readLocalState(c, &s.State)
	}
	s.captureBindings()
	i.publishValues(c, &s.State, &s.requestValues)
	c.Locals(stateKey{owner: i}, s)
	return &s.State
}

func (i *Inertia) refreshContext(c fiber.Ctx, s *core.State) {
	current := c.Context()
	if i.legacy {
		s.Context = current
		return
	}
	previous, ok := s.Context.(*lifecycleContext)
	// Custom contexts need not be comparable. Refresh those conservatively.
	if ok && reflect.TypeOf(current).Comparable() && previous.Context == current {
		return
	}
	// Keep contexts already passed to callbacks immutable and safe to read.
	s.Context = &lifecycleContext{Context: current, fiber: c}
}

func sameMap(left, right map[string]any) bool {
	return reflect.ValueOf(left).Pointer() == reflect.ValueOf(right).Pointer()
}

func (s *requestState) captureBindings() { s.capture(&s.State) }

func (v *requestValues) capture(s *core.State) {
	v.props, v.view = s.Props, s.ViewData
	v.invalidView = s.InvalidViewData
	v.pageMeta = s.LegacyPageMeta()
}

func (v *requestValues) load(s *core.State) {
	s.Props, s.ViewData, s.InvalidViewData = v.props, v.view, v.invalidView
	s.SetLegacyPageMeta(v.pageMeta)
}

// Public Locals remain the compatibility channel for direct caller writes.
// An unchanged published binding belongs to its manager, including nil clears.
// A fresh raw binding is adopted and claimed by the next manager accessing it.
func (i *Inertia) localProps(c fiber.Ctx, owned *requestValues) (map[string]any, bool) {
	props, _ := c.Locals(ContextKeyProps).(map[string]any)
	owner, ok := c.Locals(propsOwnerKey{}).(*requestValues)
	return props, !ok || owner == owned || !sameMap(props, owner.props)
}

func (i *Inertia) localView(c fiber.Ctx, owned *requestValues) (view map[string]any, invalid, readable bool) {
	raw := c.Locals(ContextKeyViewData)
	view, valid := raw.(map[string]any)
	invalid = raw != nil && !valid
	owner, ok := c.Locals(viewOwnerKey{}).(*requestValues)
	return view, invalid, !ok || owner == owned || !sameMap(view, owner.view) || invalid != owner.invalidView
}

func (i *Inertia) localMeta(c fiber.Ctx, owned *requestValues) (any, bool) {
	var local core.State
	local.SetLegacyPageMeta(c.Locals(ContextKeyPageMeta))
	meta := local.LegacyPageMeta()
	owner, ok := c.Locals(metaOwnerKey{}).(*requestValues)
	return meta, !ok || owner == owned || meta != owner.pageMeta
}

func (i *Inertia) publishProps(c fiber.Ctx, s *core.State, owner *requestValues) {
	if props, _ := c.Locals(ContextKeyProps).(map[string]any); !sameMap(props, s.Props) {
		c.Locals(ContextKeyProps, s.Props)
	}
	if c.Locals(propsOwnerKey{}) != owner {
		c.Locals(propsOwnerKey{}, owner)
	}
}

func (i *Inertia) publishView(c fiber.Ctx, s *core.State, owner *requestValues) {
	raw := c.Locals(ContextKeyViewData)
	view, valid := raw.(map[string]any)
	if !s.InvalidViewData {
		if !sameMap(view, s.ViewData) || (raw != nil && !valid) {
			c.Locals(ContextKeyViewData, s.ViewData)
		}
	} else if raw == nil || valid {
		c.Locals(ContextKeyViewData, struct{}{})
	}
	if c.Locals(viewOwnerKey{}) != owner {
		c.Locals(viewOwnerKey{}, owner)
	}
}

func (i *Inertia) publishMeta(c fiber.Ctx, s *core.State, owner *requestValues) {
	var local core.State
	local.SetLegacyPageMeta(c.Locals(ContextKeyPageMeta))
	if local.LegacyPageMeta() != s.LegacyPageMeta() {
		c.Locals(ContextKeyPageMeta, s.LegacyPageMeta())
	}
	if c.Locals(metaOwnerKey{}) != owner {
		c.Locals(metaOwnerKey{}, owner)
	}
}

func (i *Inertia) publishValues(c fiber.Ctx, s *core.State, owner *requestValues) {
	if s.Props != nil {
		i.publishProps(c, s, owner)
	}
	if s.ViewData != nil || s.InvalidViewData {
		i.publishView(c, s, owner)
	}
	if s.LegacyPageMeta() != nil {
		i.publishMeta(c, s, owner)
	}
}

func (i *Inertia) reconcileState(c fiber.Ctx, s *requestState) {
	i.reconcileBindings(c, &s.State, &s.requestValues)
}

func (i *Inertia) reconcileBindings(c fiber.Ctx, s *core.State, saved *requestValues) {
	localProps, readable := i.localProps(c, saved)
	switch {
	case !sameMap(s.Props, saved.props):
		i.publishProps(c, s, saved)
	case readable && !sameMap(localProps, saved.props):
		s.Props = localProps
		i.publishProps(c, s, saved)
	case !readable && s.Props != nil:
		i.publishProps(c, s, saved)
	}
	localView, invalid, readable := i.localView(c, saved)
	switch {
	case !sameMap(s.ViewData, saved.view) || s.InvalidViewData != saved.invalidView:
		i.publishView(c, s, saved)
	case readable && (!sameMap(localView, saved.view) || invalid != saved.invalidView):
		s.ViewData, s.InvalidViewData = localView, invalid
		i.publishView(c, s, saved)
	case !readable && (s.ViewData != nil || s.InvalidViewData):
		i.publishView(c, s, saved)
	}
	localMeta, readable := i.localMeta(c, saved)
	switch {
	case s.LegacyPageMeta() != saved.pageMeta:
		i.publishMeta(c, s, saved)
	case readable && localMeta != saved.pageMeta:
		s.SetLegacyPageMeta(localMeta)
		i.publishMeta(c, s, saved)
	case !readable && s.LegacyPageMeta() != nil:
		i.publishMeta(c, s, saved)
	}
	saved.capture(s)
}

func (i *Inertia) readLocalState(c fiber.Ctx, s *core.State) {
	if props, readable := i.localProps(c, nil); readable {
		s.Props = props
	}
	if view, invalid, readable := i.localView(c, nil); readable {
		s.ViewData, s.InvalidViewData = view, invalid
	}
	if meta, readable := i.localMeta(c, nil); readable {
		s.SetLegacyPageMeta(meta)
	}
}

func (i *Inertia) readViewData(c fiber.Ctx, s *core.State) {
	owned, _ := c.Locals(helperKey{owner: i}).(*requestValues)
	if view, invalid, readable := i.localView(c, owned); readable {
		s.ViewData, s.InvalidViewData = view, invalid
	}
}

func (i *Inertia) mutationState(c fiber.Ctx, local *core.State) *core.State {
	if s, ok := c.Locals(stateKey{owner: i}).(*requestState); ok {
		i.reconcileState(c, s)
		return &s.State
	}
	if saved, ok := c.Locals(helperKey{owner: i}).(*requestValues); ok {
		saved.load(local)
		i.reconcileBindings(c, local, saved)
	} else {
		i.readLocalState(c, local)
	}
	return local
}

func (i *Inertia) syncState(c fiber.Ctx, s *core.State) {
	if saved, ok := c.Locals(stateKey{owner: i}).(*requestState); ok {
		i.reconcileState(c, saved)
		return
	}
	saved, ok := c.Locals(helperKey{owner: i}).(*requestValues)
	if !ok {
		saved = &requestValues{}
		c.Locals(helperKey{owner: i}, saved)
	}
	i.reconcileBindings(c, s, saved)
}

// State returns the concrete per-request state for native consumers.
func (i *Inertia) State(c fiber.Ctx) *core.State { return i.state(c) }

func (i *Inertia) buildPage(c fiber.Ctx, component string, props map[string]any) (*PageDTO, error) {
	return i.buildPageWithState(c, i.state(c), component, props)
}

func (i *Inertia) buildPageWithState(c fiber.Ctx, s *core.State, component string, props map[string]any) (*PageDTO, error) {
	if i.sessionStore != nil {
		flash, err := i.sessionStore.GetFlash(c, string(ContextKeyProps))
		if err == nil {
			s.FlashData, _ = flash.(map[string]any)
		}
	}
	return i.BuildPage(s, component, props)
}

func (i *Inertia) isPrecognitionRequest(c fiber.Ctx) bool { return IsPrecognition(c) }
func (i *Inertia) shouldNoCacheResponse(c fiber.Ctx) bool {
	return strings.Contains(strings.ToLower(c.Get(fiber.HeaderCacheControl)), "no-cache")
}

func (i *Inertia) isExternalRedirect(target string) bool { return i.IsExternalRedirect(target) }
func (i *Inertia) isMethodPost(method string) bool       { return core.IsWriteMethod(method) }
func (i *Inertia) collectPrecognitionErrors(c fiber.Ctx) ValidationErrors {
	return i.PrecognitionErrors(i.state(c))
}

func filterValidationErrors(value ValidationErrors, only map[string]struct{}) ValidationErrors {
	if len(only) == 0 {
		return value
	}
	result := make(ValidationErrors)
	for key, v := range value {
		if _, ok := only[key]; ok {
			result[key] = v
		}
	}
	return result
}

func parseHeaderList(value string) map[string]struct{} {
	set := make(map[string]struct{})
	for _, key := range strings.Split(value, ",") {
		if key = strings.TrimSpace(key); key != "" {
			set[key] = struct{}{}
		}
	}
	return set
}

func (i *Inertia) renderHTML(c fiber.Ctx, s *core.State, page *PageDTO) error {
	i.applyVary(c)
	// Legacy callbacks may write view data directly to Locals while building props.
	if saved, ok := c.Locals(stateKey{owner: i}).(*requestState); ok {
		i.reconcileState(c, saved)
	} else {
		i.readViewData(c, s)
	}
	data, err := i.RenderHTML(s, page)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Send(data)
}

func (i *Inertia) renderHTMLError(c fiber.Ctx, appErr *Error, details string) error {
	tmpl, err := i.ErrorTemplate()
	if err != nil {
		_ = c.Status(500).SendString("Internal server error")
		return err
	}
	if appErr == nil {
		appErr = ErrNillable
	}
	data := map[string]any{"code": appErr.Code, "message": appErr.Message}
	if details != "" {
		data["details"] = details
	}
	body, err := core.ExecuteTemplate(tmpl, data)
	if err != nil {
		_ = c.Status(500).SendString("Internal server error")
		return err
	}
	c.Status(appErr.Code)
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	return c.Send(body)
}

func (i *Inertia) processSSR(c fiber.Ctx, page *PageDTO) (*SsrDTO, error) {
	return i.ProcessSSR(c.Context(), page)
}

func (i *Inertia) registerCSRFSharedProp() {
	if i.csrfTokenProvider == nil {
		return
	}
	core.WithCSRFEnabled(true)(i.Inertia)
	core.WithSharedProps(map[string]any{i.csrfPropName: core.LazyProp{
		Key: i.csrfPropName,
		Fn: func(ctx context.Context) (any,
			error,
		) {
			c, ok := ctx.(fiber.Ctx)
			if !ok {
				c, ok = FromContext(ctx)
			}
			if !ok {
				return nil, errors.New("inertia: Fiber CSRF callback outside request")
			}
			return i.csrfTokenProvider(c)
		},
	}})(i.Inertia)
}

func (i *Inertia) RedirectBackWithValidationErrors(c fiber.Ctx, errors ValidationErrors) error {
	i.WithValidationErrors(c, errors)
	return i.RedirectBack(c)
}

// RedirectBackWithErrors redirects back with validation errors stored in session.
func (i *Inertia) RedirectBackWithErrors(c fiber.Ctx, errors map[string]string) error {
	i.WithErrors(c, errors)
	return i.RedirectBack(c)
}

// RedirectBack redirects back to the previous page after a successful operation.
func (i *Inertia) RedirectBack(c fiber.Ctx) error {
	referer := c.Get(fiber.HeaderReferer)
	if referer == "" {
		referer = c.OriginalURL()
	}
	return i.Redirect(c, referer)
}

// Redirect handles redirects according to Inertia.js protocol.
func (i *Inertia) Redirect(c fiber.Ctx, url string) error {
	if url == "" || url == "/" {
		url = c.BaseURL()
	}
	if c.Get(HeaderInertia) != "" {
		if i.isExternalRedirect(url) {
			return i.RedirectExternal(c, url)
		}
		// For Inertia requests, use standard redirect (internal visit).
		return c.Redirect().Status(fiber.StatusFound).To(url)
	}

	// For regular requests, use standard redirect
	return c.Redirect().Status(fiber.StatusFound).To(url)
}

// RedirectExternal forces a full page reload for Inertia requests.
func (i *Inertia) RedirectExternal(c fiber.Ctx, url string) error {
	if url == "" || url == "/" {
		url = c.BaseURL()
	}

	c.Set(HeaderLocation, url)
	c.Set(fiber.HeaderLocation, url)
	return c.SendStatus(fiber.StatusConflict)
}

func (i *Inertia) renderPrecognition(c fiber.Ctx, errors ValidationErrors) error {
	addVaryHeader(c, HeaderPrecognition)
	c.Set(HeaderPrecognition, "true")
	if i.shouldNoCacheResponse(c) {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}

	if len(errors) == 0 {
		c.Set(HeaderPrecognitionSuccess, "true")
		return c.SendStatus(fiber.StatusNoContent)
	}

	payload := map[string]any{"errors": errors}
	js, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling precognition errors: %w", err)
	}

	c.Status(fiber.StatusUnprocessableEntity)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(js)
}

func (i *Inertia) renderPrecognitionError(c fiber.Ctx, errReturn *Error) error {
	errors := errReturn.ValidationErrors()
	errors = filterValidationErrors(errors, parseHeaderList(c.Get(HeaderPrecognitionValidateOnly)))
	if len(errors) > 0 {
		return i.renderPrecognition(c, errors)
	}

	addVaryHeader(c, HeaderPrecognition)
	c.Set(HeaderPrecognition, "true")
	if i.shouldNoCacheResponse(c) {
		c.Set(fiber.HeaderCacheControl, "no-cache")
	}

	status := fiber.StatusInternalServerError
	message := ErrInternal.Message
	if errReturn != nil {
		if errReturn.Code != 0 {
			status = errReturn.Code
		}
		if errReturn.Message != "" {
			message = errReturn.Message
		}
	}

	payload := map[string]any{"message": message}
	js, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling precognition error: %w", err)
	}

	c.Status(status)
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
	return c.Send(js)
}

func (i *Inertia) Render(c fiber.Ctx, component string, props map[string]any) error {
	if i.isPrecognitionRequest(c) {
		errors := i.collectPrecognitionErrors(c)
		errors = filterValidationErrors(errors, parseHeaderList(c.Get(HeaderPrecognitionValidateOnly)))
		return i.renderPrecognition(c, errors)
	}

	s := i.state(c)
	page, err := i.buildPageWithState(c, s, component, props)
	if err != nil {
		return fmt.Errorf("could not build page: %w", err)
	}

	if c.Get(HeaderInertia) != "" {
		return i.renderJSON(c, s, page)
	}

	return i.renderHTML(c, s, page)
}

// getContextKeyProps returns existing props or creates new ones.
func (i *Inertia) setFlashSessionData(c fiber.Ctx) {
	if i.sessionStore == nil {
		return
	}

	// Precognition requests should never write to flash/session.
	if i.isPrecognitionRequest(c) {
		return
	}

	status := c.Response().StatusCode()
	isRedirect := status == fiber.StatusMovedPermanently ||
		status == fiber.StatusFound ||
		status == fiber.StatusSeeOther ||
		status == fiber.StatusTemporaryRedirect ||
		status == fiber.StatusPermanentRedirect
	isInertiaLocationConflict := status == fiber.StatusConflict && len(c.Response().Header.Peek(HeaderLocation)) > 0
	if !isRedirect && !isInertiaLocationConflict {
		return
	}

	// Only persist flash-related props that are meant to survive redirects.
	var local core.State
	s := i.mutationState(c, &local)
	if len(s.Props) == 0 {
		return
	}
	flashData := s.FlashToPersist()
	if len(flashData) == 0 {
		return
	}

	if err := i.sessionStore.Flash(c, string(ContextKeyProps), flashData); err != nil {
		i.logger.ErrorContext(c, "could not set flash session props", "error", err)
	}
}

// loadFlashSessionData loads flash data from session storage.
func (i *Inertia) renderJSON(c fiber.Ctx, s *core.State, page *PageDTO) error {
	js, err := core.MarshalPage(page, s.Meta.Reset)
	if err != nil {
		return fmt.Errorf("error marshaling page: %w", err)
	}

	i.applyVary(c)
	c.Set(HeaderInertia, "true")
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	return c.Send(js)
}

// renderHTML renders the page as HTML template.
func (i *Inertia) WithProp(c fiber.Ctx, key string, value any) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithProp(s, key, value)
	i.syncState(c, s)
}

func (i *Inertia) WithViewData(c fiber.Ctx, key string, value any) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithViewData(s, key, value)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashMessages(c fiber.Ctx, flashMessages ...FlashError) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashMessages(s, flashMessages...)
	i.syncState(c, s)
}

func (i *Inertia) WithValidationErrors(c fiber.Ctx, errors ValidationErrors) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithValidationErrors(s, errors)
	i.syncState(c, s)
}

func (i *Inertia) WithErrors(c fiber.Ctx, errors map[string]string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithErrors(s, errors)
	i.syncState(c, s)
}

func (i *Inertia) WithError(c fiber.Ctx, field string, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithError(s, field, message)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashSuccess(c fiber.Ctx, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashSuccess(s, message)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashInfo(c fiber.Ctx, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashInfo(s, message)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashWarning(c fiber.Ctx, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashWarning(s, message)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashError(c fiber.Ctx, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashError(s, message)
	i.syncState(c, s)
}

func (i *Inertia) WithFlashOld(c fiber.Ctx, data map[string]any) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlashOld(s, data)
	i.syncState(c, s)
}

func (i *Inertia) WithFlash(c fiber.Ctx, key FlashLevel, message string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithFlash(s, key, message)
	i.syncState(c, s)
}

func (i *Inertia) WithLazyProp(c fiber.Ctx, key string, fn func(context.Context) (any, error)) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithLazyProp(s, key, fn)
	i.syncState(c, s)
}

func (i *Inertia) WithMatchPropsOn(c fiber.Ctx, props ...string) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithMatchPropsOn(s, props...)
	i.syncState(c, s)
}

func (i *Inertia) WithEncryptHistory(c fiber.Ctx) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithEncryptHistory(s)
	i.syncState(c, s)
}

func (i *Inertia) WithClearHistory(c fiber.Ctx) {
	var local core.State
	s := i.mutationState(c, &local)
	i.Inertia.WithClearHistory(s)
	i.syncState(c, s)
}

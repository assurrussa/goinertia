package goinertia

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"

	"github.com/assurrussa/goinertia/public"
)

type pageMeta struct {
	matchPropsOn []string
	scrollProps  map[string]ScrollPropConfig
}

type partialConfig struct {
	isPartial         bool
	hasInclude        bool
	hasExclude        bool
	include           map[string]struct{}
	exclude           map[string]struct{}
	reset             map[string]struct{}
	exceptOnce        map[string]struct{}
	forceInclude      map[string]struct{}
	scrollMergeIntent string
}

type Inertia struct {
	baseURL                   string
	rootTemplate              string
	rootHotTemplate           string
	rootErrorTemplate         string
	assetVersion              string
	sharedProps               map[string]any
	sharedFuncMap             template.FuncMap
	sharedViewData            map[string]any
	parsedTemplate            *template.Template
	parsedTemplateOnce        sync.Once
	parsedTemplateErr         error
	parsedErrorTemplate       *template.Template
	parsedErrorTemplateOnce   sync.Once
	parsedErrorTemplateErr    error
	hotURL                    string
	hotURLOnce                sync.Once
	templateFS                fs.FS
	publicFS                  fs.ReadFileFS
	ssrConfig                 SSRConfig
	ssrClient                 SSRClient
	ssrCache                  *ssrCache
	sessionStore              SessionStore // Adds session support.
	logger                    Logger
	canExposeDetails          func(ctx context.Context, headers map[string][]string) bool
	customErrorDetailsHandler func(errReturn *Error, isCanDetails bool) string
	customErrorGettingHandler func(err error) *Error
	csrfTokenCheckProvider    CSRFTokenCheckProvider
	csrfTokenProvider         CSRFTokenProvider
	csrfPropName              string
	isDev                     bool
}

func Must(inr *Inertia, err error) *Inertia {
	if err != nil {
		panic(err)
	}

	return inr
}

func NewWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	inr := New(baseURL, opts...)
	if err := inr.ParseTemplates(); err != nil {
		return nil, fmt.Errorf("failed to parse templates: %w", err)
	}

	return inr, nil
}

// New init inertia
//
// Example:
// optsInertia := []inertia.Option{inertia.WithFS(views.Templates)}
//
//		if cfg.Global.IsLocal() {
//			optsInertia = []inertia.Option{
//				inertia.WithRootTemplate("internal/adminext/views/app.gohtml"),
//				inertia.WithRootHotTemplate("internal/adminext/public/hot"),
//				inertia.WithFS(nil),
//				inertia.WithPublicFS(nil),
//	         inertia.WithCanExposeDetails(func(c fiber.Ctx) bool {
//		          admin := admin_middleware.GetAdminAuth(c)
//		          return admin != nil && admin.HasRoles("admin")
//	         }),
//			}
//		}
//		inertiaManager := inertia.New(cfg.Global.AppDomainURL, optsInertia...)
func New(baseURL string, opts ...Option) *Inertia {
	inr := &Inertia{
		baseURL:           baseURL,
		rootTemplate:      "app.gohtml",
		rootHotTemplate:   "hot",
		rootErrorTemplate: "error.gohtml",
		assetVersion:      "",
		publicFS:          public.Files,
		sharedProps:       make(map[string]any),
		parsedTemplate:    nil,
		logger:            NewLoggerAdapter(nil),
		sharedFuncMap: template.FuncMap{
			"marshal": marshal,
			"raw":     raw,
			"asset":   asset,
		},
		sharedViewData:            make(map[string]any),
		canExposeDetails:          DefaultCanExpose,
		customErrorGettingHandler: DefaultCustomGettingError,
		customErrorDetailsHandler: DefaultCustomErrorDetails,
		csrfPropName:              ContextPropsCSRFToken,
	}

	for _, o := range opts {
		o(inr)
	}

	if inr.rootHotTemplate == "" {
		inr.rootHotTemplate = "hot"
	}

	if inr.rootTemplate == "" {
		inr.rootTemplate = "app.gohtml"
	}

	if inr.rootErrorTemplate == "" {
		inr.rootErrorTemplate = "error.gohtml"
	}

	inr.registerCSRFSharedProp()

	return inr
}

func (i *Inertia) ParseTemplates() error {
	var err error

	_, err = i.createRootTemplate()
	if err != nil {
		return err
	}

	_, err = i.createRootErrorTemplate()
	if err != nil {
		return err
	}

	return nil
}

func (i *Inertia) WithProp(c fiber.Ctx, key string, value any) {
	props := i.getContextKeyProps(c)

	props[key] = value
	c.Locals(ContextKeyProps, props)
}

func (i *Inertia) WithViewData(c fiber.Ctx, key string, value any) {
	data := i.getContextKeyViewData(c)

	data[key] = value
	c.Locals(ContextKeyViewData, data)
}

// WithFlashMessages adds flashes messages.
func (i *Inertia) WithFlashMessages(c fiber.Ctx, flashMessages ...FlashError) {
	if len(flashMessages) == 0 {
		return
	}

	for _, fm := range flashMessages {
		i.WithFlash(c, fm.Level, fm.Error())
	}
}

// WithValidationErrors adds validation errors (equivalent to Django's form validation).
func (i *Inertia) WithValidationErrors(c fiber.Ctx, errors ValidationErrors) {
	if len(errors) == 0 {
		return
	}

	flatErrors := make(map[string]string)
	for field, fieldErrors := range errors {
		if len(fieldErrors) > 0 {
			flatErrors[field] = fieldErrors[0] // Take first error
		}
	}
	i.WithErrors(c, flatErrors)
}

// WithErrors adds validation errors to the response.
// Only adds to context; session is written via setFlashSessionData.
func (i *Inertia) WithErrors(c fiber.Ctx, errors map[string]string) {
	props := i.getContextKeyProps(c)

	curErrors := make(map[string]string)
	if existingErrors, exists := props[ContextPropsErrors].(map[string]string); exists {
		curErrors = existingErrors
	}

	for field, message := range errors {
		curErrors[field] = message
	}

	i.WithProp(c, ContextPropsErrors, curErrors)
}

// WithError adds a single validation error.
func (i *Inertia) WithError(c fiber.Ctx, field string, message string) {
	i.WithErrors(c, map[string]string{
		field: message,
	})
}

// WithFlashSuccess adds success flash message.
func (i *Inertia) WithFlashSuccess(c fiber.Ctx, message string) {
	i.WithFlash(c, FlashLevelSuccess, message)
}

// WithFlashInfo adds info flash message.
func (i *Inertia) WithFlashInfo(c fiber.Ctx, message string) {
	i.WithFlash(c, FlashLevelInfo, message)
}

// WithFlashWarning adds warning flash message.
func (i *Inertia) WithFlashWarning(c fiber.Ctx, message string) {
	i.WithFlash(c, FlashLevelWarning, message)
}

// WithFlashError adds error flash message.
func (i *Inertia) WithFlashError(c fiber.Ctx, message string) {
	i.WithFlash(c, FlashLevelError, message)
}

// WithFlashOld adds flash message to the response.
// Only adds to context; session is written via setFlashSessionData.
func (i *Inertia) WithFlashOld(c fiber.Ctx, data map[string]any) {
	i.WithProp(c, ContextPropsOld, data)
}

// WithFlash adds flash message to the response.
// Only adds to context; session is written via setFlashSessionData.
func (i *Inertia) WithFlash(c fiber.Ctx, key FlashLevel, message string) {
	props := i.getContextKeyProps(c)

	flash := make(map[string]string)
	if existingFlash, exists := props[ContextPropsFlash].(map[string]string); exists {
		flash = existingFlash
	}

	flash[key.String()] = message
	props[ContextPropsFlash] = flash
	c.Locals(ContextKeyProps, props)
}

// WithLazyProp adds a lazy-evaluated prop that's only computed when requested.
func (i *Inertia) WithLazyProp(c fiber.Ctx, key string, fn func(context.Context) (any, error)) {
	i.WithProp(c, key, LazyProp{Key: key, Fn: fn})
}

// WithMatchPropsOn sets matchPropsOn metadata for the response.
func (i *Inertia) WithMatchPropsOn(c fiber.Ctx, props ...string) {
	if len(props) == 0 {
		return
	}
	meta := i.getContextKeyPageMeta(c)
	for _, prop := range props {
		if prop == "" {
			continue
		}
		meta.matchPropsOn = appendUnique(meta.matchPropsOn, prop)
	}
}

// RedirectBackWithValidationErrors redirects back with multiple validation errors per field.
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

func (i *Inertia) Render(c fiber.Ctx, component string, props map[string]any) error {
	page, err := i.buildPage(c, component, props)
	if err != nil {
		return fmt.Errorf("could not build page: %w", err)
	}

	if c.Get(HeaderInertia) != "" {
		return i.renderJSON(c, page)
	}

	return i.renderHTML(c, page)
}

// getContextKeyProps returns existing props or creates new ones.
func (i *Inertia) getContextKeyProps(c fiber.Ctx) map[string]any {
	return i.getContextKey(c, ContextKeyProps)
}

// getContextKeyViewData returns existing views or creates new ones.
func (i *Inertia) getContextKeyViewData(c fiber.Ctx) map[string]any {
	return i.getContextKey(c, ContextKeyViewData)
}

// getContextKeyPageMeta returns existing page meta or creates new one.
func (i *Inertia) getContextKeyPageMeta(c fiber.Ctx) *pageMeta {
	meta := c.Locals(ContextKeyPageMeta)
	if meta != nil {
		if pm, ok := meta.(*pageMeta); ok {
			return pm
		}
	}

	pm := &pageMeta{
		scrollProps: make(map[string]ScrollPropConfig),
	}
	c.Locals(ContextKeyPageMeta, pm)
	return pm
}

// getContextKey returns existing values by key or creates new ones.
func (i *Inertia) getContextKey(c fiber.Ctx, key contextKey) map[string]any {
	ctxData := c.Locals(key)
	data := make(map[string]any)
	if ctxData != nil {
		if p, ok := ctxData.(map[string]any); ok {
			data = p
		}
	}

	return data
}

// buildPage constructs the page data with props from various sources.
func (i *Inertia) buildPage(c fiber.Ctx, component string, props map[string]any) (*PageDTO, error) {
	partial := i.parsePartialConfig(c, component)

	page := &PageDTO{
		Component: component,
		Props:     make(map[string]any),
		URL:       c.OriginalURL(),
		Version:   i.assetVersion,
	}

	// Add props in order: shared -> context -> request
	overrideKeys := i.collectOverrideKeys(c, props)
	i.addSharedProps(c, page, partial, overrideKeys)

	if err := i.addContextProps(c, page, partial); err != nil {
		return nil, err
	}

	i.addRequestProps(c, page, props, partial)
	i.applyPageMeta(c, page)
	i.ensureErrorsProp(c, page)
	i.applyErrorBag(c, page)

	return page, nil
}

// parsePartialConfig extracts partial reload configuration.
func (i *Inertia) parsePartialConfig(c fiber.Ctx, component string) *partialConfig {
	cfg := &partialConfig{
		reset:      parseHeaderList(c.Get(HeaderReset)),
		exceptOnce: parseHeaderList(c.Get(HeaderExceptOnceProps)),
	}

	partialData := strings.TrimSpace(c.Get(HeaderPartialOnly))
	partialExcept := strings.TrimSpace(c.Get(HeaderPartialExcept))
	componentMatches := c.Get(HeaderPartialComponent) == component

	if componentMatches && (partialData != "" || partialExcept != "") {
		cfg.isPartial = true
		if partialExcept != "" {
			cfg.exclude = parseHeaderList(partialExcept)
			cfg.hasExclude = true
		} else if partialData != "" {
			cfg.include = parseHeaderList(partialData)
			cfg.hasInclude = true
		}
	}

	if cfg.isPartial {
		cfg.forceInclude = map[string]struct{}{
			ContextPropsFlash:  {},
			ContextPropsOld:    {},
			ContextPropsErrors: {},
		}
		if i.csrfPropName != "" && i.csrfTokenProvider != nil {
			cfg.forceInclude[i.csrfPropName] = struct{}{}
		}
	}

	cfg.scrollMergeIntent = strings.ToLower(strings.TrimSpace(c.Get(HeaderInfiniteScrollMergeIntent)))

	return cfg
}

func (p *partialConfig) shouldIncludeProp(key string) bool {
	if p == nil {
		return true
	}
	if _, ok := p.forceInclude[key]; ok {
		return true
	}
	if !p.isPartial {
		return true
	}
	if p.hasExclude {
		if p.exclude == nil {
			return true
		}
		_, excluded := p.exclude[key]
		return !excluded
	}
	if p.hasInclude {
		if p.include == nil {
			return false
		}
		_, included := p.include[key]
		return included
	}
	return true
}

func (p *partialConfig) explicitlyIncluded(key string) bool {
	if p == nil || !p.hasInclude || p.include == nil {
		return false
	}
	_, ok := p.include[key]
	return ok
}

func (p *partialConfig) isReset(key string) bool {
	if p == nil || p.reset == nil {
		return false
	}
	_, ok := p.reset[key]
	return ok
}

func (p *partialConfig) shouldSkipOnce(onceKey string, propKey string) bool {
	if p == nil || p.exceptOnce == nil {
		return false
	}
	if _, ok := p.exceptOnce[onceKey]; !ok {
		return false
	}
	return !p.explicitlyIncluded(propKey)
}

func parseHeaderList(value string) map[string]struct{} {
	if value == "" {
		return nil
	}
	items := strings.Split(value, ",")
	set := make(map[string]struct{}, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		set[trimmed] = struct{}{}
	}
	if len(set) == 0 {
		return nil
	}
	return set
}

// setFlashSessionData writes all accumulated data to the session in one batch.
func (i *Inertia) setFlashSessionData(c fiber.Ctx) {
	if i.sessionStore == nil {
		return
	}

	if props := i.getContextKeyProps(c); len(props) > 0 {
		if err := i.sessionStore.Flash(c, string(ContextKeyProps), props); err != nil {
			i.logger.ErrorContext(c, "could not set flash session props", "error", err)
		}
	}
}

// loadFlashSessionData loads flash data from session storage.
func (i *Inertia) loadFlashSessionData(c fiber.Ctx, page *PageDTO, partial *partialConfig) {
	if i.sessionStore == nil {
		return
	}

	flashRaw, err := i.sessionStore.GetFlash(c, string(ContextKeyProps))
	if err != nil {
		return
	}

	flashData, ok := flashRaw.(map[string]any)
	if !ok {
		return
	}

	if data, ok := flashData[ContextPropsFlash].(map[string]string); ok && len(data) > 0 {
		i.setPropValue(c, page, ContextPropsFlash, data, partial)
	}

	if data, ok := flashData[ContextPropsErrors].(map[string]string); ok && len(data) > 0 {
		i.setPropValue(c, page, ContextPropsErrors, data, partial)
	}

	if data, ok := flashData[ContextPropsOld].(map[string]any); ok && len(data) > 0 {
		i.setPropValue(c, page, ContextPropsOld, data, partial)
	}
}

// addContextProps adds context-specific props to the page.
func (i *Inertia) addContextProps(c fiber.Ctx, page *PageDTO, partial *partialConfig) error {
	// Load flash data from the session first.
	i.loadFlashSessionData(c, page, partial)

	// Then add local props from context (they have priority).
	return i.addLocalContextProps(c, page, partial)
}

// addSharedProps adds shared props to the page.
func (i *Inertia) addSharedProps(c fiber.Ctx, page *PageDTO, partial *partialConfig, overrideKeys map[string]struct{}) {
	if len(overrideKeys) == 0 {
		i.addRequestProps(c, page, i.sharedProps, partial)
		return
	}

	filtered := make(map[string]any, len(i.sharedProps))
	for key, value := range i.sharedProps {
		if _, exists := overrideKeys[key]; exists {
			continue
		}
		filtered[key] = value
	}
	i.addRequestProps(c, page, filtered, partial)
}

// addLocalContextProps adds local context props to the page.
func (i *Inertia) addLocalContextProps(c fiber.Ctx, page *PageDTO, partial *partialConfig) error {
	props := i.getContextKeyProps(c)
	i.addRequestProps(c, page, props, partial)
	return nil
}

// addRequestProps adds request-specific props to the page.
func (i *Inertia) addRequestProps(c fiber.Ctx, page *PageDTO, props map[string]any, partial *partialConfig) {
	for key, value := range props {
		i.setPropValue(c, page, key, value, partial)
	}
}

func (i *Inertia) collectOverrideKeys(c fiber.Ctx, props map[string]any) map[string]struct{} {
	override := make(map[string]struct{})

	for key := range props {
		override[key] = struct{}{}
	}

	ctxProps := i.getContextKeyProps(c)
	for key := range ctxProps {
		override[key] = struct{}{}
	}

	return override
}

func (i *Inertia) registerCSRFSharedProp() {
	if i.csrfTokenProvider == nil {
		if i.csrfPropName != "" {
			delete(i.sharedProps, i.csrfPropName)
		}
		return
	}

	if i.csrfPropName == "" {
		i.csrfPropName = ContextPropsCSRFToken
	}

	propName := i.csrfPropName
	i.sharedProps[propName] = LazyProp{
		Key: propName,
		Fn: func(ctx context.Context) (any, error) {
			fiberCtx, ok := ctx.(fiber.Ctx)
			if !ok {
				return "", nil
			}

			token, err := i.csrfTokenProvider(fiberCtx)
			if err != nil {
				return "", err
			}

			return token, nil
		},
	}
}

// shouldIncludeProp determines if a prop should be included based on partial reload config.
func (i *Inertia) shouldIncludeProp(key string, partial *partialConfig) bool {
	if partial == nil {
		return true
	}
	return partial.shouldIncludeProp(key)
}

// setPropValue sets a prop value, handling lazy props appropriately.
func (i *Inertia) setPropValue(c fiber.Ctx, page *PageDTO, key string, value any, partial *partialConfig) {
	if value == nil {
		i.setNilProp(page, key, partial)
		return
	}

	if op, ok := value.(OnceProp); ok {
		next, skip := i.applyOnceProp(page, key, op, partial)
		if skip {
			return
		}

		value = next
		if value == nil {
			i.setNilProp(page, key, partial)
			return
		}
	}

	if i.handleWrappedProp(c, page, key, value, partial) {
		return
	}

	if !i.shouldIncludeProp(key, partial) {
		return
	}

	result, err := i.resolvePropValue(c, key, value)
	if err != nil {
		i.logger.WarnContext(c, "failed to evaluate prop", "key", key, "error", err)
		return
	}

	page.Props[key] = result
}

func (i *Inertia) setNilProp(page *PageDTO, key string, partial *partialConfig) {
	if i.shouldIncludeProp(key, partial) {
		page.Props[key] = nil
	}
}

func (i *Inertia) applyOnceProp(page *PageDTO, key string, op OnceProp, partial *partialConfig) (any, bool) {
	onceKey := op.Key
	if onceKey == "" {
		onceKey = key
	}
	if page.OnceProps == nil {
		page.OnceProps = make(map[string]OncePropConfig)
	}
	page.OnceProps[onceKey] = OncePropConfig{
		Prop:      key,
		ExpiresAt: op.ExpiresAt,
	}

	if partial != nil && partial.shouldSkipOnce(onceKey, key) {
		return nil, true
	}

	return op.Value, false
}

func (i *Inertia) handleWrappedProp(c fiber.Ctx, page *PageDTO, key string, value any, partial *partialConfig) bool {
	switch prop := value.(type) {
	case DeferredProp:
		return i.handleDeferredProp(c, page, key, prop, partial)
	case OptionalProp:
		return i.handleOptionalProp(c, page, key, prop, partial)
	case AlwaysProp:
		return i.handleAlwaysProp(c, page, key, prop, partial)
	case MergeProp:
		return i.handleMergeProp(c, page, key, prop, partial)
	case ScrollProp:
		return i.handleScrollProp(c, page, key, prop, partial)
	default:
		return false
	}
}

func (i *Inertia) handleDeferredProp(c fiber.Ctx, page *PageDTO, key string, prop DeferredProp, partial *partialConfig) bool {
	group := prop.Group
	if group == "" {
		group = "default"
	}
	if page.DeferredProps == nil {
		page.DeferredProps = make(map[string][]string)
	}
	page.DeferredProps[group] = appendUnique(page.DeferredProps[group], key)
	if partial == nil || !partial.explicitlyIncluded(key) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleOptionalProp(c fiber.Ctx, page *PageDTO, key string, prop OptionalProp, partial *partialConfig) bool {
	if partial == nil || !partial.explicitlyIncluded(key) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleAlwaysProp(c fiber.Ctx, page *PageDTO, key string, prop AlwaysProp, partial *partialConfig) bool {
	if partial != nil {
		if partial.forceInclude == nil {
			partial.forceInclude = make(map[string]struct{})
		}
		partial.forceInclude[key] = struct{}{}
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleMergeProp(c fiber.Ctx, page *PageDTO, key string, prop MergeProp, partial *partialConfig) bool {
	if partial == nil || !partial.isReset(key) {
		switch {
		case prop.Prepend:
			page.PrependProps = appendUnique(page.PrependProps, key)
		case prop.Deep:
			page.DeepMergeProps = appendUnique(page.DeepMergeProps, key)
		default:
			page.MergeProps = appendUnique(page.MergeProps, key)
		}
	}
	if !i.shouldIncludeProp(key, partial) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleScrollProp(c fiber.Ctx, page *PageDTO, key string, prop ScrollProp, partial *partialConfig) bool {
	if page.ScrollProps == nil {
		page.ScrollProps = make(map[string]ScrollPropConfig)
	}
	page.ScrollProps[key] = prop.Config

	if partial == nil || !partial.isReset(key) {
		if partial != nil && partial.scrollMergeIntent == "prepend" {
			page.PrependProps = appendUnique(page.PrependProps, key)
		} else {
			page.MergeProps = appendUnique(page.MergeProps, key)
		}
	}

	if !i.shouldIncludeProp(key, partial) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

// renderJSON renders the page as JSON for Inertia requests.
func (i *Inertia) renderJSON(c fiber.Ctx, page *PageDTO) error {
	js, err := json.Marshal(page)
	if err != nil {
		return fmt.Errorf("error marshaling page: %w", err)
	}

	c.Set("Vary", HeaderInertia)
	c.Set(HeaderInertia, "true")
	c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)

	return c.Send(js)
}

// renderHTML renders the page as HTML template.
func (i *Inertia) renderHTML(c fiber.Ctx, page *PageDTO) error {
	rootTemplate, err := i.createRootTemplate()
	if err != nil {
		return err
	}

	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)

	viewData, err := i.createViewData(c)
	if err != nil {
		return err
	}

	viewData["page"] = page

	if i.IsSSREnabled() {
		ssr, err := i.processSSR(c, page)
		if err != nil {
			return err
		}
		viewData["processSSR"] = ssr
	} else {
		viewData["processSSR"] = nil
	}

	var buf bytes.Buffer
	err = rootTemplate.Execute(&buf, viewData)
	if err != nil {
		return fmt.Errorf("error executing template: %w", err)
	}

	return c.Send(buf.Bytes())
}

// renderHTMLError renders the page as HTML template.
func (i *Inertia) renderHTMLError(c fiber.Ctx, appErr *Error, details string) error {
	tmpl, err := i.createRootErrorTemplate()
	if err != nil {
		i.logger.ErrorContext(c, "error creating root error template", "error", err)
		_ = c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		return err
	}

	appErrCur := ErrNillable
	if appErr != nil {
		appErrCur = appErr
	}

	c.Status(appErrCur.Code)
	c.Set(fiber.HeaderContentType, fiber.MIMETextHTMLCharsetUTF8)
	var buf bytes.Buffer
	data := map[string]any{
		"code":    appErrCur.Code,
		"message": appErrCur.Message,
	}
	if details != "" {
		data["details"] = details
	}
	err = tmpl.Execute(&buf, data)
	if err != nil {
		i.logger.ErrorContext(c, "error executing root error template", "error", err)
		_ = c.Status(fiber.StatusInternalServerError).SendString("Internal server error")
		return err
	}

	return c.Send(buf.Bytes())
}

func (i *Inertia) createRootTemplate() (*template.Template, error) {
	if i.parsedTemplate != nil {
		return i.parsedTemplate, nil
	}

	parse := func() (*template.Template, error) {
		ts := template.New(filepath.Base(i.rootTemplate)).Funcs(i.sharedFuncMap)

		var tpl *template.Template
		var err error
		if i.templateFS != nil {
			tpl, err = ts.ParseFS(i.templateFS, i.rootTemplate)
		} else {
			tpl, err = ts.ParseFiles(i.rootTemplate)
		}

		if err != nil {
			return nil, fmt.Errorf("error parsing root template: %w", err)
		}
		return tpl, nil
	}

	if i.isDev {
		return parse()
	}

	i.parsedTemplateOnce.Do(func() {
		i.parsedTemplate, i.parsedTemplateErr = parse()
	})

	return i.parsedTemplate, i.parsedTemplateErr
}

func (i *Inertia) createRootErrorTemplate() (*template.Template, error) {
	if i.parsedErrorTemplate != nil {
		return i.parsedErrorTemplate, nil
	}

	parse := func() (*template.Template, error) {
		ts := template.New(filepath.Base(i.rootErrorTemplate)).Funcs(i.sharedFuncMap)

		var tpl *template.Template
		var err error
		if i.templateFS != nil {
			tpl, err = ts.ParseFS(i.templateFS, i.rootErrorTemplate)
		} else {
			tpl, err = ts.ParseFiles(i.rootErrorTemplate)
		}

		if err != nil {
			return nil, fmt.Errorf("error parsing root error template: %w", err)
		}
		return tpl, nil
	}

	if i.isDev {
		return parse()
	}

	i.parsedErrorTemplateOnce.Do(func() {
		i.parsedErrorTemplate, i.parsedErrorTemplateErr = parse()
	})

	return i.parsedErrorTemplate, i.parsedErrorTemplateErr
}

func (i *Inertia) createViewData(c fiber.Ctx) (map[string]any, error) {
	viewData := make(map[string]any)

	// Add shared view data
	for key, value := range i.sharedViewData {
		viewData[key] = value
	}

	// Add context view data
	contextViewData := c.Locals(ContextKeyViewData)
	if contextViewData != nil {
		contextViewData, ok := contextViewData.(map[string]any)
		if !ok {
			return nil, ErrInvalidContextViewData
		}

		for key, value := range contextViewData {
			viewData[key] = value
		}
	}

	// Check Vite dev server.
	if hotURL := i.hotServerURL(); hotURL != "" {
		viewData["hotServerUrl"] = hotURL
	}

	return viewData, nil
}

func (i *Inertia) hotServerURL() string {
	readHotFile := func() string {
		publicFSRead := os.ReadFile
		if i.publicFS != nil {
			publicFSRead = i.publicFS.ReadFile
		}
		if hotFile, err := publicFSRead(i.rootHotTemplate); err == nil {
			return strings.TrimSpace(string(hotFile))
		}
		return ""
	}

	if i.isDev {
		return readHotFile()
	}

	i.hotURLOnce.Do(func() {
		i.hotURL = readHotFile()
	})

	return i.hotURL
}

func (i *Inertia) cacheLazy(c fiber.Ctx, key string, lazy LazyProp) (any, error) {
	const cacheKey = "__inertia_lazy_cache"
	cache, _ := c.Locals(cacheKey).(map[string]any)
	if cache == nil {
		cache = make(map[string]any)
		c.Locals(cacheKey, cache)
	}

	if value, ok := cache[key]; ok {
		return value, nil
	}

	result, err := lazy.Fn(c)
	if err != nil {
		return nil, err
	}

	cache[key] = result

	return result, nil
}

func (i *Inertia) resolvePropValue(c fiber.Ctx, key string, value any) (any, error) {
	switch val := value.(type) {
	case LazyProp:
		return i.cacheLazy(c, key, val)
	case func(context.Context) (any, error):
		return val(c)
	default:
		return value, nil
	}
}

func (i *Inertia) applyPageMeta(c fiber.Ctx, page *PageDTO) {
	meta := c.Locals(ContextKeyPageMeta)
	if meta == nil {
		return
	}
	pm, ok := meta.(*pageMeta)
	if !ok || pm == nil {
		return
	}

	if len(pm.matchPropsOn) > 0 {
		for _, prop := range pm.matchPropsOn {
			page.MatchPropsOn = appendUnique(page.MatchPropsOn, prop)
		}
	}

	if len(pm.scrollProps) > 0 {
		if page.ScrollProps == nil {
			page.ScrollProps = make(map[string]ScrollPropConfig)
		}
		for key, cfg := range pm.scrollProps {
			if _, exists := page.ScrollProps[key]; !exists {
				page.ScrollProps[key] = cfg
			}
		}
	}
}

func (i *Inertia) ensureErrorsProp(_ fiber.Ctx, page *PageDTO) {
	if page == nil {
		return
	}
	if _, ok := page.Props[ContextPropsErrors]; !ok {
		page.Props[ContextPropsErrors] = map[string]string{}
	}
}

func (i *Inertia) applyErrorBag(c fiber.Ctx, page *PageDTO) {
	if page == nil {
		return
	}
	bag := strings.TrimSpace(c.Get(HeaderErrorBag))
	if bag == "" {
		return
	}

	errorsVal, ok := page.Props[ContextPropsErrors]
	if !ok || errorsVal == nil {
		page.Props[ContextPropsErrors] = map[string]map[string]string{bag: {}}
		return
	}

	switch v := errorsVal.(type) {
	case map[string]string:
		page.Props[ContextPropsErrors] = map[string]map[string]string{bag: v}
	case ValidationErrors:
		flat := make(map[string]string, len(v))
		for field, errs := range v {
			if len(errs) > 0 {
				flat[field] = errs[0]
			}
		}
		page.Props[ContextPropsErrors] = map[string]map[string]string{bag: flat}
	case map[string]any:
		if _, exists := v[bag]; !exists {
			v[bag] = map[string]string{}
		}
		page.Props[ContextPropsErrors] = v
	case map[string]map[string]string:
		if _, exists := v[bag]; !exists {
			v[bag] = map[string]string{}
		}
		page.Props[ContextPropsErrors] = v
	default:
		page.Props[ContextPropsErrors] = map[string]map[string]string{bag: {}}
	}
}

func appendUnique(list []string, value string) []string {
	for _, item := range list {
		if item == value {
			return list
		}
	}
	return append(list, value)
}

func (i *Inertia) isExternalRedirect(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil || !parsed.IsAbs() {
		return false
	}
	base, err := url.Parse(i.baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return true
	}
	return !strings.EqualFold(base.Scheme, parsed.Scheme) || !strings.EqualFold(base.Host, parsed.Host)
}

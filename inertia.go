package goinertia

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/goccy/go-json"
	"github.com/gofiber/fiber/v3"
	fiberclient "github.com/gofiber/fiber/v3/client"

	"github.com/assurrussa/goinertia/public"
)

// LazyProp represents a prop that is evaluated lazily.
type LazyProp struct {
	Key string
	Fn  func(ctx context.Context) (any, error)
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
	ssrClient                 *fiberclient.Client
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
		// For Inertia requests, use 409 Conflict with X-Inertia-Location header
		c.Set(HeaderLocation, url)
		c.Set(fiber.HeaderLocation, url)
		return c.SendStatus(fiber.StatusConflict)
	}

	// For regular requests, use standard redirect
	return c.Redirect().Status(fiber.StatusFound).To(url)
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
	only := i.parsePartialOnly(c, component)

	page := &PageDTO{
		Component: component,
		Props:     make(map[string]any),
		URL:       c.OriginalURL(),
		Version:   i.assetVersion,
	}

	// Add props in order: shared -> context -> request
	overrideKeys := i.collectOverrideKeys(c, props)
	i.addSharedProps(c, page, only, overrideKeys)

	if err := i.addContextProps(c, page, only); err != nil {
		return nil, err
	}

	i.addRequestProps(c, page, props, only)

	return page, nil
}

// parsePartialOnly extracts partial reload configuration.
func (i *Inertia) parsePartialOnly(c fiber.Ctx, component string) map[string]string {
	partial := c.Get(HeaderPartialOnly)
	if partial == "" {
		return nil
	}

	only := make(map[string]string)
	if c.Get(HeaderPartialComponent) == component {
		for _, value := range strings.Split(partial, ",") {
			only[value] = value
		}
	}

	if i.csrfPropName != "" && i.csrfTokenProvider != nil {
		only[i.csrfPropName] = "1"
	}

	if len(only) >= 0 { //nolint:gocritic // maybe,  is always true
		only[ContextPropsFlash] = "1"
		only[ContextPropsOld] = "1"
		only[ContextPropsErrors] = "1"
	}

	return only
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
func (i *Inertia) loadFlashSessionData(c fiber.Ctx, page *PageDTO, only map[string]string) {
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
		if i.shouldIncludeProp(ContextPropsFlash, only) {
			i.setPropValue(c, page, ContextPropsFlash, data)
		}
	}

	if data, ok := flashData[ContextPropsErrors].(map[string]string); ok && len(data) > 0 {
		if i.shouldIncludeProp(ContextPropsErrors, only) {
			i.setPropValue(c, page, ContextPropsErrors, data)
		}
	}

	if data, ok := flashData[ContextPropsOld].(map[string]any); ok && len(data) > 0 {
		if i.shouldIncludeProp(ContextPropsOld, only) {
			i.setPropValue(c, page, ContextPropsOld, data)
		}
	}
}

// addContextProps adds context-specific props to the page.
func (i *Inertia) addContextProps(c fiber.Ctx, page *PageDTO, only map[string]string) error {
	// Load flash data from the session first.
	i.loadFlashSessionData(c, page, only)

	// Then add local props from context (they have priority).
	return i.addLocalContextProps(c, page, only)
}

// addSharedProps adds shared props to the page.
func (i *Inertia) addSharedProps(c fiber.Ctx, page *PageDTO, only map[string]string, overrideKeys map[string]struct{}) {
	if len(overrideKeys) == 0 {
		i.addRequestProps(c, page, i.sharedProps, only)
		return
	}

	filtered := make(map[string]any, len(i.sharedProps))
	for key, value := range i.sharedProps {
		if _, exists := overrideKeys[key]; exists {
			continue
		}
		filtered[key] = value
	}
	i.addRequestProps(c, page, filtered, only)
}

// addLocalContextProps adds local context props to the page.
func (i *Inertia) addLocalContextProps(c fiber.Ctx, page *PageDTO, only map[string]string) error {
	props := i.getContextKeyProps(c)
	i.addRequestProps(c, page, props, only)
	return nil
}

// addRequestProps adds request-specific props to the page.
func (i *Inertia) addRequestProps(c fiber.Ctx, page *PageDTO, props map[string]any, only map[string]string) {
	for key, value := range props {
		if !i.shouldIncludeProp(key, only) {
			continue
		}

		i.setPropValue(c, page, key, value)
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
func (i *Inertia) shouldIncludeProp(key string, only map[string]string) bool {
	return len(only) == 0 || only[key] != ""
}

// setPropValue sets a prop value, handling lazy props appropriately.
func (i *Inertia) setPropValue(c fiber.Ctx, page *PageDTO, key string, value any) {
	lazy, ok := value.(LazyProp)
	if !ok {
		page.Props[key] = value
		return
	}

	result, err := i.cacheLazy(c, key, lazy)
	if err != nil {
		i.logger.WarnContext(c, "failed to evaluate lazy prop", "key", key, "error", err)
		return
	}

	page.Props[key] = result
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

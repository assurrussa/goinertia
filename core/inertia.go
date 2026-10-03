package core

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

	"github.com/assurrussa/goinertia/public"
)

type pageMeta struct {
	matchPropsOn     []string
	scrollProps      map[string]ScrollPropConfig
	encryptHistory   *bool
	clearHistory     *bool
	nativeFlash      map[string]any
	preserveFragment *bool
	ssrDisabled      bool
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
	baseURL                 string
	baseURLParsed           *url.URL
	rootTemplate            string
	rootHotTemplate         string
	rootErrorTemplate       string
	assetVersion            string
	sharedProps             map[string]any
	sharedFuncMap           template.FuncMap
	sharedViewData          map[string]any
	parsedTemplate          *template.Template
	parsedTemplateOnce      sync.Once
	parsedTemplateErr       error
	parsedErrorTemplate     *template.Template
	parsedErrorTemplateOnce sync.Once
	parsedErrorTemplateErr  error
	hotURL                  string
	hotURLOnce              sync.Once
	templateFS              fs.FS
	publicFS                fs.ReadFileFS
	ssrConfig               SSRConfig
	ssrClient               SSRClient
	ssrCache                *ssrCache
	ssrErrorPolicy          SSRErrorPolicy
	ssrFailureHandler       SSRFailureHandler
	viteSSR                 *bool
	logger                  Logger
	csrfPropName            string
	csrfEnabled             bool
	isDev                   bool
	precognitionVary        bool
	protocolVersion         ProtocolVersion
	preserveBigIntegers     bool
}

func Must(inr *Inertia, err error) *Inertia {
	if err != nil {
		panic(err)
	}

	return inr
}

func NewWithValidation(baseURL string, opts ...Option) (*Inertia, error) {
	inr := New(baseURL, opts...)
	if inr.baseURLParsed == nil {
		return nil, ErrBaseURLEmpty
	}
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
//	         inertia.WithCanExposeDetails(func(c *State) bool {
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
		sharedViewData:   make(map[string]any),
		csrfPropName:     ContextPropsCSRFToken,
		precognitionVary: true,
	}

	for _, o := range opts {
		o(inr)
	}

	inr.NormalizeConfig()

	inr.baseURLParsed = parseInertiaBaseURL(inr.baseURL)
	if inr.baseURLParsed != nil {
		inr.baseURL = inr.baseURLParsed.String()
	}

	return inr
}

func (i *Inertia) ParseTemplates() error {
	var err error

	_, err = i.RootTemplate()
	if err != nil {
		return err
	}

	_, err = i.ErrorTemplate()
	if err != nil {
		return err
	}

	return nil
}

func (i *Inertia) WithProp(c *State, key string, value any) {
	props := i.getContextKeyProps(c)

	props[key] = value
	c.Props = props
}

func (i *Inertia) WithViewData(c *State, key string, value any) {
	data := i.getContextKeyViewData(c)

	data[key] = value
	c.ViewData = data
	c.InvalidViewData = false
}

// WithFlashMessages adds flashes messages.
func (i *Inertia) WithFlashMessages(c *State, flashMessages ...FlashError) {
	if len(flashMessages) == 0 {
		return
	}

	for _, fm := range flashMessages {
		i.WithFlash(c, fm.Level, fm.Error())
	}
}

// WithValidationErrors adds validation errors (equivalent to Django's form validation).
func (i *Inertia) WithValidationErrors(c *State, errors ValidationErrors) {
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
func (i *Inertia) WithErrors(c *State, errors map[string]string) {
	if i.IsProtocolV3() {
		i.mergeV3Errors(c, errors)
		return
	}
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
func (i *Inertia) WithError(c *State, field string, message string) {
	i.WithErrors(c, map[string]string{
		field: message,
	})
}

// WithFlashSuccess adds success flash message.
func (i *Inertia) WithFlashSuccess(c *State, message string) {
	i.WithFlash(c, FlashLevelSuccess, message)
}

// WithFlashInfo adds info flash message.
func (i *Inertia) WithFlashInfo(c *State, message string) {
	i.WithFlash(c, FlashLevelInfo, message)
}

// WithFlashWarning adds warning flash message.
func (i *Inertia) WithFlashWarning(c *State, message string) {
	i.WithFlash(c, FlashLevelWarning, message)
}

// WithFlashError adds error flash message.
func (i *Inertia) WithFlashError(c *State, message string) {
	i.WithFlash(c, FlashLevelError, message)
}

// WithFlashOld adds flash message to the response.
// Only adds to context; session is written via setFlashSessionData.
func (i *Inertia) WithFlashOld(c *State, data map[string]any) {
	i.WithProp(c, ContextPropsOld, data)
}

// WithFlash adds flash message to the response.
// Only adds to context; session is written via setFlashSessionData.
func (i *Inertia) WithFlash(c *State, key FlashLevel, message string) {
	props := i.getContextKeyProps(c)

	flash := make(map[string]string)
	if existingFlash, exists := props[ContextPropsFlash].(map[string]string); exists {
		flash = existingFlash
	}

	flash[key.String()] = message
	props[ContextPropsFlash] = flash
	c.Props = props
}

// WithLazyProp adds a lazy-evaluated prop that's only computed when requested.
func (i *Inertia) WithLazyProp(c *State, key string, fn func(context.Context) (any, error)) {
	i.WithProp(c, key, LazyProp{Key: key, Fn: fn})
}

// WithMatchPropsOn sets matchPropsOn metadata for the response.
func (i *Inertia) WithMatchPropsOn(c *State, props ...string) {
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

// WithEncryptHistory sets encryptHistory metadata for the response.
func (i *Inertia) WithEncryptHistory(c *State) {
	meta := i.getContextKeyPageMeta(c)
	value := true
	meta.encryptHistory = &value
}

// WithClearHistory sets clearHistory metadata for the response.
func (i *Inertia) WithClearHistory(c *State) {
	meta := i.getContextKeyPageMeta(c)
	value := true
	meta.clearHistory = &value
}

// RedirectBackWithValidationErrors redirects back with multiple validation errors per field.
func filterValidationErrors(errors ValidationErrors, only map[string]struct{}) ValidationErrors {
	if errors == nil || len(only) == 0 {
		return errors
	}
	filtered := make(ValidationErrors, len(errors))
	for field, msgs := range errors {
		if _, ok := only[field]; ok {
			filtered[field] = msgs
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func (i *Inertia) getContextKeyProps(c *State) map[string]any {
	if c.Props == nil {
		c.Props = make(map[string]any)
	}
	return c.Props
}

func (i *Inertia) getContextKeyViewData(c *State) map[string]any {
	if c.ViewData == nil {
		c.ViewData = make(map[string]any)
	}
	return c.ViewData
}

func (i *Inertia) getContextKeyPageMeta(c *State) *pageMeta {
	if c.pageMeta == nil {
		c.pageMeta = &pageMeta{scrollProps: make(map[string]ScrollPropConfig)}
	}
	return c.pageMeta
}

func (i *Inertia) BuildPage(c *State, component string, props map[string]any) (*PageDTO, error) {
	if i.IsProtocolV3() {
		return i.buildPageV3(c, component, props)
	}
	c.propMetadata = nil
	c.nestedMergeRoots = nil
	partial := i.parsePartialConfig(c, component)

	page := &PageDTO{
		Component: component,
		Props:     make(map[string]any),
		URL:       c.Meta.URL,
		Version:   i.assetVersion,
	}

	// Add props in order: shared -> context -> request
	overrideKeys := i.collectOverrideKeys(c, props)
	i.addSharedProps(c, page, partial, overrideKeys)

	if err := i.addContextProps(c, page, partial); err != nil {
		return nil, err
	}

	i.addRequestProps(c, page, props, partial, requestSource)
	i.applyPageMeta(c, page)
	i.ensureErrorsProp(c, page)
	i.applyErrorBag(c, page)

	return page, nil
}

// parsePartialConfig extracts partial reload configuration.
func (i *Inertia) parsePartialConfig(c *State, component string) *partialConfig {
	if c.Meta.Reset == "" && c.Meta.ExceptOnceProps == "" && c.Meta.PartialOnly == "" &&
		c.Meta.PartialExcept == "" && c.Meta.ScrollMergeIntent == "" {
		return nil
	}
	cfg := &partialConfig{
		reset:      parseHeaderList(c.Meta.Reset),
		exceptOnce: parseHeaderList(c.Meta.ExceptOnceProps),
	}

	partialData := strings.TrimSpace(c.Meta.PartialOnly)
	partialExcept := strings.TrimSpace(c.Meta.PartialExcept)
	componentMatches := c.Meta.PartialComponent == component

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
		if i.csrfPropName != "" && i.csrfEnabled {
			cfg.forceInclude[i.csrfPropName] = struct{}{}
		}
	}

	cfg.scrollMergeIntent = strings.ToLower(strings.TrimSpace(c.Meta.ScrollMergeIntent))

	return cfg
}

func (i *Inertia) shouldIncludeProp(key string, partial *partialConfig) bool {
	if partial == nil {
		return true
	}
	return partial.shouldIncludeProp(key)
}

// setPropValue sets a prop value, handling lazy props appropriately.
func (p *partialConfig) explicitlyIncluded(key string) bool {
	if p == nil || !p.hasInclude || p.include == nil {
		return false
	}
	return (propSelection{only: true, paths: p.include}).child(key).mayInclude()
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

// setFlashSessionData persists flash-related props (flash/errors/old) into the session.
// It is only needed for redirect-like responses (3xx or 409 with X-Inertia-Location),
// so we skip it for normal renders and for Precognition requests.
func (i *Inertia) loadFlashSessionData(c *State, page *PageDTO, partial *partialConfig) {
	flashData := c.FlashData
	if len(flashData) == 0 {
		return
	}
	previous := c.propSource
	c.propSource = flashSource
	defer func() { c.propSource = previous }()

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
func (i *Inertia) addContextProps(c *State, page *PageDTO, partial *partialConfig) error {
	// Load flash data from the session first.
	i.loadFlashSessionData(c, page, partial)

	// Then add local props from context (they have priority).
	return i.addLocalContextProps(c, page, partial)
}

// addSharedProps adds shared props to the page.
func (i *Inertia) addSharedProps(c *State, page *PageDTO, partial *partialConfig, overrideKeys map[string]struct{}) {
	if len(overrideKeys) == 0 {
		i.addRequestProps(c, page, i.sharedProps, partial, sharedSource)
		return
	}

	filtered := make(map[string]any, len(i.sharedProps))
	for key, value := range i.sharedProps {
		if _, exists := overrideKeys[key]; exists {
			continue
		}
		filtered[key] = value
	}
	i.addRequestProps(c, page, filtered, partial, sharedSource)
}

// addLocalContextProps adds local context props to the page.
func (i *Inertia) addLocalContextProps(c *State, page *PageDTO, partial *partialConfig) error {
	i.addRequestProps(c, page, c.Props, partial, contextSource)
	return nil
}

// addRequestProps adds request-specific props to the page.
func (i *Inertia) addRequestProps(c *State, page *PageDTO, props map[string]any, partial *partialConfig, source propSource) {
	if len(props) == 0 {
		return
	}
	previous := c.propSource
	c.propSource = source
	for key, value := range props {
		nestedMerge := hasNestedMerge(value)
		if (nestedMerge || c.nestedMergeRoots[key]) && i.shouldIncludeProp(key, partial) {
			omitUnselectedProp(c, page, key)
			if c.nestedMergeRoots == nil {
				c.nestedMergeRoots = make(map[string]bool)
			}
			c.nestedMergeRoots[key] = nestedMerge
		}
		i.setPropValue(c, page, key, value, partial)
	}
	c.propSource = previous
}

func (i *Inertia) collectOverrideKeys(c *State, props map[string]any) map[string]struct{} {
	if len(i.sharedProps) == 0 {
		return nil
	}
	override := make(map[string]struct{})

	for key := range props {
		override[key] = struct{}{}
	}

	for key := range c.Props {
		override[key] = struct{}{}
	}

	return override
}

func (i *Inertia) setPropValue(c *State, page *PageDTO, key string, value any, partial *partialConfig) {
	if value == nil {
		i.setNilProp(c, page, key, partial)
		return
	}

	if next, op := peelOnce(value, 0); op != nil {
		if _, skip := i.applyOnceProp(page, key, *op, partial); skip {
			return
		}
		value = next
		if value == nil {
			i.setNilProp(c, page, key, partial)
			return
		}
	}

	if i.handleWrappedProp(c, page, key, value, partial) {
		return
	}

	if !i.shouldIncludeProp(key, partial) {
		return
	}

	result, included, err := i.resolveSelectedPropValue(c, key, value, partial.selectionForProp(key))
	if err != nil {
		i.logger.WarnContext(c.Context, "failed to evaluate prop", "key", key, "error", err)
		return
	}
	if included {
		page.Props[key] = result
	} else {
		omitUnselectedProp(c, page, key)
	}
}

func (i *Inertia) setNilProp(c *State, page *PageDTO, key string, partial *partialConfig) {
	if !partial.selectionForProp(key).only {
		page.Props[key] = nil
	} else {
		omitUnselectedProp(c, page, key)
	}
}

func (i *Inertia) applyOnceProp(page *PageDTO, key string, op OnceProp, partial *partialConfig) (any, bool) {
	if !i.shouldIncludeProp(key, partial) {
		return nil, true
	}
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

	settings := onceValueSettings(op.Value)
	refresh := settings.fresh || (settings.refreshOnPartial && partial != nil && partial.isPartial)
	if !refresh && partial != nil && partial.shouldSkipOnce(onceKey, key) {
		return nil, true
	}

	return settings.value, false
}

func (i *Inertia) handleWrappedProp(c *State, page *PageDTO, key string, value any, partial *partialConfig) bool {
	switch prop := value.(type) {
	case DeferredProp:
		return i.handleDeferredProp(c, page, key, prop, partial)
	case OptionalProp:
		return i.handleOptionalProp(c, page, key, prop, partial)
	case AlwaysProp:
		return i.handleAlwaysProp(c, page, key, prop, partial)
	case NestedMergeProp:
		return i.handleNestedMergeProp(c, page, key, prop, partial)
	case MergeProp:
		return i.handleMergeProp(c, page, key, prop, partial)
	case ScrollProp:
		return i.handleScrollProp(c, page, key, prop, partial)
	default:
		return false
	}
}

func (i *Inertia) handleDeferredProp(c *State, page *PageDTO, key string, prop DeferredProp, partial *partialConfig) bool {
	if partial != nil && partial.isPartial {
		if partial.shouldIncludeProp(key) {
			i.setPropValue(c, page, key, prop.Value, partial)
		}
		return true
	}

	group := prop.Group
	if group == "" {
		group = defaultDeferredGroup
	}
	if page.DeferredProps == nil {
		page.DeferredProps = make(map[string][]string)
	}
	page.DeferredProps[group] = appendUnique(page.DeferredProps[group], key)
	return true
}

func (i *Inertia) handleOptionalProp(c *State, page *PageDTO, key string, prop OptionalProp, partial *partialConfig) bool {
	if partial == nil || !partial.isPartial || !partial.shouldIncludeProp(key) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleAlwaysProp(c *State, page *PageDTO, key string, prop AlwaysProp, partial *partialConfig) bool {
	if partial != nil {
		if partial.forceInclude == nil {
			partial.forceInclude = make(map[string]struct{})
		}
		partial.forceInclude[key] = struct{}{}
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	return true
}

func (i *Inertia) handleMergeProp(c *State, page *PageDTO, key string, prop MergeProp, partial *partialConfig) bool {
	if !i.shouldIncludeProp(key, partial) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	if _, resolved := page.Props[key]; !resolved {
		return true
	}
	if partial == nil || !partial.isReset(key) {
		switch {
		case prop.Prepend:
			appendPropMetadata(c, page, key, key, prependMetadata)
		case prop.Deep:
			appendPropMetadata(c, page, key, key, deepMetadata)
		default:
			appendPropMetadata(c, page, key, key, appendMetadata)
		}
	}
	return true
}

func (i *Inertia) handleScrollProp(c *State, page *PageDTO, key string, prop ScrollProp, partial *partialConfig) bool {
	if !i.shouldIncludeProp(key, partial) {
		return true
	}
	i.setPropValue(c, page, key, prop.Value, partial)
	if _, resolved := page.Props[key]; !resolved {
		return true
	}
	if page.ScrollProps == nil {
		page.ScrollProps = make(map[string]ScrollPropConfig)
	}
	cfg := prop.Config
	page.ScrollProps[key] = cfg

	if partial != nil && partial.isReset(key) {
		return true
	}
	mergePath := key
	if paginator, ok := page.Props[key].(map[string]any); ok {
		if _, hasData := paginator["data"]; hasData {
			mergePath += ".data"
		}
	}
	if partial != nil && partial.scrollMergeIntent == "prepend" {
		appendPropMetadata(c, page, key, mergePath, prependMetadata)
	} else {
		appendPropMetadata(c, page, key, mergePath, appendMetadata)
	}

	return true
}

// renderJSON renders the page as JSON for Inertia requests.
func (i *Inertia) RenderHTML(c *State, page *PageDTO) ([]byte, error) {
	rootTemplate, err := i.RootTemplate()
	if err != nil {
		return nil, err
	}

	viewData, err := i.createViewData(c)
	if err != nil {
		return nil, err
	}

	viewData["page"] = page
	viewData["pageJSON"] = pageResponseWithState(page, c)

	ssr, err := i.renderSSR(c, viewData["pageJSON"])
	if err != nil {
		return nil, err
	}
	viewData["processSSR"] = ssr
	if err := i.bootstrapViewData(viewData, ssr); err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	err = rootTemplate.Execute(&buf, viewData)
	if err != nil {
		return nil, fmt.Errorf("error executing template: %w", err)
	}

	return buf.Bytes(), nil
}

// RenderHTMLError renders the page as HTML template.
func (i *Inertia) RootTemplate() (*template.Template, error) {
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

func (i *Inertia) ErrorTemplate() (*template.Template, error) {
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

func (i *Inertia) createViewData(c *State) (map[string]any, error) {
	viewData := make(map[string]any)

	// Add shared view data
	for key, value := range i.sharedViewData {
		viewData[key] = value
	}

	// Add context view data
	if c.InvalidViewData {
		return nil, ErrInvalidContextViewData
	}
	for key, value := range c.ViewData {
		viewData[key] = value
	}

	// Check Vite dev server.
	if hotURL := i.HotServerURL(); hotURL != "" {
		viewData["hotServerUrl"] = hotURL
	}

	return viewData, nil
}

func (i *Inertia) HotServerURL() string {
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

func (i *Inertia) cacheLazy(c *State, path string, nested bool, lazy LazyProp) (any, error) {
	cache := c.lazyCache
	if cache == nil {
		cache = make(map[lazyCacheKey]any)
		c.lazyCache = cache
	}
	key := lazyCacheKey{path: path, source: c.propSource, nested: nested}

	if value, ok := cache[key]; ok {
		return value, nil
	}

	result, err := lazy.Fn(c.propContext())
	if err != nil {
		return nil, err
	}

	cache[key] = result

	return result, nil
}

func (i *Inertia) resolvePropValue(c *State, key string, value any) (any, error) {
	switch val := value.(type) {
	case LazyProp:
		result, err := i.cacheLazy(c, key, false, val)
		if err != nil {
			return nil, err
		}
		resolved, _, err := i.resolveContainer(c, key, result, 0)
		return resolved, err
	case func(context.Context) (any, error):
		result, err := val(c.propContext())
		if err != nil {
			return nil, err
		}
		resolved, _, err := i.resolveContainer(c, key, result, 0)
		return resolved, err
	case map[string]any, []any:
		resolved, _, err := i.resolveContainer(c, key, val, 0)
		return resolved, err
	default:
		return value, nil
	}
}

func (i *Inertia) applyPageMeta(c *State, page *PageDTO) {
	pm := c.pageMeta
	if pm == nil {
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

	if pm.encryptHistory != nil {
		page.EncryptHistory = *pm.encryptHistory
	}
	if pm.clearHistory != nil {
		page.ClearHistory = *pm.clearHistory
	}
}

func (i *Inertia) ensureErrorsProp(_ *State, page *PageDTO) {
	if page == nil {
		return
	}
	if _, ok := page.Props[ContextPropsErrors]; !ok {
		page.Props[ContextPropsErrors] = map[string]string{}
	}
}

func (i *Inertia) applyErrorBag(c *State, page *PageDTO) {
	if page == nil {
		return
	}
	bag := strings.TrimSpace(c.Meta.ErrorBag)
	if bag == "" {
		return
	}
	if i.IsProtocolV3() {
		page.Props[ContextPropsErrors] = map[string]any{bag: page.Props[ContextPropsErrors]}
		return
	}

	flat := flattenValidationErrors(page.Props[ContextPropsErrors])
	if flat == nil {
		page.Props[ContextPropsErrors] = map[string]map[string]string{bag: {}}
		return
	}
	page.Props[ContextPropsErrors] = map[string]map[string]string{bag: flat}
}

func (i *Inertia) IsExternalRedirect(target string) bool {
	parsed, err := url.Parse(target)
	if err != nil || (parsed.Scheme == "" && parsed.Host == "") {
		return false
	}
	base, err := url.Parse(i.baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return true
	}
	if parsed.Scheme == "" {
		parsed.Scheme = base.Scheme
	}
	return !strings.EqualFold(base.Scheme, parsed.Scheme) || !strings.EqualFold(base.Host, parsed.Host)
}

func (p *partialConfig) shouldIncludeProp(key string) bool {
	return p.selectionForProp(key).mayInclude()
}

// NormalizeConfig applies template defaults after adapter options. Call only before serving.
func (i *Inertia) NormalizeConfig() {
	if i.rootHotTemplate == "" {
		i.rootHotTemplate = "hot"
	}

	if i.rootTemplate == "" {
		i.rootTemplate = "app.gohtml"
	}

	if i.rootErrorTemplate == "" {
		i.rootErrorTemplate = "error.gohtml"
	}
}

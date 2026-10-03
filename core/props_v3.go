package core

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

type v3Resolver struct {
	engine                      *Inertia
	state                       *State
	page                        *PageDTO
	partial                     bool
	only, except, reset, loaded map[string]struct{}
}

type v3Prop struct {
	value                              any
	deferred, optional, always, rescue bool
	group                              string
	wrapped                            bool
	once                               *OnceProp
	merge                              *MergeProp
	targets                            []MergeTarget
	scroll                             *ScrollAtProp
}

func (i *Inertia) buildPageV3(c *State, component string, props map[string]any) (*PageDTO, error) {
	c.propMetadata = nil
	c.nestedMergeRoots = nil
	c.v3 = &v3PageMetadata{preserveBigIntegers: i.preserveBigIntegers}
	page := &PageDTO{Component: component, Props: make(map[string]any), URL: c.Meta.URL, Version: i.assetVersion}
	r := &v3Resolver{
		engine: i, state: c, page: page, partial: c.Meta.PartialComponent == component,
		only: parseHeaderList(c.Meta.PartialOnly), except: parseHeaderList(c.Meta.PartialExcept),
		reset: parseHeaderList(c.Meta.Reset), loaded: parseHeaderList(c.Meta.ExceptOnceProps),
	}
	values := make(map[string]any)
	sources := make(map[string]propSource)
	copyLayer := func(layer map[string]any, source propSource) {
		for key, value := range layer {
			values[key], sources[key] = value, source
		}
	}
	copyLayer(i.sharedProps, sharedSource)
	for key := range i.sharedProps {
		c.v3.sharedProps = append(c.v3.sharedProps, key)
	}
	slices.Sort(c.v3.sharedProps)
	for _, key := range []string{ContextPropsFlash, ContextPropsErrors, ContextPropsOld} {
		if value, ok := c.FlashData[key]; ok {
			values[key], sources[key] = value, flashSource
		}
	}
	copyLayer(c.Props, contextSource)
	copyLayer(props, requestSource)
	previous := c.propSource
	defer func() { c.propSource = previous }()
	for _, key := range sortedPropKeys(values) {
		c.propSource = sources[key]
		forced := key == ContextPropsErrors || key == ContextPropsFlash || key == ContextPropsOld ||
			(i.csrfEnabled && key == i.csrfPropName)
		value, included, err := r.resolve(key, nestedMapPath("", key), values[key], forced, 0)
		if err != nil {
			return nil, err
		}
		if included {
			page.Props[key] = value
		}
	}
	i.applyPageMeta(c, page)
	i.ensureErrorsProp(c, page)
	i.applyErrorBag(c, page)
	return page, nil
}

func sortedPropKeys(props map[string]any) []string {
	keys := make([]string, 0, len(props))
	for key := range props {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func matchesV3Path(paths map[string]struct{}, path string, ancestor bool) bool {
	for candidate := range paths {
		if path == candidate || strings.HasPrefix(path, candidate+".") ||
			(ancestor && strings.HasPrefix(candidate, path+".")) {
			return true
		}
	}
	return false
}

func (r *v3Resolver) selected(path string, ancestor bool) bool {
	return !r.partial || ((len(r.only) == 0 || matchesV3Path(r.only, path, ancestor)) &&
		!matchesV3Path(r.except, path, false))
}

// unwrapV3 treats wrapper ordering consistently and never modifies caller data.
func unwrapV3(value any, inherited v3Prop) (v3Prop, error) {
	p := inherited
	for range 65 {
		switch v := value.(type) {
		case DeferredProp:
			p.deferred, p.group, value = true, v.Group, v.Value
		case OptionalProp:
			p.optional, value = true, v.Value
		case AlwaysProp:
			p.always, value = true, v.Value
		case RescueProp:
			p.rescue, value = true, v.Value
		case OnceProp:
			p.once, value = &v, onceValueSettings(v.Value).value
		case MergeProp:
			p.merge, value = &v, v.Value
		case NestedMergeProp:
			p.targets, value = v.Targets, v.Value
		case ScrollProp:
			p.scroll, value = &ScrollAtProp{Value: v.Value, Config: v.Config, Path: "data"}, v.Value
		case ScrollAtProp:
			p.scroll, value = &v, v.Value
		default:
			p.wrapped = inherited.wrapped || p.deferred || p.optional || p.always || p.rescue ||
				p.once != nil || p.merge != nil || p.targets != nil || p.scroll != nil
			p.value = value
			return p, nil
		}
	}
	return p, errors.New("prop wrapper nesting exceeds 64 levels")
}

func (r *v3Resolver) resolve(path, cachePath string, value any, bypass bool, depth int) (any, bool, error) {
	return r.resolveProp(path, cachePath, value, bypass, depth, v3Prop{})
}

func (r *v3Resolver) resolveProp(
	path, cachePath string, value any, bypass bool, depth int, inherited v3Prop,
) (any, bool, error) {
	if depth > 64 {
		return nil, false, fmt.Errorf("prop %s exceeds nested resolution depth", path)
	}
	prop, err := unwrapV3(value, inherited)
	if err != nil {
		return nil, false, fmt.Errorf("prop %s: %w", path, err)
	}
	if !bypass && !prop.always && !r.selected(path, true) {
		return nil, false, nil
	}
	if r.excludeInitial(prop, path) {
		return nil, false, nil
	}
	resolved, callback, err := r.evaluate(cachePath, prop.value, depth)
	if err != nil {
		return r.failure(path, prop, err)
	}
	if callback {
		// A lazy provider may itself return wrappers, arrays or another provider.
		return r.resolveProp(path, cachePath+"c;", resolved, true, depth+1, prop)
	}
	resolved, err = r.resolveContainer(path, cachePath, resolved, bypass || prop.wrapped, depth)
	if err != nil {
		// Rescue applies to this deferred provider, not errors from independent
		// child providers after its value has already been resolved.
		return nil, false, err
	}
	r.collectMetadata(prop, path, resolved, true)
	return resolved, true, nil
}

func (r *v3Resolver) excludeInitial(prop v3Prop, path string) bool {
	if r.partial {
		return false
	}
	loaded := false
	if prop.once != nil {
		key := prop.once.Key
		if key == "" {
			key = path
		}
		_, loaded = r.loaded[key]
		loaded = loaded && r.state.Meta.Inertia != "" && !onceValueSettings(prop.once.Value).fresh
	}
	if prop.deferred || prop.optional {
		r.announceDeferred(prop, path, loaded)
		r.collectMetadata(prop, path, nil, false)
		return true
	}
	if loaded {
		r.collectOnce(prop, path)
		return true
	}
	return false
}

func (r *v3Resolver) announceDeferred(prop v3Prop, path string, loaded bool) {
	if !prop.deferred || loaded {
		return
	}
	group := prop.group
	if group == "" {
		group = defaultDeferredGroup
	}
	if r.page.DeferredProps == nil {
		r.page.DeferredProps = make(map[string][]string)
	}
	r.page.DeferredProps[group] = appendUnique(r.page.DeferredProps[group], path)
}

func (r *v3Resolver) evaluate(path string, value any, depth int) (any, bool, error) {
	if err := r.state.Context.Err(); err != nil {
		return nil, false, err
	}
	switch callback := value.(type) {
	case LazyProp:
		if callback.Fn == nil {
			return nil, true, errors.New("nil lazy callback")
		}
		result, err := r.engine.cacheLazy(r.state, path, depth != 0, callback)
		return result, true, err
	case func(context.Context) (any, error):
		if callback == nil {
			return nil, true, errors.New("nil prop callback")
		}
		result, err := callback(r.state.propContext())
		return result, true, err
	default:
		return value, false, nil
	}
}

func (r *v3Resolver) failure(path string, prop v3Prop, err error) (any, bool, error) {
	if !prop.rescue || !prop.deferred || !r.partial || r.state.Context.Err() != nil {
		return nil, false, fmt.Errorf("resolve prop %s: %w", path, err)
	}
	if logger, ok := r.engine.logger.(*LoggerAdapter); r.engine.logger == nil || (ok && logger.logger == nil) {
		slog.WarnContext(r.state.Context, "rescued deferred prop", "key", path, "error", err)
	} else {
		r.engine.logger.WarnContext(r.state.Context, "rescued deferred prop", "key", path, "error", err)
	}
	r.state.v3.rescuedProps = appendUnique(r.state.v3.rescuedProps, path)
	return nil, false, nil
}

func (r *v3Resolver) resolveContainer(path, cachePath string, value any, bypass bool, depth int) (any, error) {
	if values, ok := value.(map[string]any); ok {
		if values == nil {
			return values, nil
		}
		result := make(map[string]any, len(values))
		for _, key := range sortedPropKeys(values) {
			child, included, err := r.resolve(path+"."+key, nestedMapPath(cachePath, key), values[key], bypass, depth+1)
			if err != nil {
				return nil, err
			}
			if included {
				result[key] = child
			}
		}
		return result, nil
	}
	if values, ok := value.([]any); ok {
		if values == nil {
			return values, nil
		}
		result := make([]any, len(values))
		kept := make(map[string]any, len(values))
		for index, value := range values {
			key := strconv.Itoa(index)
			child, included, err := r.resolve(path+"."+key, cachePath+"i"+key+";", value, bypass, depth+1)
			if err != nil {
				return nil, err
			}
			if included {
				result[index] = child
				kept[key] = child
			}
		}
		if len(kept) != len(values) {
			return kept, nil
		}
		return result, nil
	}
	return r.resolveTypedContainer(path, cachePath, value, bypass, depth)
}

// Typed JSON maps and slices receive the same wrapper semantics. Structs and
// custom JSON marshalers stay opaque, retaining their own serialization rules.
func (r *v3Resolver) resolveTypedContainer(path, cachePath string, value any, bypass bool, depth int) (any, error) {
	if _, ok := value.(interface{ MarshalJSON() ([]byte, error) }); ok {
		return value, nil
	}
	v := reflect.ValueOf(value)
	if !v.IsValid() {
		return value, nil
	}
	switch v.Kind() {
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String || v.IsNil() {
			return value, nil
		}
		props := make(map[string]any, v.Len())
		for _, key := range v.MapKeys() {
			props[key.String()] = v.MapIndex(key).Interface()
		}
		return r.resolveContainer(path, cachePath, props, bypass, depth)
	case reflect.Slice, reflect.Array:
		if v.Type().Elem().Kind() == reflect.Uint8 || (v.Kind() == reflect.Slice && v.IsNil()) {
			return value, nil
		}
		values := make([]any, v.Len())
		for index := range values {
			values[index] = v.Index(index).Interface()
		}
		return r.resolveContainer(path, cachePath, values, bypass, depth)
	default:
		return value, nil
	}
}

func (r *v3Resolver) collectOnce(prop v3Prop, path string) {
	if prop.once == nil || !r.selected(path, false) {
		return
	}
	key := prop.once.Key
	if key == "" {
		key = path
	}
	if r.page.OnceProps == nil {
		r.page.OnceProps = make(map[string]OncePropConfig)
	}
	r.page.OnceProps[key] = OncePropConfig{Prop: path, ExpiresAt: prop.once.ExpiresAt}
}

func (r *v3Resolver) collectMetadata(prop v3Prop, path string, value any, resolved bool) {
	r.collectOnce(prop, path)
	if prop.scroll != nil && resolved {
		if r.page.ScrollProps == nil {
			r.page.ScrollProps = make(map[string]ScrollPropConfig)
		}
		cfg := prop.scroll.Config
		if cfg.PageName == "" {
			cfg.PageName = "page"
		}
		r.page.ScrollProps[path] = cfg
	}
	if matchesV3Path(r.reset, path, false) || !r.selected(path, false) {
		return
	}
	if prop.merge != nil {
		kind := appendMetadata
		if prop.merge.Prepend {
			kind = prependMetadata
		} else if prop.merge.Deep {
			kind = deepMetadata
		}
		appendPropMetadata(r.state, r.page, path, path, kind)
	}
	r.collectTargets(prop, path, value, resolved)
	r.collectScrollMerge(prop, path)
}

func (r *v3Resolver) collectTargets(prop v3Prop, path string, value any, resolved bool) {
	for _, target := range prop.targets {
		if resolved && !mergePathExists(value, target.Path) {
			continue
		}
		targetPath := path
		if target.Path != "" {
			targetPath += "." + target.Path
		}
		if !r.selected(targetPath, false) || matchesV3Path(r.reset, targetPath, false) {
			continue
		}
		kind := appendMetadata
		if target.Prepend {
			kind = prependMetadata
		}
		appendPropMetadata(r.state, r.page, path, targetPath, kind)
		if target.MatchOn != "" {
			appendPropMetadata(r.state, r.page, path, targetPath+"."+target.MatchOn, matchMetadata)
		}
	}
}

func (r *v3Resolver) collectScrollMerge(prop v3Prop, path string) {
	if prop.scroll != nil {
		targetPath := path
		if prop.scroll.Path != "" {
			targetPath += "." + prop.scroll.Path
		}
		if !r.selected(targetPath, false) || matchesV3Path(r.reset, targetPath, false) {
			return
		}
		kind := appendMetadata
		if r.state.Meta.ScrollMergeIntent == "prepend" {
			kind = prependMetadata
		}
		appendPropMetadata(r.state, r.page, path, targetPath, kind)
	}
}

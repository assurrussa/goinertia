package core

import (
	"context"
	"fmt"
	"strings"
)

// A zero selection includes the whole value. An only selection with no paths
// includes nothing. Paths are relative to the current JSON map.
type propSelection struct {
	only  bool
	paths map[string]struct{}
}

func (p *partialConfig) selectionForProp(key string) propSelection {
	if p == nil || !p.isPartial {
		return propSelection{}
	}
	if _, forced := p.forceInclude[key]; forced {
		return propSelection{}
	}
	if p.hasExclude {
		return (propSelection{paths: p.exclude}).child(key)
	}
	if p.hasInclude {
		return (propSelection{only: true, paths: p.include}).child(key)
	}
	return propSelection{}
}

func (s propSelection) mayInclude() bool { return !s.only || len(s.paths) > 0 }

// A token matches both an exact literal key and a nested path. For example,
// auth.user selects both a literal "auth.user" prop and auth["user"]. The rule
// applies at every map level and never depends on map iteration order. Selecting
// a parent includes its whole subtree; excluding a parent removes that subtree.
func (s propSelection) child(key string) propSelection {
	child := propSelection{only: s.only}
	for path := range s.paths {
		if path == key {
			return propSelection{only: !s.only}
		}
		if suffix, ok := strings.CutPrefix(path, key+"."); ok {
			if child.paths == nil {
				child.paths = make(map[string]struct{})
			}
			child.paths[suffix] = struct{}{}
		}
	}
	return child
}

func (i *Inertia) resolveSelectedPropValue(c *State, key string, value any, selection propSelection) (any, bool, error) {
	return i.resolveSelectedValue(c, key, value, selection, 0)
}

func (i *Inertia) resolveSelectedValue(c *State, path string, value any, selection propSelection, depth int) (any, bool, error) {
	if !selection.mayInclude() {
		return nil, false, nil
	}
	if len(selection.paths) == 0 {
		if depth == 0 {
			resolved, err := i.resolvePropValue(c, path, value)
			return resolved, true, err
		}
		resolved, _, err := i.resolveChild(c, path, value, depth)
		return resolved, true, err
	}

	var err error
	switch callback := value.(type) {
	case LazyProp:
		value, err = i.cacheLazy(c, path, depth != 0, callback)
	case func(context.Context) (any, error):
		value, err = callback(c.propContext())
	}
	if err != nil {
		return nil, false, err
	}
	return i.resolveSelectedContainer(c, path, value, selection, depth)
}

func (i *Inertia) resolveSelectedContainer(
	c *State, path string, value any, selection propSelection, depth int,
) (any, bool, error) {
	if depth > 64 {
		return nil, false, fmt.Errorf("prop %s exceeds nested callback depth", path)
	}
	values, isMap := value.(map[string]any)
	if !isMap {
		// Dotted selection traverses only JSON maps, not structs, typed maps or
		// array indexes. An except path missing from an opaque value is a no-op.
		if selection.only {
			return nil, false, nil
		}
		resolved, _, err := i.resolveContainer(c, path, value, depth)
		return resolved, true, err
	}
	if values == nil && !selection.only {
		return value, true, nil
	}

	path = nestedRootPath(path, depth)
	result := make(map[string]any, len(values))
	for key, child := range values {
		resolved, included, err := i.resolveSelectedValue(c, nestedMapPath(path, key), child, selection.child(key), depth+1)
		if err != nil {
			return nil, false, err
		}
		if included {
			result[key] = resolved
		}
	}
	if selection.only && len(result) == 0 {
		return nil, false, nil
	}
	return result, true, nil
}

// Missing selected descendants must not restore a lower-precedence layer's
// value or metadata for a prop that was actually filtered away.
func omitUnselectedProp(c *State, page *PageDTO, key string) {
	delete(page.Props, key)
	clearMergeMetadata(c, page, key)
	for group, keys := range page.DeferredProps {
		kept := keys[:0]
		for _, prop := range keys {
			if prop != key {
				kept = append(kept, prop)
			}
		}
		if len(kept) == 0 {
			delete(page.DeferredProps, group)
		} else {
			page.DeferredProps[group] = kept
		}
	}
	for onceKey, config := range page.OnceProps {
		if config.Prop == key {
			delete(page.OnceProps, onceKey)
		}
	}
}

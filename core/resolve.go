package core

import (
	"context"
	"fmt"
	"maps"
	"strconv"
)

// JSON containers are copied only when a callback changes a child. Shared host
// maps/slices are never mutated while resolving request-owned values.
func (i *Inertia) resolveContainer(c *State, path string, value any, depth int) (any, bool, error) {
	if depth > 64 {
		return nil, false, fmt.Errorf("prop %s exceeds nested callback depth", path)
	}
	switch values := value.(type) {
	case map[string]any:
		path = nestedRootPath(path, depth)
		var result map[string]any
		for key, child := range values {
			resolved, changed, err := i.resolveChild(c, nestedMapPath(path, key), child, depth+1)
			if err != nil {
				return nil, false, err
			}
			if changed {
				if result == nil {
					result = maps.Clone(values)
				}
				result[key] = resolved
			}
		}
		if result != nil {
			return result, true, nil
		}
	case []any:
		path = nestedRootPath(path, depth)
		var result []any
		for index, child := range values {
			resolved, changed, err := i.resolveChild(c, path+"i"+strconv.Itoa(index)+";", child, depth+1)
			if err != nil {
				return nil, false, err
			}
			if changed {
				if result == nil {
					result = append([]any(nil), values...)
				}
				result[index] = resolved
			}
		}
		if result != nil {
			return result, true, nil
		}
	}
	return value, false, nil
}

func nestedRootPath(path string, depth int) string {
	if depth == 0 {
		return nestedMapPath("", path)
	}
	return path
}

// Length-prefixed map segments and distinct array segments are injective even
// for keys containing dots, separators, empty strings or numeric indices.
func nestedMapPath(parent, key string) string {
	return parent + "m" + strconv.Itoa(len(key)) + ":" + key
}

func (i *Inertia) resolveChild(c *State, path string, value any, depth int) (any, bool, error) {
	var result any
	var err error
	switch child := value.(type) {
	case LazyProp:
		result, err = i.cacheLazy(c, path, true, child)
	case func(context.Context) (any, error):
		result, err = child(c.propContext())
	default:
		return i.resolveContainer(c, path, value, depth)
	}
	if err != nil {
		return nil, false, err
	}
	resolved, _, err := i.resolveContainer(c, path, result, depth)
	return resolved, true, err
}

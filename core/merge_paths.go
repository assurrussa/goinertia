package core

import "strings"

// MergeTarget describes one append or prepend target, relative to a root prop.
// An empty Path targets the root. MatchOn is relative to each array item.
type MergeTarget struct {
	Path    string
	Prepend bool
	MatchOn string
}

// NestedMergeProp marks selected nested values as mergeable. It leaves the
// original MergeProp layout and root merge constructors unchanged.
type NestedMergeProp struct {
	Value   any
	Targets []MergeTarget
}

// AppendAt appends an array at path, optionally matching items by a field.
func AppendAt(path string, matchOn ...string) MergeTarget {
	return mergeTarget(path, false, matchOn)
}

// PrependAt prepends an array at path, optionally matching items by a field.
func PrependAt(path string, matchOn ...string) MergeTarget {
	return mergeTarget(path, true, matchOn)
}

func mergeTarget(path string, prepend bool, matchOn []string) MergeTarget {
	target := MergeTarget{Path: path, Prepend: prepend}
	if len(matchOn) > 0 {
		target.MatchOn = matchOn[0]
	}
	return target
}

// MergeAt configures independent merge strategies within a JSON-shaped prop.
// Targets are copied so later changes to the caller's slice cannot alter it.
func MergeAt(value any, targets ...MergeTarget) NestedMergeProp {
	return NestedMergeProp{Value: value, Targets: append([]MergeTarget(nil), targets...)}
}

func (i *Inertia) handleNestedMergeProp(c *State, page *PageDTO, key string, prop NestedMergeProp, partial *partialConfig) bool {
	if !i.shouldIncludeProp(key, partial) {
		return true
	}
	if c.nestedMergeRoots == nil {
		c.nestedMergeRoots = make(map[string]bool)
	}
	c.nestedMergeRoots[key] = true
	i.setPropValue(c, page, key, prop.Value, partial)
	value, resolved := page.Props[key]
	if !resolved || partial.isReset(key) {
		return true
	}
	for _, target := range prop.Targets {
		if !mergePathExists(value, target.Path) {
			continue
		}
		path := key
		if target.Path != "" {
			path += "." + target.Path
		}
		if target.Prepend {
			appendPropMetadata(c, page, key, path, prependMetadata)
		} else {
			appendPropMetadata(c, page, key, path, appendMetadata)
		}
		if target.MatchOn != "" {
			appendPropMetadata(c, page, key, path+"."+target.MatchOn, matchMetadata)
		}
	}
	return true
}

func mergePathExists(value any, path string) bool {
	if path == "" {
		return true
	}
	for _, segment := range strings.Split(path, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			return false
		}
		value, ok = object[segment]
		if !ok {
			return false
		}
	}
	return true
}

// Nested merge is an additive API with replacement semantics across prop
// layers. Detect its supported wrapper chain before applying Once/defer policy.
func hasNestedMerge(value any) bool {
	for range 65 {
		switch prop := value.(type) {
		case NestedMergeProp:
			return true
		case OnceProp:
			value = onceValueSettings(prop.Value).value
		case DeferredProp:
			value = prop.Value
		case OptionalProp:
			value = prop.Value
		case AlwaysProp:
			value = prop.Value
		case MergeProp:
			value = prop.Value
		default:
			return false
		}
	}
	return false
}

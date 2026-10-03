package core

// onceSettings is carried by Value to preserve OnceProp's positional layout.
type onceSettings struct {
	value            any
	fresh            bool
	refreshOnPartial bool
}

// WithOnceFresh bypasses remembered-value omission for this response. It does
// not bypass partial selection, optional inclusion, or deferred loading.
// Omit enabled to enable freshness; an explicit false disables it.
func WithOnceFresh(enabled ...bool) OnceOption {
	return func(op *OnceProp) {
		settings := onceValueSettings(op.Value)
		settings.fresh = len(enabled) == 0 || enabled[0]
		op.Value = settings
	}
}

// WithOnceRefreshOnPartial opts into the v2 reference adapter's refresh policy:
// any selected matching partial request, including except-only, refreshes Once.
// Without it, the compatible policy refreshes only explicitly selected props.
func WithOnceRefreshOnPartial() OnceOption {
	return func(op *OnceProp) {
		settings := onceValueSettings(op.Value)
		settings.refreshOnPartial = true
		op.Value = settings
	}
}

func onceValueSettings(value any) onceSettings {
	if settings, ok := value.(onceSettings); ok {
		return settings
	}
	return onceSettings{value: value}
}

// peelOnce recognizes only documented root wrapper compositions, never nested
// map elements or typed containers. Copying wrappers preserves caller ownership.
func peelOnce(value any, depth int) (any, *OnceProp) {
	if depth > 64 {
		return value, nil
	}
	switch prop := value.(type) {
	case OnceProp:
		settings := onceValueSettings(prop.Value)
		return settings.value, &prop
	case DeferredProp:
		next, once := peelOnce(prop.Value, depth+1)
		prop.Value = next
		return prop, once
	case OptionalProp:
		next, once := peelOnce(prop.Value, depth+1)
		prop.Value = next
		return prop, once
	case MergeProp:
		next, once := peelOnce(prop.Value, depth+1)
		prop.Value = next
		return prop, once
	case NestedMergeProp:
		next, once := peelOnce(prop.Value, depth+1)
		prop.Value = next
		return prop, once
	default:
		return value, nil
	}
}

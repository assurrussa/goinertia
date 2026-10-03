package core

import "maps"

// This key is private to the session envelope, never a page prop. Keeping it
// alongside the legacy fields lets existing SessionStore bridges save and
// consume both flash APIs together without extra cookie/session operations.
const nativeFlashSessionKey = "__goinertia_native_flash"

// WithNativeFlash adds a JSON-serializable value to top-level page.flash.
// Multiple calls merge keys; the most recent value for a key wins. Current
// request values override session values. Native flash bypasses partial prop
// filtering and is separate from the legacy WithFlash helpers and props.flash.
// A SessionStore is needed only to carry values across a redirect.
func (i *Inertia) WithNativeFlash(c *State, key string, value any) {
	meta := i.getContextKeyPageMeta(c)
	if meta.nativeFlash == nil {
		meta.nativeFlash = make(map[string]any)
	}
	meta.nativeFlash[key] = value
}

func (s *State) nativeFlash() map[string]any {
	if s.Meta.Precognition != "" {
		return nil
	}
	stored, _ := s.FlashData[nativeFlashSessionKey].(map[string]any)
	var current map[string]any
	if s.pageMeta != nil {
		current = s.pageMeta.nativeFlash
	}
	if len(stored) == 0 {
		return current
	}
	if len(current) == 0 {
		return stored
	}
	merged := maps.Clone(stored)
	maps.Copy(merged, current)
	return merged
}

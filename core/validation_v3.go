package core

import "maps"

// WithAllValidationErrors retains all messages per field for a v3 response.
// WithValidationErrors keeps its historical first-message behavior. Both forms
// participate in named error bags and Precognition.
func (i *Inertia) WithAllValidationErrors(state *State, values ValidationErrors) {
	if !i.IsProtocolV3() {
		i.WithValidationErrors(state, values)
		return
	}
	if len(values) == 0 {
		return
	}
	merged := normalizeValidationErrors(i.getContextKeyProps(state)[ContextPropsErrors])
	if merged == nil {
		merged = make(ValidationErrors, len(values))
	}
	for key, messages := range values {
		merged[key] = append([]string(nil), messages...)
	}
	i.WithProp(state, ContextPropsErrors, merged)
}

func (i *Inertia) mergeV3Errors(state *State, values map[string]string) {
	current := i.getContextKeyProps(state)[ContextPropsErrors]
	merged := make(map[string]any)
	switch existing := current.(type) {
	case map[string]any:
		merged = maps.Clone(existing)
	case map[string]string:
		for key, value := range existing {
			merged[key] = value
		}
	default:
		for key, value := range normalizeValidationErrors(existing) {
			merged[key] = value
		}
	}
	if merged == nil {
		merged = make(map[string]any)
	}
	for key, value := range values {
		merged[key] = value
	}
	i.WithProp(state, ContextPropsErrors, merged)
}

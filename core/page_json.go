package core

import "github.com/goccy/go-json"

type scrollResponse struct {
	ScrollPropConfig
	Reset bool `json:"reset,omitempty"`
}

type pageJSON PageDTO

// pageResponse keeps request-only wire metadata outside the public DTO/config
// structs. Their field layouts also remain compatible with positional literals.
func pageResponse(page *PageDTO, resetHeader string, flash map[string]any) any {
	if page == nil {
		return page
	}
	reset := parseHeaderList(resetHeader)
	var scroll map[string]scrollResponse
	for key := range reset {
		if _, exists := page.ScrollProps[key]; exists {
			scroll = make(map[string]scrollResponse, len(page.ScrollProps))
			break
		}
	}
	if scroll == nil {
		if len(flash) == 0 {
			return page
		}
		return &struct {
			*pageJSON
			Flash map[string]any `json:"flash"`
		}{pageJSON: (*pageJSON)(page), Flash: flash}
	}
	for key, cfg := range page.ScrollProps {
		_, shouldReset := reset[key]
		scroll[key] = scrollResponse{ScrollPropConfig: cfg, Reset: shouldReset}
	}
	return &struct {
		*pageJSON
		ScrollProps map[string]scrollResponse `json:"scrollProps,omitempty"`
		Flash       map[string]any            `json:"flash,omitempty"`
	}{pageJSON: (*pageJSON)(page), ScrollProps: scroll, Flash: flash}
}

// MarshalPage serializes a built page with request-only scroll reset metadata.
// It does not mutate the page or pagination configuration.
func MarshalPage(page *PageDTO, resetHeader string) ([]byte, error) {
	return json.Marshal(pageResponse(page, resetHeader, nil))
}

// MarshalPageWithState serializes a built page with request-only scroll reset
// metadata and opt-in native flash. Use this instead of marshaling PageDTO
// directly when building a response with the neutral core.
// Neither the page nor its request state is mutated.
func MarshalPageWithState(page *PageDTO, state *State) ([]byte, error) {
	if state == nil {
		return MarshalPage(page, "")
	}
	return json.Marshal(pageResponseWithState(page, state))
}

func pageResponseWithState(page *PageDTO, state *State) any {
	if state == nil {
		return pageResponse(page, "", nil)
	}
	if state.v3 == nil {
		return pageResponse(page, state.Meta.Reset, state.nativeFlash())
	}
	return v3PageResponse{page: page, state: state}
}

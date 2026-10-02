package core

import "github.com/goccy/go-json"

type scrollResponse struct {
	ScrollPropConfig
	Reset bool `json:"reset,omitempty"`
}

type pageJSON PageDTO

// pageResponse keeps request-only wire metadata outside the public DTO/config
// structs. Their field layouts also remain compatible with positional literals.
func pageResponse(page *PageDTO, resetHeader string) any {
	if page == nil || resetHeader == "" || len(page.ScrollProps) == 0 {
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
		return page
	}
	for key, cfg := range page.ScrollProps {
		_, shouldReset := reset[key]
		scroll[key] = scrollResponse{ScrollPropConfig: cfg, Reset: shouldReset}
	}
	return &struct {
		*pageJSON
		ScrollProps map[string]scrollResponse `json:"scrollProps,omitempty"`
	}{pageJSON: (*pageJSON)(page), ScrollProps: scroll}
}

// MarshalPage serializes a built page with request-only scroll reset metadata.
// It does not mutate the page or pagination configuration.
func MarshalPage(page *PageDTO, resetHeader string) ([]byte, error) {
	return json.Marshal(pageResponse(page, resetHeader))
}

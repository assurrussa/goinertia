package core

import (
	"bytes"
	"fmt"
	"math/big"
	"strings"

	"github.com/goccy/go-json"
)

type v3PageResponse struct {
	page  *PageDTO
	state *State
}

type v3ScrollResponse struct {
	PageName     string `json:"pageName"`
	PreviousPage any    `json:"previousPage"`
	NextPage     any    `json:"nextPage"`
	CurrentPage  any    `json:"currentPage"`
	Reset        bool   `json:"reset"`
}

func (p v3PageResponse) MarshalJSON() ([]byte, error) {
	if p.page == nil {
		return []byte("null"), nil
	}
	meta := p.state.v3
	props, flash := any(p.page.Props), any(p.state.nativeFlash())
	if len(p.state.nativeFlash()) == 0 {
		flash = nil
	}
	if meta.preserveBigIntegers {
		var err error
		props, err = encodeBigIntegers(props)
		if err != nil {
			return nil, fmt.Errorf("encode page props: %w", err)
		}
		flash, err = encodeBigIntegers(flash)
		if err != nil {
			return nil, fmt.Errorf("encode page flash: %w", err)
		}
	}
	var scroll map[string]v3ScrollResponse
	if len(p.page.ScrollProps) > 0 {
		scroll = make(map[string]v3ScrollResponse, len(p.page.ScrollProps))
		reset := parseHeaderList(p.state.Meta.Reset)
		for key, cfg := range p.page.ScrollProps {
			scroll[key] = v3ScrollResponse{
				cfg.PageName, cfg.PreviousPage, cfg.NextPage, cfg.CurrentPage,
				matchesV3Path(reset, key, false),
			}
		}
	}
	preserveFragment := p.state.pageMeta != nil && p.state.pageMeta.preserveFragment != nil &&
		*p.state.pageMeta.preserveFragment
	return json.Marshal(struct {
		*pageJSON
		Props               any                         `json:"props"`
		Flash               any                         `json:"flash,omitempty"`
		ScrollProps         map[string]v3ScrollResponse `json:"scrollProps,omitempty"`
		RescuedProps        []string                    `json:"rescuedProps,omitempty"`
		SharedProps         []string                    `json:"sharedProps,omitempty"`
		PreserveBigIntegers bool                        `json:"preserveBigIntegers,omitempty"`
		PreserveFragment    bool                        `json:"preserveFragment,omitempty"`
	}{
		(*pageJSON)(p.page), props, flash, scroll, meta.rescuedProps, meta.sharedProps,
		meta.preserveBigIntegers, preserveFragment,
	})
}

// Process the exact JSON representation, preserving custom marshalers, tags,
// embedded-field rules, byte encodings and integer precision. Integral JSON
// tokens outside JavaScript's safe range become the official $bigint marker.
func encodeBigIntegers(value any) (any, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	return wrapBigIntegers(result), nil
}

func wrapBigIntegers(value any) any {
	switch v := value.(type) {
	case json.Number:
		if strings.ContainsAny(v.String(), ".eE") {
			return v
		}
		number, ok := new(big.Int).SetString(v.String(), 10)
		if ok && number.Abs(number).Cmp(big.NewInt(9007199254740991)) > 0 {
			return map[string]string{"$bigint": v.String()}
		}
	case map[string]any:
		for key, child := range v {
			v[key] = wrapBigIntegers(child)
		}
	case []any:
		for index, child := range v {
			v[index] = wrapBigIntegers(child)
		}
	}
	return value
}

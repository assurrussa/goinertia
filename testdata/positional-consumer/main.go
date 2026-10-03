package main

import (
	"github.com/assurrussa/goinertia"
	fiberadapter "github.com/assurrussa/goinertia/adapters/fiber"
	"github.com/assurrussa/goinertia/core"
)

// These positional literals are the pre-extraction public source contract.
var (
	_ = goinertia.MergeProp{[]int{1}, false, false}
	_ = fiberadapter.MergeProp{[]int{1}, false, false}
	_ = core.MergeProp{[]int{1}, false, false}
	_ = goinertia.OnceProp{"cache-key", nil, "value"}
	_ = fiberadapter.OnceProp{"cache-key", nil, "value"}
	_ = core.OnceProp{"cache-key", nil, "value"}
	_ = goinertia.ScrollPropConfig{"page", nil, 2, 1}
	_ = fiberadapter.ScrollPropConfig{"page", nil, 2, 1}
	_ = core.ScrollPropConfig{"page", nil, 2, 1}
	_ = goinertia.PageDTO{
		"Page", nil, "/", "", false, false, nil, nil, nil, nil, nil,
		map[string]goinertia.ScrollPropConfig{"posts": {"page", nil, 2, 1}},
		nil,
	}
)

func main() {}

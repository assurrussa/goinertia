package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/assurrussa/goinertia/core"
)

func fixtureStep(raw string) int {
	step, err := strconv.Atoi(raw)
	if err != nil || step < 1 {
		return 1
	}
	return step
}

// Deterministic values let browser tests distinguish cached props from newly
// evaluated values without process-global counters or cross-client coupling.
func onceProps(adapter, rawStep string, renamed, fresh bool) map[string]any {
	step := fixtureStep(rawStep)
	lazy := func(name string) core.LazyProp {
		return core.LazyProp{Fn: func(context.Context) (any, error) {
			return fmt.Sprintf("%s-%d", name, step), nil
		}}
	}
	props := pageProps(adapter, "Props")
	props["step"] = step
	cachedOptions := []core.OnceOption{core.WithOnceKey("stable-cache")}
	if fresh {
		cachedOptions = append(cachedOptions, core.WithOnceFresh())
	}
	props["cached"] = core.Once(lazy("cached"), cachedOptions...)
	props["refreshable"] = core.Once(lazy("refreshable"), core.WithOnceRefreshOnPartial())
	// Tests advance only the browser Date clock beyond this timestamp. No sleeps
	// or dependency on server/browser clock synchronization are needed.
	props["expiring"] = core.Once(lazy("expiring"), core.WithOnceExpiresAt(time.Now().Add(time.Hour)))
	catalog := "catalog"
	if renamed {
		catalog = "renamedCatalog"
		props["title"] = "Props renamed"
	}
	props[catalog] = core.Once(lazy("catalog"), core.WithOnceKey("shared-catalog"))
	props["deferredOnce"] = core.Defer(core.Once(lazy("deferredOnce")), "composed")
	props["onceDeferred"] = core.Once(core.Defer(lazy("onceDeferred"), "composed"))
	props["optionalOnce"] = core.Optional(core.Once(lazy("optionalOnce")))
	props["onceOptional"] = core.Once(core.Optional(lazy("onceOptional")))
	return props
}

func nestedProps(adapter, rawStep string) map[string]any {
	step := fixtureStep(rawStep)
	item := func(id int, label string) map[string]any {
		return map[string]any{"id": id, "label": label}
	}
	data := []any{item(1, "first")}
	older := []any{item(10, "tenth")}
	if step == 2 {
		data = []any{item(1, "first updated"), item(2, "second")}
		older = []any{item(9, "ninth"), item(10, "tenth updated")}
	} else if step > 2 {
		data = []any{item(3, "third")}
		older = []any{item(8, "eighth")}
	}
	props := pageProps(adapter, "Nested")
	props["feed"] = core.MergeAt(map[string]any{
		"data": data, "older": older, "page": step,
	}, core.AppendAt("data", "id"), core.PrependAt("older", "id"))
	props["selection"] = map[string]any{
		"profile": map[string]any{
			"name":  fmt.Sprintf("name-%d", step),
			"email": fmt.Sprintf("user-%d@example.test", step),
		},
		"settings": map[string]any{"theme": fmt.Sprintf("theme-%d", step)},
	}
	return props
}

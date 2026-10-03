// Package consumer exercises the public native HTTP API without Fiber imports.
package consumer

import (
	"context"
	"fmt"
	"net/http"

	"github.com/assurrussa/goinertia/adapters/nethttp"
	"github.com/assurrussa/goinertia/core"
	"github.com/assurrussa/goinertia/views"
)

// New constructs a native handler and validates templates before serving.
func New(options ...core.Option) (http.Handler, error) {
	options = append([]core.Option{core.WithFS(views.Templates), core.WithAssetVersion("v1")}, options...)
	i, err := nethttp.NewWithValidation("https://app.example", nethttp.WithCoreOptions(options...))
	if err != nil {
		return nil, err
	}
	return i.Middleware(i.Handler(func(w http.ResponseWriter, r *http.Request) error {
		i.WithProp(nethttp.State(r), "path", r.URL.Path)
		return i.Render(w, r, "Consumer", map[string]any{
			"native": core.LazyProp{Fn: func(ctx context.Context) (any, error) {
				request, ok := nethttp.Request(ctx)
				if !ok || nethttp.State(request) == nil {
					return nil, fmt.Errorf("missing native request state")
				}
				return request.URL.Path, nil
			}},
		})
	})), nil
}

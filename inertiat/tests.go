package inertiat

import (
	"html/template"
	"log/slog"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/testdata"
)

func NewForTest(baseURL string, opts ...goinertia.Option) *goinertia.Inertia {
	options := make([]goinertia.Option, 0, 12+len(opts))
	options = append(
		options,
		goinertia.WithAssetVersion("v1.0"),
		goinertia.WithRootTemplate("app.gohtml"),
		goinertia.WithRootErrorTemplate("error.gohtml"),
		goinertia.WithRootHotTemplate("public/static/hot"),
		goinertia.WithLogger(goinertia.NewLoggerAdapter(slog.Default())),
		goinertia.WithSetSharedFuncMap(template.FuncMap{
			"embed": func(val string) string {
				return "embed-test123" + val + ".html"
			},
		}),
		goinertia.WithFS(testdata.Files),
		goinertia.WithPublicFS(testdata.Files),
		goinertia.WithCanExposeDetails(goinertia.DefaultCanExpose),
		goinertia.WithCustomErrorGettingHandler(goinertia.DefaultCustomGettingError),
		goinertia.WithCustomErrorDetailsHandler(goinertia.DefaultCustomErrorDetails),
		goinertia.WithSharedViewData(map[string]any{
			"testViewDataKey": "test_view_data_VALUE",
		}),
	)

	options = append(options, opts...)

	return goinertia.New(baseURL, options...)
}

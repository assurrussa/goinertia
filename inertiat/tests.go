package inertiat

import (
	"html/template"
	"log/slog"

	"github.com/assurrussa/goinertia"
	"github.com/assurrussa/goinertia/testdata"
)

func NewForTest(baseURL string, opts ...goinertia.Option) *goinertia.Inertia {
	o := []goinertia.Option{
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
	}

	o = append(o, opts...)

	return goinertia.New(baseURL, o...)
}

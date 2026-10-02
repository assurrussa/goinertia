package fiberadapter

import (
	"html/template"
	"net/url"

	"github.com/assurrussa/goinertia/core"
)

var (
	marshal                   = core.Marshal
	raw                       = core.Raw
	asset                     = core.Asset
	sleeper                   = core.SleepContext
	appendUnique              = core.AppendUnique
	normalizeValidationErrors = core.NormalizeValidationErrors
	flattenValidationErrors   = core.FlattenValidationErrors
)

func parseInertiaBaseURL(v string) *url.URL                        { return core.ParseBaseURL(v) }
func (i *Inertia) createRootTemplate() (*template.Template, error) { return i.RootTemplate() }

func (i *Inertia) createRootErrorTemplate() (*template.Template, error) {
	return i.ErrorTemplate()
}
func (i *Inertia) hotServerURL() string { return i.HotServerURL() }

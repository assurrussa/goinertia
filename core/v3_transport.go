package core

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strings"
)

// HeaderRedirect requests an Inertia client visit while retaining URL fragments.
const HeaderRedirect = "X-Inertia-Redirect"

// WithPreserveFragment carries the incoming visit's fragment onto the response
// URL. It is request-local and only applies to the opt-in v3 protocol.
func (i *Inertia) WithPreserveFragment(state *State, preserve bool) {
	if state == nil || !i.IsProtocolV3() {
		return
	}
	if state.pageMeta == nil {
		state.pageMeta = &pageMeta{}
	}
	state.pageMeta.preserveFragment = &preserve
}

// IsPrefetch identifies the standard browser and Inertia prefetch request hints.
// Adapters pass Purpose, Sec-Purpose and X-Moz without extending RequestMeta.
func IsPrefetch(headers ...string) bool {
	for _, header := range headers {
		for _, token := range strings.FieldsFunc(header, func(r rune) bool { return r == ',' || r == ';' || r == ' ' }) {
			if strings.EqualFold(token, "prefetch") {
				return true
			}
		}
	}
	return false
}

// IsFragmentRedirect determines whether an ordinary redirect must be delivered
// as a v3 client-side redirect. Prefetches retain normal HTTP redirect semantics.
func (i *Inertia) IsFragmentRedirect(inertia string, prefetch bool, status int, location string) bool {
	if !i.IsProtocolV3() || inertia == "" || prefetch || !strings.Contains(location, "#") {
		return false
	}
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// Bootstrap renders the initial page for the app mount point. JSON is encoded
// with HTML escaping, including '<', '>', '&', U+2028 and U+2029, so user props
// cannot terminate the script or introduce an HTML comment. V3 also escapes every
// slash, matching the official bootstrap contract. SSR renderers supply
// their own complete body and should not be wrapped in another mount point.
func Bootstrap(page any, version ProtocolVersion) (template.HTML, error) {
	payload, err := json.Marshal(page)
	if err != nil {
		return "", fmt.Errorf("error marshaling bootstrap page: %w", err)
	}
	var markup string
	if version == ProtocolV3 {
		markup = `<script data-page="app" type="application/json">` +
			strings.ReplaceAll(string(payload), "/", `\/`) + `</script><div id="app"></div>`
	} else {
		markup = `<div id="app" data-page="` + html.EscapeString(string(payload)) + `"></div>`
	}
	// #nosec G203 -- Markup is library-owned; JSON and the v2 attribute are escaped above.
	return template.HTML(markup), nil
}

func (i *Inertia) bootstrapViewData(viewData map[string]any, ssr *SsrDTO) error {
	viewData["protocolV3"] = i.IsProtocolV3()
	viewData["inertiaHead"] = template.HTML("")
	if ssr != nil {
		// #nosec G203 -- SSR head/body are trusted output from the host-configured renderer.
		viewData["inertiaHead"] = template.HTML(strings.Join(ssr.Head, "\n"))
		// #nosec G203 -- Official SSR output already contains the bootstrap and mount point.
		viewData["inertiaBody"] = template.HTML(ssr.Body)
		return nil
	}
	body, err := Bootstrap(viewData["pageJSON"], i.ProtocolVersion())
	if err != nil {
		return err
	}
	viewData["inertiaBody"] = body
	return nil
}

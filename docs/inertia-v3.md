# Inertia v3 protocol

## Explicit opt-in and compatibility

The v3 contract targets the published **@inertiajs/core, vue3 and react 3.8.0**
packages (2026-10-01). V2 remains the default. Enable v3 when the browser bundle
has also migrated:

```go
inertia := nethttp.New("https://app.example",
    nethttp.WithCoreOptions(core.WithProtocolVersion(core.ProtocolV3)),
)
```

For native Fiber and the root compatibility import, use
`goinertia.WithProtocolVersion(goinertia.ProtocolV3)` directly. Configure the
engine before serving; never change the protocol of an engine concurrently.
The same application can construct separate v2 and v3 engines for separately
routed bundles. The request's asset-version string is not a protocol selector.

Existing exported `PageDTO`, `SSRConfig`, `DeferredProp`, `MergeProp`, `OnceProp`
and `ScrollPropConfig` layouts remain unchanged. The new response metadata is
request-local. Direct core consumers must use `MarshalPageWithState(page,state)`
and `RenderHTML(state,page)`, rather than `json.Marshal(page)`, to include it.

## Enable v3 in an application

Start with **goinertia v0.11.0** (`go get github.com/assurrussa/goinertia@v0.11.0`).
The [runnable v3 example](../examples/v3-app/README.md) contains both adapters,
both tested frontend frameworks, the root template and an optional Node SSR
renderer. It uses the same public API available to another Go module.

For the root Fiber API, import `github.com/assurrussa/goinertia` and add
`goinertia.WithProtocolVersion(goinertia.ProtocolV3)` to `NewWithValidation`.
For native Fiber, import `github.com/assurrussa/goinertia/adapters/fiber` and
use `fiberadapter.WithProtocolVersion(core.ProtocolV3)`. For native net/http:

```go
import (
    nethttp "github.com/assurrussa/goinertia/adapters/nethttp"
    "github.com/assurrussa/goinertia/core"
    "github.com/assurrussa/goinertia/views"
)

// Keep the existing session, CSRF, template and asset options as well.
manager, err := nethttp.NewWithValidation("http://localhost:3000",
    nethttp.WithCoreOptions(
        core.WithProtocolVersion(core.ProtocolV3),
        core.WithFS(views.Templates), // Or your application's template FS.
    ),
)
if err != nil {
    panic(err)
}
```

Migrate each application as one matched server-and-frontend change:

1. Update its Go module to v0.11.0 and its Vue or React Inertia package to
   **3.8.0**, including any direct `@inertiajs/core` dependency. Preserve the
   package-manager lockfile. React requires React/ReactDOM 19+; Vue applications
   do not need React. Inertia v3 packages require an ESM-capable build.
2. Enable `ProtocolV3` on the engine serving that frontend. `WithAssetVersion`
   identifies the built assets; setting it to `"v3"` does not select the protocol.
3. Replace a custom root template's old `data-page` mount element with
   `{{ .inertiaBody }}` and put `{{ .inertiaHead }}` in its `<head>`. Keep the
   application's stylesheet and module-script tags. Do not hand-serialize page
   JSON, wrap the helper in another mount element, or retain a duplicate mount.
   The library's default template already supports both profiles.
4. Update client event handlers from `invalid` to `httpException` and from
   `exception` to `networkError`. Replace calls to the `router.cancel()` method
   with `router.cancelAll()` and choose whether asynchronous and prefetch
   requests should also be canceled. Check the
   [official upgrade guide](https://inertiajs.com/docs/v3/getting-started/upgrade-guide)
   for client API changes used by your application. Custom HTTP exception
   handlers must handle the v3 response shape instead of assuming Axios fields
   such as `response.config.url`. If SSR is enabled, rebuild
   and restart its renderer with the same client version and hydration setup.
5. Migrate any Axios-specific Inertia request configuration. V3 uses a built-in
   XHR client; `axios.defaults` no longer configures it. For a custom CSRF cookie
   and header, pass `http: { xsrfCookieName: 'csrf_token', xsrfHeaderName:
   'X-CSRF-Token' }` to `createInertiaApp`, using the names your server issues
   and checks. Keep Axios configuration for separate Axios requests if needed.
6. Rebuild and publish matching assets with a new asset fingerprint. Verify
   initial page load, navigation, login/session/CSRF, forms and validation,
   redirects, partial/deferred/Once props, flash, and back/forward history.
   Check real hydration and failure fallback when SSR is enabled.

If a library supplies an embedded admin frontend, migrate that library's Go
engine, template and browser bundle first, then update each consuming app and
rebuild any overridden assets or templates. Unrelated backend libraries need
no protocol switch. A Go dependency update alone cannot migrate an embedded
v2 frontend.

## Server protocol acceptance matrix

“Covered” below identifies executable tests, not a substitute for recording a
completed run. A release or merge claiming v3 support requires all acceptance
gates at the end of this document against its exact candidate SHA.

| Protocol feature | Implementation / public API | Acceptance evidence |
|---|---|---|
| HTML bootstrap | Script `data-page="app" type="application/json"`, then mount div; JSON escapes slash, `<`, `>`, `&`, U+2028/2029, without HTML entities | Core hostile-prop/bootstrap tests; Vue and React CSR browser startup |
| JSON, Vary, cache headers | Existing native lifecycle; JSON `X-Inertia:true`; distinct HTML/control responses | Both adapter fixtures and browser navigation |
| Asset mismatch | GET-only 409, `X-Inertia-Location` and current `X-Inertia-Version`; Precognition exempt | Adapter tests; foreground reload and background no-reload browser cases |
| Ordinary/external/fragment redirects | Write 303 policy retained; external 409 Location; non-prefetch fragment 409 `X-Inertia-Redirect` | Raw and helper redirects, prefetch, method, fragment browser cases |
| Incoming fragment preservation | `WithPreserveFragment(state,true)` / Fiber context equivalent emits `preserveFragment` | Wire and browser URL assertions |
| Partial reloads | Matching component; dotted only then except; always/errors immune | Core selection tests, both adapters, real client dotted deep merge |
| Recursive values | Wrappers/providers in JSON maps, typed string-key maps and slices; injective lazy cache keys; copy ownership | Core nested, typed, callback, excluded-evaluation, cycle-depth and race tests |
| Deferred and optional | Groups; full-load omission; selected partial resolution; no reannouncement on partial | Nested/wrapper composition and real deferred requests |
| Deferred rescue | `Rescue(Defer(callback))` or `Defer(Rescue(callback))`; report error, omit value, `rescuedProps` | Error/cancellation unit tests and real client rescue/retry |
| Nonrescued provider errors | Returned to native error handling, not silently omitted | Core propagation and safe adapter error response tests |
| Once | Custom key/expiry, initial cached omission, fresh override; any selected partial refreshes | Unit and browser cache/reload/expiry/cross-component tests |
| Append/prepend/deep merge | Existing constructors work recursively; `MergeAt` supports mixed nested targets and item identity | Nested labels, client merge and reset tests |
| Reset | Root/path reset suppresses merge/match labels; scroll reset boolean | Core reset and browser replacement |
| Infinite scroll | `Scroll` defaults to `.data`; `ScrollAt(value,cfg,path)` supports custom or empty root path; cursor metadata and intent | Custom path, prepend, reset, deferred composition tests |
| Shared props | Top-level registered shared keys emitted as `sharedProps` | Wire metadata and real instant-visit sharing |
| Validation | Always errors object, first-message helper; `WithAllValidationErrors` preserves arrays; named bags and session redirects | Core shape/precedence, native session/Precognition, browser forms |
| Native flash | `WithNativeFlash`; top-level `flash`, events and history stripping | Existing session tests; real client event/history checks |
| History | `WithEncryptHistory`, `WithClearHistory` | Wire tests and browser encrypted state/back-forward |
| Big integers (optional) | `WithPreserveBigIntegers(true)` emits `$bigint` markers for unsafe integral JSON tokens plus `preserveBigIntegers` | Signed/unsigned bounds, custom JSON shape, flash and real client revival |
| Correct error status | `RenderWithStatus` renders host error component with shared props; unhandled v3 errors are safe HTTP exceptions | Status/shared-prop adapter tests and client exception event |
| Production SSR | Official ESM server POST `/render`, `{head,body}`; complete v3 SSR body with hydration marker and `data-inertia` head | Real Vue/React render followed by browser hydration |
| SSR errors | Report structured render/connection diagnostics; v3 CSR fallback; explicit override available | Retry/invalid/error/cancel/fallback unit tests and browser fallback |
| Per-request SSR control | `WithSSRDisabled`; does not mutate shared engine | Independent/concurrent request tests |
| Development SSR | Vite hot endpoint `/__inertia_ssr`, explicit `WithViteSSR` override; production endpoint otherwise | Transport endpoint/cache tests; optional host Vite setup |
| CSRF/Precognition | Host-supplied token/check callbacks, 204/422, field selection and Vary | Existing adapter tests retained plus form browser fixture |
| Cancellation and request isolation | Request lifecycle reaches lazy/SSR; no shared request state | Race suite, canceled provider/SSR, interrupted browser visits |

## Versioned differences

- V2 keeps its `data-page` attribute bootstrap and historical except precedence.
  V3 uses the script bootstrap and intersects only/except.
- V2 keeps historical once omission unless explicitly selected or opted into
  refresh. V3 ignores remembered-once exclusions on matching partial requests. Once
  metadata accompanies every successfully included value, including a selected
  descendant of a wrapped container or an Always/provider bypass. This preserves
  refreshed expiry and custom-key ownership when a prop is renamed. The pinned
  reference adapter filters this metadata more narrowly; this edge case is
  covered by dedicated Go and real-client regressions.
- V2 keeps log-and-omit callback errors. V3 propagates errors unless rescued.
  A canceled request is never rescued into a success response.
- V3 resolves nested wrappers in maps and slices, including wrappers returned
  by providers. A provider or protocol wrapper containing a container bypasses child partial
  filtering, matching the pinned reference adapter. Select the parent path
  when its entire returned value is desired. Plain map children are filtered
  before providers run.
- V3 `Scroll` uses the conventional `data` array path. Use `ScrollAt(..., "")`
  for an array directly at the prop root. Explicit `Defer(Scroll(...))` defers
  loading; neither official v3 Scroll nor this adapter defers by default.
- V2 SSR errors continue propagating. V3 reports and falls back to CSR by
  default. Both can use the additive per-request and explicit error-policy APIs.
- V3 validation bags preserve either string or array-valued errors. The old
  first-message convenience helper and its v2 behavior remain unchanged.

Recursive traversal deliberately does not reinterpret arbitrary Go structs or
custom JSON marshalers as framework property providers. They retain their normal
JSON serialization. Use JSON-shaped maps/slices to embed protocol wrappers.
Arrays with omitted elements become sparse numeric-key JSON objects; fully
included arrays remain arrays. This preserves indexes and absent-path semantics
for Deferred/Once/rescue, rather than incorrectly marking an omitted value as
loaded by sending null. For a previously complete array, a dotted-only reload
may replace its root: the pinned client deep-merges objects, not arrays. Reload
the entire array or use explicit merge metadata when retaining siblings matters.
Go ORM paginator normalization, dotted-key declaration convenience, dependency
injection and Laravel property-provider interfaces are host/framework concerns;
they are not extra fields or required operations in the wire protocol.

## Frontend and host-owned features

React v3 requires React 19+. Inertia v3 packages are ESM-only. The versioned
browser fixture isolates React 19 from the retained v2 React 18 fixtures.
Client event renames (`httpException`, `networkError`), `router.cancelAll()`, XHR transport,
`useHttp`, optimistic updates, layouts, `<Deferred>`, `<InfiniteScroll>`, instant
visits, polling and prefetch caching belong to the client. The adapter supplies
their documented HTTP and page metadata; it does not reimplement client logic.

The host owns routing, page resolution, form validation, session storage,
CSRF-token generation/cookie attributes, authorization, safe redirect targets,
asset version generation, Vite/SSR process management and registered error-page
components. Inertia is not an authorization boundary: never return a secret
because a partial or deferred request happened to select its name.

## Reproducible acceptance

1. Full Go test/vet/build/lint, race x5, generated code/format and tidy checks.
2. Pinned v2 protocol replay and independent net/http consumer, including its
   dependency prohibition on Fiber/fasthttp.
3. Retained **240** browser cases at v2.3.18/2.3.28 × Vue/React × native
   Fiber/nethttp × CSR/real SSR.
4. V3.8.0 Vue/React matrix across those same adapters/rendering modes, including
   failure, cancellation, history, redirects, metadata and security cases.
5. Example client and production SSR builds with exact dependency lockfiles.
6. Codex review resolved with no blocking findings and required repository
   checks successful for the candidate SHA. Publication is not a release.

The browser runner is described in
[integration/browser/README.md](../integration/browser/README.md). Unit tests,
SSR-only renders and bundle builds must never be recorded as browser hydration
passes. If the environment cannot launch Chromium, record that exact blocker
and run the frozen candidate in an authorized browser-capable executor before
the support claim or merge.

## Sources checked 2026-10-03

- [Published v3.8.0 release](https://github.com/inertiajs/inertia/releases/tag/v3.8.0)
- [Pinned client page contract](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/types.ts)
- [Pinned bootstrap](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/domUtils.ts),
  [SSR body](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/ssrUtils.ts),
  [response handling](https://github.com/inertiajs/inertia/blob/v3.8.0/packages/core/src/response.ts)
- [Protocol specification](https://github.com/inertiajs/docs/blob/main/v3/core-concepts/the-protocol.mdx),
  audited blob `f6b2ba25ea58417a713f370c809725489cd01518`
- [Reference props resolver](https://github.com/inertiajs/inertia-laravel/blob/4da52b72da39396cad1d3ec8523b6c649009456e/src/PropsResolver.php)
- [Reference BigInt encoding](https://github.com/inertiajs/inertia-laravel/blob/4da52b72da39396cad1d3ec8523b6c649009456e/src/PreservesBigIntegers.php)
- [Upgrade guide](https://github.com/inertiajs/docs/blob/main/v3/getting-started/upgrade-guide.mdx),
  audited blob `1cdafd44afc20f3597ff852012bf0a080d330d72`

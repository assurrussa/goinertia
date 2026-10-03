# Protocol coverage and implementation roadmap

The architecture decision retains a neutral core and both native adapters.
This matrix audits server behavior; it does not certify an Inertia client or
browser integration from successful JSON fixtures alone.

## Version contract

The default contract preserves the existing **Inertia v2.x subset**, including
its historical root/Fiber behavior. Published client versions **2.3.18** and
**2.3.28** remain pinned in the regression suite. The tables below describe that
v2 baseline, rather than limitations of the new v3 profile.

Explicit `WithProtocolVersion(ProtocolV3)` enables the **Inertia 3.8.0 server
protocol**. Its source-backed [feature and acceptance matrix](inertia-v3.md)
records every v3 field, versioned difference, host responsibility and required
browser gate. A passing JSON or SSR-only fixture is never evidence of hydration.

Sources were checked on 2026-10-03. No new public Go release/tag is asserted.

Implemented means the server behavior exists and is covered by local tests.
Partial means a documented subset or compatibility difference remains.
Missing means no library implementation. Host means the surrounding application
owns that policy. A core cell marked “policy” requires adapter lifecycle wiring.

## Shared protocol and rendering

| Behavior | Core | Root / legacy Fiber | Native Fiber | Native HTTP | Evidence or limit |
|---|---|---|---|---|---|
| v2 initial HTML and JSON page responses | Implemented | Implemented | Implemented | Implemented | Shared protocol fixtures; templates use the v2 data-page attribute |
| HTML escaping, templates, assets/hot file | Implemented | Implemented | Implemented | Implemented | Existing rendering/template suite; dev configuration is set before serving |
| X-Inertia response and Vary/cache policy | Policy | Implemented | Implemented | Implemented | JSON, HTML, conflict, direct Fiber render; HTTP preserves multiple Vary lines and wildcard |
| GET asset version conflict, location | Policy | Implemented | Implemented | Implemented | 409 + X-Inertia-Location; non-GET and Precognition skip the check |
| Internal/external/scheme-relative redirects | Policy | Implemented | Implemented | Implemented | Shared fixtures; hosts validate destinations |
| Write redirect normalization | Policy | Implemented | Implemented | Implemented | Historical 301/302 to 303 for POST/PUT/PATCH/DELETE; 307/308 stay explicit |
| Empty successful handler response | Host | Partial | Partial | Partial | No automatic redirect-back for an empty 200; HTTP commits a normal empty response |
| Top-level partial only/except and component match | Implemented | Implemented | Implemented | Implemented | Mismatched component returns a full page; reserved errors/flash/old/CSRF remain included |
| Both only and except supplied | Partial | Partial | Partial | Partial | Historical except precedence is retained; the official adapter applies only, then excludes |
| Dotted/nested partial selection | Implemented | Implemented | Implemented | Implemented | JSON-map paths filter before callbacks; literal dotted keys also match; typed containers and array indexes stay opaque |
| Lazy callbacks and per-request result cache | Implemented | Implemented | Implemented | Implemented | Root callback receives actual fiber.Ctx; native callbacks receive lifecycle context |
| Optional/deferred full visit and explicit only | Implemented | Implemented | Implemented | Implemented | Omit on full visit, announce deferred groups, evaluate explicit matching only |
| Optional/deferred except-only selection | Implemented | Implemented | Implemented | Implemented | Matching except reload evaluates eligible wrappers and skips excluded ones; corrected v2 behavior |
| Deferred announcements on partial responses | Implemented | Implemented | Implemented | Implemented | Partial responses no longer schedule another deferred group |
| Wrapper composition and nesting | Partial | Partial | Partial | Partial | Lazy callbacks in map[string]any and []any containers resolve with copy-on-change ownership; nested prop wrappers/typed Go containers are not recursively interpreted |
| Root append/prepend/deep merge labels | Implemented | Implemented | Implemented | Implemented | Selected/resolved props only; missing or failed values no longer leave merge/scroll labels |
| Nested merge targets and mixed strategies | Implemented | Implemented | Implemented | Implemented | Additive MergeAt with AppendAt/PrependAt targets; selected/resolved paths only; root reset suppresses wrapper labels |
| Matching items metadata | Partial | Partial | Partial | Partial | MergeAt target matching is selected/resolved/reset-aware; legacy WithMatchPropsOn remains host-supplied and unvalidated |
| Reset root merge labels | Implemented | Implemented | Implemented | Implemented | Selected value is re-resolved and append/prepend/deep label omitted |
| Infinite scroll | Partial | Partial | Partial | Partial | Manual metadata, map paginator data path, append/prepend intent and reset flag work; no typed paginator normalization, custom data path or deferred-by-default scroll |
| Once keys, expiry metadata and explicit reload | Partial | Partial | Partial | Partial | Key/expiry, explicit only, WithOnceFresh and supported root compositions work; WithOnceRefreshOnPartial opts into reference except-only refresh; legacy default preserved |
| Errors object, first error, named bag | Implemented | Implemented | Implemented | Implemented | Empty object, flattening and named bag fixtures; hosts provide validation/session source |
| History clear/encrypt flags | Implemented | Implemented | Implemented | Implemented | Server emits flags; encryption and browser storage are client responsibilities |
| Shared/context/request prop precedence | Implemented | Implemented | Implemented | Implemented | Existing override/lazy tests; startup configuration is immutable during serving |
| Legacy flash/old/errors through session | Policy | Implemented | Implemented | Implemented | Redirect persistence and consume-once tests; these remain ordinary page props |
| Native flash outside browser history | Policy | Implemented | Implemented | Implemented | Opt-in WithNativeFlash emits top-level page.flash; legacy props.flash remains unchanged; history/events belong to the pinned v2 client |
| Prefetch/poll/load-visible request transport | Partial | Partial | Partial | Partial | Ordinary Inertia requests work; Purpose-specific behavior, history/client caching and scheduling belong to the host/client |
| Precognition server responses | Policy | Implemented | Implemented | Implemented | 204/422, validate-only, Vary and checker-only tests; no Laravel validation integration |

Advanced prop comparisons use the official
[partial reloads](https://inertiajs.com/docs/v2/data-props/partial-reloads),
[merge](https://inertiajs.com/docs/v2/data-props/merging-props),
[scroll](https://inertiajs.com/docs/v2/data-props/infinite-scroll), and
[once](https://inertiajs.com/docs/v2/data-props/once-props) descriptions.
The [official 2.x Response implementation](https://github.com/inertiajs/inertia-laravel/blob/2.x/src/Response.php)
clarifies partial filtering, metadata selection and scroll reset. This is a
reference implementation, not a dependency. The audit also checked pinned
commit `67666f22924515f44173dfa8bd4a608580519999`. The linked branch can change.
The optional-only wording in the topic docs is narrower than actual reference
adapter filtering. Except-only reloads now resolve eligible wrappers as a v2
correctness fix; full visits still omit them. Existing root callbacks, shared
data ownership and only/except precedence remain unchanged.

## Lifecycle, host hooks and SSR

| Behavior | Core | Root / legacy Fiber | Native Fiber | Native HTTP | Evidence or limit |
|---|---|---|---|---|---|
| Request state isolation | Implemented | Implemented | Implemented | Implemented | Parallel protocol fixtures and warm-page tests; external HTTP consumer runs 32 simultaneous real requests |
| Lifecycle cancellation | Implemented | Host context | Host context | Implemented | Fiber requires c.SetContext; native client cancellation and downstream WithContext values/cancellation reach lazy/SSR/Request helpers |
| Streaming and response commitment | Adapter | Fiber lifecycle | Fiber lifecycle | Implemented | HTTP writes directly; headers/cookies/status fixed before commitment; library page serialization precedes write |
| Optional writer capabilities | Adapter | Fiber lifecycle | Fiber lifecycle | Implemented | Flusher/Hijacker/Pusher/CloseNotifier, ReaderFrom/StringWriter, Unwrap/ResponseController; informational responses and commitment through transparent Unwrap wrappers tested |
| Flash session ownership and rotation | Adapter | Implemented + host | Implemented + host | Host hooks | Raw Fiber acquisition released; middleware ownership borrowed; GoAdmin bridge owns Release; HTTP host owns cookies/storage/rotation |
| CSRF token and checker | Adapter | Host hooks | Host hooks | Host hooks | Independent token injection/write checking; host owns token generation and cookie policy |
| Error mapping and presentation | Host defaults | Configurable | Configurable | Partial | Native HTTP maps core Error/ValidationError and uses safe details; no matching custom error/exposure callback options |
| SSR POST and owned response body | Implemented | Implemented | Implemented | Implemented | Native HTTP default transport; custom injection; body read before Close, ownership regression and actual loopback tests |
| SSR retry, timeout, cancellation, cache | Implemented | Implemented | Implemented | Implemented | Parallel retries do not Reset shared client; lifecycle context, cache copies and race tests |
| SSR failure policy | Implemented | Implemented | Implemented | Implemented | Existing errors propagate; no silent CSR fallback |
| Per-request SSR disable / Vite v3 development integration | Missing | Missing | Missing | Missing | Enable/DisableSSR are startup configuration, not concurrent request controls |
| Cold template publication | Implemented | Implemented | Implemented | Implemented | sync.Once publication; concurrent cold root/error template tests |
| Configuration changes during serving | Unsupported | Unsupported | Unsupported | Unsupported | Configure options and shared values before serving; State is not concurrently mutable |

The library supplies server hooks for
[CSRF](https://inertiajs.com/docs/v2/security/csrf-protection),
[history](https://inertiajs.com/docs/v2/security/history-encryption), and
[SSR](https://inertiajs.com/docs/v2/advanced/server-side-rendering).
It does not implement host authentication, credentials or client encryption.
Session and CSRF fixtures use local data only.

## Explicit v3 profile and acceptance

The prior v3-specific bootstrap, redirect, recursive-wrapper, rescue and SSR gaps
are implemented behind the explicit profile. See [Inertia v3](inertia-v3.md) for
the exact source pins, APIs and complete candidate acceptance matrix.

V2 continues preserving its historical only/except and Once policies, public
struct layouts, callback lifecycle and SSR error behavior. New metadata is kept
in request-local state rather than added to positional public structs. Native
adapters serialize `MarshalPageWithState`; direct core callers must do the same.

Required checks include full fmt/vet/lint/build/tests, race x5, pinned v2 replay,
independent HTTP consumer and its no-Fiber dependency graph, the unchanged 240
v2 browser regressions, and the v3 Vue/React × Fiber/nethttp × CSR/real SSR matrix.
Browser startup, hydration, exception, cancellation, fragment/version redirects,
flash/history and deferred/Once/merge/rescue flows must pass on the frozen
candidate before a full support claim. See the browser README for execution.
No generic framework interface, whole-response buffer or adapter bridge is added.

## Independent HTTP consumer

`integration/nethttp-consumer` is a separate module requiring only goinertia directly.
Its production and test imports use core, adapters/nethttp, views and the standard
library; neither imports the root facade, mocks or Fiber. From that directory:

```sh
go mod tidy
go vet ./...
go build ./...
go test -race -count=5 ./...
go list -deps -test ./... > dependencies.txt
```

The compiled/test graph must contain no `github.com/gofiber/` or
`github.com/valyala/fasthttp` package. The parent module still declares Fiber for
the compatibility API; that module requirement is distinct from native HTTP's
compiled/runtime imports. The nested module is excluded from the parent
`go test ./...`; `make check` and the existing required Go workflow run its
vet/build/race/dependency gates explicitly. No frontend/browser test results
are asserted by this consumer.

A focused pinned v2.3.18 client replay is required by `make check` and the Go
workflow through `make protocol-replay`. It can also be run separately with
`GOINERTIA_CLIENT_REPLAY=1 go test -run '^TestPinnedClientProtocolReplay$' ./`.
See [replay provenance and limits](../integration/protocol-replay/README.md).
It covers grouped deferred scheduling, preservation of an excluded deep-merge
prop, paginator append/prepend and scroll reset. Browser/bootstrap/history and hydration are covered separately by the real
browser matrix; this focused replay makes no claims about them.

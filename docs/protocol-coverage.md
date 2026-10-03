# Protocol coverage and implementation roadmap

The architecture decision retains a neutral core and both native adapters.
This matrix audits server behavior; it does not certify an Inertia client or
browser integration from successful JSON fixtures alone.

## Version contract

The compatibility target of this change is the existing **Inertia v2.x wire
contract subset**, including the historical root/Fiber behavior described below.
It is not a claim of complete v2 support or compatibility with every v2 minor.
The pinned client regression target is **Inertia v2.3.18**, commit
`ed9b159a5857663211580e081d6bd90520f17bad`. Its actual merge, deferred scheduler
and scroll reset methods are replayed against responses from all three public
adapter entrypoints. This does not certify the entire v2.3.18 client. The forward
implementation target is the **Inertia v3 server protocol** documented on the
audit date.
The official documentation defaults to v3 and explicitly marks v2 as legacy.
See the [documentation index](https://inertiajs.com/docs/llms.txt),
[v2 protocol](https://inertiajs.com/docs/v2/core-concepts/the-protocol), and
[v3 upgrade guide](https://inertiajs.com/docs/v3/getting-started/upgrade-guide).
Sources were checked on 2026-10-03; no latest patch release or public Go tag is
asserted. The examples now pin 2.3.28. A separate [real browser matrix](../integration/browser/README.md)
exercises published Vue/React clients at exact 2.3.18 and 2.3.28, each against
native Fiber/net/http with CSR and actual SSR hydration. Its CI jobs are a merge
gate, independent of the historical focused method replay. This matrix does not
certify every client feature or an unrestricted v2/v3 compatibility claim.

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
| Dotted/nested partial selection | Missing | Missing | Missing | Missing | Maps are filtered by their literal top-level keys; no recursive path resolver |
| Lazy callbacks and per-request result cache | Implemented | Implemented | Implemented | Implemented | Root callback receives actual fiber.Ctx; native callbacks receive lifecycle context |
| Optional/deferred full visit and explicit only | Implemented | Implemented | Implemented | Implemented | Omit on full visit, announce deferred groups, evaluate explicit matching only |
| Optional/deferred except-only selection | Implemented | Implemented | Implemented | Implemented | Matching except reload evaluates eligible wrappers and skips excluded ones; corrected v2 behavior |
| Deferred announcements on partial responses | Implemented | Implemented | Implemented | Implemented | Partial responses no longer schedule another deferred group |
| Wrapper composition and nesting | Partial | Partial | Partial | Partial | Lazy callbacks in map[string]any and []any containers resolve with copy-on-change ownership; nested prop wrappers/typed Go containers are not recursively interpreted |
| Root append/prepend/deep merge labels | Implemented | Implemented | Implemented | Implemented | Selected/resolved props only; missing or failed values no longer leave merge/scroll labels |
| Nested merge targets and mixed strategies | Missing | Missing | Missing | Missing | Merge/Prepend/DeepMerge operate on the root prop; no append-at-path API |
| Matching items metadata | Partial | Partial | Partial | Partial | WithMatchPropsOn emits host-supplied paths; no wrapper-level match API or reset/partial path validation |
| Reset root merge labels | Implemented | Implemented | Implemented | Implemented | Selected value is re-resolved and append/prepend/deep label omitted |
| Infinite scroll | Partial | Partial | Partial | Partial | Manual metadata, map paginator data path, append/prepend intent and reset flag work; no typed paginator normalization, custom data path or deferred-by-default scroll |
| Once keys, expiry metadata and explicit reload | Partial | Partial | Partial | Partial | Key/expiry and explicit only work; metadata follows partial selection; no force-refresh option and except-only skip differs from reference |
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

## v3-specific gaps

The [v3 protocol](https://inertiajs.com/docs/v3/core-concepts/the-protocol)
and [upgrade guide](https://inertiajs.com/docs/v3/getting-started/upgrade-guide)
introduce requirements beyond the compatible template/DTO contract:

| Feature | Core | Legacy Fiber | Native Fiber | Native HTTP |
|---|---|---|---|---|
| Script-element initial page and safe JSON/script termination | Missing default | Missing default | Missing default | Missing default |
| X-Inertia-Redirect fresh Inertia GET response | Missing | Missing | Missing | Missing |
| Explicit deferred rescue and rescuedProps | Missing | Missing | Missing | Missing |
| Recursive nested wrappers and dotted selection/metadata | Missing | Missing | Missing | Missing |
| Native flash / onFlash | v2 subset | v2 subset | v2 subset | v2 subset |
| Exception page with correct error status/shared props | Partial | Partial | Partial | Partial |

Lazy callback errors currently log and omit the value, preserving old behavior.
This is not v3 deferred rescue: there is no explicit rescue choice or rescuedProps
signal. The [deferred error policy](https://inertiajs.com/docs/v3/data-props/deferred-props)
needs a versioned design before changing that default. The
[native flash API](https://inertiajs.com/docs/v2/data-props/flash-data) also exists
in later v2 clients; `WithNativeFlash` opts into its top-level wire contract.
Normal props.flash remains in history and must not be advertised as its
equivalent. See [flash behavior and session lifecycle](flash.md).
v3 error pages require dedicated tests against
[exception handling](https://inertiajs.com/docs/v3/advanced/error-handling).

## Acceptance gates and phased work

The compatible adapter PR is blocked by regressions in its supported subset,
request ownership/cancellation, session/CSRF, response commitment, or the existing
Fiber/GoAdmin API. The audit fixes partial deferred re-announcement, metadata for
excluded props and unresolved merge/scroll values, eligible except wrappers,
nested lazy containers, paginator data paths/reset, and multiple HTTP Vary lines.
HTTP fixes also bind downstream contexts and follow Unwrap when checking
commitment, preventing an error template from being appended to a sent body.
These have targeted shared/core/HTTP regressions. The old Fiber test requiring
once metadata for an unselected prop now asserts its absence, matching the
reference adapter's metadata filter. Existing helper signatures and facade
aliases remain. ScrollPropConfig retains its original four-field layout, including
positional literals. The request-only reset flag is encoded by an internal wire
view, rather than added to the public DTO/config. Raw JSON, default HTML and SSR
fixtures cover it. For direct core consumers, BuildPage returns the compatible
DTO; `core.MarshalPageWithState(page, state)` creates the request wire response
including opt-in native flash and scroll reset. The older
`core.MarshalPage(page, state.Meta.Reset)` remains available for scroll reset
only. Plain json.Marshal(PageDTO) lacks request-only metadata. The adapters perform
this step, and replay fixtures keep raw response JSON. Nested lazy cache keys use
source layers and length-prefixed
map/array paths; literal dots and context/request callbacks cannot alias.
Full fmt/vet/lint/build, existing Fiber tests, shared fixtures, race x5, local SSR and session checks, external
consumers, GoAdmin 0.7 source integration and performance comparisons remain
required after code changes.

The outstanding rows above block a **complete protocol support claim**, even
when the compatible PR gates pass. They are scoped follow-up work, not hidden
HTTP/Fiber parity failures introduced by extraction. Work in this order:

1. Add explicit protocol/version policy with v2 defaults preserved. Extend the pinned
   v2 replay with full v2 and v3 client fixtures; exercise browser bootstrap, redirects, partial
   requests, history, native flash and SSR hydration on both server adapters.
2. Complete common prop semantics: dotted selection, nested merge targets,
   custom scroll paths/defer metadata, once refresh, and versioned only/except/error
   behavior. Use one core implementation and the same adapter fixtures.
3. Add v3 bootstrap, redirect and deferred rescue/error contracts beyond the
   opt-in native flash now available. Preserve legacy flash and callback compatibility. Give HTTP native
   error policy hooks where host customization is needed.
4. Validate v3 SSR development/per-request control and the broader client matrix.
   Re-run lifecycle/race, external consumers and performance budgets for every
   material change. Publication/release acceptance is separate from these tests.

No new generic framework interface, whole-response buffer or adapter-to-adapter
bridge is needed to complete these features.

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
prop, paginator append/prepend and scroll reset. Full browser/bootstrap/history
and hydration remain roadmap acceptance gates, not results claimed here.

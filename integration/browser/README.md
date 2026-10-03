# Real browser contract matrix

This fixture runs published Inertia clients, real Vue/React rendering and real
Go servers. It is separate from the focused method replay in `../protocol-replay`.
A selected version chooses an explicit server protocol as well as the client.

## Exact profiles

| Client | Framework runtime | Native adapters | Bootstrap |
|---|---|---|---|
| 2.3.18 | Vue 3.5.22 / React 18.3.1 | Fiber + net/http | CSR + real SSR hydration |
| 2.3.28 | Vue 3.5.22 / React 18.3.1 | Fiber + net/http | CSR + real SSR hydration |
| 3.8.0 | Vue 3.5.22 / React 19.3.0 | Fiber + net/http | CSR + real SSR hydration |

The existing `package.json`/lock retain both v2 adapters and React 18. The
`v3/package.json`/lock install a separate React 19 dependency tree. All adapters
require the same exact core version as themselves. Build-time ESM resolution
selects that profile for browser **and** SSR bundles. There are no copied client
methods, runtime version substitutions, peer-dependency overrides or checksum
bypasses. Existing v2 public examples remain pinned to 2.3.28;
[`examples/v3-app`](../../examples/v3-app/README.md) provides separate v3 profiles.

Official sources verified on 2026-10-03:

- [v3.8.0 release, 2026-10-01](https://github.com/inertiajs/inertia/releases/tag/v3.8.0)
- [v3 upgrade guide, including React 19 and script bootstrap](https://inertiajs.com/docs/v3/getting-started/upgrade-guide)
- [v3 wire protocol](https://inertiajs.com/docs/v3/core-concepts/the-protocol)
- [v2.3.18 release](https://github.com/inertiajs/inertia/releases/tag/v2.3.18)
- [v2.3.28 release](https://github.com/inertiajs/inertia/releases/tag/v2.3.28)
- Published npm manifests and integrity hashes in each committed lockfile.

These framework runtime versions are fixture pins, not claims about minimum
supported versions. Svelte is not tested here. The root Fiber compatibility
facade retains its Go/replay regression gates.

## One bounded full run

Requires Go 1.26, Node 24, npm and Playwright Chromium:

```sh
cd integration/browser
npm ci
npm ci --prefix v3
npx playwright install --with-deps chromium
npm run matrix
```

After installing Chromium, `make browser` runs that same complete matrix from
the repository root. `npm run matrix -- v2` runs the four historical profiles;
`npm run matrix -- v3` runs the two new profiles. This is a bounded sequential
run, stops at the first failure and disables retries. It does not schedule
background checks or create external services. Ports 18985 and 19000–19003 must
be free. The optional `PLAYWRIGHT_CHROMIUM_EXECUTABLE` names an already installed
Chromium binary; omit it to use Playwright's managed Chromium.

For one profile:

```sh
INERTIA_VERSION=3.8.0 FRAMEWORK=react npm run build
npm run verify:ssr
npm test
```

Each v2 profile selects the unchanged 15-scenario regression set across four
adapter/bootstrap combinations: **240 historical browser scenarios** total.
Each v3 profile selects that shared set plus 16 v3 scenarios: **248 v3 scenarios**
selected in total, of which four intentionally skip the SSR-only failure test
in CSR projects. Reports and traces are separated by version/framework in
`playwright-report/` and `test-results/`, so later profiles do not overwrite
prior evidence. The existing four automatic CI profiles remain v2; adding v3
coverage does not silently expand automatic workflow spending.

## Exercised contracts

Shared v2/v3 scenarios cover initial markup and hydration preserving the heading
DOM node; JavaScript/hydration errors; navigation without document reload and
Back/Forward; real `useForm` errors and corrected submission; redirect/session
flash; native `onFlash` and global flash events; native flash stripped from
history; deferred loading; append/prepend/reset; Once reuse, expiry, freshness,
renamed keys and optional/deferred composition; dotted selection.

The additional v3 scenarios exercise:

- Script-element bootstrap with hostile script terminators and Unicode.
- `409 + X-Inertia-Redirect`, fresh GET and fragments; explicit `preserveFragment`.
- Interrupted/replaced visits and the public `cancelAll()` API.
- HTTP 404/500 Inertia pages, shared props and `onHttpException` callbacks.
- Recursive nested wrappers, dotted deferred metadata, explicit rescue and retry.
- Sparse array omissions observed through actual React/Vue `Deferred` components;
  dotted array partials and rescued indexes. A previously complete array follows
  the pinned client's replacement semantics; sparse updates are not represented
  as placeholder `null` values.
- Only/except intersection, v3 Once refresh, and dotted client deep merging.
- Foreground asset conflicts that reload versus background conflicts that notify
  without reloading; version response headers.
- Native history encryption and restoration; clear-history metadata.
- Reported SSR failure followed by CSR mounting, per-request SSR disable, and a
  subsequent independently hydrated document.
- Official BigInt parsing and exact values through SSR/CSR.
- `sharedProps` consumed during an actual instant visit.
- All validation messages in a named error bag across redirect/session storage.
- Custom scroll data paths, append/prepend intent and scroll reset metadata.

`server/v3_test.go` also verifies wire behavior directly, including the reference
adapter's provider-returned-container policy. Providers own their returned
container's selection; plain nested maps are filtered by dotted request paths.
The real browser assertions retain version-specific differences: v2 shallow
root replacement versus v3 deep merging for dotted partial targets, and v2
opt-in versus v3 default selected-Once refresh on partial visits.

## Server versus frontend responsibilities

| Feature family | Go/server responsibility | Client or host responsibility |
|---|---|---|
| Bootstrap, redirects, errors, props, flash | Correct HTTP/wire payload and session lifecycle | Rendering, callbacks and DOM |
| Deferred, Once, merge, scroll | Selection/evaluation and metadata | Scheduling, caching, merging and observers |
| Encryption, history and fragments | Flags, URLs and version headers | Crypto, browser storage, scrolling and restoration |
| SSR | Transport, timeouts/failure reporting and request policy | Framework renderer and hydration; host supervises process |
| Instant visits | Shared-prop metadata | Intermediate component swap and optimistic page UI |
| Forms/useHttp, optimistic updates, layouts, transitions | Ordinary validated HTTP responses | Framework hooks/state, rollback and visual transitions |
| Prefetch, polling, WhenVisible/WhenMounted | Normal Inertia transport, version/flash policy | Scheduling, caching, visibility and component lifecycle |
| Vite development integration | Configured SSR endpoint/hot-file support | Optional frontend plugin/build setup, owned by the host |

The last frontend-only families are not advertised as separate Go APIs or fully
covered frontend test suites. This matrix demonstrates the listed server-client
contracts, not every possible Inertia UI component or application policy.

## Verification limits

`npm run verify:ssr` invokes the real framework/Inertia renderer without launching
a browser. It is useful build/SSR evidence, **not a hydration pass**. A restricted
runner may forbid Chromium's process/socket setup. In that case keep the browser
gate pending and run the same frozen candidate in an authorized supported
executor before merge; never substitute method replay or SSR-only rendering.
All sessions and form data are synthetic, servers bind to loopback, and the
fixture must not be exposed as a production application.

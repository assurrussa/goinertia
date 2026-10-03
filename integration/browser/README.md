# Real browser contract matrix

This fixture runs the published Inertia clients, real Vue/React rendering and
real Go HTTP servers. It is separate from the small method replay in
`../protocol-replay` and makes no claim of complete Inertia support.

## Pinned matrix

| Client | Framework | Transports | Bootstrap |
|---|---|---|---|
| 2.3.18 | Vue 3.5.22, React 18.3.1 | native Fiber, native net/http | CSR, SSR + hydration |
| 2.3.28 | Vue 3.5.22, React 18.3.1 | native Fiber, native net/http | CSR, SSR + hydration |

Both adapters of each published client depend on the **same exact core version**.
The historical client uses npm aliases, so `npm ci` installs both complete
versions from one lockfile; no runtime version substitution or copied client
methods are used. All four public examples pin their Inertia adapter to 2.3.28.
React and Vue versions here are tested fixture versions, not minimum supported
versions or claims about the latest releases. Svelte and full v3 are outside this
matrix. The root Fiber facade retains the existing Go/replay regression gates.

Official sources checked on 2026-10-03:

- [2.3.18 release](https://github.com/inertiajs/inertia/releases/tag/v2.3.18)
- [2.3.28 release](https://github.com/inertiajs/inertia/releases/tag/v2.3.28)
- [v2 native flash contract](https://inertiajs.com/docs/v2/data-props/flash-data)
- Published `@inertiajs/core`, `@inertiajs/vue3`, and `@inertiajs/react` npm
  manifests at those exact versions; integrity hashes are in `package-lock.json`.

## Run

Requires Go 1.26, Node 24, npm and a supported Playwright Chromium installation:

```sh
cd integration/browser
npm ci
npx playwright install --with-deps chromium
INERTIA_VERSION=2.3.28 FRAMEWORK=vue npm run build
npm test
```

Repeat the build/test commands for `FRAMEWORK=react` and `INERTIA_VERSION=2.3.18`,
or run `make browser` at the repository root after installing Chromium. The Go
workflow has four independently reported matrix jobs. Each runs all four
transport/bootstrap combinations, for 24 browser scenarios per job (96 total).
Failures preserve Playwright traces/reports in CI artifacts. Retries are disabled.
The servers bind to loopback only and use disposable, random-cookie test sessions.
Do not expose this fixture as a production application.

Covered flows:

- Initial CSR bootstrap and real SSR markup, client hydration preserving the
  heading DOM node, hydration/JavaScript error checks.
- Inertia transitions without document reload, Back and Forward restoration.
- Deferred network requests and real client partial merge/reset behavior.
- `useForm` validation errors, corrected resubmission and write redirects.
- Native `page.flash`, `onFlash` and global flash event; legacy `props.flash`
  coexistence; native history stripping and legacy history preservation;
  session consume-once behavior on reload.

The fixture SSR renderer uses the actual framework `renderToString` and Inertia
`createInertiaApp`. The Go engine calls it through its normal SSR transport.
Tests use synthetic values and no external account or service.

A restricted runner may be able to build clients, run Go fixtures and render
SSR but forbid launching Chromium. Those checks do **not** establish a browser
pass or hydration compatibility. In that case the real browser CI jobs must pass
before merge; do not report method replay as equivalent evidence.

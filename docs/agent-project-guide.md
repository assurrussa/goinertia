# Agent Project Guide

This guide captures project-specific facts that future agents need before
changing `goinertia`. It complements `AGENTS.md`, `README.md`, and the topic
docs in this directory.

## Purpose

`goinertia` has a neutral core, a compatible Fiber root facade and native Fiber/net/http adapters. It is a Go adapter for Inertia.js. It lets a Go/Fiber
backend own routing and controllers while Vue, React, or Svelte owns the
client-side page components.

The module path is `github.com/assurrussa/goinertia`. The current Go module
targets Go `1.26` and Fiber `v3.3.0`.

The architecture decision retains the neutral core and both native lifecycles.
The compatible wire contract is a documented Inertia v2.x subset; current
official documentation targets v3. Read [protocol coverage](protocol-coverage.md)
before claiming full protocol/client compatibility. The separate
`integration/nethttp-consumer` module validates the public HTTP API and its compiled
dependency graph without root/Fiber imports. `make check` and the Go workflow
run it and the pinned Node 24 client replay explicitly. The separate
`integration/browser` fixture tests published Vue/React clients at 2.3.18 and
2.3.28 with both native adapters and CSR/SSR hydration. `make browser` and four
CI matrix jobs exercise it; Chromium is required. A replay or SSR-only render
is not a substitute for the browser jobs.

## Public Surface

The root package provides the production API:

- Constructors: `New`, `NewWithValidation`, and `Must`.
- Rendering and middleware: `Render`, `Middleware`, and
  `MiddlewareErrorListener`.
- Redirect helpers: `Redirect`, `RedirectBack`, `RedirectExternal`,
  `RedirectBackWithErrors`, and `RedirectBackWithValidationErrors`.
- Prop helpers: `WithProp`, `WithViewData`, `WithLazyProp`,
  `WithMatchPropsOn`, `WithEncryptHistory`, `WithClearHistory`, `Defer`,
  `Optional`, `Always`, `Merge`, `Prepend`, `DeepMerge`, `Scroll`, and `Once`.
- Flash and validation helpers: `WithNativeFlash` (top-level `page.flash`),
  `WithFlashSuccess`, `WithFlashInfo`,
  `WithFlashWarning`, `WithFlashError`, `WithFlashOld`, `WithErrors`,
  `WithError`, and `WithValidationErrors`.
- Options: template/public filesystems, templates, asset version, session
  store, logger, shared template funcs, shared view data, shared props, error
  detail callbacks, CSRF providers, SSR config, dev mode, and Precognition
  `Vary` behavior.
- Extension contracts: `SessionStore`, generic `FiberSessionAdapter`,
  `Logger`, `SSRClient`, `CSRFTokenProvider`, and `CSRFTokenCheckProvider`.

The `inertiat` package is test support, not production runtime. It provides
test apps, request helpers, page decoding, and a mock session store.

## Runtime Behavior To Preserve

The middleware handles both regular browser requests and Inertia XHR requests.
For Inertia requests it adds `Vary: X-Inertia`, checks asset versions for GET
requests, sends `409` plus `X-Inertia-Location` for version conflicts, rewrites
write-method redirects from `302` to `303`, and preserves explicit
`Cache-Control: no-cache` echo behavior.

Precognition requests bypass asset-version conflicts and session flash writes.
Successful validation-only requests return `204` with
`Precognition-Success: true`; failing requests return validation errors.

Props are built from shared props, context props, and request props. Context
and request keys override shared props. The `errors` prop is always present,
with an empty object when no errors exist.

Session flash persistence is limited to redirect-like responses and
`409 + X-Inertia-Location`. Normal renders do not write flash data into the
session.

SSR is optional. It posts the built page payload to an SSR endpoint, supports
custom headers and a custom `SSRClient`, retries retryable failures, and can
cache SSR responses by payload hash.

Templates default to `app.gohtml`, `error.gohtml`, and `hot`. Dev mode reparses
templates and rereads the Vite hot file on each request.

## Examples

Use examples as integration references:

- `examples/basic-app`: Fiber + Vue + no SSR.
- `examples/basic-app-ssr`: Fiber + Vue + SSR endpoint.
- `examples/basic-app-react`: Fiber + React + no SSR.
- `examples/basic-app-react-ssr`: Fiber + React + SSR endpoint.

Each example installs frontend dependencies inside its own directory. Build the
selected frontend bundle there, then run the matching Go example from the repo
root with `go run ./examples/<name>`.

## Verification

Use focused tests while iterating:

- `go test ./...` for normal verification.
- `go test -race -count=5 ./...` for shared behavior, middleware, session, SSR,
  or cache changes.
- `go test -bench=. -benchmem ./...` or `make bench-all` when changing hot
  request paths.
- `go generate ./...` after changing interfaces that feed generated mocks.

Use `make check` before a release-style handoff when the required local tools
are installed.

## Documentation Placement

Put user-facing feature details in topic docs:

- Setup and rendering: `docs/basic.md`.
- Options: `docs/options.md`.
- Flash/session behavior: `docs/flash.md`.
- Validation and redirects: `docs/validation.md`.
- Precognition: `docs/precognition.md`.
- History flags: `docs/history.md`.
- External redirects and `409`: `docs/redirect-409.md`.
- Lazy/deferred/optional props: `docs/lazy-props.md`.
- Shared lazy props: `docs/shared-lazy.md`.
- SSR: `docs/ssr.md`.
- Uploads: `docs/uploads.md`.

Keep future agent-only decisions in the root `implementation-notes.md`.

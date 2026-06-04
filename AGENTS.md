# Repository Guidelines

## Project Contract

`goinertia` is a reusable Go module at `github.com/assurrussa/goinertia`.
It is a Fiber v3 adapter for Inertia.js applications, with support for
Inertia protocol headers, page rendering, partial reloads, lazy/deferred/once
props, merge and scroll metadata, flash and validation helpers, CSRF hooks,
error handling, and optional SSR.

Treat this repository as a library, not an application host. Keep public API
changes small, documented, and covered by tests because downstream Fiber
applications import the root package directly.

## Source Order

Local verified files win over shared wiki notes for commands, public APIs,
config keys, supported imports, runtime behavior, and release gates.

Read context in this order:

1. `AGENTS.md`, `README.md`, and `docs/agent-project-guide.md`.
2. The relevant topic document in `docs/`.
3. `go.mod`, `Makefile`, package code, tests, examples, and generated mocks.
4. Shared wiki pages only after local grounding.

Use `$project-context-router` for work that needs cross-project context or the
shared wiki. The shared wiki root is `/Users/amir/agents/agent-context`.
Relevant pages are:

- `/Users/amir/agents/agent-context/streams/wiki/index.md`
- `/Users/amir/agents/agent-context/streams/wiki/glossary.md`
- `/Users/amir/agents/agent-context/streams/wiki/platforms/goinertia.md`
- `/Users/amir/agents/agent-context/streams/wiki/platforms/gofiber.md`

If local docs or code conflict with the shared wiki, treat the wiki as stale.
When the task includes documentation upkeep and the shared context is writable,
update the relevant platform page after verification. Keep shared pages concise
and contract-focused; do not copy full local README sections into wiki.

## Repository Layout

- Root package: production `goinertia` API and protocol behavior.
- `inertiat/`: reusable test helpers for Fiber/Inertia requests and mock
  session support.
- `views/`, `public/`, and `testdata/`: embedded defaults and test fixtures.
- `examples/basic-app`: Vue client example without SSR.
- `examples/basic-app-ssr`: Vue client example with SSR.
- `examples/basic-app-react`: React client example without SSR.
- `examples/basic-app-react-ssr`: React client example with SSR.
- `docs/`: user-facing topic documentation plus agent project notes.
- `mocks/`: generated mocks from `go generate ./...`.

## Commands

- Fast verification: `go test ./...`
- Race verification: `go test -race -count=5 ./...`
- Benchmarks: `make bench-all`
- Generate mocks: `go generate ./...`
- Full local gate: `make check`

`make check` runs tidy, generate, formatting, vet, lint, tests, race tests,
and HTML coverage. It expects local tools such as `gofumpt`, `gci`, and
`golangci-lint`.

Example applications have their own `package.json` files. Install and build
frontend assets from the selected example directory, then run the Go example
from the repository root. The default example port is `8383`, configurable via
`-port` or `PORT` where the target command supports it.

## Development Rules

- Preserve Fiber v3 signatures; handlers use `fiber.Ctx`.
- Prefer `NewWithValidation` for examples and startup paths that should fail
  fast on invalid base URL or templates.
- Keep `New` lenient for callers that need delayed template parsing.
- Keep Inertia protocol behavior covered when touching middleware, redirects,
  headers, asset version conflicts, partial reloads, merge metadata,
  Precognition, flash/session behavior, or SSR.
- Do not add host-application routing, authentication, storage, or frontend
  build policy to the root package. Those remain host responsibilities.
- Session-backed flash and validation flows require `WithSessionStore`.
- SSR is optional and must remain injectable through `SSRClient` for tests and
  host customization.
- When changing public behavior, update the matching `docs/` topic and add or
  adjust tests in the same change.

## Documentation Rules

- Keep `README.md` as the high-level product and quick-start entrypoint.
- Keep detailed feature behavior in topic files under `docs/`.
- Keep agent-facing project facts in `docs/agent-project-guide.md`.
- Keep implementation decisions and tradeoffs in `implementation-notes.md`.
- Do not duplicate large README sections into shared wiki or agent notes.

# Implementation Notes

## 2026-06-04 Project Initialization Docs

Scope: initialized agent-facing project documentation for the existing
`goinertia` repository. No production code was changed.

Decisions:

- Kept `AGENTS.md` as the operational contract for future agents instead of
  copying the full README.
- Added `docs/agent-project-guide.md` for project-specific facts that are too
  detailed for `AGENTS.md` but useful before implementation work.
- Updated `docs/README.md` so the new agent guide is discoverable from the docs
  index and fixed the existing duplicate `redirect-409.md` entry while there.
- Wrote docs in English to match the existing README and topic documentation.
- Verified the shared `goinertia` and `gofiber` wiki pages against the local
  README/code at a summary level. They matched the current project role, so no
  shared wiki update was needed.

Tradeoffs:

- Did not document every public method in `AGENTS.md`; detailed API inventory
  lives in `docs/agent-project-guide.md`.
- Did not run frontend example builds during this docs-only initialization.

## 2026-06-04 Lint Configuration Follow-up

Scope: fixed the `goconst` lint failure reported by `make lint`.

Decision:

- Added `golangci-lint` v2 `linters.exclusions.rules` for `goconst` in
  `_test.go` files and `examples/`. The reported duplicates were test fixture
  values and demo UI labels; extracting them into constants would reduce
  readability without improving the production library contract.

Verification note:

- The exclusion syntax was checked against current `golangci-lint`
  documentation through Context7 before editing.
- `make lint` passes outside the sandbox. Inside the sandbox, Make/Go can fail
  before linting while trying to write Go module stat cache under
  `/Users/amir/go/pkg/mod/cache`.

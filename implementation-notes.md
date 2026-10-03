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

## 2026-10-02 Compatible native adapters

- Preserved root imports as aliases/delegation to the native Fiber adapter;
  neutral DTOs, props, concrete request metadata/state, templates/assets and
  SSR policy now live in core. One module and Go 1.26 are retained.
- Correctness fixes precede extraction in separate commits: 5561699 and
  1f4701b. SSR uses net/http with owned bodies and lifecycle cancellation;
  retry no longer mutates/reset a shared transport. Custom injection remains.
- Root lazy callbacks still receive fiber.Ctx. Native callbacks receive a
  lifecycle context plus typed adapter helpers. Legacy metadata Locals remain
  usable. Constructor defaults and public mock symbols are preserved.
- Raw sessions release acquired values; custom bridges retain ownership;
  middleware sessions are borrowed. GoAdmin 0.7 source integration verifies
  breadcrumbs, CSRF and rotation/flash through its own SessionBridge.
- HTTP response policy executes before commitment and preserves streaming and
  optional writer capabilities. Whole arbitrary responses are not buffered.
- Validation includes make check and build on Go 1.26.7, external consumer,
  existing Fiber suite, shared protocol fixtures and race tests. GoAdmin 0.7
  uses its own Go 1.27.1/Fiber 3.5 dependency graph with a local replace.
- Five benchmark comparisons ran after the other test jobs finished, using
  the same corrected-baseline harness, Go 1.26.7 and four execution threads.
  Small Fiber requests and flash show measurable time/allocation increases;
  larger pages have smaller differences and parallel timings vary. Required
  pooled-string ownership is measured separately. See docs/benchmarks.md.
- No public release, retag, merge, host migration or production action is part
  of this implementation. Publication is a separate authorization boundary.

## 2026-10-02 Profiling follow-up

The initial extraction was published as a WIP draft after direct authorization.
Flash allocation profiles identified unnecessary State/context/metadata capture
on mutation-only requests. Those helpers now use a temporary concrete State and
synchronize existing native state. Full owned metadata is deferred to rendering.
Vary checks avoid token split allocations; page reads avoid empty local maps,
override maps without shared props and empty partial configurations. Legacy
callbacks need no additional neutral context helper. HTML uses one State per
render while honoring direct legacy Locals view-data writes.

SSR/body/metadata/session ownership remains intact without unsafe aliases or
State pooling. Full make check/build, protocol/race, external consumer and
GoAdmin source gates pass. Results and remaining acceptance limits are in
docs/benchmarks.md; the product/maintenance decision is in docs/core-tradeoffs.md.


## V2 prop follow-up (stage 3)

Nested selection traverses JSON maps before lazy resolution, with deterministic
literal-dot dual matching and preserved except precedence. Required ancestor
callbacks still execute. Filtered containers are request-owned. MergeAt supplies
new target metadata without adding fields to positional MergeProp literals.
Once freshness settings are held in a private Value envelope through existing
OnceOption, preserving OnceProp's public layout. Root Deferred/Optional/Merge
composition is supported; arbitrary wrappers in nested containers are not.
The official v2 selected-partial Once refresh rule is opt-in, preserving existing
except-only behavior. Full v3 migration is a later, separately scoped task.

Merge metadata ownership is tracked per page in request-private State rather
than inferred from dotted wire paths. Literal dotted root names and nested
paths can emit identical labels. Removing one root removes only its ownership;
other emitting roots retain their label, while unrelated plain values cannot
inherit stale merge instructions. New MergeAt values replace lower-layer data
and wrapper metadata before Once/deferred selection; old wrapper replacement
semantics outside MergeAt remain unchanged.

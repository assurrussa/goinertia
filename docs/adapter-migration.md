# Compatible adapter migration

Baseline: master a8255f25bf46be9a01011d489c9e1191f1d5e0c6; Go directive 1.26.

1. Correct the middleware benchmark to assert a successful response. Fix SSR
   body ownership, cancellation and retry reset, raw session lifetime,
   checker-only CSRF, cold templates, Vary and scheme-relative redirects.
   Preserve this corrected Fiber implementation as the performance baseline.
2. Extract a concrete request metadata/state and shared page/props, template,
   assets and SSR policy into `core`. Move framework lifecycle to
   `adapters/fiber`; retain the root API as a compatibility facade.
3. Implement native `adapters/nethttp` with policy before response commitment,
   explicit session/CSRF hooks and writer capability passthrough. Run shared
   protocol fixtures against both adapters.
4. Run the existing Fiber suite, race/SSR/session/cold-template checks,
   fmt/vet/lint/build and external consumer/GoAdmin 0.7 integration. Compare
   corrected old Fiber, new Fiber and native HTTP using a common benchmark
   matrix when the host is sufficiently idle.

Legacy prop callbacks receive Fiber Ctx; neutral callbacks receive lifecycle
context. SSR errors remain errors. Raw Fiber Store sessions transfer ownership;
custom SessionStore implementations retain their own lifetime. No host project
changes are part of this migration.

The first corrected baseline passed `go test -race -count=5 ./...` on the local
Go 1.27.1 compiler. The module directive is unchanged. Go 1.26 compiler validation
is a separate gate. Benchmark timing is deferred while concurrent API/GeoIP work
uses the host. No performance claims or public tag validation have been made.

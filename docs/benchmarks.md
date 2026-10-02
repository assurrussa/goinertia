# Adapter comparison benchmarks

Corrected original Fiber baseline: `1f4701b572f77d8049d1a0c90e873d081b02fa7a`.
This includes the successful middleware benchmark, SSR/session ownership and
protocol corrections before extraction. The initial checkpoint is
`5561699ca2b2db4430de7a4d0a233e75cd843be2`.

The common `comparison_benchmark_test.go` is copied unchanged into a checkout
of the corrected baseline. Run on an idle host, with the same compiler, CPU
settings, dependencies and Go cache state in both checkouts:

```sh
go test -run '^$' -bench '^BenchmarkCorrectedFiber$' -benchmem -count=10 ./
go test -run '^$' -bench '^Benchmark(CorrectedFiber|NativeFiber|NativeHTTP|FiberMetadataOwnership)$' -benchmem -count=10 ./
```

Save separate result files for old Fiber, the new legacy facade, native Fiber
and native HTTP. Compare each scenario's ns/op, B/op and allocs/op with benchstat.
The baseline and new Fiber use direct native Fiber/fasthttp request dispatch.
HTTP uses native Handler dispatch and a reusable fixture writer. These are
adapter/framework measurements, not network round-trip throughput.

The matrix covers warm JSON/HTML with 0/10/100 props and 0/1/10 evaluated lazy
props where the count fits, sequential and parallel dispatch, flash redirects,
SSR cache hits, retry and cancellation policy. Warmup asserts status and lazy
counts. Flash uses equivalent stateless cookie/session fixtures; it does not
measure a database or production session backend. SSR cases use custom local
fixtures: retry checks policy, cancellation uses a cancelled lifecycle context,
and cache uses a warmed shared cache. Actual network ownership, cancellation
and parallel retries are separately checked against loopback SSR servers.
SSR retry/cancel cases are sequential; parallel coverage applies to warm page
rendering. Logger fixtures discard output consistently.

`BenchmarkFiberMetadataOwnership` isolates pooled Fiber string copies. These
copies ensure safe ownership and must not be described as adapter-only overhead.
The original middleware benchmark includes app.Test overhead; the comparison
matrix uses direct dispatch and must not be mixed with its timing numbers.

## Local comparison, 2026-10-02

After the concurrent API/GeoIP test jobs finished, all cases passed a one-iteration
correctness preflight. Five old/new series then ran sequentially on an Apple M5
Pro, Go 1.26.7, darwin/arm64, `GOMAXPROCS=4`, `-benchtime=200ms`, `-benchmem`.
The two checkouts used the same common harness and warm build/dependency caches.
The host had 85–89% idle CPU before the run, 69–72% during a sampled parallel
run, and 82–84% afterward, without active swap I/O in those samples. No other
processes were stopped. Timing had been deferred while the other jobs ran.

Medians from five samples follow; each cell is **ns/op / B/op / allocs/op**.
The [complete 32-scenario matrix](benchmark-results.csv) also includes the new
legacy facade, all lazy counts and all parallel cases.

| Scenario | Corrected old Fiber | Native Fiber | Native HTTP |
|---|---:|---:|---:|
| JSON, 0 props | 1312 / 1289 / 23 | 1517 / 1655 / 28 | 1058 / 1866 / 23 |
| JSON, 10 props | 2558 / 2731 / 30 | 2807 / 2889 / 34 | 2347 / 3100 / 29 |
| JSON, 100 props, 10 lazy | 17875 / 20254 / 48 | 18276 / 20397 / 51 | 17684 / 20374 / 46 |
| JSON, 100 props, 10 lazy, parallel | 7840 / 20289 / 48 | 8264 / 20436 / 51 | 8023 / 20400 / 46 |
| HTML, 10 props | 8106 / 11788 / 129 | 8526 / 11941 / 131 | 8044 / 12188 / 131 |
| HTML, 100 props, 10 lazy | 29534 / 42208 / 148 | 30337 / 42341 / 149 | 28993 / 41472 / 149 |
| Flash redirect | 877 / 1545 / 19 | 1444 / 1961 / 25 | 1148 / 2424 / 24 |
| SSR cache hit | 9848 / 13035 / 156 | 10276 / 13187 / 158 | 9799 / 13415 / 158 |
| SSR retry fixture | 10882 / 14076 / 172 | 11185 / 14231 / 174 | 11059 / 14462 / 174 |
| SSR cancellation fixture | 3273 / 4379 / 60 | 3449 / 4531 / 62 | 3082 / 4731 / 59 |

These measurements show a real cost for small Fiber requests and flash: native
Fiber medians increase by 15.6% for empty JSON and 64.7% for flash, with five and
six additional allocations respectively. The new root facade's empty JSON is
1653 ns/op, 1655 B/op, 28 allocs/op; flash is 1432 ns/op, 1961 B/op, 25 allocs/op.
State, lifecycle helpers and owned metadata contribute to this change. The
standalone ownership fixture with 14 populated string fields measures 506 ns/op,
168 B/op and 14 allocs/op; it is a separate workload and cannot be subtracted
from the page results to claim pure adapter overhead.

Benchstat on the five samples detects changes in several small-request/flash
cases and some parallel cases. Some HTML parallel samples vary by more than
20%; short runs on a shared workstation do not establish a stable production
percentage. Larger sequential page cases often have no statistically resolved
timing difference in this sample. Allocation increases remain measurable.
No zero-loss claim, network SSR throughput claim or performance acceptance
threshold is implied. Repeat longer runs on a dedicated host before making
deployment capacity or performance budget decisions.

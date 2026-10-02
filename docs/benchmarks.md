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

## Initial extraction comparison, 2026-10-02 (02e9250)

After the concurrent API/GeoIP test jobs finished, all cases passed a one-iteration
correctness preflight. Five old/new series then ran sequentially on an Apple M5
Pro, Go 1.26.7, darwin/arm64, `GOMAXPROCS=4`, `-benchtime=200ms`, `-benchmem`.
The two checkouts used the same common harness and warm build/dependency caches.
The host had 85–89% idle CPU before the run, 69–72% during a sampled parallel
run, and 82–84% afterward, without active swap I/O in those samples. No other
processes were stopped. Timing had been deferred while the other jobs ran.

Medians from five samples follow; each cell is **ns/op / B/op / allocs/op**.
The [complete 32-scenario matrix](benchmark-initial-results.csv) also includes the new
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

## Optimization follow-up (e8491e9)

Allocation profiles of the same flash fixture identified eager State/context/
metadata capture, allocating Vary token splits and unused page-building maps.
Prop/flash mutation now uses a temporary concrete State for Locals, capturing
owned protocol metadata only for rendering or explicit native State access.
Vary checks scan without split allocations. Empty partial configuration and
unused local/override maps stay absent. Legacy callbacks keep Fiber Ctx without
an additional neutral helper; native callbacks retain that helper. HTML rendering
reuses the State already used to build its page and still refreshes direct legacy
Locals view data. Required metadata and SSR body ownership remain intact.

The initial comparison is preserved above. The follow-up uses the same corrected
original SHA, unchanged harness, compiler and four execution threads, with five
sequential series whose old/new order alternates. The ownership benchmark is a
separate workload rather than a subtraction from adapter timing. See also
[the architecture and maintenance decision](core-tradeoffs.md).


Follow-up medians, **ns/op / B/op / allocs/op**:

| Scenario | Old Fiber ns / B / allocs | New facade ns / B / allocs | Native Fiber ns / B / allocs | Native HTTP ns / B / allocs |
|---|---:|---:|---:|---:|
| JSON/props=0/lazy=0 | 1306 / 1289 / 23 | 1333 / 1221 / 14 | 1264 / 1269 / 15 | 950 / 1721 / 19 |
| JSON/props=10/lazy=0 | 2530 / 2731 / 30 | 2175 / 1998 / 17 | 2196 / 2047 / 18 | 1864 / 2499 / 22 |
| JSON/props=100/lazy=10 | 17811 / 20250 / 48 | 13456 / 13199 / 28 | 13674 / 13253 / 29 | 13224 / 13460 / 33 |
| JSON/props=100/lazy=10/parallel | 7648 / 20284 / 48 | 5322 / 13236 / 28 | 5457 / 13281 / 29 | 5410 / 13492 / 33 |
| HTML/props=10/lazy=0 | 8141 / 11790 / 129 | 7669 / 11157 / 118 | 7668 / 11205 / 119 | 7364 / 11584 / 124 |
| HTML/props=100/lazy=10 | 28551 / 42175 / 148 | 24115 / 35193 / 130 | 24449 / 35236 / 131 | 23959 / 34549 / 136 |
| flash/props=10/lazy=0 | 861 / 1545 / 19 | 687 / 1297 / 9 | 698 / 1297 / 9 | 1090 / 2392 / 22 |
| SSR-cache/props=10/lazy=0 | 9862 / 13032 / 156 | 9193 / 12400 / 145 | 9292 / 12448 / 146 | 8986 / 12814 / 151 |
| SSR-retry/props=10/lazy=0 | 10663 / 14075 / 172 | 10132 / 13439 / 161 | 10136 / 13490 / 162 | 9822 / 13856 / 167 |
| SSR-cancel/props=10/lazy=0 | 3170 / 4379 / 60 | 2835 / 3750 / 49 | 2857 / 3798 / 50 | 2553 / 4130 / 52 |

The [complete optimization matrix](benchmark-optimized-results.csv) includes every lazy/parallel
case. CPU idle was 81–85% before that run and 73–84% during the sampled
window. A few swap-in pages were observed before timing; none in the sampled
mid-run window. All five series passed status and per-request lazy-count checks.

Benchstat detects no statistically significant time regression across the 32
measured scenarios for either the root facade or native Fiber. Empty facade JSON
medians are 1306 → 1333 ns/op, with no resolved timing difference (p=0.310);
allocations decrease 23 → 14 and bytes 1289 → 1221. Native Fiber empty JSON is
1264 ns/op, 1269 B/op, 15 allocations. Flash improves from 861 ns/op, 1545 B/op,
19 allocations to 687/1297/9 for the facade and 698/1297/9 for native Fiber.
Empty HTML has no resolved timing difference in that sample. Larger
JSON/HTML pages have measurable improvements. The standalone ownership fixture
measures 481 ns/op, 168 B/op and 14 allocations, with required copies preserved.

Some raw samples still have substantial variance. No resolved regression in a
short workstation matrix proves neither exact equality nor a universal zero-loss
guarantee. Native HTTP flash remains costlier than old Fiber (1090 ns/op, 2392
B/op, 22 allocations versus 861/1545/19), reflecting the native lifecycle and
fixture differences; it is not hidden behind the Fiber results. The architecture direction now retains both adapters; measured budgets still
require review.


## Protocol audit follow-up

The final audit retains the same corrected old Fiber baseline and unchanged
common harness. Five old/new series alternate execution order, with Go 1.26.7,
GOMAXPROCS=4 and 200ms per case on the same Apple M5 Pro. No Go checks run
concurrently with timing, and no unrelated process is stopped. Sampled CPU idle
is 90–96% before, 67–74% during and 78–87% after. Some old samples still vary
substantially; the raw series is retained rather than silently replaced.

Final medians, **ns/op / B/op / allocs/op**:

| Scenario | Old Fiber ns / B / allocs | New facade ns / B / allocs | Native Fiber ns / B / allocs | Native HTTP ns / B / allocs |
|---|---:|---:|---:|---:|
| JSON/props=0/lazy=0 | 1261 / 1289 / 23 | 1297 / 1221 / 14 | 1244 / 1269 / 15 | 970 / 1722 / 19 |
| JSON/props=10/lazy=0 | 2430 / 2731 / 30 | 2130 / 1998 / 17 | 2180 / 2046 / 18 | 1904 / 2499 / 22 |
| JSON/props=100/lazy=10 | 17398 / 20266 / 48 | 13274 / 13204 / 28 | 13599 / 13251 / 29 | 13199 / 13460 / 33 |
| JSON/props=100/lazy=10/parallel | 7578 / 20294 / 48 | 5341 / 13232 / 28 | 5392 / 13281 / 29 | 5419 / 13491 / 33 |
| HTML/props=10/lazy=0 | 7894 / 11787 / 129 | 7462 / 11157 / 118 | 7544 / 11205 / 119 | 7351 / 11586 / 124 |
| HTML/props=100/lazy=10 | 27893 / 42159 / 148 | 23985 / 35176 / 130 | 24077 / 35246 / 131 | 23667 / 34546 / 136 |
| flash/props=10/lazy=0 | 848 / 1545 / 19 | 677 / 1297 / 9 | 676 / 1297 / 9 | 1116 / 2392 / 22 |
| SSR-cache/props=10/lazy=0 | 9381 / 13033 / 156 | 9032 / 12402 / 145 | 9026 / 12450 / 146 | 9005 / 12814 / 151 |
| SSR-retry/props=10/lazy=0 | 10409 / 14078 / 172 | 10050 / 13440 / 161 | 10086 / 13490 / 162 | 9928 / 13858 / 167 |
| SSR-cancel/props=10/lazy=0 | 3094 / 4379 / 60 | 2810 / 3750 / 49 | 2843 / 3798 / 50 | 2546 / 4130 / 52 |

The [current complete matrix](benchmark-results.csv) covers all 32 scenarios.
The root facade has no statistically significant time regression in that
matrix; its empty JSON timing is unresolved (p=0.135). Native Fiber has one
short-run signal: empty parallel JSON +3.09% (p=0.016). A focused five-round,
old/native alternating 1s check does not resolve that parallel difference
(p=0.151), but does find +3.42% for sequential empty native JSON (p=0.016).
The focused means are about 1.22 versus 1.26 microseconds; allocations remain
23 versus 15. These mixed signals on a shared workstation warrant a small
empty-request cost/uncertainty budget, not a universal no-regression claim.
The focused series samples 64–82% idle CPU; a few swap-in pages occur in that
sample. No broad matrix is discarded in favor of the focused result.

Native HTTP flash remains costlier than old Fiber: 1116/2392/22 versus
848/1545/19. The separate mandatory metadata copy fixture is 483 ns/op,
168 B/op and 14 allocations and is not subtracted from adapter measurements.
Nested lazy maps/slices now require copies when callbacks change children;
this ownership cost is not measured by the flat-prop warm-page matrix and must
not be called adapter-only overhead. There is no production/network throughput
or full-protocol performance claim.

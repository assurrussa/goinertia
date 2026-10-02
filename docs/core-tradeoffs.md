# Core and adapter tradeoffs

The split is a protocol/rendering engine with two native lifecycles. It does
not turn Fiber into HTTP or introduce a universal framework context. The
existing root package remains a Fiber compatibility API. One module still
requires Fiber; only the compiled core/native HTTP graphs are Fiber-free.

## Shared implementation and maintenance cost

Core owns the page DTO, shared/local/request prop precedence, partial and
deferred/optional/always/once props, merge/scroll/history metadata, validation
bags, templates/assets, SSR retries/cache/ownership and protocol policy. Both
adapters execute this code. Those features are substantial reusable behavior;
their extraction is more than a common interface around two handlers.

The baseline had approximately 2774 physical Go lines in its production root
files. The split has approximately 2317 in core, 1223 in Fiber, 606 in HTTP and
179 in the root facade: roughly 4325 total, including comments and excluding
tests/generated mocks. This is about 56% more source, not a free abstraction.
Fiber aliases/options and facade forwarding account for part of the increase.
The HTTP lifecycle includes precommit session cookies and response status,
streaming, informational responses and optional writer capabilities. These
semantics need their own tests and future review even when protocol fixtures
are shared. The native HTTP writer alone is roughly 240 lines.

## What caused the initial slowdown

Allocation profiles identified avoidable work in the first extraction:

- Flash/prop helpers created a complete request State, context helper and
  owned protocol metadata even when the request only redirected. Such helpers
  now operate on a temporary concrete State containing existing Locals; a full
  owned State is created when rendering or explicitly requested by a consumer.
- Repeated Vary checks allocated token slices and copied current header strings.
  They now scan tokens without allocation; an empty header gets the canonical
  combined value once. Direct rendering still applies the required policy.
- Page building eagerly created unused local/override maps and an empty partial
  configuration. Reads now leave optional maps nil, override keys are collected
  only when shared props exist, and absent partial policy needs no allocation.
- Legacy callbacks already receive the actual Fiber context, so they do not
  also need the neutral adapter helper added to their lifecycle context. Native
  callbacks retain that helper; SSR always receives lifecycle context.

Required metadata ownership and SSR body copies remain. No unsafe aliases,
reflection, universal context or shared State pooling were introduced to recover
performance. Current numbers and uncertainty are in [benchmarks.md](benchmarks.md).

## Recommendation

Keep a small neutral protocol/rendering core if native HTTP support has an
actual consumer or a concrete near-term use. The shared behavior above and
independent compiled HTTP graph justify that boundary, provided both lifecycles
receive maintenance and the measured performance budget is accepted.

If all supported hosts remain Fiber/GoAdmin and HTTP has no planned consumer,
prefer Fiber-only for the public product, or retain only narrow neutral DTO,
prop, template and SSR pieces. Maintaining the second lifecycle just for a
future claim of universality adds source and tests without current user value.
Leaving this as a draft is reasonable until that product decision is made.

Performance improvement alone does not prove the split is the better design:
the Vary/map optimizations could also be applied to a Fiber-only implementation.
The decision is reuse and maintenance cost, not whether extraction work has
already been done. A larger adapter matrix or a new universal HTTP framework is
outside this compatible stage.

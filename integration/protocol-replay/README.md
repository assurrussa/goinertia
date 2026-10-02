# Pinned client protocol replay

`client-v2.3.18.ts` contains unchanged MIT-licensed methods/blocks from
[`inertiajs/inertia` v2.3.18](https://github.com/inertiajs/inertia/tree/ed9b159a5857663211580e081d6bd90520f17bad),
commit `ed9b159a5857663211580e081d6bd90520f17bad`:

- `packages/core/src/response.ts`: mergeProps and its matching helpers.
- `packages/core/src/router.ts`: loadDeferredProps.
- `packages/core/src/infiniteScroll/data.ts`: resetState and success listener.

The adjacent LICENSE preserves the upstream notice. Dependencies (`get`, `set`,
current page, reload dispatch and browser events) are small doubles. Client
merging, deferred group dispatch and scroll reset code is copied unchanged.
The Go test generates actual responses from legacy Fiber, native Fiber and
native HTTP; Node replays them through these client methods. This is a focused
browser-independent regression replay, not a complete browser/DOM or hydration
test and not an assertion of full client compatibility.

Run from the module root with Node 24 available:

```sh
GOINERTIA_CLIENT_REPLAY=1 go test -run '^TestPinnedClientProtocolReplay$' -count=1 ./
```

The test fails if Node is missing and runs with a bounded timeout. `make check`
and the existing required Go workflow run `make protocol-replay` with Node 24;
the plain Go suite still skips it. The replay uses raw adapter response JSON,
so wire-only fields are not lost in a public DTO decode/re-encode round trip. No
npm install, frontend asset build, provider or remote service is used.

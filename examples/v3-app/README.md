# Inertia v3: Vue/React, Fiber/net/http, CSR/SSR

This runnable example uses official Inertia **3.8.0**, Vue **3.5.22** or React
**19.3.0**, and explicitly opts into `core.ProtocolV3`. Its lockfile and React 19
runtime are separate from the existing v2 examples, which remain on 2.3.28.

Requires Go 1.26 and Node 24. From this directory:

```sh
npm ci
FRAMEWORK=vue npm run build
# Or: FRAMEWORK=react npm run build
```

Then, from the repository root, choose a native adapter:

```sh
go run ./examples/v3-app -adapter fiber -port 8383
# Or: go run ./examples/v3-app -adapter nethttp -port 8383
```

Open http://127.0.0.1:8383. The Home/About links perform Inertia visits. Nested
`details.message` is deferred, while `details.catalog` uses a shared Once key.
The template uses `.inertiaHead` and `.inertiaBody`; the engine emits a safe
`script[type=application/json][data-page=app]` bootstrap.

## Real SSR and hydration

Build either framework as above. Start its SSR renderer in this directory:

```sh
npm run ssr
```

In another terminal, from the repository root:

```sh
go run ./examples/v3-app -adapter nethttp -ssr-url http://127.0.0.1:13714/render
# -adapter fiber uses the same renderer and template.
```

The renderer binds only to loopback. Go requests its `/render` endpoint using
the library's normal SSR transport. The browser uses `hydrateRoot` for React or
`createSSRApp` for Vue when markup is present, and mounts normally for CSR.
`SSR_PORT` changes the Node port; pass the matching URL to Go. Restart both
processes after switching frontend builds.

A full document request to `/without-ssr` demonstrates per-request SSR disable.
Ordinary Inertia navigation already returns JSON and does not invoke SSR.
If SSR fails, v3 falls back to CSR and the configured failure callback logs the
failure. Set `WithSSRErrorPolicy(core.SSRErrorPropagate)` if the host should fail
instead. No error is silently swallowed.

This is a small integration example, not a production application template.
Production hosts own authentication, sessions, CSRF, URL authorization, asset
fingerprinting, process supervision and observability. The explicit HTTP SSR
bridge is independent of Laravel or Vite's optional development plugin.

The [browser matrix](../../integration/browser/README.md) exercises both native
adapters and both bootstrap modes with the actual published clients, alongside
the preserved v2 clients. A successful frontend build alone does not establish
hydration compatibility.

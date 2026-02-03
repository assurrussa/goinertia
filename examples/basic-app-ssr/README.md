# basic-app-ssr

A minimal Fiber + goinertia SSR example with a tiny Vue app and SSR setup.

## Run

### 1) Install frontend deps

```
cd examples/basic-app-ssr
npm install
```

### 2) Build SSR bundle

```
npm run build:ssr
```

Optional: rebuild SSR bundle on changes:

```
npm run dev:ssr
```

### 3) Start the SSR server

```
npm run serve:ssr
```

### 4) Run the Go server (from repo root)

```
go run ./examples/basic-app-ssr
```

Then open http://localhost:8383

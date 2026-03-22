# basic-app-react-ssr

A minimal Fiber + goinertia SSR example with a React client.

## Run

### 1) Install frontend deps

```bash
cd examples/basic-app-react-ssr
npm install
```

### 2) Build the client and SSR bundles

```bash
npm run build
npm run build:ssr
```

Optional: rebuild the SSR bundle on changes:

```bash
npm run dev:ssr
```

### 3) Start the SSR server

```bash
npm run serve:ssr
```

### 4) Run the Go server from repo root

```bash
go run ./examples/basic-app-react-ssr
```

Then open http://localhost:8383

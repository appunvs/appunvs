# appunvs browser

SvelteKit SPA client for [appunvs](../README.md). Runs entirely in the browser
and speaks the same protojson wire format as the other clients, so it can act
as a **provider**, a **connector**, or **both** simultaneously.

Local data lives in [wa-sqlite](https://github.com/rhashimoto/wa-sqlite)
(SQLite compiled to WebAssembly, persisted through an IndexedDB VFS). The relay
connection is a WebSocket at `GET <relay>/ws?token=…&last_seq=…`.

## Prerequisites

- Node.js 20+
- A running appunvs relay reachable from the browser (defaults to
  `http://localhost:8080`).

## Quick start

```sh
npm install
npm run dev
```

Open the printed URL. On first boot the app will:
1. Generate a device ID and persist it in `localStorage`.
2. `POST <relay>/auth/register` to obtain a JWT and `user_id`.
3. Initialize the wa-sqlite database (stored in IndexedDB).
4. Open the WebSocket and begin syncing.

## Environment

- `VITE_RELAY_BASE` — base URL of the relay, e.g. `http://localhost:8080` or
  `https://relay.example.com`. Scheme is auto-converted to `ws://` / `wss://`
  for the WebSocket endpoint. Defaults to `http://localhost:8080`.

Set it when developing or building:

```sh
VITE_RELAY_BASE=https://relay.example.com npm run build
```

## Scripts

- `npm run dev` — Vite dev server with HMR.
- `npm run build` — static SPA build into `build/`. Uses
  `@sveltejs/adapter-static` with `fallback: index.html` so any host that can
  serve static files + a 200-OK fallback will work.
- `npm run preview` — preview the built site.
- `npm run check` — TypeScript + Svelte type checking via `svelte-check`.

## wa-sqlite loading note

wa-sqlite ships a `.wasm` binary alongside its async ESM wrapper. To keep the
main bundle small we:

- exclude it from Vite's dep optimizer (`optimizeDeps.exclude: ['wa-sqlite']`
  in `vite.config.ts`), and
- load it via **dynamic `import()`** in `src/lib/db/sqlite.ts`, only after the
  UI has mounted.

The IDB-backed VFS (`IDBBatchAtomicVFS`) is also dynamically imported. In
production you can either bundle the `.wasm` asset with Vite or fetch it from
a CDN — wa-sqlite's loader picks it up from the same URL as the ESM wrapper.
If you deploy to a restrictive CSP, be sure to allow `wasm-unsafe-eval` and
the origin hosting the wasm.

## File layout

```
src/
  lib/
    pb/wire.ts        # hand-written protojson mirror of appunvs.proto
    db/sqlite.ts      # wa-sqlite wrapper (dynamic import, serialized stmts)
    db/records.ts     # records table CRUD + pub/sub
    relay/client.ts   # WebSocket client with exp-backoff reconnect
    sync/engine.ts    # provider/connector role routing + seq-gap handling
    stores.ts         # svelte writables for UI state
    config.ts         # relay base URL
  routes/
    +layout.ts        # ssr=false, csr=true, prerender=false
    +layout.svelte    # bootstrap sequence
    +page.svelte      # UI
```

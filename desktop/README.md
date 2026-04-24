# appunvs desktop

Tauri 2 client for appunvs. Acts as both provider and connector: WebSocket and
SQLite live in Rust; SvelteKit talks to them only through Tauri `invoke` +
event listeners.

## Prerequisites

- Rust 1.77+ (`rustup`)
- Node.js 20+
- Tauri 2 system dependencies for your OS — see
  https://v2.tauri.app/start/prerequisites/
  - Linux: `webkit2gtk-4.1`, `libsoup-3.0`, `libayatana-appindicator3`,
    `build-essential`, `pkg-config`
  - macOS: Xcode command line tools
  - Windows: WebView2 runtime + MSVC build tools

## First-time setup

```bash
cd desktop
npm install
# Icons are not committed — generate them once from a PNG source:
npx @tauri-apps/cli icon path/to/source.png
```

## Running

```bash
# Dev (SvelteKit + Tauri window with hot reload)
npm run tauri dev

# Production bundle
npm run tauri build
```

Set the relay URL via env var (defaults to `http://localhost:8080`):

```bash
APPUNVS_RELAY_BASE=http://localhost:8080 npm run tauri dev
```

## Architecture

```
src/            SvelteKit front-end (strict TS, CSR only)
  lib/tauri.ts    typed invoke/listen wrappers
  lib/stores.ts   connState / role / records / lastSeq writables
  lib/types.ts    Record / SyncStatus shapes
src-tauri/      Rust back-end
  wire.rs         protojson-mirroring Message + enums
  db/             rusqlite records table + apply_broadcast
  relay/          WebSocket actor + /auth/register client
  sync/           role-aware engine (seq checks, rebroadcast)
  commands/       #[tauri::command] entry points
```

Data persists in the platform app-data directory:

- Linux: `~/.local/share/com.appunvs.desktop/appunvs.db`
- macOS: `~/Library/Application Support/com.appunvs.desktop/appunvs.db`
- Windows: `%APPDATA%\com.appunvs.desktop\appunvs.db`

The device registers with the relay on first launch; the returned JWT +
`device_id` are cached in `config.json` next to the database.

## Config + logging

- `APPUNVS_RELAY_BASE` — relay HTTP base URL (e.g. `http://localhost:8080`)
- `APPUNVS_LOG` — `tracing-subscriber` env filter
  (e.g. `APPUNVS_LOG=appunvs_desktop_lib=debug,info`)

## Testing

No tests yet; run `cargo check` inside `src-tauri/` for a fast type check.

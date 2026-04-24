# desktop — Tauri 2 shell

This directory is the macOS / Windows / Linux native shell for appunvs. It
wraps the Expo Web export from `app/` and exposes a small Rust core for
local concerns the WebView cannot reach (secure storage of session tokens,
keychain integration, deep-link / QR scanner intents).

The frontend itself lives in `../app` and is built once for all three
surfaces (browser, desktop, mobile). The Tauri config in
`src-tauri/tauri.conf.json` points `frontendDist` at `../../app/dist`,
which is what `npm --prefix ../app run web:export` produces.

## Dev loop

```bash
# In one terminal
npm --prefix ../app run web:dev      # Expo web on :8081

# In another
cargo tauri dev                      # Tauri shell pointed at :8081
```

## Production build

```bash
npm --prefix ../app run web:export   # ../../app/dist
cargo tauri build                    # bundles for the host OS
```

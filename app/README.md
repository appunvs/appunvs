# app — Expo monorepo for browser, desktop, mobile

Single React Native + Expo project that builds three surfaces from one
codebase:

| Surface  | Build                                                  |
| -------- | ------------------------------------------------------ |
| Mobile   | `npx expo run:ios` / `npx expo run:android` (dev client) |
| Browser  | `npm run web:dev` (dev) / `npm run web:export` (static) |
| Desktop  | `cargo tauri dev` from `../desktop/src-tauri/` (wraps Expo Web) |

## Layout

```
app/
├── app.json            # Expo manifest
├── package.json
├── babel.config.js
├── metro.config.js
├── tsconfig.json
├── app/                # expo-router file-based routing
│   ├── _layout.tsx
│   ├── index.tsx       # redirect → /(tabs)/chat
│   ├── (tabs)/
│   │   ├── _layout.tsx # responsive: side bar on >=720dp web/desktop, bottom on mobile
│   │   ├── chat.tsx    # AI chat input + transcript
│   │   ├── stage.tsx   # isolated runtime that loads a Box's bundle
│   │   └── profile.tsx # account + box list (create / publish / pair)
│   └── pair/[code].tsx # deep-link landing for QR claim
└── src/
    ├── lib/
    │   ├── api.ts      # relay HTTP client + RELAY_URL resolver
    │   ├── auth.ts     # SecureStore (native) / localStorage (web) token store
    │   ├── box.ts      # /box + /pair typed client
    │   └── ai.ts       # /ai/turn streaming client (currently a stub)
    ├── proto/chat.ts   # hand-mirrored ChatTurnEvent
    ├── state/
    │   ├── chat.ts     # zustand store for the transcript
    │   └── box.ts      # active-box selection
    └── stage/
        ├── runtime.types.ts  # contract (StageRuntimeProps)
        ├── runtime.ts        # TypeScript-only re-export
        ├── runtime.native.tsx# WebView fallback (until isolated Hermes lands)
        └── runtime.web.tsx   # iframe sandbox="allow-scripts"
```

## Stage runtime contract

> Loading a bundle into Stage MUST NOT be able to reach the host app's
> state, tokens, MMKV stores, or file system.

- **Web**: `<iframe sandbox="allow-scripts">`, no `allow-same-origin`.
- **Native**: today, `react-native-webview` with cookies / DOM storage
  disabled.  The next slice replaces this with a dedicated isolated Hermes
  runtime (a custom native module — same surface, true RN render).

Either way, `runtime.types.ts` is the immovable contract; both
implementations conform to `StageRuntimeProps`.

## Configuring the relay URL

Set `EXPO_PUBLIC_RELAY_URL` in the environment used by `expo start`, or
add `expo.extra.relayUrl` in `app.json`.  Defaults to
`http://localhost:8080`.

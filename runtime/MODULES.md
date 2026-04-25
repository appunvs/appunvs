# SubRuntime Module Allowlist

The list of native modules the host links statically into the binary so
AI-generated RN bundles can `import` them inside a SubRuntime. This is
the **execution contract** for AI bundles: anything outside this list
fails the sandbox build (Metro `externals`) and cannot be reached at
runtime.

The list is **conservative**: every entry expands the binary, lengthens
SubRuntime spawn time, and adds an attack surface to audit. We add to
it deliberately, not by default.

## Tier 1 — required for any meaningful UI

These ship in v0 of the SubRuntime work. AI bundles can rely on them
landing day-one.

| Module | Purpose |
| --- | --- |
| `react` | hooks, context, components |
| `react-native` | View / Text / Image / ScrollView / Pressable / TextInput / FlatList / SectionList / KeyboardAvoidingView |
| `react-native-reanimated` | UI-thread animation worklets |
| `react-native-gesture-handler` | swipe, pan, long-press, double-tap |
| `react-native-screens` | native screen container (perf for nested navigators) |
| `react-native-safe-area-context` | safe area insets on iPhone notch / Android display cutout |
| `react-native-svg` | inline vector graphics |
| `react-native-mmkv` | namespace-isolated key-value store |

8 modules — covers ~90% of "make a useful screen" cases.

## Tier 2 — common but not in v0

Land when the v0 surface stabilizes; AI bundles can opt-in once the
host binary version supports them. Tier-2 imports must be guarded by
runtime version checks until widely deployed.

| Module | Purpose |
| --- | --- |
| `expo-image` | better image rendering, perceptual quality, gif/webp |
| `expo-blur` | iOS-style frosted blur |
| `expo-haptics` | tap / impact / notification feedback |
| `expo-linking` | deep links (within whitelist) |
| `react-native-webview` | embed web content (carefully — sub-sub-iframe risk) |

5 modules — covers the polish layer.

## Tier 3 — capability-gated

These need a permission prompt at first use. AI bundle imports them like
any other module, but native side intercepts and prompts the user.

| Module | Purpose | Permission |
| --- | --- | --- |
| `expo-camera` | live camera frames + capture | Camera |
| `react-native-vision-camera` | high-perf camera, ML preview | Camera |
| (future) | Microphone, Photos, Contacts, Location | per-permission |

## Forbidden — never expose

These break the sandbox model and have no curated alternative inside the
SubRuntime. AI bundle that imports any of these fails the sandbox build.

| Module | Reason |
| --- | --- |
| `expo-secure-store` / `react-native-keychain` | host's auth tokens live here; sub-runtime exposure = token exfiltration vector |
| `react-native-fs` (raw) | filesystem root access; sub-runtime gets `SubRuntimeStorage` (namespace-scoped) instead |
| `expo-file-system` (raw) | same |
| `expo-updates` | sub-runtime cannot self-update; that's the host's job |
| `expo-notifications` | only host should schedule notifications |
| `react-native-keychain` | duplicate of secure-store concern |
| Direct fetch to arbitrary hosts | sub-runtime gets `SubRuntimeNetwork` that's pinned to relay endpoints |

## Bridge surfaces (custom replacements for forbidden APIs)

These are appunvs-native modules the host injects into every SubRuntime.
They give AI bundles equivalent capability without the security blast
radius of the upstream module.

| Module | Replaces | Behaviour |
| --- | --- | --- |
| `@appunvs/sub-storage` | MMKV / SecureStore / FileSystem | namespace-scoped k/v + small blob; no escape |
| `@appunvs/sub-network` | fetch | only `https://relay.example/box/*` and `/ws`; namespace_token auto-injected |
| `@appunvs/sub-publish` | (none) | one-shot helper that calls back into the host's `publish_box` flow |

## Versioning

This file plus `version.json` are the **runtime ABI**. When a module is
added (e.g., `expo-image` lands in v0.4), bump `runtime.sdk_version` in
`version.json` and stamp every AI bundle with the runtime version it
was built against. The host binary's loader rejects any AI bundle whose
`min_runtime` is higher than the host's bundled SDK version.

## Discussion log

- 2026-04-25 — initial allowlist drafted; 8/5/2 split chosen to balance
  AI capability vs binary size + audit cost.

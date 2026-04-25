# runtime

The appunvs **native shell** — two platform-native iOS + Android apps that
host the user-facing UI and embed a sandboxed Hermes runtime to load
AI-generated bundles.

This directory is the **host application** itself, not a separate library.
All UI (Chat / Stage / Profile) lives in native code: SwiftUI on iOS,
Jetpack Compose on Android. The previous Expo + react-native-web
prototype (`app/`) is gone — each platform has its own native UI codebase
with its own design system implementation against shared tokens.

## Layout

```
runtime/
├── README.md            ← this file
├── MODULES.md           ← native module allowlist for AI bundles
├── version.json         ← runtime SDK version
├── ios/                 ← Swift + SwiftUI Xcode project
│   ├── project.yml      ← XcodeGen spec; xcodeproj is generated
│   ├── Runtime/         ← Swift source tree
│   └── README.md
└── android/             ← Kotlin + Compose Gradle project
    ├── settings.gradle.kts
    ├── build.gradle.kts
    ├── app/             ← module
    └── README.md
```

When the **SubRuntime** native module lands (PR D), each platform gets a
`modules/SubRuntime/` subdirectory containing the native bridge that
spawns a second `jsi::Runtime` (Hermes) per Stage. The host UI itself
stays pure native; only the Stage tab content runs JS.

## Why native, not RN

- Best perf, no RN bridge tax for the host UI
- Native look-and-feel (SF Symbols, Dynamic Type, system gestures, system fonts)
- Smaller binary, native debugging, native crash reporting
- The RN runtime exists only inside SubRuntimes for AI bundles, not for the
  host

The cost is two UI codebases instead of one — accepted.

## Phases (see PR sequence in main README)

| # | Phase | Status |
| --- | --- | --- |
| 0 | Scaffold (this PR) — empty SwiftUI/Compose 3-tab apps, design tokens, CI hookup | ✅ |
| 1 | Port UI: BoxSwitcher / Bubble / ToolCall / EmptyState / QuotaBar plus Chat / Stage / Profile screens, both platforms | 🔜 |
| 2 | Network + state: HTTP client to relay, AsyncStorage equivalent, Box list cache | 🔜 |
| 3 | SubRuntime native module: spawn/load/destroy second Hermes runtime, Fabric surface mount | 🔜 |
| 4 | HostBridge: white-listed natives exposed to AI bundles | 🔜 |

## Local development

### iOS

Prerequisites: macOS 14+, Xcode 16+, [XcodeGen](https://github.com/yonaskolb/XcodeGen)
(`brew install xcodegen`).

```bash
cd runtime/ios
xcodegen generate                         # produces Runtime.xcodeproj
open Runtime.xcodeproj                     # or: xcodebuild -scheme Runtime
```

The Xcode project file is **not** committed; regenerate from
`project.yml` whenever scheme / settings change.

### Android

Prerequisites: JDK 17+, Android Studio Hedgehog or later, Android SDK 35.

```bash
cd runtime/android
gradle wrapper --gradle-version 8.10       # one-time, on a machine with gradle installed
./gradlew assembleDebug                    # or open in Android Studio
```

Gradle wrapper jar is **not** committed; the bootstrap above is a one-time
ritual per clone (no need to re-run unless gradle version changes).

## Build outputs

| Output | Produced by | Consumed by |
| --- | --- | --- |
| iOS app `.ipa` | `xcodebuild -archivePath ...` then `xcodebuild -exportArchive` | App Store / TestFlight |
| Android app `.aab` | `./gradlew bundleRelease` | Google Play |
| (later) `RuntimeSDK.xcframework` | `packaging/build-ios.sh` | Third-party hosts that want to embed our runtime |
| (later) `runtime.aar` | `packaging/build-android.sh` | Third-party hosts on Android |

The third-party SDK outputs are deferred until the host product is
shipping — single producer (us), single consumer (us), no need for the
ABI compatibility ceremony yet.

## License

MIT (matches main repo). Runtime stays in the monorepo until it has
external consumers; at that point we'll split to `appunvs/runtime` and
re-license the SDK side independently.

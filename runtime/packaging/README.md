# Runtime SDK packaging

This directory holds the build pipeline that turns `runtime/` into the
two artifacts the host shell consumes:

| Artifact | Built by | Linked into |
| --- | --- | --- |
| `RuntimeSDK.xcframework` | `build-ios.sh` | `appunvs/ios/` (host iOS app) |
| `runtime.aar` | `build-android.sh` | `appunvs/android/` (host Android app) |

## Status

Both scripts are **stubs** today (PR D2-prep).  PR D2 lands the real
build steps once the SubRuntime native module exists (the C++/JSI layer
that spawns a Hermes context per Box).

## Inputs

- `runtime/ios/` — RN init's iOS project; we extend it with the
  SubRuntime ObjC++ module and build it as a static framework
- `runtime/android/` — RN init's Android Gradle project; we extend it
  with the SubRuntime JNI module and assemble it as a library AAR
- `runtime/src/` — TypeScript host bridge declarations and the dev
  harness; the SDK ships no JS itself (the host loads its own bundle, AI
  bundles load theirs)
- `runtime/MODULES.md` — the Tier 1 allowlist of native modules to
  bundle into the SDK

## Outputs

`build/ios/RuntimeSDK.xcframework` and `build/android/runtime.aar`.
Both gitignored — these are produced on demand by the shipping pipeline.

## Versioning

The SDK version lives in `runtime/version.json` (`runtime.sdk_version`).
Bump it when the curated module list, the host bridge surface
(`src/HostBridge.ts`), or the SubRuntime ABI changes.  Host shells pin
to a specific SDK version range; AI bundles record which SDK they were
built against so the host can refuse to load incompatible bundles.

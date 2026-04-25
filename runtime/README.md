# runtime — appunvs runtime SDK

This is the **JS/Hermes side** of appunvs: a React Native 0.85 project
that builds into a single artifact per platform
(`RuntimeSDK.xcframework` and `runtime.aar`) which the host shell links.
**The shipped app under `appunvs/` does not contain RN by itself** —
it links this SDK and uses it to mount AI-generated bundles inside the
Stage tab.

A separate, smaller pipeline in `sandbox/` is what the **relay** uses
to compile AI-generated source into a bundle the SDK can load.

## Layout

```
runtime/
├── ios/                  ← RN init's iOS subdir; produces the framework
├── android/              ← RN init's Android subdir; produces the AAR
├── package.json          ← RN 0.85 + curated Tier 1 modules
├── babel.config.js       ← stock RN preset
├── metro.config.js       ← stock metro (dev / harness only)
├── app.json
├── tsconfig.json
├── index.js              ← RN entry; loads src/
│
├── src/
│   ├── index.tsx         ← host JS entry — placeholder UI shown when no
│   │                       AI bundle is loaded
│   ├── HostBridge.ts     ← TypeScript declarations for the JSI surface
│   │                       the host injects into every SubRuntime
│   └── TestHarness.tsx   ← dev-only screen for ad-hoc bundle loading
│
├── packaging/            ← SDK build pipeline
│   ├── build-ios.sh      → RuntimeSDK.xcframework
│   ├── build-android.sh  → runtime.aar
│   └── README.md
│
├── sandbox/              ← relay-side bundle build pipeline
│   ├── Dockerfile        ← image with allowlisted deps installed
│   ├── metro.config.js   ← rejects imports outside the allowlist
│   └── build-bundle.sh   ← AI source in → index.bundle out
│
├── version.json          ← runtime SDK version + ABI
├── MODULES.md            ← Tier 1 / 2 / 3 module allowlist
├── ARCHITECTURE.md       ← three-paths comparison + outputs ↔ consumers
├── README.md             ← this file
└── LICENSE
```

## How the three pieces relate

```
┌─ runtime/            (this directory)
│  │
│  ├─ packaging/  ─→  RuntimeSDK.xcframework + runtime.aar
│  │                        │
│  │                        └─→ linked by appunvs/{ios,android}
│  │                              │
│  │                              └─→ host's Stage tab mounts a
│  │                                    SubRuntime view backed by
│  │                                    one Hermes per Box
│  │
│  └─ sandbox/    ─→  docker image
│                          │
│                          └─→ used by relay/internal/sandbox
│                                to compile AI source → index.bundle
│                                  │
│                                  └─→ served via /_artifacts
│                                       and pulled by SubRuntime
│                                       at Stage-tab load time
```

## Local development

### Run the runtime by itself (dev harness)

```bash
cd runtime
npm install                              # one-time
npm run ios       # or: npm run android
```

You'll see `TestHarness.tsx` — a dev screen with a textbox to paste a
bundle URL.  PR D2 wires the actual SubRuntime mount; today the harness
is UI-only.

### Build the SDK artifact (PR D2)

```bash
./packaging/build-ios.sh        → build/ios/RuntimeSDK.xcframework
./packaging/build-android.sh    → build/android/runtime.aar
```

Both stub today.

### Build the sandbox image (PR D2)

```bash
docker build -t appunvs/sandbox sandbox/
docker run --rm -v "$(pwd)/example-ai-source:/work" appunvs/sandbox
# → /work/index.bundle
```

## Versioning

`version.json`'s `runtime` is the SDK release version.  `sdk_version` is
the integer ABI version — bump it when:

- a curated module is added or removed (`MODULES.md` change)
- the host bridge surface (`src/HostBridge.ts`) gains, loses, or
  reshapes a method
- the SubRuntime's bundle-load contract changes

The host shell pins to a specific `sdk_version` range; AI bundles record
which `sdk_version` they were built against so the host can reject
incompatible bundles.

## Why a monorepo

The runtime SDK + the host shells + the relay all change together at
this stage.  When the SDK gets external consumers we'll split it to
`appunvs/runtime` as its own repo with semver releases.

## License

MIT (matches main repo).

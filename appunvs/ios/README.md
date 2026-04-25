# appunvs/ios

Native iOS host shell — Swift + SwiftUI.  This is the binary that ships
to the App Store; it links the runtime SDK (`runtime/` → `RuntimeSDK.xcframework`)
to gain JS/Hermes capability inside the Stage tab, but the host UI itself
(Chat, Profile) is pure SwiftUI with zero JS.

The Xcode project is generated from `project.yml` via [XcodeGen](https://github.com/yonaskolb/XcodeGen);
the `.xcodeproj` itself is **not committed** so every checkout regenerates
from the canonical YAML.

## Bootstrap

```bash
brew install xcodegen                     # one-time, per dev machine
cd appunvs/ios
xcodegen generate                         # → produces Runtime.xcodeproj
open Runtime.xcodeproj                    # or: xcodebuild -scheme Runtime
```

CI does the same — `.github/workflows/native.yml` runs `xcodegen generate`
then `xcodebuild build` on a `macos-15` runner.

## Layout

```
ios/
├── project.yml           ← XcodeGen spec; edit this, never the .xcodeproj
├── Runtime/              ← Swift source tree
│   ├── RuntimeApp.swift  ← @main entry; auth gate → TabView with 3 tabs
│   ├── Net/              ← URLSession-backed clients for the relay's REST
│   │                       and SSE endpoints
│   ├── State/            ← AuthStore, BoxStore, ChatStore, AppState
│   ├── Theme/Theme.swift ← color/spacing/radius tokens (light + dark)
│   ├── UI/               ← reusable components (Bubble, BoxSwitcher, ...)
│   ├── Screens/          ← ChatView, StageView, ProfileView, LoginView
│   └── Assets.xcassets/  ← AppIcon + AccentColor
├── .gitignore
└── README.md             ← this file
```

## Source layout philosophy

- `Theme/` is the design-system layer; token primitives only.
- `UI/` holds reusable components (Bubble, Card, Badge, BoxSwitcher).
- `Screens/` is one file per top-level destination; sub-views live
  alongside unless they grow too big to inline.
- `State/` holds host-wide observable state.
- `Net/` holds the typed wrappers around the relay's REST + SSE surface.
- The future `SubRuntime/` directory (PR D2) houses the ObjC++ wrapper
  around the runtime SDK that mounts a Hermes-backed view inside the
  Stage tab.

## Running

Local: `xcodebuild -scheme Runtime -destination 'platform=iOS Simulator,name=iPhone 16'` or open in Xcode and ⌘R.

CI: see `.github/workflows/native.yml`.

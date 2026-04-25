# runtime/ios

Native iOS host shell — Swift + SwiftUI. The Xcode project is generated
from `project.yml` via [XcodeGen](https://github.com/yonaskolb/XcodeGen);
the `.xcodeproj` itself is **not committed** so every checkout regenerates
from the canonical YAML.

## Bootstrap

```bash
brew install xcodegen                     # one-time, per dev machine
cd runtime/ios
xcodegen generate                         # → produces Runtime.xcodeproj
open Runtime.xcodeproj                    # or: xcodebuild -scheme Runtime
```

CI does the same — `.github/workflows/native.yml` runs `xcodegen generate`
then `xcodebuild build` on a `macos-latest` runner.

## Layout

```
ios/
├── project.yml           ← XcodeGen spec; edit this, never the .xcodeproj
├── Runtime/              ← Swift source tree
│   ├── RuntimeApp.swift  ← @main entry; TabView with 3 tabs
│   ├── Theme/Theme.swift ← color/spacing/radius tokens (light + dark)
│   ├── State/            ← observable host state (theme override etc.)
│   ├── Screens/          ← ChatView, StageView, ProfileView placeholders
│   └── Assets.xcassets/  ← AppIcon + AccentColor
├── .gitignore
└── README.md             ← this file
```

## Source layout philosophy

- `Theme/` is the design-system layer; only token primitives live here.
  Reusable components (Button styles, Card, Badge, ToolCall, Bubble,
  …) land in `UI/` once PR C ports them from the prior RN tokens.
- `Screens/` is one file per top-level tab; sub-views live alongside
  unless they grow too big to inline.
- `State/` holds host-wide observable state — auth tokens, active Box
  reference, theme override.  Per-screen state stays inside the screen.
- The future `SubRuntime/` directory (PR D) houses the native module
  that spawns and renders sandboxed Hermes runtimes for AI bundles.

## Phasing

Today the screens are **placeholders** — `Image` + `Text` only. Each
will get a real implementation in PR C as the corresponding RN
component is ported. Specifically:

| iOS file | Source RN file (deleted with `app/`) |
| --- | --- |
| Theme/Theme.swift | app/src/theme/colors.ts + spacing.ts + radius.ts |
| Screens/ChatView.swift | app/src/screens/ChatPanel.tsx + ai.ts |
| Screens/StageView.swift | app/src/screens/StagePanel.tsx + stage/runtime.* |
| Screens/ProfileView.swift | app/app/(tabs)/profile.tsx |
| (PR C) UI/Bubble.swift | app/src/components/Bubble.tsx |
| (PR C) UI/ToolCall.swift | app/src/components/ToolCall.tsx |
| (PR C) UI/BoxSwitcher.swift | app/src/components/BoxSwitcher.tsx |
| (PR C) UI/QuotaBar.swift | app/src/components/QuotaBar.tsx |

## Running

Local: `xcodebuild -scheme Runtime -destination 'platform=iOS Simulator,name=iPhone 16'` or open in Xcode and ⌘R.

CI: see `.github/workflows/native.yml`.

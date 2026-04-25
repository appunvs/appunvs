# runtime/android

Native Android host shell — Kotlin + Jetpack Compose. Single-Activity
architecture; the Activity hosts the entire Compose tree with three
top-level tabs (Chat / Stage / Profile).

## Bootstrap

Prerequisites: JDK 17+, Android SDK 35, Android Studio Hedgehog or later.

```bash
cd runtime/android
gradle wrapper --gradle-version 8.10       # one-time, on a host with Gradle installed
./gradlew assembleDebug                    # or open in Android Studio
```

The Gradle wrapper jar (`gradle/wrapper/gradle-wrapper.jar`) is
**not committed** — every clone runs `gradle wrapper` once to
materialize it. This avoids checking a binary into git that's
trivially regenerable.

CI does the same in `.github/workflows/native.yml` via
`gradle/actions/setup-gradle@v4`, which installs Gradle and runs
`assembleDebug` directly without going through the wrapper.

## Layout

```
android/
├── settings.gradle.kts            ← project + included modules
├── build.gradle.kts               ← root, plugin versions only
├── gradle.properties              ← JVM args, AndroidX, Kotlin style
├── gradle/libs.versions.toml      ← centralized dependency versions
├── app/                           ← single host module
│   ├── build.gradle.kts           ← Android plugin + Compose
│   ├── proguard-rules.pro
│   └── src/main/
│       ├── AndroidManifest.xml
│       ├── java/com/appunvs/runtime/
│       │   ├── MainActivity.kt    ← @Composable entry; TabView
│       │   ├── theme/             ← Color.kt + Theme.kt (light + dark tokens)
│       │   ├── state/AppState.kt  ← ViewModel; theme override + DataStore
│       │   └── screens/           ← ChatScreen / StageScreen / ProfileScreen placeholders
│       └── res/
│           ├── drawable/ic_launcher.xml      ← vector launcher (placeholder)
│           ├── values/strings.xml            ← app name
│           ├── values/themes.xml             ← Activity theme
│           └── xml/                          ← backup + extraction rules
├── .gitignore
└── README.md                                  ← this file
```

## Phasing

Today the screens are **placeholders**. PR C ports the design-system
components (Bubble / ToolCall / EmptyState / QuotaBar / BoxSwitcher)
from the prior RN tokens into reusable `@Composable`s under a future
`ui/` package, then wires them into the screens.

| Kotlin file | Source RN file (deleted with `app/`) |
| --- | --- |
| `theme/Color.kt` | `app/src/theme/colors.ts` |
| `theme/Theme.kt` | `app/src/theme/{spacing,radius,ThemeProvider}.ts` |
| `state/AppState.kt` | `app/src/theme/store.ts` |
| `screens/ChatScreen.kt` | `app/src/screens/ChatPanel.tsx` |
| `screens/StageScreen.kt` | `app/src/screens/StagePanel.tsx` |
| `screens/ProfileScreen.kt` | `app/app/(tabs)/profile.tsx` |

PR D adds `runtime/android/app/src/main/java/com/appunvs/runtime/subruntime/`
with the JNI native module that spawns a sandboxed Hermes runtime.

## Running

Local: `./gradlew installDebug` deploys to a connected device or
emulator; `./gradlew :app:assembleDebug` produces an APK at
`app/build/outputs/apk/debug/app-debug.apk`.

Compose previews: open any screen in Android Studio and use the
preview pane (the placeholder screens don't ship with `@Preview`
blocks yet — added in PR C alongside the real components).

CI: see `.github/workflows/native.yml` — runs on Ubuntu with Java 17
and Android SDK 35.

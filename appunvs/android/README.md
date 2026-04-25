# appunvs/android

Native Android host shell — Kotlin + Jetpack Compose.  This is the
binary that ships to Google Play; it links the runtime SDK
(`runtime/` → `runtime.aar`) to gain JS/Hermes capability inside the
Stage tab, but the host UI itself (Chat, Profile) is pure Compose with
zero JS.

Single-Activity architecture; the Activity hosts the entire Compose tree
with three top-level tabs (Chat / Stage / Profile), gated by an
auth screen.

## Bootstrap

Prerequisites: JDK 17+, Android SDK 35, Android Studio Hedgehog or later.

```bash
cd appunvs/android
gradle wrapper --gradle-version 8.10       # one-time, on a host with Gradle installed
./gradlew assembleDebug                    # or open in Android Studio
```

The Gradle wrapper jar (`gradle/wrapper/gradle-wrapper.jar`) is
**not committed** — every clone runs `gradle wrapper` once to
materialize it.

CI uses `gradle/actions/setup-gradle@v4` and runs `assembleDebug`
without the wrapper.

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
│       │   ├── MainActivity.kt    ← @Composable entry; auth gate + TabView
│       │   ├── net/               ← Retrofit + OkHttp clients for the relay
│       │   ├── state/             ← AuthRepo, BoxRepo, ChatViewModel, AppState
│       │   ├── theme/             ← Color.kt + Theme.kt (light + dark tokens)
│       │   ├── ui/                ← reusable composables (Bubble, BoxSwitcher, ...)
│       │   └── screens/           ← ChatScreen / StageScreen / ProfileScreen / LoginScreen
│       └── res/
│           ├── drawable/ic_launcher.xml      ← vector launcher (placeholder)
│           ├── values/strings.xml            ← app name
│           ├── values/themes.xml             ← Activity theme
│           └── xml/                          ← backup + extraction + network-security rules
├── .gitignore
└── README.md                                 ← this file
```

## Source layout philosophy

- `theme/` is the design-system layer; token primitives only.
- `ui/` holds reusable composables (Bubble, AppBadge, BoxSwitcher).
- `screens/` is one file per top-level destination.
- `state/` holds host-wide observable state (ViewModels).
- `net/` holds the Retrofit interface, OkHttp client, and SSE consumer
  for the relay.
- The future `subruntime/` package (PR D2) houses the JNI bridge to the
  runtime SDK that mounts a Hermes-backed view inside the Stage tab.

## Running

Local: `./gradlew installDebug` deploys to a connected device or
emulator; `./gradlew :app:assembleDebug` produces an APK at
`app/build/outputs/apk/debug/app-debug.apk`.

CI: see `.github/workflows/native.yml` — runs on Ubuntu with Java 17
and Android SDK 35.

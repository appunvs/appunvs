// Runtime SDK Android library module.  Lives inside the RN init's
// gradle project so it can pick up react-android + hermes-android.
//
// D3.b: declares the React Native + Hermes deps that the SDK needs
// to compile + link against.  RuntimeView impl in this PR is still
// the placeholder from D2.c — D3.c starts using these deps.
plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.appunvs.runtimesdk"
    // 35 matches the host shell (appunvs/android/app/build.gradle.kts).
    // The RN init's root buildscript wants 36 for its own dev-harness
    // app, but library modules can pin lower without a problem — and
    // 35 saves us from installing two SDK platforms in CI.
    compileSdk = 35

    defaultConfig {
        minSdk = 24
        consumerProguardFiles("consumer-rules.pro")
        // AndroidJUnitRunner — the standard runner for instrumented
        // tests (androidx.test).  Required for ActivityScenario +
        // UI Automator to work in the test APK.
        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(getDefaultProguardFile("proguard-android-optimize.txt"))
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }
}

// Module-level repositories — required because RN init's settings.gradle
// doesn't centralize repos via dependencyResolutionManagement.
repositories {
    google()
    mavenCentral()
}

dependencies {
    // React Native runtime — gives ReactHost, JSI, the Fabric C++
    // surface.  Published to mavenCentral as of RN 0.71+.  Pinned to
    // RN 0.85.2 to match runtime/package.json.
    implementation("com.facebook.react:react-android:0.85.2")

    // Hermes engine — bundles libhermes.so per ABI (arm64-v8a /
    // armeabi-v7a / x86_64).  We run AI bundles inside this engine.
    implementation("com.facebook.react:hermes-android:0.85.2")

    // Tier 1 native modules — see runtime/MODULES.md.  Each ships its
    // own android library that the RN gradle plugin's autolinking
    // (autolinkLibrariesFromCommand in runtime/android/settings.gradle)
    // already includes as a `:react-native-X` project module.  We add
    // them as implementation deps here so they're packaged inside
    // runtime.aar and registered with the SDK's React runtime when the
    // host app launches.
    implementation(project(":react-native-reanimated"))
    // react-native-worklets is reanimated 4's mandatory peer (split out
    // of the reanimated package itself); the gradle plugin autolinks it
    // as :react-native-worklets when reanimated 4 is in dependencies.
    implementation(project(":react-native-worklets"))
    implementation(project(":react-native-gesture-handler"))
    implementation(project(":react-native-screens"))
    implementation(project(":react-native-safe-area-context"))
    implementation(project(":react-native-svg"))
    implementation(project(":react-native-mmkv"))

    // --- D3.c.4 instrumented UI test deps ---
    //
    // androidx.test.ext:junit + AndroidJUnit4 runner so we can
    // @RunWith(AndroidJUnit4::class) and emit JUnit XML for CI.
    androidTestImplementation("androidx.test.ext:junit:1.2.1")
    // ActivityScenario for spinning up FixtureActivity from the test.
    androidTestImplementation("androidx.test:core:1.6.1")
    androidTestImplementation("androidx.test:runner:1.6.2")
    androidTestImplementation("androidx.test:rules:1.6.1")
    // UI Automator drives the system-wide accessibility tree — finds
    // the rendered RN <Text> by its visible text.  Cheaper to use than
    // Espresso here since RN's view hierarchy uses non-standard
    // ReactTextView types Espresso's withText matcher trips over.
    androidTestImplementation("androidx.test.uiautomator:uiautomator:2.3.0")
}

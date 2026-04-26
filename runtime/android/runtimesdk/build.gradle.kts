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
}

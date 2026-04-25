// Runtime SDK Android library module.  Lives inside the RN init's
// gradle project so it can pick up the RN gradle plugin + the
// react-android / hermes-android artifacts when D3.b adds them.
//
// The plugin classpaths come from the root build.gradle's
// `buildscript { dependencies { classpath(...) } }` — no need to
// repeat versions here.
//
// D3.a empty shell: pure kotlin library, no RN deps.  D3.b adds the
// react-android + hermes-android implementation deps.
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

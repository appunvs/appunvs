// Module-level Gradle build for the host shell.
//
// Compose-only UI with a single Activity holding the three top-level
// tabs (Chat / Stage / Profile).  The Stage tab links the runtime SDK
// (built from runtime/ → runtime.aar) to mount AI-generated bundles
// inside a Hermes-backed view; the rest of the host UI is pure Compose.

plugins {
    alias(libs.plugins.android.application)
    alias(libs.plugins.kotlin.android)
    alias(libs.plugins.kotlin.compose)
    alias(libs.plugins.kotlin.serialization)
}

android {
    namespace = "com.appunvs.runtime"
    compileSdk = 35

    defaultConfig {
        applicationId = "com.appunvs.runtime"
        minSdk = 24
        targetSdk = 35
        versionCode = 1
        versionName = "0.0.1"

        testInstrumentationRunner = "androidx.test.runner.AndroidJUnitRunner"
    }

    buildTypes {
        release {
            isMinifyEnabled = false
            proguardFiles(
                getDefaultProguardFile("proguard-android-optimize.txt"),
                "proguard-rules.pro"
            )
        }
    }

    compileOptions {
        sourceCompatibility = JavaVersion.VERSION_17
        targetCompatibility = JavaVersion.VERSION_17
    }

    kotlinOptions {
        jvmTarget = "17"
    }

    buildFeatures {
        compose = true
        // Generate BuildConfig so DEBUG-only entry points (Design tokens
        // preview reachable from Profile) can branch on `BuildConfig.DEBUG`.
        buildConfig = true
    }

    packaging {
        resources {
            excludes += "/META-INF/{AL2.0,LGPL2.1}"
        }
    }
}

dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.androidx.lifecycle.runtime.ktx)
    implementation(libs.androidx.lifecycle.viewmodel.compose)
    implementation(libs.androidx.activity.compose)

    // Compose BOM — pins every compose artifact to the same train.
    implementation(platform(libs.androidx.compose.bom))
    implementation(libs.androidx.ui)
    implementation(libs.androidx.ui.graphics)
    implementation(libs.androidx.ui.tooling.preview)
    implementation(libs.androidx.material3)
    implementation(libs.androidx.material.icons.extended)

    // DataStore for theme override persistence (UserDefaults equivalent).
    implementation(libs.androidx.datastore.preferences)

    // Material XML themes — `Theme.Material3.DayNight.NoActionBar` is
    // resolved here; Compose Material3 (above) provides runtime
    // components, this provides the splash/window theme XML.
    implementation(libs.google.material)

    // Network: Retrofit + OkHttp + Kotlinx serialization.  The kotlinx
    // converter is hand-written (see net/JsonConverterFactory.kt) — the
    // community ports drift between Kotlin/serialization releases and
    // it's a 30-line file we own anyway.  OkHttp is used directly for
    // the /ai/turn SSE stream so we get line-by-line reads without a
    // dedicated SSE library.
    implementation(libs.retrofit)
    implementation(libs.okhttp)
    implementation(libs.okhttp.logging)
    implementation(libs.kotlinx.serialization.json)
    implementation(libs.kotlinx.coroutines.android)

    // EncryptedSharedPreferences for the device token (Keychain equivalent).
    implementation(libs.androidx.security.crypto)

    // Runtime SDK — built by `runtime/packaging/build-android.sh` into
    // runtime/build/android/runtime.aar.  File-path link today (single
    // monorepo, single producer); when the SDK splits to its own repo,
    // this becomes `implementation("io.<brand>:runtime:1.0.0")` from a
    // Maven coordinate — single line change.  CI runs the SDK build
    // BEFORE this gradle invocation.
    //
    // Path: this file lives at appunvs/android/app/build.gradle.kts —
    // three levels deep — so `../../../` is the repo root.
    implementation(files("../../../runtime/build/android/runtime.aar"))

    debugImplementation(libs.androidx.ui.tooling)

    testImplementation(libs.junit)
    androidTestImplementation(libs.androidx.junit)
    androidTestImplementation(platform(libs.androidx.compose.bom))
}

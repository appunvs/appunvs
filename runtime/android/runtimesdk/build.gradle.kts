// Runtime SDK Android library module.  Lives inside the RN init's
// gradle project so it can pick up react-android + hermes-android
// from the React Native gradle plugin's classpath.
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
    // The RN init's root buildscript wants 36 for the dev-harness app
    // module, but library modules can pin lower without a problem —
    // and 35 saves us from installing two SDK platforms in CI.
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

dependencies {
    // React Native runtime + JNI shim — gives us ReactHost, JSI, the
    // Fabric C++ surface, and the Java↔C++ bridge classes.  Published
    // to mavenCentral as of RN 0.71+.  Pinned here to RN 0.85.2 to
    // match runtime/package.json.
    implementation("com.facebook.react:react-android:0.85.2")

    // Hermes engine — bundles libhermes.so per ABI (arm64-v8a /
    // armeabi-v7a / x86_64).  React-android pulls JSI; hermes-android
    // is the JS engine we run AI bundles in.
    implementation("com.facebook.react:hermes-android:0.85.2")
}

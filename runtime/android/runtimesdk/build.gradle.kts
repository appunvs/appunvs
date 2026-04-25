// Runtime SDK Android library module.  Lives inside the RN init's
// gradle project so it can pick up the RN gradle plugin + the
// react-android / hermes-android artifacts when D3.b adds them.
//
// The plugin classpaths come from the root build.gradle's `buildscript {
// dependencies { classpath(...) } }` — no need to repeat versions here.
plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.appunvs.runtimesdk"
    // Match the RN init's compileSdk (36) so AGP doesn't whine about
    // mixed compileSdk between modules.
    compileSdk = 36

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

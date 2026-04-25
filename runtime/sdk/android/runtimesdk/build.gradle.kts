plugins {
    id("com.android.library")
    id("org.jetbrains.kotlin.android")
}

android {
    namespace = "com.appunvs.runtimesdk"
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

    // No Compose, no React Native — D2.a is a pure kotlin shell exposing
    // a single `RuntimeSDK.hello()` static method.  PR D2.c adds the JNI
    // bridge to Hermes + RN's C++ runtime; at that point we link
    // libhermes.so / libreact_render.so etc. via dynamic features.
}

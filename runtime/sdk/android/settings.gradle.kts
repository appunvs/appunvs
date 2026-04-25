// Standalone Gradle project for the runtime SDK library.  Lives next to
// (not inside) the RN init's `runtime/android/` so we don't inherit the
// React Native gradle plugin requirement (which needs `npm install`
// to materialize @react-native/gradle-plugin under node_modules/).
//
// PR D2.c may collapse this back into the RN init's project once the
// SDK actually links Hermes + React Native — at which point the npm
// dependency is unavoidable anyway.
pluginManagement {
    repositories {
        gradlePluginPortal()
        google()
        mavenCentral()
    }
}

dependencyResolutionManagement {
    repositoriesMode.set(RepositoriesMode.FAIL_ON_PROJECT_REPOS)
    repositories {
        google()
        mavenCentral()
    }
}

rootProject.name = "runtime-sdk"
include(":runtimesdk")

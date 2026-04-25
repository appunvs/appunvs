// Top-level Gradle build file.  Plugin versions live here; per-module
// Gradle files apply them.  Keep this minimal — modules own their
// own dependency lists.

plugins {
    alias(libs.plugins.android.application) apply false
    alias(libs.plugins.kotlin.android) apply false
    alias(libs.plugins.kotlin.compose) apply false
    alias(libs.plugins.kotlin.serialization) apply false
}

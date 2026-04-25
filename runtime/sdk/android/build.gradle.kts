// Root project for the runtime SDK library build.  Plugin versions
// declared once here, modules `apply` them.  Match the host shell's
// AGP / Kotlin pinning so AAR consumers don't hit dual-runtime issues.
plugins {
    id("com.android.library") version "8.7.2" apply false
    id("org.jetbrains.kotlin.android") version "2.0.21" apply false
}

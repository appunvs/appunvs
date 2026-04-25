// RuntimeSDK — public surface for the runtime SDK Android library.
//
// D2.a empty shell: exposes one static method (`hello()`) so the host
// can link the AAR, call into it, and prove the packaging pipeline
// (gradle :runtimesdk:assembleRelease → aar → host link) works
// end-to-end.
//
// PR D2.c replaces this with the real surface:
//
//   class RuntimeView(context: Context) : ViewGroup(context) {
//       suspend fun loadBundle(url: String)
//       fun reset()
//   }
//
// At that point the JNI layer for Hermes + React Native's C++ runtime
// gets wired in via libruntimesdk.so under jniLibs/.
package com.appunvs.runtimesdk

object RuntimeSDK {
    /** SDK release version string (matches `runtime/version.json`). */
    const val VERSION: String = "0.1.0"

    /**
     * Returns a stable identifier for this SDK build.  Host code calls
     * this in D2.b to verify the AAR linked correctly.
     */
    fun hello(): String = "hello from runtime SDK (D2.a empty shell)"
}

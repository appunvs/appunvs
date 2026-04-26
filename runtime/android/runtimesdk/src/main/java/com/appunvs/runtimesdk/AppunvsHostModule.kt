// AppunvsHostModule — Android side of the bridge between AI bundles'
// `@appunvs/host` imports and the host app's capabilities.
//
// D3.e.1: scaffolding (sdkVersion + echo smoke).
// D3.e.2 (this PR): identity flows in via the constructor — each
// per-RuntimeView ReactHost gets its own AppunvsHostPackage(identity),
// which constructs an AppunvsHostModule with that identity, and the
// module exposes it as a `identity` constant.
//
// Storage / network / publish are still no-ops here; D3.e.{3,4,5} land
// the real surfaces.
package com.appunvs.runtimesdk

import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod

class AppunvsHostModule(
    reactContext: ReactApplicationContext,
    private val identity: RuntimeBoxIdentity,
) : ReactContextBaseJavaModule(reactContext) {

    // The JS-side native module name.  AI bundles reach it via
    // `NativeModules.AppunvsHost`.  KEEP IN SYNC with the iOS
    // RCT_EXPORT_MODULE(AppunvsHost) name.
    override fun getName(): String = NAME

    // Constants are exposed once, when the module is initialised, and
    // available from JS as `NativeModules.AppunvsHost.sdkVersion` /
    // `.identity`.  sdkVersion is the SDK ABI version (not host app,
    // not bundle).  identity mirrors BoxIdentity in HostBridge.ts.
    override fun getConstants(): Map<String, Any> = mapOf(
        "sdkVersion" to "0.1.0",
        "identity" to mapOf(
            "boxID"   to identity.boxID,
            "version" to identity.version,
            "title"   to identity.title,
        ),
    )

    // Smoke-test method: `host()._echo(s)` round-trips a string through
    // the bridge.  D3.e.4 / D3.e.5 will replace this with real
    // network.request / publish.publish methods; for now it lets us
    // verify the JS→native→JS path works end to end.
    @ReactMethod
    fun echo(message: String?, promise: Promise) {
        if (message == null) {
            promise.reject("E_NIL_MESSAGE", "echo: null message")
            return
        }
        promise.resolve(message)
    }

    companion object {
        const val NAME = "AppunvsHost"
    }
}

// AppunvsHostModule — Android side of the bridge between AI bundles'
// `@appunvs/host` imports and the host app's capabilities.
//
// D3.e.1 (this PR): scaffolding only.  Exposes a `sdkVersion` constant
// and an `echo` smoke-test method.  Real surfaces (identity, storage,
// network, publish) land in D3.e.{2,3,4,5}.
//
// One instance per RuntimeView's ReactHost (the SDK constructs hosts
// individually in RuntimeView.kt and passes our package list).  Per-
// instance identity (boxID etc.) is NOT here yet — D3.e.2 will thread
// it through loadBundle into the constructor.
package com.appunvs.runtimesdk

import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod

class AppunvsHostModule(reactContext: ReactApplicationContext) :
    ReactContextBaseJavaModule(reactContext) {

    // The JS-side native module name.  AI bundles reach it via
    // `NativeModules.AppunvsHost`.  KEEP IN SYNC with the iOS
    // RCT_EXPORT_MODULE(AppunvsHost) name.
    override fun getName(): String = NAME

    // Constants are exposed once, when the module is initialised, and
    // available from JS as `NativeModules.AppunvsHost.sdkVersion`.
    // This is the SDK ABI version, not the host app or the bundle.
    override fun getConstants(): Map<String, Any> = mapOf(
        "sdkVersion" to "0.1.0",
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

// AppunvsHostModule — Android side of the bridge between AI bundles'
// `@appunvs/host` imports and the host app's capabilities.
//
// D3.e.1: scaffolding (sdkVersion + echo smoke).
// D3.e.2: identity threaded in via the constructor.
// D3.e.3: storage backed by react-native-mmkv on the JS side.
// D3.e.{4,5} (this PR): network.request() + publish() — bridge methods
// that delegate to host-registered handler functions.  SDK doesn't make
// HTTP calls or know what publish means; the host's HTTP client / relay
// flow does, and registers a function here at app launch via
// AppunvsHostModule.registerRequestHandler / .registerPublishHandler.
//
// What this PR does NOT do:
//   - SubNetwork.subscribe (SSE) — separate architecture chunk.
//   - host shell wiring (registering handlers in appunvs/android) —
//     each host's responsibility, not SDK's.
package com.appunvs.runtimesdk

import com.facebook.react.bridge.Arguments
import com.facebook.react.bridge.Promise
import com.facebook.react.bridge.ReactApplicationContext
import com.facebook.react.bridge.ReactContextBaseJavaModule
import com.facebook.react.bridge.ReactMethod
import com.facebook.react.bridge.ReadableMap
import com.facebook.react.bridge.WritableMap

/** Host-supplied handler called from JS via host().network.request().
 *  Path is already prefixed with /box/{id}/ by the JS layer; host fills
 *  in baseURL + auth.  body is the request body (JSON or raw text), or
 *  null for GET.  Handler completes by invoking [callback] with either
 *  a response WritableMap { status, headers, body } or an error.
 */
typealias AppunvsRequestHandler = (
    method: String,
    path: String,
    body: String?,
    callback: (response: WritableMap?, error: Throwable?) -> Unit,
) -> Unit

/** Host-supplied handler for host().publish.publish().  Completes with
 *  a response WritableMap { version, ok } or error.
 */
typealias AppunvsPublishHandler = (
    message: String?,
    callback: (response: WritableMap?, error: Throwable?) -> Unit,
) -> Unit

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
    // the bridge.  Kept around for D3.e.{4,5} CI smoke; an AI bundle
    // wouldn't normally call it.
    @ReactMethod
    fun echo(message: String?, promise: Promise) {
        if (message == null) {
            promise.reject("E_NIL_MESSAGE", "echo: null message")
            return
        }
        promise.resolve(message)
    }

    // D3.e.4: network.request().  JS already prefixed `path` with
    // /box/{id}/.  Delegates to the host-registered handler — SDK
    // doesn't make HTTP calls itself.
    @ReactMethod
    fun request(method: String, path: String, body: String?, promise: Promise) {
        val handler = requestHandler
        if (handler == null) {
            promise.reject(
                "E_NO_REQUEST_HANDLER",
                "host hasn't registered a network request handler " +
                    "(see AppunvsHostModule.registerRequestHandler)",
            )
            return
        }
        handler(method, path, body) { response, error ->
            if (error != null) {
                promise.reject("E_REQUEST_FAILED", error.message ?: "request failed", error)
            } else {
                promise.resolve(response ?: Arguments.createMap())
            }
        }
    }

    // D3.e.5: publish().  Same pattern — host's relay client knows what
    // publish means; SDK is a relay.
    @ReactMethod
    fun publish(message: String?, promise: Promise) {
        val handler = publishHandler
        if (handler == null) {
            promise.reject(
                "E_NO_PUBLISH_HANDLER",
                "host hasn't registered a publish handler " +
                    "(see AppunvsHostModule.registerPublishHandler)",
            )
            return
        }
        handler(message) { response, error ->
            if (error != null) {
                promise.reject("E_PUBLISH_FAILED", error.message ?: "publish failed", error)
            } else {
                promise.resolve(response ?: Arguments.createMap().apply {
                    putString("version", "")
                    putBoolean("ok", false)
                })
            }
        }
    }

    companion object {
        const val NAME = "AppunvsHost"

        /** Per-process slots for host-registered handlers.  Hosts call
         *  these once at app launch; pass null to clear (e.g. tests).
         */
        @Volatile
        var requestHandler: AppunvsRequestHandler? = null

        @Volatile
        var publishHandler: AppunvsPublishHandler? = null

        /** Convenience setter to match the iOS Obj-C API shape. */
        @JvmStatic
        fun registerRequestHandler(handler: AppunvsRequestHandler?) {
            requestHandler = handler
        }

        @JvmStatic
        fun registerPublishHandler(handler: AppunvsPublishHandler?) {
            publishHandler = handler
        }
    }
}

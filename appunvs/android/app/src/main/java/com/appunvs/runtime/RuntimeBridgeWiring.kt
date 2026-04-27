// RuntimeBridgeWiring — registers the SDK's AppunvsHostModule callbacks
// against this host shell's OkHttpClient and publish flow.
//
// Called from MainActivity.onCreate() (or wherever AuthRepo becomes
// available with a live token source).  Without these registrations,
// AI bundles loaded into a RuntimeView would call
// host().network.request() / .publish() and get
// "host hasn't registered a ... handler" errors.
//
// What this wires:
//   - request handler -> AuthRepo's auth-interceptor-wrapped OkHttpClient.
//                         Token rotation is automatic.
//   - publish handler -> stub returning { version: "", ok: false }
//                        (relay's publish endpoint not yet defined; the
//                         contract is wired so AI bundles get a real
//                         response shape rather than a "no handler"
//                         rejection — when the relay surface lands,
//                         swap this for the real call).
//
// What's NOT wired here (deliberately):
//   - SubNetwork.subscribe (SSE) — the SDK side of subscribe needs a
//     RCTEventEmitter migration + a generic SSE consumer; that's
//     the only D3.e remainder, tracked as a separate slice.
//
// Registrations are per-process; calling register(http:) again replaces
// the previous handlers (e.g., on sign-out + re-sign-in the new
// http takes over).
package com.appunvs.runtime

import com.appunvs.runtime.net.NetConfig
import com.appunvs.runtimesdk.AppunvsHostModule
import com.facebook.react.bridge.Arguments
import com.facebook.react.bridge.WritableMap
import okhttp3.MediaType.Companion.toMediaTypeOrNull
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.IOException
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException

object RuntimeBridgeWiring {

    /// Long-lived scope for the request handler closures.  The SDK
    /// keeps the handler reference alive for the process lifetime so
    /// we use SupervisorJob — one failed request doesn't take the
    /// whole bridge down.
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    fun register(http: OkHttpClient) {
        registerRequest(http)
        registerPublish()
    }

    private fun registerRequest(http: OkHttpClient) {
        AppunvsHostModule.registerRequestHandler { method, path, body, callback ->
            scope.launch {
                try {
                    val resolvedPath = if (path.startsWith("/")) path.drop(1) else path
                    val url = NetConfig.relayBaseURL.trimEnd('/') + "/" + resolvedPath
                    val req = Request.Builder()
                        .url(url)
                        .method(
                            method,
                            body?.takeIf { it.isNotEmpty() }
                                ?.toRequestBody("application/json".toMediaTypeOrNull()),
                        )
                        .header("Accept", "application/json")
                        .build()

                    val (status, headers, bodyText) = executeAsync(http, req)
                    val response: WritableMap = Arguments.createMap().apply {
                        putInt("status", status)
                        putString("body", bodyText)
                        val headersMap = Arguments.createMap()
                        for ((k, v) in headers) headersMap.putString(k, v)
                        putMap("headers", headersMap)
                    }
                    callback(response, null)
                } catch (t: Throwable) {
                    callback(null, t)
                }
            }
        }
    }

    private fun registerPublish() {
        AppunvsHostModule.registerPublishHandler { _, callback ->
            // No relay-side publish endpoint defined yet.  AI bundles
            // calling publish() get an explicit ok=false response so
            // they can surface "publish unavailable" — better than a
            // "no handler" rejection because the JS side can branch
            // on .ok.  Replace with a real relay call when defined.
            val response = Arguments.createMap().apply {
                putString("version", "")
                putBoolean("ok", false)
            }
            callback(response, null)
        }
    }

    /// Execute an OkHttp request as a suspending function.  Wraps the
    /// async API rather than calling the blocking .execute() — keeps
    /// the bridge responsive even on slow networks.
    private suspend fun executeAsync(
        http: OkHttpClient,
        req: Request,
    ): Triple<Int, Map<String, String>, String> = suspendCancellableCoroutine { cont ->
        val call = http.newCall(req)
        cont.invokeOnCancellation { call.cancel() }
        call.enqueue(object : okhttp3.Callback {
            override fun onFailure(call: okhttp3.Call, e: IOException) {
                cont.resumeWithException(e)
            }
            override fun onResponse(call: okhttp3.Call, response: okhttp3.Response) {
                response.use { r ->
                    val headers = r.headers.toMultimap()
                        .mapValues { it.value.firstOrNull() ?: "" }
                    val body = r.body?.string() ?: ""
                    cont.resume(Triple(r.code, headers, body))
                }
            }
        })
    }
}

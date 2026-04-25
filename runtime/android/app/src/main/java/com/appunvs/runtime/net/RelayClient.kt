// RelayClient — wires OkHttp + Retrofit + Kotlinx serialization, and
// installs an Authorization-header interceptor that pulls the current
// token from a `TokenSource` callback.  One client per process; the
// callback indirection lets the AuthRepo rotate the token (session →
// device → cleared on signout) without re-instantiating Retrofit.
package com.appunvs.runtime.net

import kotlinx.serialization.json.Json
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import retrofit2.Retrofit
import retrofit2.converter.kotlinx.serialization.asConverterFactory

fun interface TokenSource {
    fun current(): String?
}

object RelayClient {
    val json: Json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = true
    }

    /// Build a fresh Retrofit-backed RelayApi + the OkHttp client used
    /// by the AI SSE consumer.  Both share one `tokenSource`.
    fun build(tokenSource: TokenSource): RelayBundle {
        val auth = Interceptor { chain ->
            val req = chain.request()
            val tok = tokenSource.current()
            val out = if (tok != null) {
                req.newBuilder().addHeader("Authorization", "Bearer $tok").build()
            } else {
                req
            }
            chain.proceed(out)
        }

        val ok = OkHttpClient.Builder()
            .addInterceptor(auth)
            .build()

        val retrofit = Retrofit.Builder()
            .baseUrl(NetConfig.relayBaseURL)
            .client(ok)
            .addConverterFactory(json.asConverterFactory("application/json".toMediaType()))
            .build()

        val api = retrofit.create(RelayApi::class.java)
        return RelayBundle(api = api, http = ok)
    }
}

/// Pair returned from RelayClient.build — the typed Retrofit interface
/// for REST calls, plus the underlying OkHttp client used by the AI SSE
/// consumer for streaming reads.
data class RelayBundle(
    val api: RelayApi,
    val http: OkHttpClient,
)

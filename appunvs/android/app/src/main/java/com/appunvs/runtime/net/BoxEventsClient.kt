// BoxEventsClient — long-lived SSE consumer for GET /box/events.
//
// Opened once at sign-in (MainActivity LaunchedEffect) and torn down at
// sign-out / process exit.  Receives `bundle_ready` events for any box
// owned by the authenticated user; the consumer typically calls
// `boxRepo.refresh()` so Stage's reactive binding to activeBox.bundleURL
// flips and RuntimeView re-mounts the new bundle.
//
// Reconnect: this client keeps trying.  Network drops, server restarts,
// proxy timeouts — all are handled by an internal exponential-backoff
// reconnect loop.  On a fresh (re)connect we emit a `Reconnected` event
// so the consumer can `boxRepo.refresh()` and pick up any events missed
// while disconnected (publishes during the gap fan out to subscribers
// that don't exist; we recover by polling once).
//
// Heartbeats (`event: heartbeat`) keep the TCP connection alive through
// reverse proxies; they're consumed silently and never reach the caller.
//
// Surface: `events()` returns a cold `Flow` the caller drives via
// `collect { }`.  Cancellation via the collecting coroutine's scope
// cancels the in-flight HTTP call (OkHttp's response.use propagates).
package com.appunvs.runtime.net

import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json
import kotlin.coroutines.coroutineContext
import okhttp3.OkHttpClient
import okhttp3.Request

@Serializable
data class BoxBundleReadyEvent(
    val type: String,
    val box_id: String,
    val version: String,
    val uri: String,
    val content_hash: String,
    val size_bytes: Long,
)

/// What the events() Flow emits.
sealed class BoxStreamEvent {
    data class BundleReady(val payload: BoxBundleReadyEvent) : BoxStreamEvent()
    /// Emitted on every successful (re)connect AFTER the first.  The
    /// caller uses this as a hint to re-fetch the box list and catch up
    /// on events that may have fired while disconnected.
    object Reconnected : BoxStreamEvent()
}

class BoxEventsClient(
    private val http: OkHttpClient,
    private val baseURL: String = NetConfig.relayBaseURL,
    private val json: Json = RelayClient.json,
) {
    /// Backoff schedule for reconnect attempts, in milliseconds.  Walks
    /// the array and stays at the last value indefinitely.  6 entries
    /// totals ~62s before settling at the 30s cap.
    private val backoffMs = longArrayOf(1_000, 2_000, 5_000, 10_000, 20_000, 30_000)

    fun events(): Flow<BoxStreamEvent> = flow {
        var attempt = 0
        var firstConnect = true
        while (true) {
            val connected = runOnce(firstConnect, this)
            if (connected) {
                attempt = 0
                firstConnect = false
            } else {
                attempt = (attempt + 1).coerceAtMost(backoffMs.size - 1)
            }
            delay(backoffMs[attempt.coerceAtMost(backoffMs.lastIndex)])
        }
    }.flowOn(Dispatchers.IO)

    /// One open-stream-and-read attempt.  Returns true if the response
    /// hit an OK status so the caller can reset its backoff counter.
    /// Errors during read mid-stream still count as connected (the
    /// stream just dropped — backoff resets next iteration).
    ///
    /// Cancellation: OkHttp's execute()/readUtf8Line() are blocking
    /// and don't observe coroutine cancellation on their own.  We bind
    /// call.cancel() to the producing coroutine's Job — when the
    /// LaunchedEffect that hosts our flow cancels, the job completes,
    /// the handler fires, and the in-flight HTTP request aborts
    /// (closing the socket; readUtf8Line throws IOException; the
    /// catch below funnels back through the flow with no crash).
    private suspend fun runOnce(
        firstConnect: Boolean,
        emitter: kotlinx.coroutines.flow.FlowCollector<BoxStreamEvent>,
    ): Boolean {
        val req = Request.Builder()
            .url("$baseURL/box/events")
            .header("Accept", "text/event-stream")
            .get()
            .build()

        val call = http.newCall(req)
        val handle = coroutineContext[Job]?.invokeOnCompletion { call.cancel() }
        return try {
            call.execute().use { resp ->
                if (!resp.isSuccessful) return false
                val source = resp.body?.source() ?: return false

                if (!firstConnect) {
                    emitter.emit(BoxStreamEvent.Reconnected)
                }

                var event = "message"
                while (!source.exhausted()) {
                    val line = source.readUtf8Line() ?: break
                    if (line.isEmpty()) {
                        event = "message"
                        continue
                    }
                    if (line.startsWith("event: ")) {
                        event = line.removePrefix("event: ")
                        continue
                    }
                    if (line.startsWith("data: ")) {
                        val payload = line.removePrefix("data: ")
                        parse(event, payload)?.let { emitter.emit(it) }
                    }
                }
                true
            }
        } catch (ce: CancellationException) {
            // Coroutine cancellation must propagate — don't swallow.
            throw ce
        } catch (_: Throwable) {
            false
        } finally {
            handle?.dispose()
        }
    }

    private fun parse(event: String, payload: String): BoxStreamEvent? {
        return when (event) {
            "bundle_ready" -> runCatching {
                BoxStreamEvent.BundleReady(json.decodeFromString(BoxBundleReadyEvent.serializer(), payload))
            }.getOrNull()
            "heartbeat", "message" -> null
            else -> null
        }
    }
}

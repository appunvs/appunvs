// AISSEClient — Server-Sent Events consumer for POST /ai/turn.
//
// We read the response body line-by-line through OkHttp's `BufferedSource`
// rather than pulling in a dedicated SSE library — the relay's grammar
// is small (one `event:` + one `data:` per record, blank line separator)
// and we want the streamed-frame surface to look the same on both
// platforms.
//
// The flow is cold: collecting starts the HTTP request; the upstream
// connection is closed on cancellation or after the terminal frame.
package com.appunvs.runtime.net

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.flow.flowOn
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.intOrNull
import kotlinx.serialization.json.jsonPrimitive
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody

sealed class AIFrame {
    data class Token(val turnID: String, val text: String) : AIFrame()
    data class ToolCall(val turnID: String, val callID: String, val name: String, val argsJSON: String) : AIFrame()
    data class ToolResult(val turnID: String, val callID: String, val resultJSON: String, val isError: Boolean) : AIFrame()
    data class Finished(val turnID: String, val stopReason: String, val tokensIn: Int, val tokensOut: Int) : AIFrame()
    data class Err(val turnID: String, val message: String) : AIFrame()
}

class AISSEClient(
    private val http: OkHttpClient,
    private val baseURL: String = NetConfig.relayBaseURL,
    private val json: Json = RelayClient.json,
) {

    fun turn(boxID: String, text: String): Flow<AIFrame> = flow {
        val body = json.encodeToString(
            AITurnRequest.serializer(),
            AITurnRequest(boxID = boxID, text = text),
        ).toRequestBody("application/json".toMediaType())

        val req = Request.Builder()
            .url("$baseURL/ai/turn")
            .header("Accept", "text/event-stream")
            .post(body)
            .build()

        http.newCall(req).execute().use { resp ->
            if (!resp.isSuccessful) {
                val msg = resp.body?.string()?.take(2048) ?: "(no body)"
                throw IllegalStateException("HTTP ${resp.code}: $msg")
            }
            val source = resp.body?.source() ?: throw IllegalStateException("empty body")
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
                    parse(event, payload)?.let { frame ->
                        emit(frame)
                        if (frame is AIFrame.Finished || frame is AIFrame.Err) return@use
                    }
                }
            }
        }
    }.flowOn(Dispatchers.IO)

    private fun parse(event: String, payload: String): AIFrame? {
        val obj = runCatching { json.parseToJsonElement(payload) as JsonObject }
            .getOrNull() ?: return null
        val turnID = obj["turn_id"]?.jsonPrimitive?.contentOrNull ?: ""
        return when (event) {
            "token" -> AIFrame.Token(
                turnID = turnID,
                text = obj["text"]?.jsonPrimitive?.contentOrNull ?: "",
            )
            "tool_call" -> AIFrame.ToolCall(
                turnID = turnID,
                callID = obj["call_id"]?.jsonPrimitive?.contentOrNull ?: "",
                name = obj["name"]?.jsonPrimitive?.contentOrNull ?: "",
                argsJSON = obj["args_json"]?.jsonPrimitive?.contentOrNull ?: "",
            )
            "tool_result" -> AIFrame.ToolResult(
                turnID = turnID,
                callID = obj["call_id"]?.jsonPrimitive?.contentOrNull ?: "",
                resultJSON = obj["result_json"]?.jsonPrimitive?.contentOrNull ?: "",
                isError = obj["is_error"]?.jsonPrimitive?.boolean ?: false,
            )
            "finished" -> AIFrame.Finished(
                turnID = turnID,
                stopReason = obj["stop_reason"]?.jsonPrimitive?.contentOrNull ?: "",
                tokensIn = obj["tokens_in"]?.jsonPrimitive?.intOrNull ?: 0,
                tokensOut = obj["tokens_out"]?.jsonPrimitive?.intOrNull ?: 0,
            )
            "error" -> AIFrame.Err(
                turnID = turnID,
                message = obj["error"]?.jsonPrimitive?.contentOrNull ?: "unknown error",
            )
            else -> null
        }
    }
}

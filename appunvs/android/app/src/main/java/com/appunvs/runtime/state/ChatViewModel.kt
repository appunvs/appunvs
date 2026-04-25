// ChatViewModel — chat transcript + AI streaming.  Mirrors iOS
// ChatStore.  Per-box transcripts live in memory; switching boxes
// switches the visible transcript without losing prior turns.
package com.appunvs.runtime.state

import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import androidx.compose.runtime.snapshots.SnapshotStateList
import androidx.compose.runtime.toMutableStateList
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.appunvs.runtime.net.AIFrame
import com.appunvs.runtime.net.AISSEClient
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.launch
import java.util.UUID

enum class ChatRole { USER, ASSISTANT, SYSTEM }

data class ChatMessage(
    val id: String,
    val role: ChatRole,
    var text: String,
    var pending: Boolean = false,
)

class ChatViewModel(private val sse: AISSEClient) : ViewModel() {

    private val transcripts = mutableStateMapOf<String, SnapshotStateList<ChatMessage>>()
    var sending by mutableStateOf(false)
        private set
    var lastError by mutableStateOf<String?>(null)
    private var currentJob: Job? = null

    fun messages(boxID: String?): List<ChatMessage> =
        if (boxID == null) emptyList() else transcripts[boxID] ?: emptyList()

    fun send(boxID: String, text: String) {
        currentJob?.cancel()
        sending = true
        val transcript = transcripts.getOrPut(boxID) { mutableListOf<ChatMessage>().toMutableStateList() }
        transcript += ChatMessage(rid(), ChatRole.USER, text)
        val assistant = ChatMessage(rid(), ChatRole.ASSISTANT, "", pending = true)
        transcript += assistant

        currentJob = viewModelScope.launch {
            sse.turn(boxID, text)
                .catch { e ->
                    finalize(boxID, assistant.id)
                    appendSystem(boxID, "× ${e.message ?: e.javaClass.simpleName}")
                    lastError = e.message
                }
                .collect { frame ->
                    when (frame) {
                        is AIFrame.Token       -> appendToken(boxID, assistant.id, frame.text)
                        is AIFrame.ToolCall    -> appendSystem(boxID, "› ${frame.name} ${frame.argsJSON}")
                        is AIFrame.ToolResult  -> if (frame.isError) appendSystem(boxID, "✗ tool failed")
                        is AIFrame.Finished    -> finalize(boxID, assistant.id)
                        is AIFrame.Err         -> {
                            finalize(boxID, assistant.id)
                            appendSystem(boxID, "× ${frame.message}")
                            lastError = frame.message
                        }
                    }
                }
            finalize(boxID, assistant.id)
            sending = false
        }
    }

    fun cancel() {
        currentJob?.cancel()
        sending = false
    }

    fun clear(boxID: String) {
        transcripts[boxID]?.clear()
    }

    // MARK: - private

    private fun appendToken(boxID: String, id: String, text: String) {
        val transcript = transcripts[boxID] ?: return
        val idx = transcript.indexOfFirst { it.id == id }
        if (idx == -1) return
        // SnapshotStateList notifies on element replacement — mutating
        // the in-place data class field doesn't trigger recomposition.
        transcript[idx] = transcript[idx].copy(text = transcript[idx].text + text)
    }

    private fun appendSystem(boxID: String, text: String) {
        val transcript = transcripts[boxID] ?: return
        transcript += ChatMessage(rid(), ChatRole.SYSTEM, text)
    }

    private fun finalize(boxID: String, id: String) {
        val transcript = transcripts[boxID] ?: return
        val idx = transcript.indexOfFirst { it.id == id }
        if (idx == -1) return
        val cur = transcript[idx]
        transcript[idx] = cur.copy(
            text = if (cur.text.isEmpty()) "(no response)" else cur.text,
            pending = false,
        )
    }

    private fun rid(): String = UUID.randomUUID().toString().replace("-", "")
}

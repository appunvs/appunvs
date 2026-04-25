// MockData — in-memory fixtures used while the network layer is being
// designed.  Real /box and /pair clients arrive in a follow-up PR;
// this lets the screens render end-to-end against realistic shapes.
//
// Field names mirror box.proto / pair.proto so swapping in the real
// network store is a 1:1 lift.
package com.appunvs.runtime.state

import androidx.compose.runtime.Composable
import androidx.compose.runtime.MutableState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateListOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.lifecycle.ViewModel

enum class BoxState { DRAFT, PUBLISHED, ARCHIVED }

data class Box(
    val id: String,
    val title: String,
    val state: BoxState,
    val currentVersion: String,
    val updatedAt: Long,
)

enum class ChatRole { USER, ASSISTANT, SYSTEM }

data class ChatMessage(
    val id: String,
    val role: ChatRole,
    val text: String,
    val pending: Boolean = false,
)

class MockStore : ViewModel() {
    val boxes = mutableStateListOf<Box>()
    val messages = mutableStateListOf<ChatMessage>()
    var activeBox by mutableStateOf<Box?>(null)
        private set

    init {
        val now = System.currentTimeMillis()
        val demo = Box(
            id = "box_demo",
            title = "demo-counter",
            state = BoxState.DRAFT,
            currentVersion = "",
            updatedAt = now,
        )
        boxes += demo
        boxes += Box("box_todo",  "todo-app",     BoxState.PUBLISHED, "v3", now - 3_600_000)
        boxes += Box("box_color", "color-picker", BoxState.ARCHIVED,  "v1", now - 86_400_000)
        activeBox = demo

        messages += ChatMessage(rid(), ChatRole.USER, "做一个计数器 app")
        messages += ChatMessage(rid(), ChatRole.ASSISTANT, "好的,我来写。先建一个 index.tsx 加 Button。")
    }

    fun setActive(box: Box) {
        activeBox = box
    }

    fun appendUser(text: String) {
        messages += ChatMessage(rid(), ChatRole.USER, text)
        messages += ChatMessage(rid(), ChatRole.ASSISTANT, "（模拟回复 — 真 AI 在 PR D 接入）")
    }

    fun createBox(title: String): Box {
        val b = Box(
            id = "box_${rid().take(8)}",
            title = title,
            state = BoxState.DRAFT,
            currentVersion = "",
            updatedAt = System.currentTimeMillis(),
        )
        boxes.add(0, b)
        activeBox = b
        return b
    }

    private fun rid(): String =
        java.util.UUID.randomUUID().toString().replace("-", "")
}

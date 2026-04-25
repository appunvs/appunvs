// ChatScreen — header chip (BoxSwitcher) + transcript + composer.
// Backed by ChatViewModel (real /ai/turn SSE) and BoxRepo (real /box).
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material3.Button
import androidx.compose.material3.Divider
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.state.BoxRepo
import com.appunvs.runtime.state.ChatRole
import com.appunvs.runtime.state.ChatViewModel
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.Bubble
import com.appunvs.runtime.ui.BubbleRole
import com.appunvs.runtime.ui.BoxSwitcher
import com.appunvs.runtime.ui.EmptyState
import com.appunvs.runtime.ui.NewBoxSheet

@Composable
fun ChatScreen(
    boxRepo: BoxRepo,
    chat: ChatViewModel,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    var draft by remember { mutableStateOf("") }
    var newBoxOpen by remember { mutableStateOf(false) }
    val activeBoxID = boxRepo.activeBox?.boxID
    val messages = chat.messages(activeBoxID)

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage),
    ) {
        // Header chip
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.l.dp, vertical = Spacing.s.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            BoxSwitcher(
                repo = boxRepo,
                onNewBox = { newBoxOpen = true },
            )
        }
        Divider(color = colors.borderDefault)

        // Transcript
        if (activeBoxID == null) {
            EmptyState(
                title = "选个 Box 开始",
                hint = "每个 Box 是一个独立项目，对话历史与代码都和它绑定。",
                modifier = Modifier.weight(1f),
            )
        } else if (messages.isEmpty()) {
            EmptyState(
                title = "和 AI 说点什么",
                hint = "比如\"做一个计数器 app\"。",
                modifier = Modifier.weight(1f),
            )
        } else {
            val listState = rememberLazyListState()
            LaunchedEffect(messages.size) {
                if (messages.isNotEmpty()) {
                    listState.animateScrollToItem(messages.lastIndex)
                }
            }
            LazyColumn(
                state = listState,
                verticalArrangement = Arrangement.spacedBy(Spacing.s.dp),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(Spacing.l.dp),
                modifier = Modifier.weight(1f),
            ) {
                items(messages, key = { it.id }) { msg ->
                    Bubble(
                        role = msg.role.bubbleRole(),
                        text = msg.text,
                        pending = msg.pending,
                    )
                }
            }
        }

        // Composer
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(colors.bgCard)
                .padding(Spacing.s.dp),
            verticalAlignment = Alignment.Bottom,
        ) {
            OutlinedTextField(
                value = draft,
                onValueChange = { draft = it },
                placeholder = { Text("描述一个改动…") },
                modifier = Modifier
                    .weight(1f)
                    .heightIn(max = 120.dp),
            )
            Spacer(Modifier.width(Spacing.s.dp))
            Button(
                onClick = {
                    val t = draft.trim()
                    val box = boxRepo.activeBox
                    if (t.isNotEmpty() && box != null) {
                        chat.send(box.boxID, t)
                        draft = ""
                    }
                },
                enabled = draft.trim().isNotEmpty()
                    && activeBoxID != null
                    && !chat.sending,
            ) { Text(if (chat.sending) "…" else "发送") }
        }
    }

    if (newBoxOpen) {
        NewBoxSheet(
            onDismiss = { newBoxOpen = false },
            onCreate = { title ->
                boxRepo.create(title)
                newBoxOpen = false
            },
        )
    }
}

private fun ChatRole.bubbleRole(): BubbleRole = when (this) {
    ChatRole.USER      -> BubbleRole.USER
    ChatRole.ASSISTANT -> BubbleRole.ASSISTANT
    ChatRole.SYSTEM    -> BubbleRole.SYSTEM
}

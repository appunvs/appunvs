// ChatScreen — header chip (BoxSwitcher) + transcript + composer.
// AI is mock today via MockStore.appendUser; real /ai/turn SSE
// arrives in the network/auth follow-up PR.
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
import androidx.compose.material3.MaterialTheme
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

import com.appunvs.runtime.state.ChatRole
import com.appunvs.runtime.state.MockStore
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.Bubble
import com.appunvs.runtime.ui.BubbleRole
import com.appunvs.runtime.ui.BoxSwitcher
import com.appunvs.runtime.ui.EmptyState
import com.appunvs.runtime.ui.NewBoxSheet

@Composable
fun ChatScreen(
    store: MockStore,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    var draft by remember { mutableStateOf("") }
    var newBoxOpen by remember { mutableStateOf(false) }

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
                store = store,
                onNewBox = { newBoxOpen = true },
            )
        }
        Divider(color = colors.borderDefault)

        // Transcript
        if (store.activeBox == null) {
            EmptyState(
                title = "选个 Box 开始",
                hint = "每个 Box 是一个独立项目，对话历史与代码都和它绑定。",
                modifier = Modifier.weight(1f),
            )
        } else if (store.messages.isEmpty()) {
            EmptyState(
                title = "和 AI 说点什么",
                hint = "比如\"做一个计数器 app\"。",
                modifier = Modifier.weight(1f),
            )
        } else {
            val listState = rememberLazyListState()
            LaunchedEffect(store.messages.size) {
                if (store.messages.isNotEmpty()) {
                    listState.animateScrollToItem(store.messages.lastIndex)
                }
            }
            LazyColumn(
                state = listState,
                verticalArrangement = Arrangement.spacedBy(Spacing.s.dp),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(Spacing.l.dp),
                modifier = Modifier.weight(1f),
            ) {
                items(store.messages, key = { it.id }) { msg ->
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
                    if (t.isNotEmpty()) {
                        store.appendUser(t)
                        draft = ""
                    }
                },
                enabled = draft.trim().isNotEmpty(),
            ) { Text("发送") }
        }
    }

    if (newBoxOpen) {
        NewBoxSheet(
            onDismiss = { newBoxOpen = false },
            onCreate = { title ->
                store.createBox(title)
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

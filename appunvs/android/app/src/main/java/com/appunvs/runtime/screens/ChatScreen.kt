// ChatScreen — chat detail reached by drilling into a box from
// BoxesScreen.  Header is a back-arrow + box title row (replaces the
// old BoxSwitcher chip); body is the scrolling transcript; footer is
// the composer.  Same store wiring as before.
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
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.outlined.ArrowBack
import androidx.compose.material3.Button
import androidx.compose.material3.Divider
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
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
import com.appunvs.runtime.theme.AppType
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.Bubble
import com.appunvs.runtime.ui.BubbleRole
import com.appunvs.runtime.ui.EmptyState

@Composable
fun ChatScreen(
    boxRepo: BoxRepo,
    chat: ChatViewModel,
    boxID: String,
    onBack: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    var draft by remember { mutableStateOf("") }
    val box = boxRepo.boxes.firstOrNull { it.boxID == boxID }
    val messages = chat.messages(boxID)

    LaunchedEffect(boxID) {
        box?.let(boxRepo::setActive)
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage),
    ) {
        // Header — back chevron + title.  Replaces the old BoxSwitcher
        // chip now that box selection lives in the dedicated Boxes tab.
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = Spacing.s.dp, vertical = Spacing.s.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            IconButton(onClick = onBack) {
                Icon(
                    Icons.AutoMirrored.Outlined.ArrowBack,
                    contentDescription = "返回",
                    tint = colors.textPrimary,
                )
            }
            Text(
                text = box?.title ?: "Box",
                style = AppType.bodyEmphasis.copy(color = colors.textPrimary),
            )
        }
        Divider(color = colors.borderDefault)

        // Transcript
        if (messages.isEmpty()) {
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
                    if (t.isNotEmpty()) {
                        chat.send(boxID, t)
                        draft = ""
                    }
                },
                enabled = draft.trim().isNotEmpty() && !chat.sending,
            ) { Text(if (chat.sending) "…" else "发送") }
        }
    }
}

private fun ChatRole.bubbleRole(): BubbleRole = when (this) {
    ChatRole.USER      -> BubbleRole.USER
    ChatRole.ASSISTANT -> BubbleRole.ASSISTANT
    ChatRole.SYSTEM    -> BubbleRole.SYSTEM
}

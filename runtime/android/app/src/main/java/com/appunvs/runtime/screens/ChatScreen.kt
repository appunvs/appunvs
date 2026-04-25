// ChatScreen — placeholder.  PR C ports BoxSwitcher + Bubble +
// ToolCall + Composer from the prior RN component.
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing

@Composable
fun ChatScreen(modifier: Modifier = Modifier) {
    val colors = LocalAppColors.current
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier.padding(Spacing.xxl.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Icon(
                imageVector = Icons.Outlined.ChatBubbleOutline,
                contentDescription = null,
                tint = colors.brandDark,
                modifier = Modifier.padding(bottom = Spacing.s.dp),
            )
            Text(
                text = "Chat",
                style = MaterialTheme.typography.titleLarge.copy(color = colors.textPrimary),
            )
            Text(
                text = "待 PR C: 把 Bubble / ToolCall / Composer 从 RN 端口过来。",
                style = MaterialTheme.typography.bodyMedium.copy(color = colors.textSecondary),
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = Spacing.m.dp),
            )
        }
    }
}

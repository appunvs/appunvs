// Bubble — chat message.  User messages right-align with brand fill;
// assistant / system messages left-align with bordered surface.  Soft
// "tail" corner trims the relevant top corner per role.
package com.appunvs.runtime.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing

enum class BubbleRole { USER, ASSISTANT, SYSTEM }

@Composable
fun Bubble(
    role: BubbleRole,
    text: String,
    pending: Boolean = false,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    val isUser = role == BubbleRole.USER

    val bg = when (role) {
        BubbleRole.USER      -> colors.brandDark
        BubbleRole.ASSISTANT -> colors.bgCard
        BubbleRole.SYSTEM    -> colors.bgInput
    }
    val fg = if (isUser) Color.White else colors.textPrimary

    val shape = if (isUser) {
        RoundedCornerShape(
            topStart = Radius.xl.dp,
            bottomStart = Radius.xl.dp,
            bottomEnd = Radius.xl.dp,
            topEnd = Radius.s.dp,
        )
    } else {
        RoundedCornerShape(
            topStart = Radius.s.dp,
            bottomStart = Radius.xl.dp,
            bottomEnd = Radius.xl.dp,
            topEnd = Radius.xl.dp,
        )
    }

    val display = if (text.isEmpty() && pending) "…" else text

    Row(
        modifier = modifier.fillMaxWidth(),
        horizontalArrangement = if (isUser) Arrangement.End else Arrangement.Start,
        verticalAlignment = Alignment.Top,
    ) {
        Text(
            text = display,
            color = fg,
            modifier = Modifier
                .widthIn(max = 360.dp)
                .clip(shape)
                .background(bg)
                .let {
                    if (!isUser) it.border(BorderStroke(1.dp, colors.borderDefault), shape) else it
                }
                .padding(horizontal = Spacing.l.dp, vertical = Spacing.m.dp),
        )
    }
}

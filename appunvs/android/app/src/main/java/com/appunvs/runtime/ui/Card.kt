// Card — themed surface wrapper with optional border.  Use this rather
// than `Box(modifier.background(...))` so radius / padding / border
// stay token-driven.
package com.appunvs.runtime.ui

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing

@Composable
fun AppCard(
    modifier: Modifier = Modifier,
    padding: Int = Spacing.l,
    cornerRadius: Int = Radius.l,
    bordered: Boolean = false,
    content: @Composable () -> Unit,
) {
    val colors = LocalAppColors.current
    val shape = RoundedCornerShape(cornerRadius.dp)
    Box(
        modifier = modifier
            .clip(shape)
            .background(colors.bgCard)
            .let { base ->
                if (bordered) base.border(BorderStroke(1.dp, colors.borderDefault), shape) else base
            }
            .padding(padding.dp),
    ) {
        content()
    }
}

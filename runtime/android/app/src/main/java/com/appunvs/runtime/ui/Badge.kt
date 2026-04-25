// Badge — pill-shaped status label.  Tone semantics map to theme
// tokens, never raw colors at the call site.
package com.appunvs.runtime.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing

enum class BadgeTone { NEUTRAL, INFO, SUCCESS, WARNING, DANGER }

@Composable
fun AppBadge(
    label: String,
    tone: BadgeTone = BadgeTone.NEUTRAL,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    val (bg, fg) = when (tone) {
        BadgeTone.NEUTRAL -> colors.bgInput   to colors.textSecondary
        BadgeTone.INFO    -> colors.brandPale to colors.brandDark
        BadgeTone.SUCCESS -> colors.brandPale to colors.semanticSuccess
        BadgeTone.WARNING -> colors.bgInput   to colors.semanticWarning
        BadgeTone.DANGER  -> colors.bgInput   to colors.semanticDanger
    }
    Text(
        text = label,
        color = fg,
        fontSize = 13.sp,
        fontWeight = FontWeight.SemiBold,
        modifier = modifier
            .clip(RoundedCornerShape(percent = 50))
            .background(bg)
            .padding(horizontal = Spacing.s.dp, vertical = 2.dp),
    )
}

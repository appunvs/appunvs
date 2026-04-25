// QuotaBar — usage progress with warning escalation at ≥90%.
package com.appunvs.runtime.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxHeight
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors

@Composable
fun QuotaBar(
    label: String,
    used: Int,
    cap: Int,
    unit: String? = null,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    val ratio = if (cap == 0) 0f else (used.toFloat() / cap).coerceIn(0f, 1f)
    val danger = ratio >= 0.9f
    val fill = if (danger) colors.semanticWarning else colors.brandDark

    Column(modifier = modifier) {
        Row(modifier = Modifier.fillMaxWidth().padding(bottom = 4.dp)) {
            Text(
                text = label,
                style = MaterialTheme.typography.bodySmall.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.SemiBold,
                ),
                modifier = Modifier.weight(1f),
            )
            Text(
                text = buildString {
                    append(used)
                    append(" / ")
                    append(cap)
                    if (unit != null) {
                        append(' ')
                        append(unit)
                    }
                },
                style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
            )
        }
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(6.dp)
                .clip(RoundedCornerShape(50))
                .background(colors.bgInput),
        ) {
            Box(
                modifier = Modifier
                    .fillMaxWidth(fraction = ratio)
                    .fillMaxHeight()
                    .background(fill),
            )
        }
    }
}

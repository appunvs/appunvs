// EmptyState — canonical zero-state shape.  Title + optional hint +
// optional action slot.  Matches the iOS counterpart's signature.
package com.appunvs.runtime.ui

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.widthIn
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
fun EmptyState(
    title: String,
    hint: String? = null,
    modifier: Modifier = Modifier,
    action: @Composable () -> Unit = {},
) {
    val colors = LocalAppColors.current
    Box(
        modifier = modifier
            .fillMaxSize()
            .padding(Spacing.xxl.dp),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = title,
                style = MaterialTheme.typography.titleLarge.copy(color = colors.textPrimary),
                textAlign = TextAlign.Center,
            )
            if (hint != null) {
                Text(
                    text = hint,
                    style = MaterialTheme.typography.bodyMedium.copy(color = colors.textSecondary),
                    textAlign = TextAlign.Center,
                    modifier = Modifier
                        .padding(top = Spacing.m.dp)
                        .widthIn(max = 360.dp),
                )
            }
            Box(modifier = Modifier.padding(top = Spacing.m.dp)) {
                action()
            }
        }
    }
}

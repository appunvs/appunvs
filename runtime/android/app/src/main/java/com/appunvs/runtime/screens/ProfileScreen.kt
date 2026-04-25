// ProfileScreen — minimal account center.  Sections:
//
//   1. Account header (placeholder identity)
//   2. Theme override picker (functional today; persists via AppState)
//
// Quotas / devices / billing land in PR C alongside the network
// client.  Box list management is intentionally NOT here — it lives
// in the Chat tab's BoxSwitcher chip.
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.state.AppState
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing

@Composable
fun ProfileScreen(
    modifier: Modifier = Modifier,
    state: AppState,
) {
    val colors = LocalAppColors.current
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage)
            .padding(Spacing.l.dp),
        verticalArrangement = Arrangement.spacedBy(Spacing.l.dp),
    ) {
        Text(
            text = "个人中心",
            style = MaterialTheme.typography.headlineMedium.copy(color = colors.textPrimary),
        )

        // Account card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = colors.bgCard),
            shape = RoundedCornerShape(Radius.l.dp),
        ) {
            Row(
                modifier = Modifier.padding(Spacing.l.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Box(
                    modifier = Modifier
                        .size(56.dp)
                        .clip(CircleShape)
                        .background(colors.brandPale),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = "u",
                        style = MaterialTheme.typography.titleLarge.copy(color = colors.brandDark),
                    )
                }
                Spacer(Modifier.width(Spacing.l.dp))
                Column {
                    Text(
                        text = "未登录用户",
                        style = MaterialTheme.typography.titleMedium.copy(color = colors.textPrimary),
                    )
                    Text(
                        text = "guest@local",
                        style = MaterialTheme.typography.bodyMedium.copy(color = colors.textSecondary),
                    )
                }
            }
        }

        // Theme picker card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = colors.bgCard),
            shape = RoundedCornerShape(Radius.l.dp),
        ) {
            Column(modifier = Modifier.padding(Spacing.l.dp)) {
                Text(
                    text = "主题",
                    style = MaterialTheme.typography.titleSmall.copy(color = colors.textSecondary),
                    modifier = Modifier.padding(bottom = Spacing.s.dp),
                )
                ThemePicker(state = state)
            }
        }

        Text(
            text = "appunvs · v0.0.1",
            style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
        )
    }
}

@Composable
private fun ThemePicker(state: AppState) {
    val options = listOf(
        AppState.ThemeOverride.SYSTEM to "跟随系统",
        AppState.ThemeOverride.LIGHT  to "浅色",
        AppState.ThemeOverride.DARK   to "深色",
    )
    SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
        options.forEachIndexed { index, (value, label) ->
            SegmentedButton(
                selected = state.themeOverride == value,
                onClick = { state.setTheme(value) },
                shape = SegmentedButtonDefaults.itemShape(index = index, count = options.size),
            ) {
                Text(label)
            }
        }
    }
}

// ProfileScreen — account center.  Sections:
//
//   1. Account header (placeholder identity)
//   2. Today's usage (mock numbers)
//   3. Theme override (functional, persisted via DataStore)
//   4. Devices (placeholder)
//   5. Footer (sign-out placeholder + version)
//
// Box list lives in the Chat tab's BoxSwitcher chip — not here.
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
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Divider
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.SegmentedButton
import androidx.compose.material3.SegmentedButtonDefaults
import androidx.compose.material3.SingleChoiceSegmentedButtonRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.state.AppState
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.AppBadge
import com.appunvs.runtime.ui.AppCard
import com.appunvs.runtime.ui.BadgeTone
import com.appunvs.runtime.ui.QuotaBar

@Composable
fun ProfileScreen(
    state: AppState,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage)
            .verticalScroll(rememberScrollState())
            .padding(Spacing.l.dp),
        verticalArrangement = Arrangement.spacedBy(Spacing.l.dp),
    ) {
        Text(
            text = "个人中心",
            style = MaterialTheme.typography.headlineMedium.copy(
                color = colors.textPrimary,
                fontWeight = FontWeight.Bold,
            ),
        )

        AccountCard()
        UsageCard()
        ThemeCard(state = state)
        DevicesCard()
        Footer()
    }
}

@Composable
private fun AccountCard() {
    val colors = LocalAppColors.current
    AppCard(modifier = Modifier.fillMaxWidth()) {
        Row(verticalAlignment = Alignment.CenterVertically) {
            Box(
                modifier = Modifier
                    .size(56.dp)
                    .clip(CircleShape)
                    .background(colors.brandPale),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = "u",
                    style = MaterialTheme.typography.titleLarge.copy(
                        color = colors.brandDark,
                        fontWeight = FontWeight.Bold,
                    ),
                )
            }
            Spacer(Modifier.width(Spacing.l.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "未登录用户",
                    style = MaterialTheme.typography.titleMedium.copy(color = colors.textPrimary),
                )
                Text(
                    text = "guest@local",
                    style = MaterialTheme.typography.bodyMedium.copy(color = colors.textSecondary),
                )
            }
            AppBadge("Free", tone = BadgeTone.INFO)
        }
    }
}

@Composable
private fun UsageCard() {
    val colors = LocalAppColors.current
    AppCard(modifier = Modifier.fillMaxWidth()) {
        Column(verticalArrangement = Arrangement.spacedBy(Spacing.m.dp)) {
            Text(
                text = "本月用量",
                style = MaterialTheme.typography.titleSmall.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.SemiBold,
                ),
            )
            QuotaBar(label = "对话", used = 0, cap = 300)
            QuotaBar(label = "存储", used = 0, cap = 5_120, unit = "MB")
        }
    }
}

@Composable
private fun ThemeCard(state: AppState) {
    val colors = LocalAppColors.current
    val options = listOf(
        AppState.ThemeOverride.SYSTEM to "跟随系统",
        AppState.ThemeOverride.LIGHT  to "浅色",
        AppState.ThemeOverride.DARK   to "深色",
    )
    AppCard(modifier = Modifier.fillMaxWidth()) {
        Column {
            Text(
                text = "主题",
                style = MaterialTheme.typography.titleSmall.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.SemiBold,
                ),
                modifier = Modifier.padding(bottom = Spacing.s.dp),
            )
            SingleChoiceSegmentedButtonRow(modifier = Modifier.fillMaxWidth()) {
                options.forEachIndexed { index, (value, label) ->
                    SegmentedButton(
                        selected = state.themeOverride == value,
                        onClick = { state.setTheme(value) },
                        shape = SegmentedButtonDefaults.itemShape(index = index, count = options.size),
                    ) { Text(label) }
                }
            }
        }
    }
}

@Composable
private fun DevicesCard() {
    val colors = LocalAppColors.current
    AppCard(modifier = Modifier.fillMaxWidth(), padding = 0) {
        Column(modifier = Modifier.fillMaxWidth()) {
            Text(
                text = "设备",
                style = MaterialTheme.typography.titleSmall.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.SemiBold,
                ),
                modifier = Modifier.padding(Spacing.l.dp),
            )
            Divider(color = colors.borderDefault)
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(Spacing.l.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Column(modifier = Modifier.weight(1f)) {
                    Text(
                        text = "当前设备",
                        style = MaterialTheme.typography.bodyLarge.copy(
                            color = colors.textPrimary,
                            fontWeight = FontWeight.SemiBold,
                        ),
                    )
                    Text(
                        text = "此刻活跃",
                        style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
                    )
                }
                AppBadge("本机", tone = BadgeTone.INFO)
            }
        }
    }
}

@Composable
private fun Footer() {
    val colors = LocalAppColors.current
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.fillMaxWidth().padding(top = Spacing.l.dp),
    ) {
        TextButton(onClick = { /* network/auth wiring lands later */ }) {
            Text(text = "退出登录", color = colors.textSecondary)
        }
        Text(
            text = "appunvs · v0.0.1 (dev)",
            style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
        )
    }
}

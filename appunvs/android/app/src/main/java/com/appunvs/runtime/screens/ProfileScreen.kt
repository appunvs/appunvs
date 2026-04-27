// ProfileScreen — account center.  Sections:
//
//   1. Account header (real email from /auth/me when available)
//   2. Today's usage (mock numbers — billing surface lands later)
//   3. Theme override (functional, persisted via DataStore)
//   4. Devices (real list from /auth/me)
//   5. Footer (sign-out + version)
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
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.BuildConfig
import com.appunvs.runtime.state.AppState
import com.appunvs.runtime.state.AuthRepo
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.AppBadge
import com.appunvs.runtime.ui.AppCard
import com.appunvs.runtime.ui.BadgeTone
import com.appunvs.runtime.ui.QuotaBar

@Composable
fun ProfileScreen(
    state: AppState,
    auth: AuthRepo,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current

    // Local toggle for the design-tokens preview reachable from
    // Footer's DEBUG-only link.  Pure local state so we don't have to
    // wire a NavController just for this dev affordance.
    var showTokens by remember { mutableStateOf(false) }
    if (showTokens) {
        Column(modifier = modifier.fillMaxSize().background(colors.bgPage)) {
            TextButton(onClick = { showTokens = false }) {
                Text(text = "← 返回", color = colors.brandDark)
            }
            TokensPreviewScreen()
        }
        return
    }

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

        AccountCard(auth = auth)
        UsageCard()
        ThemeCard(state = state)
        DevicesCard(auth = auth)
        Footer(auth = auth, onShowTokens = { showTokens = true })
    }
}

@Composable
private fun AccountCard(auth: AuthRepo) {
    val colors = LocalAppColors.current
    val email = auth.me?.email ?: "—"
    val name = email.substringBefore('@', missingDelimiterValue = "已登录")
    val initial = name.take(1).uppercase()
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
                    text = initial,
                    style = MaterialTheme.typography.titleLarge.copy(
                        color = colors.brandDark,
                        fontWeight = FontWeight.Bold,
                    ),
                )
            }
            Spacer(Modifier.width(Spacing.l.dp))
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = name,
                    style = MaterialTheme.typography.titleMedium.copy(color = colors.textPrimary),
                )
                Text(
                    text = email,
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
private fun DevicesCard(auth: AuthRepo) {
    val colors = LocalAppColors.current
    val devices = auth.me?.devices.orEmpty()
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
            if (devices.isEmpty()) {
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
            } else {
                devices.forEach { dev ->
                    Row(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(Spacing.l.dp),
                        verticalAlignment = Alignment.CenterVertically,
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = dev.platform,
                                style = MaterialTheme.typography.bodyLarge.copy(
                                    color = colors.textPrimary,
                                    fontWeight = FontWeight.SemiBold,
                                ),
                            )
                            Text(
                                text = dev.id,
                                style = MaterialTheme.typography.bodySmall.copy(
                                    color = colors.textSecondary,
                                    fontFamily = FontFamily.Monospace,
                                ),
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                    Divider(color = colors.borderDefault)
                }
            }
        }
    }
}

@Composable
private fun Footer(auth: AuthRepo, onShowTokens: () -> Unit) {
    val colors = LocalAppColors.current
    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = Modifier.fillMaxWidth().padding(top = Spacing.l.dp),
    ) {
        TextButton(onClick = { auth.signOut() }) {
            Text(text = "退出登录", color = colors.semanticDanger)
        }
        Text(
            text = "appunvs · v0.0.1 (dev)",
            style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
        )
        if (BuildConfig.DEBUG) {
            // Hidden behind DEBUG so the design-tokens preview ships
            // out of release builds.  Used while iterating on Theme.kt
            // / Typography.kt to see every token in one screen.
            TextButton(onClick = onShowTokens) {
                Text(
                    text = "Design tokens →",
                    color = colors.textSecondary,
                    style = MaterialTheme.typography.bodySmall,
                )
            }
        }
    }
}

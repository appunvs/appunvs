// BoxesScreen — entry tab.  Mirrors iOS BoxesView; replaces the
// previous Chat tab as the primary navigation.  Tapping a row drills
// into ChatScreen for that box (handled in MainActivity via
// chatBoxID state).
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.border
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Add
import androidx.compose.material.icons.outlined.Inventory2
import androidx.compose.material.icons.outlined.QrCodeScanner
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.alpha
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.net.BoxWire
import com.appunvs.runtime.state.BoxRepo
import com.appunvs.runtime.theme.AppType
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtime.ui.AppBadge
import com.appunvs.runtime.ui.BadgeTone
import com.appunvs.runtime.ui.EmptyState

@Composable
fun BoxesScreen(
    boxRepo: BoxRepo,
    onSelect: (String) -> Unit,
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
        // Header
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(
                text = "我的 Box",
                style = AppType.display.copy(color = colors.textPrimary),
            )
            Box(modifier = Modifier.weight(1f))
            IconButton(onClick = {
                boxRepo.create("new-box")
            }) {
                Icon(
                    Icons.Outlined.Add,
                    contentDescription = "新建 Box",
                    tint = colors.brandDark,
                )
            }
        }

        // List
        if (boxRepo.boxes.isEmpty()) {
            EmptyState(
                title = "没有 Box",
                hint = "点右上角 + 新建一个项目。",
            )
        } else {
            Column(verticalArrangement = Arrangement.spacedBy(Spacing.s.dp)) {
                boxRepo.boxes.forEach { box ->
                    BoxRow(
                        box = box,
                        isActive = box.boxID == boxRepo.activeBox?.boxID,
                        onClick = {
                            boxRepo.setActive(box)
                            onSelect(box.boxID)
                        },
                    )
                }
            }
        }

        // Scan hint
        Row(
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(Spacing.m.dp),
            modifier = Modifier
                .fillMaxWidth()
                .alpha(0.55f),
        ) {
            Icon(
                Icons.Outlined.QrCodeScanner,
                contentDescription = null,
                tint = colors.textSecondary,
            )
            Text(
                text = "扫码看别人的 app",
                style = AppType.body.copy(color = colors.textSecondary),
                modifier = Modifier.weight(1f),
            )
            AppBadge("即将上线", tone = BadgeTone.NEUTRAL)
        }
    }
}

@Composable
private fun BoxRow(
    box: BoxWire,
    isActive: Boolean,
    onClick: () -> Unit,
) {
    val colors = LocalAppColors.current
    Row(
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(Spacing.m.dp),
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .background(colors.bgCard, RoundedCornerShape(Radius.l.dp))
            .border(
                width = if (isActive) 1.dp else 0.5.dp,
                color = if (isActive) colors.brandDark else colors.borderDefault,
                shape = RoundedCornerShape(Radius.l.dp),
            )
            .padding(horizontal = Spacing.l.dp, vertical = Spacing.m.dp),
    ) {
        // Icon badge
        Box(
            modifier = Modifier
                .size(40.dp)
                .background(colors.brandPale, RoundedCornerShape(Radius.m.dp)),
            contentAlignment = Alignment.Center,
        ) {
            Icon(
                Icons.Outlined.Inventory2,
                contentDescription = null,
                tint = colors.brandDark,
                modifier = Modifier.size(20.dp),
            )
        }
        // Title + version
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = box.title,
                style = AppType.bodyEmphasis.copy(color = colors.textPrimary),
            )
            Text(
                text = "v" + box.currentVersion.ifEmpty { "—" },
                style = AppType.caption.copy(
                    color = colors.textSecondary,
                    fontFamily = androidx.compose.ui.text.font.FontFamily.Monospace,
                ),
            )
        }
        AppBadge(box.state, tone = badgeTone(box.state))
    }
}

private fun badgeTone(state: String): BadgeTone = when (state) {
    "published" -> BadgeTone.SUCCESS
    "draft"     -> BadgeTone.WARNING
    else        -> BadgeTone.NEUTRAL
}

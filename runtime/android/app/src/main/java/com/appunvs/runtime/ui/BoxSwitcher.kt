// BoxSwitcher — Chat header chip + ModalBottomSheet listing boxes.
// Tapping the chip opens a sheet with: the box list (active marked
// with a brand check), a "+ New box" row at the bottom, and a
// disabled "Scan QR" placeholder for the connector flow.
package com.appunvs.runtime.ui

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.widthIn
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.AddCircle
import androidx.compose.material.icons.filled.ArrowDropDown
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.QrCodeScanner
import androidx.compose.material3.Divider
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.launch

import com.appunvs.runtime.state.Box
import com.appunvs.runtime.state.BoxState
import com.appunvs.runtime.state.MockStore
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun BoxSwitcher(
    store: MockStore,
    onNewBox: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val colors = LocalAppColors.current
    var sheetOpen by remember { mutableStateOf(false) }
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()

    Row(
        modifier = modifier
            .clip(RoundedCornerShape(Radius.m.dp))
            .clickable { sheetOpen = true }
            .padding(horizontal = Spacing.s.dp, vertical = Spacing.xs.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(
            text = store.activeBox?.title ?: "选择 Box",
            style = MaterialTheme.typography.bodyLarge.copy(
                color = colors.textPrimary,
                fontWeight = FontWeight.SemiBold,
            ),
            maxLines = 1,
            modifier = Modifier.widthIn(max = 220.dp),
        )
        Icon(
            imageVector = Icons.Filled.ArrowDropDown,
            contentDescription = null,
            tint = colors.textSecondary,
        )
    }

    if (sheetOpen) {
        ModalBottomSheet(
            onDismissRequest = { sheetOpen = false },
            sheetState = sheetState,
            containerColor = colors.bgCard,
        ) {
            Column(modifier = Modifier.padding(bottom = Spacing.xl.dp)) {
                Text(
                    text = "我的 Box",
                    style = MaterialTheme.typography.titleMedium.copy(
                        color = colors.textPrimary,
                        fontWeight = FontWeight.Bold,
                    ),
                    modifier = Modifier.padding(horizontal = Spacing.l.dp, vertical = Spacing.s.dp),
                )

                store.boxes.forEach { box ->
                    BoxRow(
                        box = box,
                        isActive = box.id == store.activeBox?.id,
                        onTap = {
                            store.setActive(box)
                            scope.launch {
                                sheetState.hide()
                                sheetOpen = false
                            }
                        },
                    )
                    Divider(color = colors.borderDefault)
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable {
                            scope.launch {
                                sheetState.hide()
                                sheetOpen = false
                                onNewBox()
                            }
                        }
                        .padding(horizontal = Spacing.l.dp, vertical = Spacing.m.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        imageVector = Icons.Filled.AddCircle,
                        contentDescription = null,
                        tint = colors.brandDark,
                    )
                    Spacer(Modifier.width(Spacing.m.dp))
                    Text(
                        text = "新建 Box",
                        style = MaterialTheme.typography.bodyLarge.copy(
                            color = colors.textPrimary,
                            fontWeight = FontWeight.SemiBold,
                        ),
                    )
                }

                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = Spacing.l.dp, vertical = Spacing.m.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Icon(
                        imageVector = Icons.Filled.QrCodeScanner,
                        contentDescription = null,
                        tint = colors.textSecondary,
                    )
                    Spacer(Modifier.width(Spacing.m.dp))
                    Text(
                        text = "扫码看别人的 app",
                        style = MaterialTheme.typography.bodyLarge.copy(
                            color = colors.textSecondary,
                            fontWeight = FontWeight.SemiBold,
                        ),
                    )
                    Spacer(Modifier.weight(1f))
                    AppBadge("即将上线", tone = BadgeTone.NEUTRAL)
                }
            }
        }
    }
}

@Composable
private fun BoxRow(
    box: Box,
    isActive: Boolean,
    onTap: () -> Unit,
) {
    val colors = LocalAppColors.current
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onTap() }
            .background(if (isActive) colors.brandPale else colors.bgCard)
            .padding(horizontal = Spacing.l.dp, vertical = Spacing.m.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(
                text = box.title,
                style = MaterialTheme.typography.bodyLarge.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.SemiBold,
                ),
                maxLines = 1,
            )
            Text(
                text = "v${box.currentVersion.ifEmpty { "—" }}",
                style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
            )
        }
        AppBadge(box.state.label, tone = box.state.tone)
        if (isActive) {
            Spacer(Modifier.width(Spacing.s.dp))
            Icon(
                imageVector = Icons.Filled.Check,
                contentDescription = "active",
                tint = colors.brandDark,
            )
        }
    }
}

private val BoxState.label: String
    get() = when (this) {
        BoxState.DRAFT     -> "draft"
        BoxState.PUBLISHED -> "published"
        BoxState.ARCHIVED  -> "archived"
    }

private val BoxState.tone: BadgeTone
    get() = when (this) {
        BoxState.DRAFT     -> BadgeTone.WARNING
        BoxState.PUBLISHED -> BadgeTone.SUCCESS
        BoxState.ARCHIVED  -> BadgeTone.NEUTRAL
    }

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun NewBoxSheet(
    onDismiss: () -> Unit,
    onCreate: (String) -> Unit,
) {
    val colors = LocalAppColors.current
    val sheetState = rememberModalBottomSheetState()
    val scope = rememberCoroutineScope()
    var title by remember { mutableStateOf("") }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
        containerColor = colors.bgCard,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(Spacing.l.dp),
            verticalArrangement = Arrangement.spacedBy(Spacing.m.dp),
        ) {
            Text(
                text = "新建 Box",
                style = MaterialTheme.typography.titleMedium.copy(
                    color = colors.textPrimary,
                    fontWeight = FontWeight.Bold,
                ),
            )
            androidx.compose.material3.OutlinedTextField(
                value = title,
                onValueChange = { title = it },
                placeholder = { Text("比如 todo-app") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
            )
            Text(
                text = "Box 是一个独立项目, 对话历史和源代码都和它绑定。",
                style = MaterialTheme.typography.bodySmall.copy(color = colors.textSecondary),
            )
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(Spacing.s.dp),
            ) {
                androidx.compose.material3.OutlinedButton(
                    onClick = {
                        scope.launch {
                            sheetState.hide()
                            onDismiss()
                        }
                    },
                    modifier = Modifier.weight(1f),
                ) { Text("取消") }
                androidx.compose.material3.Button(
                    onClick = {
                        val t = title.trim()
                        if (t.isNotEmpty()) {
                            onCreate(t)
                            scope.launch {
                                sheetState.hide()
                                onDismiss()
                            }
                        }
                    },
                    enabled = title.trim().isNotEmpty(),
                    modifier = Modifier.weight(1f),
                ) { Text("创建") }
            }
        }
    }
}

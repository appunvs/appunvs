// StageScreen — D2.e ties Stage to the active Box.  When the user
// switches Boxes via the Chat tab's BoxSwitcher, Stage tracks
// `boxRepo.activeBox` and reloads RuntimeView with the new box's
// bundle URL.
//
// The bundle URL today is *derived* from the box id (and the
// configured relay base URL) since the box-list endpoint doesn't
// surface BundleRef.uri on the list shape — only the `/box/{id}`
// detail endpoint does.  D3 either fetches that detail when active
// box changes, or BoxRepo grows a per-box-detail cache.  This screen's
// contract stays "give me the URL for the active box and I'll mount
// it."
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.PlayCircleOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView

import com.appunvs.runtime.net.NetConfig
import com.appunvs.runtime.state.BoxRepo
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtimesdk.RuntimeView

@Composable
fun StageScreen(
    boxRepo: BoxRepo,
    modifier: Modifier = Modifier,
) {
    val activeBox = boxRepo.activeBox
    val bundleUrl = remember(activeBox) {
        activeBox?.let { box ->
            val version = box.currentVersion.ifEmpty { "draft" }
            "${NetConfig.relayBaseURL}/_artifacts/${box.boxID}/$version/index.bundle"
        }
    }

    if (bundleUrl == null) {
        NoBoxState(modifier = modifier)
        return
    }

    AndroidView(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black),
        factory = { ctx ->
            RuntimeView(ctx).also { view ->
                view.loadBundle(bundleUrl, null)
            }
        },
        update = { view ->
            if (view.currentBundleURL != bundleUrl) {
                view.loadBundle(bundleUrl, null)
            }
        },
    )
}

@Composable
private fun NoBoxState(modifier: Modifier = Modifier) {
    val colors = LocalAppColors.current
    Box(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black),
        contentAlignment = Alignment.Center,
    ) {
        Column(
            modifier = Modifier.padding(Spacing.xxl.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(Spacing.m.dp),
        ) {
            Icon(
                imageVector = Icons.Outlined.PlayCircleOutline,
                contentDescription = null,
                tint = Color(0xFF707070),
            )
            Text(
                text = "挑一个 Box",
                style = MaterialTheme.typography.titleLarge.copy(color = Color.White),
            )
            Text(
                text = "从 Chat tab 顶上的 Box 切换器选一个，bundle 在这里跑。",
                style = MaterialTheme.typography.bodyMedium.copy(color = Color(0xFFB0B0B0)),
                textAlign = TextAlign.Center,
            )
        }
    }
}

// StageScreen — D2.d wires the host Stage tab to mount the runtime
// SDK's RuntimeView (a FrameLayout subclass) via Compose's AndroidView.
//
// Today the bundle URL is hardcoded so the host can be visually
// verified end-to-end without the relay / Box flow.  D2.e replaces
// the hardcoded URL with the active box's bundle URL from BoxRepo
// and reacts to changes (loadBundle on update).
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.viewinterop.AndroidView

import com.appunvs.runtimesdk.RuntimeView

/// Hardcoded bundle URL for D2.d.  D2.e binds this to the active
/// box's bundle URL from BoxRepo.activeBox?.currentVersion.
private const val DEMO_BUNDLE_URL =
    "https://relay.example/_artifacts/box_demo/v1/index.bundle"

@Composable
fun StageScreen(modifier: Modifier = Modifier) {
    AndroidView(
        modifier = modifier
            .fillMaxSize()
            .background(Color.Black),
        factory = { ctx ->
            RuntimeView(ctx).also { view ->
                view.loadBundle(DEMO_BUNDLE_URL, null)
            }
        },
        update = { view ->
            // No-op for D2.d — bundle URL is constant.  D2.e reads
            // the active box's URL from a remembered MutableState
            // and calls view.loadBundle when it changes.
            if (view.currentBundleURL != DEMO_BUNDLE_URL) {
                view.loadBundle(DEMO_BUNDLE_URL, null)
            }
        },
    )
}

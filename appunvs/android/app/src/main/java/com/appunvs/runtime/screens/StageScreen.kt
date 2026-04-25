// StageScreen — D2.b smoke test.  The host now links the runtime SDK
// AAR (built from runtime/sdk/android/) and calls into its hello
// method to prove the linkage works end-to-end.
//
// D2.c widens the SDK to expose a real RuntimeView; D2.d mounts that
// here in place of the hello text.  D2.e wires the active Box's
// bundle URL through to RuntimeView.loadBundle(...).
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing
import com.appunvs.runtimesdk.RuntimeSDK

@Composable
fun StageScreen(modifier: Modifier = Modifier) {
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
        ) {
            Icon(
                imageVector = Icons.Outlined.PlayCircleOutline,
                contentDescription = null,
                tint = colors.brandLight,
                modifier = Modifier.padding(bottom = Spacing.s.dp),
            )
            Text(
                text = "Stage",
                style = MaterialTheme.typography.titleLarge.copy(color = Color.White),
            )
            Text(
                text = RuntimeSDK.hello(),
                style = MaterialTheme.typography.bodyMedium.copy(
                    color = Color(0xFFB0B0B0),
                    fontFamily = FontFamily.Monospace,
                ),
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = Spacing.m.dp),
            )
            Text(
                text = "D2.c will replace this with a real RuntimeView mount.",
                style = MaterialTheme.typography.bodySmall.copy(color = Color(0xFF707070)),
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = Spacing.s.dp),
            )
        }
    }
}

// StageScreen — placeholder.  PR D wires a native AndroidView that
// hosts a sandboxed Hermes runtime to render the active Box's
// bundle.  Today shows an empty-state pointing forward.
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
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Spacing

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
                text = "待 PR D: SubRuntime native module 接入。",
                style = MaterialTheme.typography.bodyMedium.copy(color = Color(0xFFB0B0B0)),
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(top = Spacing.m.dp),
            )
        }
    }
}

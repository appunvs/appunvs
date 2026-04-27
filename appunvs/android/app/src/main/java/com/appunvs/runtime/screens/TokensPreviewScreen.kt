// TokensPreviewScreen — reachable from Profile in DEBUG builds.
// Mirrors iOS's TokensPreviewView: every design token rendered in one
// scroll surface so iterating on the design system is a tight loop.
//
// Sections:
//   - Color tokens: brand / text / surface / semantic
//   - Spacing scale: bars sized to each value
//   - Radius scale: rounded squares at each radius
//   - Typography scale: a sample sentence in each token
package com.appunvs.runtime.screens

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.unit.dp

import com.appunvs.runtime.theme.AppColors
import com.appunvs.runtime.theme.AppType
import com.appunvs.runtime.theme.LocalAppColors
import com.appunvs.runtime.theme.Radius
import com.appunvs.runtime.theme.Spacing

@Composable
fun TokensPreviewScreen(modifier: Modifier = Modifier) {
    val colors = LocalAppColors.current
    Column(
        modifier = modifier
            .fillMaxSize()
            .background(colors.bgPage)
            .verticalScroll(rememberScrollState())
            .padding(Spacing.l.dp),
        verticalArrangement = Arrangement.spacedBy(Spacing.xxl.dp),
    ) {
        ColorsSection(colors)
        SpacingSection(colors)
        RadiusSection(colors)
        TypographySection(colors)
    }
}

@Composable
private fun ColorsSection(colors: AppColors) {
    SectionHeader("Color", colors)
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.s.dp)) {
        ColorRow("brandDark",       colors.brandDark, colors)
        ColorRow("brandLight",      colors.brandLight, colors)
        ColorRow("brandPale",       colors.brandPale, colors)
        TokenDivider(colors)
        ColorRow("textPrimary",     colors.textPrimary, colors)
        ColorRow("textSecondary",   colors.textSecondary, colors)
        TokenDivider(colors)
        ColorRow("bgPage",          colors.bgPage, colors)
        ColorRow("bgCard",          colors.bgCard, colors)
        ColorRow("bgInput",         colors.bgInput, colors)
        ColorRow("borderDefault",   colors.borderDefault, colors)
        TokenDivider(colors)
        ColorRow("semanticSuccess", colors.semanticSuccess, colors)
        ColorRow("semanticWarning", colors.semanticWarning, colors)
        ColorRow("semanticDanger",  colors.semanticDanger, colors)
        ColorRow("semanticInfo",    colors.semanticInfo, colors)
    }
}

@Composable
private fun SpacingSection(colors: AppColors) {
    SectionHeader("Spacing", colors)
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.s.dp)) {
        SpacingRow("xs",   Spacing.xs,   colors)
        SpacingRow("s",    Spacing.s,    colors)
        SpacingRow("m",    Spacing.m,    colors)
        SpacingRow("l",    Spacing.l,    colors)
        SpacingRow("xl",   Spacing.xl,   colors)
        SpacingRow("xxl",  Spacing.xxl,  colors)
        SpacingRow("xxxl", Spacing.xxxl, colors)
        SpacingRow("huge", Spacing.huge, colors)
    }
}

@Composable
private fun RadiusSection(colors: AppColors) {
    SectionHeader("Radius", colors)
    Row(horizontalArrangement = Arrangement.spacedBy(Spacing.m.dp)) {
        RadiusSwatch("s",    Radius.s,    colors)
        RadiusSwatch("m",    Radius.m,    colors)
        RadiusSwatch("l",    Radius.l,    colors)
        RadiusSwatch("xl",   Radius.xl,   colors)
        RadiusSwatch("pill", Radius.pill, colors)
    }
}

@Composable
private fun TypographySection(colors: AppColors) {
    SectionHeader("Typography", colors)
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.s.dp)) {
        TypeRow("display",      AppType.display,      colors)
        TypeRow("title",        AppType.title,        colors)
        TypeRow("heading",      AppType.heading,      colors)
        TypeRow("body",         AppType.body,         colors)
        TypeRow("bodyEmphasis", AppType.bodyEmphasis, colors)
        TypeRow("caption",      AppType.caption,      colors)
        TypeRow("label",        AppType.label,        colors)
        TypeRow("mono",         AppType.mono,         colors)
    }
}

// MARK: - Building blocks

@Composable
private fun SectionHeader(title: String, colors: AppColors) {
    Text(
        text = title,
        style = AppType.heading,
        color = colors.textPrimary,
        modifier = Modifier.padding(bottom = Spacing.xs.dp),
    )
}

@Composable
private fun TokenDivider(colors: AppColors) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .height(1.dp)
            .background(colors.borderDefault)
            .padding(vertical = Spacing.xs.dp),
    )
}

@Composable
private fun ColorRow(name: String, color: Color, colors: AppColors) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(Spacing.m.dp),
    ) {
        Box(
            modifier = Modifier
                .size(36.dp)
                .background(color, RoundedCornerShape(Radius.s.dp)),
        )
        Column {
            Text(
                text = name,
                style = AppType.bodyEmphasis,
                color = colors.textPrimary,
            )
            Text(
                text = "color: $color",
                style = AppType.caption,
                color = colors.textSecondary,
            )
        }
    }
}

@Composable
private fun SpacingRow(name: String, value: Int, colors: AppColors) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(Spacing.m.dp),
    ) {
        Text(
            text = name,
            style = AppType.bodyEmphasis,
            color = colors.textPrimary,
            modifier = Modifier.width(48.dp),
        )
        Box(
            modifier = Modifier
                .width(value.dp)
                .height(18.dp)
                .background(colors.brandDark),
        )
        Spacer(modifier = Modifier.width(Spacing.s.dp))
        Text(
            text = "${value}dp",
            style = AppType.caption,
            color = colors.textSecondary,
        )
    }
}

@Composable
private fun RadiusSwatch(name: String, value: Int, colors: AppColors) {
    Column(verticalArrangement = Arrangement.spacedBy(Spacing.xs.dp)) {
        Box(
            modifier = Modifier
                .size(56.dp)
                .background(
                    colors.brandLight,
                    RoundedCornerShape(minOf(value, 28).dp),
                ),
        )
        Text(
            text = name,
            style = AppType.caption,
            color = colors.textSecondary,
        )
    }
}

@Composable
private fun TypeRow(name: String, style: TextStyle, colors: AppColors) {
    Column {
        Text(
            text = "The quick brown fox",
            style = style,
            color = colors.textPrimary,
        )
        Text(
            text = name,
            style = AppType.caption,
            color = colors.textSecondary,
        )
    }
}

// Theme color tokens — port of the prior RN tokens (`app/src/theme/colors.ts`)
// to Compose `Color`.  We expose two parallel palette objects (`LightColors`
// + `DarkColors`); `RuntimeTheme` picks one based on the user's override
// and the system trait.
package com.appunvs.runtime.theme

import androidx.compose.ui.graphics.Color

/// Strongly-typed token surface so screens can refer to
/// `colors.brandDark` instead of `Color(0xFF...)` and theme switches
/// stay in one place.
data class AppColors(
    val brandDark: Color,
    val brandLight: Color,
    val brandPale: Color,
    val textPrimary: Color,
    val textSecondary: Color,
    val bgPage: Color,
    val bgCard: Color,
    val bgInput: Color,
    val borderDefault: Color,
    val semanticSuccess: Color,
    val semanticWarning: Color,
    val semanticDanger: Color,
    val semanticInfo: Color,
)

val LightColors = AppColors(
    brandDark       = Color(0xFF0B505A),
    brandLight      = Color(0xFF6FC0CC),
    brandPale       = Color(0xFFE9F4F5),
    textPrimary     = Color(0xFF152127),
    textSecondary   = Color(0xFF557280),
    bgPage          = Color(0xFFF2F6F6),
    bgCard          = Color(0xFFFFFFFF),
    bgInput         = Color(0xFFE9EFF0),
    borderDefault   = Color(0xFFDAE4E6),
    semanticSuccess = Color(0xFF1F7A4D),
    semanticWarning = Color(0xFFA65A0E),
    semanticDanger  = Color(0xFFB23A3A),
    semanticInfo    = Color(0xFF155E96),
)

val DarkColors = AppColors(
    brandDark       = Color(0xFF4FB0BE),
    brandLight      = Color(0xFF167C8C),
    brandPale       = Color(0xFF14353B),
    textPrimary     = Color(0xFFE8F0F2),
    textSecondary   = Color(0xFF9AB0B8),
    bgPage          = Color(0xFF0B1418),
    bgCard          = Color(0xFF152127),
    bgInput         = Color(0xFF1E2D33),
    borderDefault   = Color(0xFF243339),
    semanticSuccess = Color(0xFF5BD391),
    semanticWarning = Color(0xFFF0B45A),
    semanticDanger  = Color(0xFFF08585),
    semanticInfo    = Color(0xFF7CB6E5),
)

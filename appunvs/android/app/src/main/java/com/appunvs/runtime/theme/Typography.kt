// Typography — shared semantic type tokens with the iOS side.  Names
// are the same (display / title / heading / body / bodyEmphasis /
// caption / label / mono); values are explicit because Compose has no
// native equivalent of Apple's Dynamic Type scale, so we hand-pick sp
// values that visually match iOS at default text-size settings.
//
// Future: respect the user's accessibility scale via
// `LocalDensity.current.fontScale`.  Out of scope for this baseline.
package com.appunvs.runtime.theme

import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

/// Typography scale.  Each token is a [TextStyle] you can pass directly
/// to `Text(style = AppType.body)`.  Values mirror iOS's
/// `Typography.swift` so the two platforms render at the same visual
/// rank.
object AppType {
    /// Page hero — splash / onboarding marquee.
    val display = TextStyle(
        fontSize = 28.sp,
        fontWeight = FontWeight.Bold,
        lineHeight = 36.sp,
    )

    /// Tab / screen title.
    val title = TextStyle(
        fontSize = 22.sp,
        fontWeight = FontWeight.Bold,
        lineHeight = 28.sp,
    )

    /// Section heading inside a screen.
    val heading = TextStyle(
        fontSize = 17.sp,
        fontWeight = FontWeight.SemiBold,
        lineHeight = 22.sp,
    )

    /// Default reading copy.
    val body = TextStyle(
        fontSize = 17.sp,
        fontWeight = FontWeight.Normal,
        lineHeight = 22.sp,
    )

    /// Body with emphasis (button labels, primary metadata).
    val bodyEmphasis = TextStyle(
        fontSize = 17.sp,
        fontWeight = FontWeight.SemiBold,
        lineHeight = 22.sp,
    )

    /// Tertiary / metadata text under primary content.
    val caption = TextStyle(
        fontSize = 12.sp,
        fontWeight = FontWeight.Normal,
        lineHeight = 16.sp,
    )

    /// Tag / pill / form-field label.
    val label = TextStyle(
        fontSize = 13.sp,
        fontWeight = FontWeight.SemiBold,
        lineHeight = 16.sp,
    )

    /// Code / IDs / hashes — fixed pitch.
    val mono = TextStyle(
        fontSize = 14.sp,
        fontWeight = FontWeight.Normal,
        fontFamily = FontFamily.Monospace,
        lineHeight = 20.sp,
    )
}

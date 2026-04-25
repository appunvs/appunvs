// RuntimeTheme — wraps Material3 ColorScheme + provides our typed token
// surface via CompositionLocal.  Screens reach for `LocalAppColors.current`
// to get the strongly-typed palette; for any Material component we also
// derive a `ColorScheme` so M3 renders sensible defaults.
package com.appunvs.runtime.theme

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.CompositionLocalProvider
import androidx.compose.runtime.staticCompositionLocalOf

import com.appunvs.runtime.state.AppState

val LocalAppColors = staticCompositionLocalOf { LightColors }

/// Spacing scale — keep numbers out of view code.
object Spacing {
    const val xs   = 4
    const val s    = 8
    const val m    = 12
    const val l    = 16
    const val xl   = 20
    const val xxl  = 24
    const val xxxl = 32
    const val huge = 48
}

/// Corner radius scale.  `xl` reserved for chat bubbles.
object Radius {
    const val s    = 6
    const val m    = 10
    const val l    = 12
    const val xl   = 14
    const val pill = 999
}

@Composable
fun RuntimeTheme(
    themeOverride: AppState.ThemeOverride = AppState.ThemeOverride.SYSTEM,
    content: @Composable () -> Unit,
) {
    val isDark = when (themeOverride) {
        AppState.ThemeOverride.SYSTEM -> isSystemInDarkTheme()
        AppState.ThemeOverride.LIGHT  -> false
        AppState.ThemeOverride.DARK   -> true
    }
    val palette = if (isDark) DarkColors else LightColors

    val materialScheme = if (isDark) {
        darkColorScheme(
            primary       = palette.brandDark,
            secondary     = palette.brandLight,
            background    = palette.bgPage,
            surface       = palette.bgCard,
            onPrimary     = palette.bgPage,
            onSecondary   = palette.textPrimary,
            onBackground  = palette.textPrimary,
            onSurface     = palette.textPrimary,
        )
    } else {
        lightColorScheme(
            primary       = palette.brandDark,
            secondary     = palette.brandLight,
            background    = palette.bgPage,
            surface       = palette.bgCard,
            onPrimary     = androidx.compose.ui.graphics.Color.White,
            onSecondary   = palette.textPrimary,
            onBackground  = palette.textPrimary,
            onSurface     = palette.textPrimary,
        )
    }

    CompositionLocalProvider(LocalAppColors provides palette) {
        MaterialTheme(colorScheme = materialScheme, content = content)
    }
}

// Public theme surface.  Importers should never reach into colors.ts /
// typography.ts directly; everything flows through `useTheme()`.

export { ThemeProvider, useTheme } from './ThemeProvider';
export { useThemeOverrideStore, type ThemeOverride } from './store';
export type { Theme } from './tokens';
export type { Colors } from './colors';
export type { TypographyVariantName } from './typography';

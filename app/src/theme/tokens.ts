// Combined theme object consumed by every styled component via
// `useTheme()`.  Keep this surface stable: components branch on
// `theme.scheme` rather than reading any platform / Appearance API
// directly, so a future "always-dark mode for connector flow" or
// per-Box theming hook drops in here without touching consumers.

import { darkColors, lightColors, type Colors } from './colors';
import { radius } from './radius';
import { spacing } from './spacing';
import { typography } from './typography';

export interface Theme {
  scheme:     'light' | 'dark';
  colors:     Colors;
  spacing:    typeof spacing;
  radius:     typeof radius;
  typography: typeof typography;
}

export const lightTheme: Theme = {
  scheme:     'light',
  colors:     lightColors,
  spacing,
  radius,
  typography,
};

export const darkTheme: Theme = {
  scheme:     'dark',
  colors:     darkColors,
  spacing,
  radius,
  typography,
};

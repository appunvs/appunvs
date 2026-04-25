// Typography variants.  Body sizes are deliberately small (15/13) so the
// chat tab can fit dense conversation comfortably; bump up if user
// research says so.  Custom fonts (Inter, Noto Sans SC) are intentionally
// not loaded in v1 — we lean on platform system fonts to keep startup
// fast and avoid an `expo-font` setup pass.  Add when accent typography
// becomes a brand requirement.

import { Platform, type TextStyle } from 'react-native';

const monoFamily = Platform.select({
  ios:     'Menlo',
  android: 'monospace',
  default: 'JetBrains Mono, SF Mono, Menlo, Consolas, ui-monospace, monospace',
});

export interface TypographyVariant {
  fontSize:   number;
  fontWeight: TextStyle['fontWeight'];
  lineHeight: number;
  fontFamily?: string;
  letterSpacing?: number;
}

export const typography: Record<
  'displayLg' | 'h1' | 'h2' | 'h3' |
  'body' | 'bodyStrong' |
  'caption' | 'captionStrong' |
  'mono' | 'monoSm',
  TypographyVariant
> = {
  displayLg:     { fontSize: 32, fontWeight: '700', lineHeight: 40 },
  h1:            { fontSize: 24, fontWeight: '700', lineHeight: 30 },
  h2:            { fontSize: 19, fontWeight: '700', lineHeight: 26 },
  h3:            { fontSize: 16, fontWeight: '600', lineHeight: 22 },
  body:          { fontSize: 15, fontWeight: '400', lineHeight: 22 },
  bodyStrong:    { fontSize: 15, fontWeight: '600', lineHeight: 22 },
  caption:       { fontSize: 13, fontWeight: '400', lineHeight: 18 },
  captionStrong: { fontSize: 13, fontWeight: '600', lineHeight: 18 },
  mono:          { fontSize: 13, fontWeight: '400', lineHeight: 18, fontFamily: monoFamily },
  monoSm:        { fontSize: 12, fontWeight: '400', lineHeight: 16, fontFamily: monoFamily },
};

export type TypographyVariantName = keyof typeof typography;

// Themed Text — wraps RN Text with a typography variant + colour token.
// All in-app prose should go through this, never raw `<Text>`.
import { Text as RNText, type TextProps as RNTextProps, type TextStyle, type StyleProp } from 'react-native';

import { useTheme } from '@/theme';
import type { Colors } from '@/theme';
import type { TypographyVariantName } from '@/theme/typography';

type ColorToken = keyof Colors;

export interface TextProps extends RNTextProps {
  variant?: TypographyVariantName;   // defaults to body
  color?: ColorToken;                // defaults to textPrimary
  align?: TextStyle['textAlign'];
}

export function Text({
  variant = 'body',
  color = 'textPrimary',
  align,
  style,
  children,
  ...rest
}: TextProps) {
  const theme = useTheme();
  const variantStyle = theme.typography[variant];
  const composed: StyleProp<TextStyle> = [
    variantStyle,
    { color: theme.colors[color], textAlign: align },
    style,
  ];
  return <RNText style={composed} {...rest}>{children}</RNText>;
}

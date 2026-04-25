// Themed Button.  Three variants:
//
//   primary   — solid brandDark on filled background (CTA)
//   secondary — outlined, low-contrast bg, brand text (Cancel, secondary action)
//   ghost     — text-only, no background until pressed (toolbar, list rows)
//
// Sizes: sm / md (default) / lg.  Loading state replaces children with a
// spinner and disables press.  Icons go before/after text via `iconLeft`
// / `iconRight` slots — render them with vector-icons in the caller, the
// button just slots them.
import { ActivityIndicator, Pressable, StyleSheet, type PressableProps, type ViewStyle, type StyleProp } from 'react-native';
import { useState, type ReactNode } from 'react';

import { useTheme } from '@/theme';
import { Text } from './Text';

type Variant = 'primary' | 'secondary' | 'ghost';
type Size = 'sm' | 'md' | 'lg';

export interface ButtonProps extends Omit<PressableProps, 'children' | 'style'> {
  label: string;
  variant?: Variant;
  size?: Size;
  loading?: boolean;
  iconLeft?: ReactNode;
  iconRight?: ReactNode;
  fullWidth?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function Button({
  label,
  variant = 'primary',
  size = 'md',
  loading = false,
  iconLeft,
  iconRight,
  fullWidth,
  disabled,
  style,
  ...rest
}: ButtonProps) {
  const theme = useTheme();
  const [pressed, setPressed] = useState(false);

  const sizing = (() => {
    switch (size) {
      case 'sm': return { padV: theme.spacing.xs, padH: theme.spacing.m, gap: theme.spacing.xs };
      case 'lg': return { padV: theme.spacing.m,  padH: theme.spacing.xl, gap: theme.spacing.s  };
      default:   return { padV: theme.spacing.s,  padH: theme.spacing.l, gap: theme.spacing.s  };
    }
  })();

  // Resolve the (bg, fg, border) triplet for this state.  All three are
  // raw hex strings — no theme-colors-key indirection — so consumers
  // (Text style, ActivityIndicator color) can use them directly.
  const palette: { bg: string; fg: string; border: string } = (() => {
    const c = theme.colors;
    if (disabled) {
      return { bg: c.bgInput, fg: c.textSecondary, border: 'transparent' };
    }
    switch (variant) {
      case 'primary':
        return {
          bg: pressed ? c.brandLight : c.brandDark,
          fg: theme.scheme === 'light' ? '#FFFFFF' : c.bgPage,
          border: 'transparent',
        };
      case 'secondary':
        return {
          bg: pressed ? c.brandPale : 'transparent',
          fg: c.brandDark,
          border: c.brandDark,
        };
      case 'ghost':
      default:
        return {
          bg: pressed ? c.brandPale : 'transparent',
          fg: c.textPrimary,
          border: 'transparent',
        };
    }
  })();

  const composed: StyleProp<ViewStyle> = [
    styles.base,
    {
      paddingVertical: sizing.padV,
      paddingHorizontal: sizing.padH,
      backgroundColor: palette.bg,
      borderColor: palette.border,
      borderWidth: variant === 'secondary' ? 1 : 0,
      borderRadius: theme.radius.m,
      gap: sizing.gap,
      width: fullWidth ? '100%' : undefined,
      opacity: disabled || loading ? 0.7 : 1,
    },
    style,
  ];

  const labelVariant = size === 'sm' ? 'captionStrong' : 'bodyStrong';

  return (
    <Pressable
      accessibilityRole="button"
      onPressIn={() => setPressed(true)}
      onPressOut={() => setPressed(false)}
      disabled={disabled || loading}
      style={composed}
      {...rest}
    >
      {loading
        ? <ActivityIndicator size="small" color={palette.fg} />
        : <>
            {iconLeft}
            <Text variant={labelVariant} style={{ color: palette.fg }}>{label}</Text>
            {iconRight}
          </>}
    </Pressable>
  );
}

const styles = StyleSheet.create({
  base: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
  },
});

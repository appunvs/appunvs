// Surface primitive.  Wraps content in a themed card with optional
// border + padding + radius.  Don't reach for `View` + manual styling
// for grouped content; use Card so the borders / radius tokenize.
import { View, type ViewProps, type ViewStyle, type StyleProp } from 'react-native';

import { useTheme } from '@/theme';
import type { SpacingKey } from '@/theme/spacing';
import type { RadiusKey } from '@/theme/radius';

export interface CardProps extends ViewProps {
  padding?: SpacingKey | 'none';
  radius?: RadiusKey;
  bordered?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function Card({
  padding = 'l',
  radius: radiusToken = 'l',
  bordered = false,
  style,
  children,
  ...rest
}: CardProps) {
  const theme = useTheme();
  const composed: StyleProp<ViewStyle> = [
    {
      backgroundColor: theme.colors.bgCard,
      borderRadius: theme.radius[radiusToken],
      padding: padding === 'none' ? 0 : theme.spacing[padding],
      borderWidth: bordered ? 1 : 0,
      borderColor: bordered ? theme.colors.borderDefault : undefined,
    },
    style,
  ];
  return <View style={composed} {...rest}>{children}</View>;
}

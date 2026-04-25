// Square pressable for icon-only actions (toolbar, header, sheet rows).
// Uses the same press-state palette as Button ghost variant.
import { Pressable, type PressableProps, type StyleProp, type ViewStyle } from 'react-native';
import { useState, type ReactNode } from 'react';

import { useTheme } from '@/theme';

type Size = 'sm' | 'md' | 'lg';

export interface IconButtonProps extends Omit<PressableProps, 'children' | 'style'> {
  children: ReactNode;
  size?: Size;
  variant?: 'ghost' | 'filled';
  style?: StyleProp<ViewStyle>;
}

export function IconButton({
  children,
  size = 'md',
  variant = 'ghost',
  disabled,
  style,
  ...rest
}: IconButtonProps) {
  const theme = useTheme();
  const [pressed, setPressed] = useState(false);

  const dim = size === 'sm' ? 32 : size === 'lg' ? 48 : 40;
  const bg = (() => {
    if (disabled) return 'transparent';
    if (variant === 'filled') {
      return pressed ? theme.colors.brandLight : theme.colors.brandDark;
    }
    return pressed ? theme.colors.brandPale : 'transparent';
  })();

  return (
    <Pressable
      accessibilityRole="button"
      onPressIn={() => setPressed(true)}
      onPressOut={() => setPressed(false)}
      disabled={disabled}
      style={[
        {
          width: dim,
          height: dim,
          borderRadius: theme.radius.m,
          backgroundColor: bg,
          alignItems: 'center',
          justifyContent: 'center',
          opacity: disabled ? 0.5 : 1,
        },
        style,
      ]}
      {...rest}
    >
      {children}
    </Pressable>
  );
}

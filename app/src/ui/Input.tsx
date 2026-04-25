// Themed TextInput.  Wraps RN TextInput with consistent paddings,
// background, focus ring, and placeholder color.  Use for every form
// field; don't compose raw <TextInput>.
import { useState } from 'react';
import {
  TextInput,
  StyleSheet,
  type TextInputProps,
  type StyleProp,
  type ViewStyle,
} from 'react-native';

import { useTheme } from '@/theme';

export interface InputProps extends Omit<TextInputProps, 'style'> {
  style?: StyleProp<ViewStyle>;
  invalid?: boolean;        // red border when something upstream failed validation
}

export function Input({
  style,
  invalid,
  onFocus,
  onBlur,
  multiline,
  ...rest
}: InputProps) {
  const theme = useTheme();
  const [focused, setFocused] = useState(false);

  const composed = [
    styles.base,
    {
      backgroundColor: theme.colors.bgInput,
      borderColor: invalid
        ? theme.colors.semanticDanger
        : focused
        ? theme.colors.brandDark
        : theme.colors.borderDefault,
      borderRadius: theme.radius.m,
      color: theme.colors.textPrimary,
      paddingVertical: multiline ? theme.spacing.m : theme.spacing.s,
      paddingHorizontal: theme.spacing.m,
      minHeight: multiline ? 64 : 40,
    },
    theme.typography.body,
    style as object,
  ];

  return (
    <TextInput
      {...rest}
      multiline={multiline}
      placeholderTextColor={theme.colors.textSecondary}
      onFocus={(e) => { setFocused(true); onFocus?.(e); }}
      onBlur={(e) => { setFocused(false); onBlur?.(e); }}
      style={composed}
    />
  );
}

const styles = StyleSheet.create({
  base: {
    borderWidth: 1,
  },
});

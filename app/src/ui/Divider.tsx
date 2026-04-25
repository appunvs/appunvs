// Horizontal divider — 1px hairline using the theme border token.
// Use for separating list rows / sections inside a Card.
import { View, type ViewStyle, type StyleProp } from 'react-native';

import { useTheme } from '@/theme';

export function Divider({ style }: { style?: StyleProp<ViewStyle> }) {
  const theme = useTheme();
  return (
    <View
      style={[
        {
          height: 1,
          backgroundColor: theme.colors.borderDefault,
        },
        style,
      ]}
    />
  );
}

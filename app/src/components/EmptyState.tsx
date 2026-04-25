// Standardized empty / zero-state.  Every screen with a "nothing yet"
// shape goes through this — keeps voice and layout consistent.
import { View, type StyleProp, type ViewStyle } from 'react-native';
import type { ReactNode } from 'react';

import { useTheme } from '@/theme';
import { Text } from '@/ui';

export interface EmptyStateProps {
  title: string;
  hint?: string;
  action?: ReactNode;       // typically a <Button> from @/ui
  style?: StyleProp<ViewStyle>;
}

export function EmptyState({ title, hint, action, style }: EmptyStateProps) {
  const theme = useTheme();
  return (
    <View
      style={[
        {
          flex: 1,
          alignItems: 'center',
          justifyContent: 'center',
          padding: theme.spacing.xxl,
          gap: theme.spacing.m,
        },
        style,
      ]}
    >
      <Text variant="h2" align="center">{title}</Text>
      {hint
        ? <Text color="textSecondary" align="center" style={{ maxWidth: 360 }}>{hint}</Text>
        : null}
      {action
        ? <View style={{ marginTop: theme.spacing.m }}>{action}</View>
        : null}
    </View>
  );
}

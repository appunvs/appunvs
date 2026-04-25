// Compact pill for state labels (draft / published / running / failed).
// Tone presets:
//
//   neutral  — subdued, used for muted metadata
//   info     — soft brand-pale, used for informational tags
//   success  — used for "published" / "ok"
//   warning  — used for "draft" / pending
//   danger   — used for errors
//
import { View, type StyleProp, type ViewStyle } from 'react-native';

import { useTheme } from '@/theme';
import { Text } from './Text';

export type BadgeTone = 'neutral' | 'info' | 'success' | 'warning' | 'danger';

export interface BadgeProps {
  label: string;
  tone?: BadgeTone;
  style?: StyleProp<ViewStyle>;
}

export function Badge({ label, tone = 'neutral', style }: BadgeProps) {
  const theme = useTheme();
  const c = theme.colors;
  const palette: Record<BadgeTone, { bg: string; fg: string }> = {
    neutral: { bg: c.bgInput,    fg: c.textSecondary },
    info:    { bg: c.brandPale,  fg: c.brandDark },
    success: { bg: c.brandPale,  fg: c.semanticSuccess },
    warning: { bg: c.bgInput,    fg: c.semanticWarning },
    danger:  { bg: c.bgInput,    fg: c.semanticDanger },
  };
  const { bg, fg } = palette[tone];
  return (
    <View
      style={[
        {
          alignSelf: 'flex-start',
          paddingHorizontal: theme.spacing.s,
          paddingVertical: 2,
          borderRadius: theme.radius.pill,
          backgroundColor: bg,
        },
        style,
      ]}
    >
      <Text variant="captionStrong" style={{ color: fg }}>{label}</Text>
    </View>
  );
}

// Quota usage bar for the Profile tab.  Shows a label, a fraction
// (e.g. "73 / 100 today"), and a colored fill that escalates from
// brand-pale → brand-dark as usage approaches the cap.  At ≥90% the
// bar flips to the warning tone so heavy users see the wall coming.
import { View, type StyleProp, type ViewStyle } from 'react-native';

import { useTheme } from '@/theme';
import { Text } from '@/ui';

export interface QuotaBarProps {
  label: string;
  used: number;
  cap: number;
  unit?: string;            // optional suffix in the right-side count, e.g. "MB"
  style?: StyleProp<ViewStyle>;
}

export function QuotaBar({ label, used, cap, unit, style }: QuotaBarProps) {
  const theme = useTheme();
  const ratio = cap === 0 ? 0 : Math.max(0, Math.min(1, used / cap));
  const danger = ratio >= 0.9;
  const fillColor = danger
    ? theme.colors.semanticWarning
    : theme.colors.brandDark;

  return (
    <View style={[{ gap: theme.spacing.xs }, style]}>
      <View style={{ flexDirection: 'row', justifyContent: 'space-between' }}>
        <Text variant="captionStrong">{label}</Text>
        <Text variant="caption" color="textSecondary">
          {used.toLocaleString()} / {cap.toLocaleString()}{unit ? ` ${unit}` : ''}
        </Text>
      </View>
      <View
        style={{
          height: 6,
          borderRadius: theme.radius.pill,
          backgroundColor: theme.colors.bgInput,
          overflow: 'hidden',
        }}
      >
        <View
          style={{
            width: `${ratio * 100}%`,
            height: '100%',
            backgroundColor: fillColor,
          }}
        />
      </View>
    </View>
  );
}

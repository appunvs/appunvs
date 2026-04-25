// Chat message bubble.  User messages are right-aligned with the brand
// fill; assistant / system messages are left-aligned with a muted
// surface.  No avatars in v1 — too heavy for the dense chat aesthetic.
import { View, type StyleProp, type ViewStyle } from 'react-native';

import { useTheme } from '@/theme';
import { Text } from '@/ui';

export type BubbleRole = 'user' | 'assistant' | 'system';

export interface BubbleProps {
  role: BubbleRole;
  text: string;
  pending?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function Bubble({ role, text, pending, style }: BubbleProps) {
  const theme = useTheme();
  const isUser = role === 'user';

  const bg = isUser
    ? theme.colors.brandDark
    : theme.colors.bgCard;
  const fg = isUser
    ? (theme.scheme === 'light' ? '#FFFFFF' : theme.colors.bgPage)
    : theme.colors.textPrimary;

  return (
    <View
      style={[
        {
          alignSelf: isUser ? 'flex-end' : 'flex-start',
          maxWidth: '85%',
          paddingHorizontal: theme.spacing.l,
          paddingVertical: theme.spacing.m,
          backgroundColor: bg,
          borderRadius: theme.radius.xl,
          borderTopRightRadius: isUser ? theme.radius.s : theme.radius.xl,
          borderTopLeftRadius: isUser ? theme.radius.xl : theme.radius.s,
          borderWidth: isUser ? 0 : 1,
          borderColor: theme.colors.borderDefault,
        },
        style,
      ]}
    >
      <Text style={{ color: fg }}>
        {text || (pending ? '…' : '')}
      </Text>
    </View>
  );
}

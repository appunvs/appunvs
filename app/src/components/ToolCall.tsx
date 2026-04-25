// Collapsible tool_use / tool_result block in chat.  Default state is
// collapsed: one line showing tool icon + name + a short args preview +
// status (pending / ok / error).  Tap the row to expand the full
// arguments JSON and the tool result body in monospace.
//
// Status is derived from props:
//   - has `result` and `isError` → done (success / failure)
//   - no result yet                → pending
//
// Keep this component dumb — no fetching, no streaming logic.  Parent
// owns the timeline; this just renders one entry.
import { useState, useMemo } from 'react';
import { Pressable, View, ScrollView, type ViewStyle, type StyleProp } from 'react-native';
import Animated, { useAnimatedStyle, useSharedValue, withTiming } from 'react-native-reanimated';

import { useTheme } from '@/theme';
import { Badge, Text } from '@/ui';

export interface ToolCallProps {
  name: string;
  argsJson: string;        // raw JSON string from tool_call
  result?: string;          // raw JSON string from tool_result; undefined while running
  isError?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function ToolCall({ name, argsJson, result, isError, style }: ToolCallProps) {
  const theme = useTheme();
  const [open, setOpen] = useState(false);
  const rotation = useSharedValue(0);

  const status = result === undefined ? 'pending' : isError ? 'error' : 'ok';

  const argsPreview = useMemo(() => previewArgs(argsJson), [argsJson]);

  const chevronStyle = useAnimatedStyle(() => ({
    transform: [{ rotate: `${rotation.value}deg` }],
  }));

  const toggle = () => {
    rotation.value = withTiming(open ? 0 : 90, { duration: 220 });
    setOpen((prev) => !prev);
  };

  const tone = status === 'ok' ? 'success' : status === 'error' ? 'danger' : 'info';
  const label = status === 'ok' ? 'ok' : status === 'error' ? 'error' : 'running';

  return (
    <View
      style={[
        {
          backgroundColor: theme.colors.bgCard,
          borderRadius: theme.radius.l,
          borderWidth: 1,
          borderColor: theme.colors.borderDefault,
          overflow: 'hidden',
        },
        style,
      ]}
    >
      <Pressable
        onPress={toggle}
        style={{
          paddingVertical: theme.spacing.s,
          paddingHorizontal: theme.spacing.m,
          flexDirection: 'row',
          alignItems: 'center',
          gap: theme.spacing.s,
        }}
      >
        <Animated.View style={chevronStyle}>
          <Text variant="captionStrong" color="textSecondary">▸</Text>
        </Animated.View>
        <Text variant="monoSm" color="brandDark">{name}</Text>
        <Text
          variant="monoSm"
          color="textSecondary"
          numberOfLines={1}
          style={{ flex: 1 }}
        >
          {argsPreview}
        </Text>
        <Badge label={label} tone={tone} />
      </Pressable>
      {open && (
        <View
          style={{
            paddingHorizontal: theme.spacing.m,
            paddingBottom: theme.spacing.m,
            gap: theme.spacing.s,
            borderTopWidth: 1,
            borderTopColor: theme.colors.borderDefault,
            paddingTop: theme.spacing.s,
          }}
        >
          <Text variant="captionStrong" color="textSecondary">arguments</Text>
          <ScrollView horizontal showsHorizontalScrollIndicator={false}>
            <Text variant="monoSm">{prettyJson(argsJson)}</Text>
          </ScrollView>
          {result !== undefined && (
            <>
              <Text
                variant="captionStrong"
                color={isError ? 'semanticDanger' : 'textSecondary'}
              >
                {isError ? 'error' : 'result'}
              </Text>
              <ScrollView horizontal showsHorizontalScrollIndicator={false}>
                <Text variant="monoSm">{prettyJson(result)}</Text>
              </ScrollView>
            </>
          )}
        </View>
      )}
    </View>
  );
}

// previewArgs collapses a json blob into a one-line `key=value, key=value`
// shorthand for the row.  Falls back to the raw string when parsing fails.
function previewArgs(raw: string): string {
  try {
    const obj = JSON.parse(raw);
    if (obj === null || typeof obj !== 'object') return String(obj);
    const pairs = Object.entries(obj as Record<string, unknown>)
      .slice(0, 3)
      .map(([k, v]) => `${k}=${shorten(String(v))}`);
    return pairs.join(', ');
  } catch {
    return raw.slice(0, 80);
  }
}

function prettyJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

function shorten(s: string, max = 24): string {
  if (s.length <= max) return s;
  return s.slice(0, max) + '…';
}

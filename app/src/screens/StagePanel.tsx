// StagePanel — host for the isolated runtime that loads a Box's
// bundle.  Used by:
//
//   - the Stage tab (always, full-bleed)
//   - the Chat tab on wide screens (right side panel)
//
// The runtime itself is platform-split (`@/stage/runtime`); this
// component owns the chrome around it: the version-status header,
// empty / error states, and reload control.
import { useState, useCallback } from 'react';
import {
  ActivityIndicator,
  View,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import { useTheme } from '@/theme';
import { Badge, Button, IconButton, Text } from '@/ui';
import { EmptyState } from '@/components';
import { useActiveBox } from '@/state/box';
import { StageRuntime } from '@/stage/runtime';
import type { BundleRef } from '@/lib/box';

export interface StagePanelProps {
  showHeader?: boolean;
  style?: StyleProp<ViewStyle>;
}

export function StagePanel({ showHeader = true, style }: StagePanelProps) {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const box = useActiveBox();
  const [error, setError] = useState<string | null>(null);
  // Bumping the key force-remounts the runtime — used by the reload icon.
  const [reloadNonce, setReloadNonce] = useState(0);

  const onReload = useCallback(() => {
    setError(null);
    setReloadNonce((n) => n + 1);
  }, []);

  if (!box || !box.current?.uri) {
    return (
      <View style={[{ flex: 1, backgroundColor: theme.colors.bgPage }, style]}>
        {showHeader ? (
          <Header
            insets={insets}
            box={box ?? null}
            current={null}
            onReload={onReload}
            disabled
          />
        ) : null}
        <EmptyState
          title="还没有可加载的 bundle"
          hint="用 Chat 创建一个 Box 并发布；或在 Profile 里扫码加载别人的。"
          action={
            box
              ? <Button label="去 Chat 编辑" onPress={() => router.push('/(tabs)/chat')} />
              : <Button label="新建 Box" onPress={() => router.push('/box/new')} />
          }
        />
      </View>
    );
  }

  return (
    <View style={[{ flex: 1, backgroundColor: '#000' }, style]}>
      {showHeader ? (
        <Header
          insets={insets}
          box={box}
          current={box.current}
          onReload={onReload}
        />
      ) : null}
      {error ? (
        <View style={{ flex: 1, padding: theme.spacing.xxl, alignItems: 'center', justifyContent: 'center' }}>
          <Text color="semanticDanger" align="center">{error}</Text>
          <View style={{ marginTop: theme.spacing.l }}>
            <Button label="重新加载" variant="secondary" onPress={onReload} />
          </View>
        </View>
      ) : (
        <StageRuntime
          key={reloadNonce}
          bundle={box.current}
          onError={(e) => setError(e.message)}
          fallback={<ActivityIndicator color={theme.colors.brandLight} />}
        />
      )}
    </View>
  );
}

interface HeaderProps {
  insets: { top: number };
  box: { title: string } | null;
  current: BundleRef | null;
  onReload: () => void;
  disabled?: boolean;
}

function Header({ insets, box, current, onReload, disabled }: HeaderProps) {
  const theme = useTheme();
  return (
    <View
      style={{
        paddingTop: insets.top + theme.spacing.xs,
        paddingHorizontal: theme.spacing.l,
        paddingBottom: theme.spacing.xs,
        flexDirection: 'row',
        alignItems: 'center',
        gap: theme.spacing.s,
        backgroundColor: theme.colors.bgPage,
        borderBottomWidth: 1,
        borderBottomColor: theme.colors.borderDefault,
      }}
    >
      <Text variant="captionStrong" numberOfLines={1} style={{ flex: 1 }}>
        {box?.title ?? 'Stage'}
        {current
          ? <>{'  '}<Text variant="caption" color="textSecondary">v{current.version}</Text></>
          : null}
      </Text>
      {current
        ? <Badge label={current.build_state} tone={current.build_state === 'succeeded' ? 'success' : 'warning'} />
        : null}
      <IconButton size="sm" onPress={onReload} disabled={disabled}>
        <Text variant="captionStrong" color="textPrimary">↻</Text>
      </IconButton>
    </View>
  );
}

// BoxSwitcher — the Chat header chip + bottom-sheet that lets the user
// switch active Box without leaving Chat.  Sheet contents:
//
//   - search field (placeholder for v1)
//   - list of all boxes for this namespace, current one marked with a
//     trailing brand check
//   - footer actions: "+ New box" → /box/new, "Pair to existing" →
//     /pair (camera; deferred to a follow-up but the row is here)
//
// State source: zustand `useActiveBoxStore` for current; `useBoxes()`
// hook (TanStack Query against /box list) for the full set.  Both
// exist already — this component just composes them.
import { useState } from 'react';
import { Modal, Pressable, ScrollView, View } from 'react-native';
import { useRouter } from 'expo-router';
import { useQuery } from '@tanstack/react-query';

import { useTheme } from '@/theme';
import { Text, IconButton, Divider, Badge } from '@/ui';
import { listBoxes, type Box } from '@/lib/box';
import { useActiveBoxStore } from '@/state/box';

export function BoxSwitcher() {
  const theme = useTheme();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const active = useActiveBoxStore((s) => s.box);
  const setActive = useActiveBoxStore((s) => s.setActive);

  const { data, refetch } = useQuery({
    queryKey: ['boxes'],
    queryFn: async () => listBoxes(),
    enabled: open,
  });

  const boxes = data?.boxes ?? [];

  return (
    <>
      <Pressable
        onPress={() => { setOpen(true); void refetch(); }}
        accessibilityRole="button"
        style={{
          flexDirection: 'row',
          alignItems: 'center',
          gap: theme.spacing.xs,
          paddingHorizontal: theme.spacing.s,
          paddingVertical: theme.spacing.xs,
          borderRadius: theme.radius.m,
        }}
      >
        <Text variant="bodyStrong" numberOfLines={1} style={{ maxWidth: 220 }}>
          {active?.title ?? '选择 Box'}
        </Text>
        <Text color="textSecondary">▾</Text>
      </Pressable>

      <Modal
        visible={open}
        transparent
        animationType="slide"
        onRequestClose={() => setOpen(false)}
      >
        <Pressable
          onPress={() => setOpen(false)}
          style={{ flex: 1, backgroundColor: '#0007', justifyContent: 'flex-end' }}
        >
          <Pressable
            onPress={() => { /* swallow taps inside the sheet */ }}
            style={{
              backgroundColor: theme.colors.bgCard,
              borderTopLeftRadius: theme.radius.xl,
              borderTopRightRadius: theme.radius.xl,
              paddingTop: theme.spacing.m,
              paddingBottom: theme.spacing.xl,
              maxHeight: '80%',
            }}
          >
            {/* drag handle */}
            <View style={{ alignItems: 'center', marginBottom: theme.spacing.s }}>
              <View style={{
                width: 36, height: 4, borderRadius: 2,
                backgroundColor: theme.colors.borderDefault,
              }} />
            </View>

            <View style={{ paddingHorizontal: theme.spacing.l, paddingBottom: theme.spacing.s }}>
              <Text variant="h3">我的 Box</Text>
            </View>

            <ScrollView>
              {boxes.length === 0 ? (
                <View style={{ padding: theme.spacing.xl, alignItems: 'center' }}>
                  <Text color="textSecondary">还没有 Box，新建一个开始。</Text>
                </View>
              ) : (
                boxes.map((b) => (
                  <BoxRow
                    key={b.box_id}
                    box={b}
                    isActive={active?.box_id === b.box_id}
                    onSelect={() => {
                      setActive({ box: b });
                      setOpen(false);
                    }}
                  />
                ))
              )}
            </ScrollView>

            <Divider style={{ marginTop: theme.spacing.s }} />
            <SheetActionRow
              icon="＋"
              label="新建 Box"
              onPress={() => { setOpen(false); router.push('/box/new'); }}
            />
            <SheetActionRow
              icon="📷"
              label="扫码看别人的 app"
              onPress={() => { setOpen(false); router.push('/(tabs)/profile'); }}
              disabled
            />
          </Pressable>
        </Pressable>
      </Modal>
    </>
  );
}

function BoxRow({
  box,
  isActive,
  onSelect,
}: {
  box: Box;
  isActive: boolean;
  onSelect: () => void;
}) {
  const theme = useTheme();
  const tone = box.state === 'published' ? 'success' : box.state === 'archived' ? 'neutral' : 'warning';
  return (
    <Pressable
      onPress={onSelect}
      style={{
        paddingVertical: theme.spacing.m,
        paddingHorizontal: theme.spacing.l,
        flexDirection: 'row',
        alignItems: 'center',
        gap: theme.spacing.m,
        backgroundColor: isActive ? theme.colors.brandPale : 'transparent',
      }}
    >
      <View style={{ flex: 1 }}>
        <Text variant="bodyStrong" numberOfLines={1}>{box.title}</Text>
        <Text variant="caption" color="textSecondary" numberOfLines={1}>
          v{box.current_version || '—'}
        </Text>
      </View>
      <Badge label={box.state} tone={tone} />
      {isActive ? <Text color="brandDark">✓</Text> : null}
    </Pressable>
  );
}

function SheetActionRow({
  icon,
  label,
  onPress,
  disabled,
}: {
  icon: string;
  label: string;
  onPress: () => void;
  disabled?: boolean;
}) {
  const theme = useTheme();
  return (
    <Pressable
      onPress={onPress}
      disabled={disabled}
      style={({ pressed }) => ({
        paddingVertical: theme.spacing.m,
        paddingHorizontal: theme.spacing.l,
        flexDirection: 'row',
        alignItems: 'center',
        gap: theme.spacing.m,
        backgroundColor: pressed ? theme.colors.brandPale : 'transparent',
        opacity: disabled ? 0.4 : 1,
      })}
    >
      <Text variant="h3" color="brandDark">{icon}</Text>
      <Text variant="bodyStrong">{label}</Text>
      {disabled
        ? <Badge label="即将上线" tone="neutral" style={{ marginLeft: 'auto' }} />
        : null}
    </Pressable>
  );
}

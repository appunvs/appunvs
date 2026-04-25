// /box/new — minimal create-Box flow.  Two fields:
//
//   - title: required, free text
//   - runtime: hidden in v1 (always rn_bundle); the input lives in
//     the schema for forward compatibility.
//
// On success we set the new Box as active and route back to Chat.
// Errors render inline; no full-screen error states for this small
// flow.
import { useState } from 'react';
import { View } from 'react-native';
import { useRouter } from 'expo-router';
import { useSafeAreaInsets } from 'react-native-safe-area-context';

import { useTheme } from '@/theme';
import { Button, Card, Input, Text } from '@/ui';
import { createBox } from '@/lib/box';
import { useActiveBoxStore } from '@/state/box';

export default function NewBoxScreen() {
  const theme = useTheme();
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const setActive = useActiveBoxStore((s) => s.setActive);

  const [title, setTitle] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const onCreate = async () => {
    const t = title.trim();
    if (!t) return;
    setBusy(true);
    setError(null);
    try {
      const r = await createBox({ title: t });
      setActive(r);
      router.replace('/(tabs)/chat');
    } catch (e) {
      setError(String(e));
      setBusy(false);
    }
  };

  return (
    <View
      style={{
        flex: 1,
        backgroundColor: theme.colors.bgPage,
        paddingTop: insets.top + theme.spacing.l,
        paddingHorizontal: theme.spacing.l,
        paddingBottom: insets.bottom + theme.spacing.l,
        gap: theme.spacing.l,
      }}
    >
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: theme.spacing.s }}>
        <Button label="返回" variant="ghost" size="sm" onPress={() => router.back()} />
      </View>

      <Text variant="h1">新建 Box</Text>
      <Text color="textSecondary">
        Box 是一个独立项目，对话历史和源代码都跟它绑定。命名后即可进入 Chat 与 AI
        协作。
      </Text>

      <Card>
        <Text variant="captionStrong" color="textSecondary">名称</Text>
        <Input
          value={title}
          onChangeText={setTitle}
          placeholder="比如 todo-app"
          autoFocus
          style={{ marginTop: theme.spacing.s }}
          maxLength={64}
        />
        {error
          ? <Text variant="caption" color="semanticDanger" style={{ marginTop: theme.spacing.s }}>{error}</Text>
          : null}
        <View style={{ flexDirection: 'row', gap: theme.spacing.s, marginTop: theme.spacing.l }}>
          <Button
            label="创建并进入 Chat"
            onPress={onCreate}
            disabled={!title.trim() || busy}
            loading={busy}
          />
          <Button label="取消" variant="ghost" onPress={() => router.back()} />
        </View>
      </Card>
    </View>
  );
}

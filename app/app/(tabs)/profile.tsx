import { useEffect, useState } from 'react';
import {
  View, Text, FlatList, Pressable, StyleSheet, ActivityIndicator, TextInput, Alert,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import { useActiveBoxStore } from '@/state/box';
import { listBoxes, createBox, publishBox, issuePair } from '@/lib/box';
import type { Box } from '@/lib/box';

// Profile tab — account/device settings + a list of the user's Boxes.
// Tapping a row makes that Box the "active" one for Chat + Stage.  A
// long-press opens the publish / pair sheet (placeholder Alert for now).
export default function ProfileScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const setActive = useActiveBoxStore((s) => s.setActive);
  const active = useActiveBoxStore((s) => s.box);
  const [boxes, setBoxes] = useState<Box[] | null>(null);
  const [draftTitle, setDraftTitle] = useState('');
  const [busy, setBusy] = useState(false);

  const refresh = async () => {
    setBusy(true);
    try {
      const res = await listBoxes();
      setBoxes(res.boxes ?? []);
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    refresh();
  }, []);

  const onCreate = async () => {
    const title = draftTitle.trim();
    if (!title) return;
    setBusy(true);
    try {
      const r = await createBox({ title });
      setDraftTitle('');
      await refresh();
      setActive(r);
      router.push('/(tabs)/chat');
    } finally {
      setBusy(false);
    }
  };

  const onPublish = async (b: Box) => {
    setBusy(true);
    try {
      const r = await publishBox(b.box_id);
      Alert.alert('Published', `${b.title} → v${r.current?.version ?? '?'}`);
      await refresh();
    } catch (e) {
      Alert.alert('Publish failed', String(e));
    } finally {
      setBusy(false);
    }
  };

  const onPair = async (b: Box) => {
    try {
      const r = await issuePair({ box_id: b.box_id, ttl_sec: 300 });
      Alert.alert('Pairing code', `${r.short_code}\nexpires in 5 min`);
    } catch (e) {
      Alert.alert('Pair failed', String(e));
    }
  };

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <Text style={styles.h1}>Boxes</Text>
      <View style={styles.createRow}>
        <TextInput
          style={styles.input}
          placeholder="new box title"
          placeholderTextColor="#888"
          value={draftTitle}
          onChangeText={setDraftTitle}
        />
        <Pressable style={styles.create} onPress={onCreate} disabled={busy}>
          <Text style={styles.createText}>Create</Text>
        </Pressable>
      </View>
      {busy && <ActivityIndicator style={{ marginVertical: 8 }} />}
      <FlatList
        data={boxes ?? []}
        keyExtractor={(b) => b.box_id}
        contentContainerStyle={{ gap: 8, padding: 12 }}
        renderItem={({ item }) => (
          <Pressable
            style={[styles.row, active?.box_id === item.box_id && styles.rowActive]}
            onPress={() => setActive({ box: item })}
          >
            <View style={{ flex: 1 }}>
              <Text style={styles.rowTitle}>{item.title}</Text>
              <Text style={styles.rowMeta}>
                {item.state} · v{item.current_version || '—'}
              </Text>
            </View>
            <Pressable onPress={() => onPair(item)} style={styles.miniBtn}>
              <Text style={styles.miniBtnText}>Pair</Text>
            </Pressable>
            <Pressable onPress={() => onPublish(item)} style={styles.miniBtn}>
              <Text style={styles.miniBtnText}>Publish</Text>
            </Pressable>
          </Pressable>
        )}
        ListEmptyComponent={
          <Text style={styles.empty}>No boxes yet — create one above.</Text>
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root:        { flex: 1, backgroundColor: '#0b0d10' },
  h1:          { color: '#f4f5f7', fontSize: 22, fontWeight: '700', padding: 16 },
  createRow:   { flexDirection: 'row', paddingHorizontal: 12, gap: 8 },
  input:       { flex: 1, color: '#f4f5f7', backgroundColor: '#161a20', padding: 10, borderRadius: 8 },
  create:      { backgroundColor: '#2e6cdf', paddingHorizontal: 16, justifyContent: 'center', borderRadius: 8 },
  createText:  { color: '#fff', fontWeight: '600' },
  row:         { flexDirection: 'row', alignItems: 'center', gap: 8, padding: 12, borderRadius: 10, backgroundColor: '#161a20' },
  rowActive:   { borderColor: '#2e6cdf', borderWidth: 1 },
  rowTitle:    { color: '#f4f5f7', fontSize: 16, fontWeight: '600' },
  rowMeta:     { color: '#9aa3ad', fontSize: 12, marginTop: 2 },
  miniBtn:     { backgroundColor: '#222831', paddingHorizontal: 10, paddingVertical: 6, borderRadius: 6 },
  miniBtnText: { color: '#9aa3ad', fontSize: 12, fontWeight: '600' },
  empty:       { color: '#9aa3ad', textAlign: 'center', padding: 24 },
});

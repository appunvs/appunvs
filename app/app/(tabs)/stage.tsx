import { useState } from 'react';
import { View, Text, Pressable, StyleSheet, ActivityIndicator } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useRouter } from 'expo-router';

import { useActiveBox } from '@/state/box';
import { StageRuntime } from '@/stage/runtime';

// Stage tab — hosts the isolated runtime that loads the current Box's
// bundle.  The runtime is platform-specific (see src/stage/runtime.*.ts);
// this screen is a thin frame around it that handles the empty / loading /
// error states.
export default function StageScreen() {
  const insets = useSafeAreaInsets();
  const router = useRouter();
  const box = useActiveBox();
  const [error, setError] = useState<string | null>(null);

  if (!box || !box.current?.uri) {
    return (
      <View style={[styles.empty, { paddingTop: insets.top }]}>
        <Text style={styles.emptyTitle}>No bundle loaded</Text>
        <Text style={styles.emptyHint}>
          Pair to a published Box (Profile → Pair) or publish from Chat first.
        </Text>
        <Pressable style={styles.cta} onPress={() => router.push('/(tabs)/profile')}>
          <Text style={styles.ctaText}>Open Profile</Text>
        </Pressable>
      </View>
    );
  }

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      <View style={styles.statusBar}>
        <Text style={styles.statusText} numberOfLines={1}>
          {box.title} · v{box.current.version}
        </Text>
      </View>
      {error ? (
        <View style={styles.error}>
          <Text style={styles.errorText}>{error}</Text>
        </View>
      ) : (
        <StageRuntime
          bundle={box.current}
          onError={(e) => setError(e.message)}
          fallback={<ActivityIndicator />}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root:       { flex: 1, backgroundColor: '#000' },
  statusBar:  { paddingVertical: 6, paddingHorizontal: 12, backgroundColor: '#101418' },
  statusText: { color: '#9aa3ad', fontSize: 12 },
  empty:      { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24, gap: 12, backgroundColor: '#0b0d10' },
  emptyTitle: { color: '#f4f5f7', fontSize: 20, fontWeight: '600' },
  emptyHint:  { color: '#9aa3ad', textAlign: 'center', maxWidth: 320 },
  cta:        { backgroundColor: '#2e6cdf', paddingHorizontal: 16, paddingVertical: 10, borderRadius: 8 },
  ctaText:    { color: '#fff', fontWeight: '600' },
  error:      { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  errorText:  { color: '#ff6b6b', textAlign: 'center' },
});

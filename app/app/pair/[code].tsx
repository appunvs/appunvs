import { useEffect, useState } from 'react';
import { View, Text, ActivityIndicator, StyleSheet, Pressable } from 'react-native';
import { useLocalSearchParams, useRouter } from 'expo-router';

import { claimPair } from '@/lib/box';
import { useActiveBoxStore } from '@/state/box';

// Deep-link landing for `appunvs://pair/<code>` and the equivalent web URL.
// On a connector device the QR scanner just opens this route; the screen
// claims the code and forwards the user to the Stage tab.
export default function PairCode() {
  const { code } = useLocalSearchParams<{ code: string }>();
  const router = useRouter();
  const setActive = useActiveBoxStore((s) => s.setActive);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!code) return;
    (async () => {
      try {
        const r = await claimPair(code);
        if (!r.bundle) {
          setError('Pairing succeeded but the box has no published bundle yet.');
          return;
        }
        setActive({
          box: {
            box_id: r.box_id,
            title: r.box_id,
            state: 'published',
            current_version: r.bundle.version,
            namespace: '',
            provider_device_id: '',
            runtime: 'rn_bundle',
            created_at: 0,
            updated_at: 0,
          },
          current: r.bundle,
        });
        router.replace('/(tabs)/stage');
      } catch (e) {
        setError(String(e));
      }
    })();
  }, [code]);

  return (
    <View style={styles.root}>
      {error ? (
        <>
          <Text style={styles.err}>{error}</Text>
          <Pressable style={styles.cta} onPress={() => router.replace('/(tabs)/profile')}>
            <Text style={styles.ctaText}>Back</Text>
          </Pressable>
        </>
      ) : (
        <>
          <ActivityIndicator />
          <Text style={styles.note}>claiming {code}…</Text>
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  root:    { flex: 1, alignItems: 'center', justifyContent: 'center', gap: 12, backgroundColor: '#0b0d10' },
  note:    { color: '#9aa3ad' },
  err:     { color: '#ff6b6b', textAlign: 'center', padding: 24 },
  cta:     { backgroundColor: '#2e6cdf', paddingHorizontal: 16, paddingVertical: 10, borderRadius: 8 },
  ctaText: { color: '#fff', fontWeight: '600' },
});

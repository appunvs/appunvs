// TestHarness — dev-only screen that lets you load an AI bundle by URL
// without going through the full Chat → publish → Stage flow.
//
// Reachable only when running the runtime as a standalone app (`npm run
// ios` / `npm run android` from this directory).  The host shell never
// mounts this — it has its own Stage tab that fetches bundles from the
// relay's pair-claim endpoint.
//
// Future: paste a bundle URL → press Load → the harness asks the
// SubRuntime native module (PR D2) to spawn a fresh runtime, fetch the
// bundle, and mount it in place of this harness.  Today it's a UI stub
// so the placeholder structure is committed.
import React, { useState } from 'react';
import {
  SafeAreaView,
  StyleSheet,
  Text,
  TextInput,
  TouchableOpacity,
  View,
} from 'react-native';
import { host } from './HostBridge';

export function TestHarness(): React.JSX.Element {
  const [url, setUrl] = useState<string>('');
  const [status, setStatus] = useState<string>('idle');
  const sdk = host().sdkVersion;

  const onLoad = () => {
    if (!url.trim()) {
      setStatus('enter a bundle URL first');
      return;
    }
    // PR D2: hand off to the SubRuntime native module.
    setStatus(`SubRuntime not wired yet — would load ${url.trim()}`);
  };

  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.body}>
        <Text style={styles.title}>runtime · dev harness</Text>
        <Text style={styles.subtitle}>SDK {sdk}</Text>

        <Text style={styles.label}>Bundle URL</Text>
        <TextInput
          value={url}
          onChangeText={setUrl}
          placeholder="https://relay.example/_artifacts/box_xxx/v3/index.bundle"
          placeholderTextColor="#557280"
          style={styles.input}
          autoCapitalize="none"
          autoCorrect={false}
        />

        <TouchableOpacity style={styles.button} onPress={onLoad}>
          <Text style={styles.buttonLabel}>Load</Text>
        </TouchableOpacity>

        <Text style={styles.status}>{status}</Text>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0B1418' },
  body: { flex: 1, padding: 24, gap: 12 },
  title: { color: '#E8F0F2', fontSize: 22, fontWeight: '700' },
  subtitle: { color: '#9AB0B8', fontSize: 13, marginBottom: 16 },
  label: { color: '#9AB0B8', fontSize: 12, fontWeight: '600' },
  input: {
    backgroundColor: '#1E2D33',
    color: '#E8F0F2',
    padding: 12,
    borderRadius: 8,
    fontFamily: 'Menlo',
    fontSize: 12,
  },
  button: {
    backgroundColor: '#4FB0BE',
    padding: 14,
    borderRadius: 8,
    alignItems: 'center',
    marginTop: 4,
  },
  buttonLabel: { color: '#0B1418', fontSize: 15, fontWeight: '700' },
  status: { color: '#557280', fontSize: 12, fontStyle: 'italic', marginTop: 8 },
});

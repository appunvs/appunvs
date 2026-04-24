import { useEffect, useState } from 'react';
import { StyleSheet, View, Text } from 'react-native';
import { WebView } from 'react-native-webview';

import type { StageRuntimeProps } from './runtime.types';

// Native StageRuntime — fallback path until the dedicated isolated Hermes
// runtime native module lands.  See runtime.types.ts for the contract.
//
// Important: WebView shares the OS network stack but NOT the host JS heap.
// We deliberately do NOT injectJavaScript() any host data; the bundle URI
// is loaded as-is and runs in its own JS context owned by the WebKit /
// WebView2 process.
export function StageRuntime({ bundle, onError, fallback }: StageRuntimeProps) {
  const [ready, setReady] = useState(false);

  // Reset on URI change so a version bump triggers a clean reload.
  useEffect(() => { setReady(false); }, [bundle.uri]);

  return (
    <View style={styles.root}>
      {!ready && (
        <View style={styles.loading}>
          {fallback ?? <Text style={styles.note}>loading bundle…</Text>}
        </View>
      )}
      <WebView
        source={{ uri: bundle.uri }}
        onLoadEnd={() => setReady(true)}
        onError={(e) => onError?.(new Error(e.nativeEvent.description ?? 'webview error'))}
        // Lock the WebView down: no third-party schemes, no file:// URLs,
        // no JS bridge.  When we swap to an isolated Hermes runtime the
        // sandbox model gets stricter still.
        originWhitelist={[bundle.uri.split('/').slice(0, 3).join('/')]}
        javaScriptEnabled
        domStorageEnabled={false}
        thirdPartyCookiesEnabled={false}
        sharedCookiesEnabled={false}
        style={styles.web}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root:    { flex: 1, backgroundColor: '#000' },
  web:     { flex: 1, backgroundColor: 'transparent' },
  loading: { ...StyleSheet.absoluteFillObject, alignItems: 'center', justifyContent: 'center' },
  note:    { color: '#9aa3ad' },
});

export default StageRuntime;

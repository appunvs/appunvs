// Host JS entry — the React tree that the host shell mounts when no AI
// bundle is loaded yet.  Two reasons it exists at all:
//
//   1. Dev convenience.  Running `npm run ios` / `npm run android` from
//      this directory boots the runtime with NO bundle wired in; this
//      placeholder is what you see, with a textbox to paste a bundle URL
//      for ad-hoc testing.  See `TestHarness.tsx`.
//
//   2. Default surface.  When the host's Stage tab mounts `RuntimeView`
//      before the user picks a Box, this is the "no bundle loaded yet"
//      empty state.  Tells the user to pick a Box from the Chat tab.
//
// Once a bundle is loaded into a SubRuntime, this entry is no longer
// active — the bundle's own root component takes over.
import React from 'react';
import { SafeAreaView, StyleSheet, Text, View } from 'react-native';
import { TestHarness } from './TestHarness';

export default function App(): React.JSX.Element {
  // The runtime is "alone" only when run from `npm run ios|android`
  // here.  Inside the host shell, the host gates whether to mount this
  // entry vs a bundle-driven SubRuntime, so detecting `__DEV__` here is
  // a good-enough signal for now.
  if (__DEV__) {
    return <TestHarness />;
  }
  return (
    <SafeAreaView style={styles.container}>
      <View style={styles.center}>
        <Text style={styles.title}>appunvs runtime</Text>
        <Text style={styles.hint}>
          Pick a Box from the Chat tab to load it here.
        </Text>
      </View>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  container: { flex: 1, backgroundColor: '#0B1418' },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  title: { color: '#E8F0F2', fontSize: 22, fontWeight: '700', marginBottom: 8 },
  hint: { color: '#9AB0B8', fontSize: 14, textAlign: 'center' },
});

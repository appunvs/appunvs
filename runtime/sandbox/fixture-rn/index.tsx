// runtime/sandbox/fixture-rn/index.tsx
//
// Tiny React Native bundle used by D3.c.4 UI tests to assert that
// RuntimeView.loadBundle() actually evaluates a JS bundle and renders
// React components.  Does NOT exercise any native modules — that's
// D3.d's territory; this fixture stays on plain View / Text only.
//
// Public registration name: "RuntimeRoot" — must match the moduleName
// passed by RuntimeView.mm and RuntimeView.kt.
//
// testID anchors used by the platform UI tests:
//   - "runtime-root"     → root <View>
//   - "runtime-greeting" → <Text> containing "Hello from D3.c"
import React from 'react';
import { AppRegistry, StyleSheet, Text, View } from 'react-native';

function RuntimeRoot() {
  return (
    <View style={styles.container} testID="runtime-root">
      <Text style={styles.greeting} testID="runtime-greeting">
        Hello from D3.c
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    alignItems: 'center',
    justifyContent: 'center',
    backgroundColor: 'black',
  },
  greeting: {
    color: 'white',
    fontSize: 24,
    fontWeight: 'bold',
  },
});

AppRegistry.registerComponent('RuntimeRoot', () => RuntimeRoot);

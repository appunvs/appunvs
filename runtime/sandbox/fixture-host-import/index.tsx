// runtime/sandbox/fixture-host-import/index.tsx
//
// Smoke fixture that imports from @appunvs/host and uses its sdkVersion
// at render time.  Bundling with sandbox/metro.config.js exercises:
//
//   1. The metro resolveRequest shim that maps the bare specifier
//      '@appunvs/host' → runtime/src/HostBridge.ts (so AI bundles can
//      import the bridge without us publishing a real npm package).
//   2. The ALLOWED_MODULES allowlist: react / react-native / @appunvs/host
//      should pass; anything else would throw at metro time.
//
// Distinct from sandbox/fixture-rn/index.tsx (which is the plain
// RuntimeView render fixture for D3.c.4 UI tests and uses the DEFAULT
// metro config, not the sandbox one).
import React from 'react';
import { AppRegistry, StyleSheet, Text, View } from 'react-native';
import { host } from '@appunvs/host';

function HostImportRoot() {
  // Read at render time so the bundler doesn't tree-shake the host import
  // away.  Inside RuntimeView, host().sdkVersion comes from the native
  // AppunvsHostModule's constantsToExport.
  const sdkVersion = host().sdkVersion;
  return (
    <View style={styles.container} testID="host-import-root">
      <Text style={styles.text} testID="host-import-sdk-version">
        SDK {sdkVersion} via @appunvs/host
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
  text: {
    color: 'white',
    fontSize: 18,
  },
});

AppRegistry.registerComponent('RuntimeRoot', () => HostImportRoot);

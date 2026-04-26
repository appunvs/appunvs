// Sandbox metro config — used when relay's sandbox builder bundles an
// AI-generated source tree.  Differs from the dev-time runtime
// metro.config.js in two ways:
//
//   1. `resolver.blockList` rejects every npm package outside the
//      curated allowlist.  AI bundles can `import 'react'` and
//      `import 'react-native-reanimated'`, but `import 'lodash'` fails
//      the build before bytecode is emitted.
//
//   2. `resolver.assetExts` strips out anything that would let an AI
//      bundle ship arbitrary native binaries (`.so`, `.dylib`).
//
// The bundle that comes out is loaded by the host into a SubRuntime
// (PR D2) — so the build-time block is the **first** line of defence;
// the SubRuntime's stripped-down module registry is the second.
const path = require('path');
const { getDefaultConfig, mergeConfig } = require('@react-native/metro-config');

// Allowlist must match `runtime/MODULES.md` Tier 1 (and later Tier 2).
// Add to this list ONLY by also updating MODULES.md and bumping
// `runtime.sdk_version` in `version.json`.
const ALLOWED_MODULES = new Set([
  'react',
  'react/jsx-runtime',
  'react/jsx-dev-runtime',
  'react-native',
  'react-native-safe-area-context',
  'react-native-screens',
  'react-native-gesture-handler',
  'react-native-reanimated',
  // Reanimated 4 split out the worklets runtime into its own package;
  // bundles using reanimated transitively import it.  Not in
  // runtime/MODULES.md as a Tier 1 entry because AI bundles never need
  // to import it directly — but metro must resolve it.
  'react-native-worklets',
  'react-native-svg',
  'react-native-mmkv',
  // The host bridge — resolved at metro time to the in-tree TypeScript
  // surface (runtime/src/HostBridge.ts).  At RUNTIME inside a SubRuntime
  // its `host()` impl reads from `NativeModules.AppunvsHost` (D3.e.1+);
  // outside RuntimeView it falls back to the dev stub also in that file.
  '@appunvs/host',
]);

// Where the @appunvs/host bare specifier resolves to.  The contract
// lives in runtime/src/HostBridge.ts (typing surface + dev stub +
// NativeModules-backed live impl); metro maps the specifier here so
// AI bundles can `import { host } from '@appunvs/host'` without us
// publishing a real package.
const APPUNVS_HOST_FILE = path.resolve(
  __dirname,
  '..',
  'src',
  'HostBridge.ts',
);

const config = {
  resolver: {
    // Treat every non-allowlisted bare import as missing.
    resolveRequest: (context, moduleName, platform) => {
      // Map the host-bridge specifier to the in-tree TS file.  Done before
      // the allowlist check so the rest of the function doesn't have to
      // special-case the @-scoped package against a real node_modules path.
      if (moduleName === '@appunvs/host') {
        return { type: 'sourceFile', filePath: APPUNVS_HOST_FILE };
      }
      const isRelative = moduleName.startsWith('./') || moduleName.startsWith('../');
      const isAbsolute = path.isAbsolute(moduleName);
      if (isRelative || isAbsolute) {
        return context.resolveRequest(context, moduleName, platform);
      }
      // Strip subpath imports like 'react-native/Libraries/...' to root pkg
      // for allowlist comparison — but keep the full name for resolution.
      const root = moduleName.startsWith('@')
        ? moduleName.split('/').slice(0, 2).join('/')
        : moduleName.split('/')[0];
      if (!ALLOWED_MODULES.has(moduleName) && !ALLOWED_MODULES.has(root)) {
        throw new Error(
          `[sandbox] import '${moduleName}' is not in the allowlist; see runtime/MODULES.md`
        );
      }
      return context.resolveRequest(context, moduleName, platform);
    },
  },
};

module.exports = mergeConfig(getDefaultConfig(__dirname), config);

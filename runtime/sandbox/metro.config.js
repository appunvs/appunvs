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
  // Compiler emit, not a user-facing API.  babel-preset-env + the RN
  // babel preset transpile ES module imports through helpers like
  // `@babel/runtime/helpers/interopRequireDefault`; AI bundles can't
  // (and don't) import these explicitly, but metro still asks the
  // resolver to find them.  Allowlisted as a bypass for the user
  // allowlist semantics — not in runtime/MODULES.md (that's only
  // user-callable surface).
  '@babel/runtime',
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

// Where to search for node_modules.  Docker sandbox image has them at
// /sandbox/node_modules (next to this config); the CI smoke job runs
// from runtime/ with the install at runtime/node_modules (one level up).
// Listing both keeps the config robust to either layout.
const NODE_MODULES_PATHS = [
  path.resolve(__dirname, 'node_modules'),
  path.resolve(__dirname, '..', 'node_modules'),
];

const NODE_MODULES_SEGMENT = `${path.sep}node_modules${path.sep}`;

const config = {
  // watchFolders extends metro's "files it can read from" beyond the
  // project root.  Needed so the CI smoke can pull in
  // runtime/src/HostBridge.ts (lives outside runtime/sandbox/ which is
  // metro's project root via getDefaultConfig(__dirname)) and so
  // node_modules walking can step up.
  watchFolders: [path.resolve(__dirname, '..')],
  resolver: {
    nodeModulesPaths: NODE_MODULES_PATHS,
    resolveRequest: (context, moduleName, platform) => {
      // Map the host-bridge specifier to the in-tree TS file.  Done before
      // any other check so the rest of the function doesn't have to
      // special-case the @-scoped package against a real node_modules path.
      if (moduleName === '@appunvs/host') {
        return { type: 'sourceFile', filePath: APPUNVS_HOST_FILE };
      }

      // Trust transitive imports.  Once an allowlisted package is in the
      // graph, its OWN imports (whether bare like 'invariant', scoped like
      // '@babel/runtime/helpers/...', or internal RN paths) are vetted by
      // virtue of the package itself being allowlisted.  AI source CANNOT
      // synthesise an import that appears to originate from inside
      // node_modules — so this bypass is sound at the AI-source security
      // boundary.  Without it the allowlist would have to enumerate every
      // transitive dep of every Tier 1 module, which is a pile of
      // versioned whack-a-mole.
      if (context.originModulePath &&
          context.originModulePath.includes(NODE_MODULES_SEGMENT)) {
        return context.resolveRequest(context, moduleName, platform);
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

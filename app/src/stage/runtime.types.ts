// Type-only entry point for the Stage runtime contract.  The actual
// component is resolved by Metro's platform extension rules:
//
//   web      -> runtime.web.tsx
//   native   -> runtime.native.tsx
//
// Both files MUST export a React component called StageRuntime that
// implements StageRuntimeProps below.  Nothing else may rely on the
// implementation details of either platform.
//
// The contract — see docs/architecture.md for the rationale:
//
//   "Loading a bundle into Stage MUST NOT be able to reach the host app's
//    state, tokens, MMKV stores, or file system."
//
// Today the native fallback uses react-native-webview, which inherits the
// host process's network identity but NOT its JS heap or stores.  A future
// slice replaces it with a dedicated isolated Hermes runtime; the contract
// stays the same.
import type { ReactNode } from 'react';

import type { BundleRef } from '@/lib/box';

export interface StageRuntimeProps {
  bundle: BundleRef;
  onError?: (err: Error) => void;
  fallback?: ReactNode;
}

// The component itself lives in runtime.native.tsx / runtime.web.tsx and
// is consumed via `import { StageRuntime } from '@/stage/runtime'`.  Type
// imports go through this file:
//   `import type { StageRuntimeProps } from '@/stage/runtime.types'`

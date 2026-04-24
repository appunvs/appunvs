import { useEffect, useRef } from 'react';
import { View, StyleSheet, Text } from 'react-native';

import type { StageRuntimeProps } from './runtime.types';

// Web StageRuntime — uses an iframe with strict sandbox attributes.  The
// host page and the iframe content live in the same browser tab but have
// distinct JS contexts; window-level message-passing requires explicit
// postMessage, which the runner code is not given a target for.
//
// We deliberately omit `allow-same-origin` so the bundle cannot read
// document.cookie / localStorage / IndexedDB belonging to the host origin
// even if the artifact CDN happens to be the same origin (it shouldn't be
// in production, but defense in depth).
export function StageRuntime({ bundle, onError, fallback }: StageRuntimeProps) {
  const ref = useRef<HTMLIFrameElement | null>(null);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;
    const handler = () => onError?.(new Error('iframe load failed'));
    node.addEventListener('error', handler);
    return () => node.removeEventListener('error', handler);
  }, [onError]);

  return (
    <View style={styles.root}>
      {fallback ? <View style={styles.fallback}>{fallback}</View> : null}
      {/* eslint-disable-next-line react-native/no-inline-styles */}
      <iframe
        ref={ref}
        src={bundle.uri}
        title="appunvs-stage"
        sandbox="allow-scripts"
        referrerPolicy="no-referrer"
        style={{
          flex: 1,
          width: '100%',
          height: '100%',
          border: 0,
          background: '#000',
        } as unknown as React.CSSProperties}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root:     { flex: 1, backgroundColor: '#000' },
  fallback: { ...StyleSheet.absoluteFillObject, alignItems: 'center', justifyContent: 'center' },
});

export default StageRuntime;

// Type-only fallback so a stale `import type { Text }` from this file
// doesn't pollute the public surface.
export type _Unused = Text;

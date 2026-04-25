// HostBridge — TypeScript declarations for the capabilities the host
// app injects into every AI bundle's SubRuntime.
//
// AI bundles `import { ... } from '@appunvs/host'` to reach these.
// The host side registers each function as a JSI binding when it spawns
// a SubRuntime; the metro sandbox config (sandbox/metro.config.js) lists
// `@appunvs/host` as an external so bundle build doesn't try to find a
// real package.
//
// This file is the **contract** — versioned alongside `version.json`'s
// `runtime.sdk_version`.  Adding a method here is an ABI bump.
//
// See MODULES.md for the full module allowlist (Tier 1 / 2 / 3 / Forbidden).
//
// NOTE: today this is a typing surface only — the native-side bindings
// land in PR D2 (SubRuntime native module).  AI bundles can `import` it,
// but at dev time inside the runtime harness the import resolves to the
// `__stub` below.

export interface BoxIdentity {
  /** Stable per-Box id assigned by the relay. */
  readonly boxID: string;
  /** Bundle version string (e.g. `"v3"`).  Empty string for unbuilt drafts. */
  readonly version: string;
  /** Short title for display, mirrors `BoxWire.title` from the relay. */
  readonly title: string;
}

/** Capability-scoped k/v store.  Replaces `MMKV` / `SecureStore` for AI bundles. */
export interface SubStorage {
  getString(key: string): Promise<string | null>;
  setString(key: string, value: string): Promise<void>;
  remove(key: string): Promise<void>;
  /** Returns the keys this Box has written.  No cross-Box visibility. */
  keys(): Promise<string[]>;
}

/** Pinned-endpoint network surface.  Replaces raw `fetch` for AI bundles. */
export interface SubNetwork {
  /** GET/POST against `/box/{id}/*` endpoints owned by this Box's namespace. */
  request(path: string, init?: { method?: string; body?: unknown }): Promise<Response>;
  /** Subscribe to live events for this Box.  Resolves to a closer fn. */
  subscribe(handler: (event: unknown) => void): Promise<() => void>;
}

/** Helper that asks the host to run the publish flow for the active Box. */
export interface SubPublish {
  /** Resolves once the relay reports the build finished (success or failure). */
  publish(message?: string): Promise<{ version: string; ok: boolean }>;
}

/** Top-level shape the host injects.  Available as `globalThis.__APPUNVS__`. */
export interface HostBridge {
  identity: BoxIdentity;
  storage: SubStorage;
  network: SubNetwork;
  publish: SubPublish;
  /** Runtime SDK version this Box was loaded into. */
  sdkVersion: string;
}

declare global {
  // eslint-disable-next-line no-var
  var __APPUNVS__: HostBridge | undefined;
}

/** Dev-time stub.  Returns the live host bridge when present, otherwise
 *  a no-op stand-in so the runtime harness can render without crashing. */
export function host(): HostBridge {
  if (globalThis.__APPUNVS__) {
    return globalThis.__APPUNVS__;
  }
  return __stub;
}

const __stub: HostBridge = {
  identity: { boxID: 'box_dev', version: '', title: 'dev harness' },
  storage: {
    async getString() { return null; },
    async setString() {},
    async remove() {},
    async keys() { return []; },
  },
  network: {
    async request() { throw new Error('SubNetwork unavailable in dev harness'); },
    async subscribe() { return () => {}; },
  },
  publish: {
    async publish() { return { version: '', ok: false }; },
  },
  sdkVersion: '0.1.0',
};

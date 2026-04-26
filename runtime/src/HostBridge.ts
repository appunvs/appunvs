// HostBridge — TypeScript declarations for the capabilities the host
// app injects into every AI bundle's SubRuntime.
//
// AI bundles `import { ... } from '@appunvs/host'` to reach these.
// The metro sandbox config (sandbox/metro.config.js) lists
// `@appunvs/host` as an external so bundle build doesn't try to find a
// real package.
//
// This file is the **contract** — versioned alongside `version.json`'s
// `runtime.sdk_version`.  Adding a method here is an ABI bump.
//
// See MODULES.md for the full module allowlist (Tier 1 / 2 / 3 / Forbidden).
//
// D3.e.1 wiring: when running inside RuntimeView, the SDK registers a
// native module called `AppunvsHost` (see runtime/ios/SDK/AppunvsHostModule.mm
// and runtime/android/runtimesdk/src/main/java/.../AppunvsHostModule.kt).
// `host()` below detects it via `NativeModules.AppunvsHost` and returns a
// bridge that surfaces native-side data.
// D3.e.2: identity flowed through loadBundle into the module's constants.
// D3.e.3: storage backed by react-native-mmkv with per-box namespace.
// D3.e.{4,5} (this PR): network.request() + publish() — wrap calls to
//   AppunvsHostModule.request / .publish, which delegate to a host-
//   registered handler closure.  SDK never makes HTTP calls itself;
//   host's HTTPClient owns auth + endpoints + retries.  network.subscribe
//   stays a TODO (SSE is its own architecture).
//
// In environments without the native module (web preview, jest, the
// runtime harness), `host()` falls back to the dev stub at the bottom
// of this file.

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

// Lazy-required so this module is safe to import in non-RN environments
// (jest, web preview, ssr): the require itself doesn't resolve unless
// `host()` actually executes the lookup branch.
type NativeRequestResponse = {
  status: number;
  headers: Record<string, string>;
  body: string;
};

type NativePublishResponse = {
  version: string;
  ok: boolean;
};

type NativeAppunvsHost = {
  sdkVersion: string;
  identity?: { boxID?: unknown; version?: unknown; title?: unknown };
  echo(message: string): Promise<string>;
  request(method: string, path: string, body: string | null): Promise<NativeRequestResponse>;
  publish(message: string | null): Promise<NativePublishResponse>;
};

function nativeAppunvsHost(): NativeAppunvsHost | null {
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const { NativeModules } = require('react-native') as {
      NativeModules: Record<string, unknown>;
    };
    const mod = NativeModules?.AppunvsHost;
    if (mod && typeof (mod as { sdkVersion?: unknown }).sdkVersion === 'string') {
      return mod as NativeAppunvsHost;
    }
  } catch {
    // react-native not available — fall through to dev stub.
  }
  return null;
}

function asString(x: unknown): string {
  return typeof x === 'string' ? x : '';
}

// MMKV-backed SubStorage scoped to a single box.  MMKV ships with the
// SDK as a Tier 1 native module (see runtime/MODULES.md) so it's
// available whenever we're inside a RuntimeView.  `id` parameter
// gives us a separate file per Box — AI bundle has no API for picking
// a different namespace, since the boxID derives from the identity
// native handed us as a read-only constant.
//
// Lazy-required so jest / web preview / non-RN environments can still
// load this module without a `react-native-mmkv` import resolving.
function makeMmkvStorage(boxID: string): SubStorage | null {
  try {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    const mod = require('react-native-mmkv') as {
      MMKV: new (cfg: { id: string }) => MMKVLike;
    };
    const store = new mod.MMKV({ id: `appunvs-box-${boxID || 'dev'}` });
    return {
      async getString(key) {
        return store.getString(key) ?? null;
      },
      async setString(key, value) {
        store.set(key, value);
      },
      async remove(key) {
        store.delete(key);
      },
      async keys() {
        return store.getAllKeys();
      },
    };
  } catch {
    return null;
  }
}

// Subset of the MMKV instance API we use.  Avoids a hard import-time
// dependency on react-native-mmkv's types in non-RN environments.
interface MMKVLike {
  getString(key: string): string | undefined;
  set(key: string, value: string): void;
  delete(key: string): void;
  getAllKeys(): string[];
}

/** Returns the host bridge: from globalThis.__APPUNVS__ when an upstream
 *  has set one, otherwise from the AppunvsHost native module when running
 *  inside RuntimeView, otherwise from the in-process dev stub. */
export function host(): HostBridge {
  if (globalThis.__APPUNVS__) {
    return globalThis.__APPUNVS__;
  }
  const native = nativeAppunvsHost();
  if (native) {
    // D3.e.1: sdkVersion + smoke methods.
    // D3.e.2: identity flows through native too — RuntimeView
    // staged it before bridge init, AppunvsHostModule exposed it as a
    // constant.
    // D3.e.3: storage backed by MMKV scoped to identity.boxID.
    // D3.e.4: network.request() routes through native to a host-
    //   registered handler.  network.subscribe stays a TODO (SSE).
    // D3.e.5: publish() routes through native to a host-registered
    //   handler.
    const id = native.identity ?? {};
    const identity = {
      boxID:   asString(id.boxID),
      version: asString(id.version),
      title:   asString(id.title),
    };
    const storage = makeMmkvStorage(identity.boxID) ?? __unimplementedStorage;
    return {
      ...__stub,
      sdkVersion: native.sdkVersion,
      identity,
      storage,
      network: makeNetwork(native, identity.boxID),
      publish: makePublish(native),
    };
  }
  return __stub;
}

// D3.e.4: SubNetwork backed by AppunvsHostModule.request.  Path
// scoping to /box/{boxID}/* happens here on the JS side — we already
// have the boxID from identity, so prefixing it client-side keeps the
// native module's request handler simple (host just needs to know
// baseURL + auth, not the box-routing convention).  network.subscribe
// is still a TODO (SSE has its own architecture; not a one-shot RPC).
function makeNetwork(native: NativeAppunvsHost, boxID: string): SubNetwork {
  const scoped = boxID || 'dev';
  return {
    async request(path, init) {
      const fullPath = path.startsWith('/box/')
        ? path
        : `/box/${scoped}${path.startsWith('/') ? path : '/' + path}`;
      const method = init?.method ?? 'GET';
      let body: string | null = null;
      if (init?.body !== undefined && init.body !== null) {
        body = typeof init.body === 'string' ? init.body : JSON.stringify(init.body);
      }
      const native_response = await native.request(method, fullPath, body);
      // Wrap in a real Response so AI bundles can use .json() / .text() /
      // .ok / .status normally.  Hermes ships the Web Response constructor.
      return new Response(native_response.body, {
        status: native_response.status,
        headers: native_response.headers,
      });
    },
    async subscribe() {
      throw new Error(
        'SubNetwork.subscribe not implemented yet — SSE is its own design slice',
      );
    },
  };
}

// D3.e.5: SubPublish backed by AppunvsHostModule.publish.
function makePublish(native: NativeAppunvsHost): SubPublish {
  return {
    async publish(message) {
      const result = await native.publish(message ?? null);
      return { version: result.version, ok: result.ok };
    },
  };
}

// Defensive fallback when react-native-mmkv require fails inside a
// RuntimeView SubRuntime — should be unreachable in practice (MMKV is
// pinned in runtime/package.json + linked via D3.d) but better to fail
// loud than no-op silently and let the AI bundle read/write nothing.
const __unimplementedStorage: SubStorage = {
  async getString() { throw new Error('SubStorage unavailable: react-native-mmkv failed to load'); },
  async setString() { throw new Error('SubStorage unavailable: react-native-mmkv failed to load'); },
  async remove() { throw new Error('SubStorage unavailable: react-native-mmkv failed to load'); },
  async keys() { throw new Error('SubStorage unavailable: react-native-mmkv failed to load'); },
};

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

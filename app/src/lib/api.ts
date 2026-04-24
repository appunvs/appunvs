// Thin HTTP client around the relay.  Reads the relay base URL from
// RELAY_URL (Expo extra) and falls back to localhost:8080 for dev.
import Constants from 'expo-constants';

import { getDeviceToken, getSessionToken } from './auth';

export const RELAY_URL: string =
  (Constants.expoConfig?.extra?.relayUrl as string | undefined) ??
  process.env.EXPO_PUBLIC_RELAY_URL ??
  'http://localhost:8080';

export type AuthMode = 'device' | 'session' | 'none';

export interface RequestOpts {
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE';
  auth?: AuthMode;          // default 'device'
  body?: unknown;           // JSON-serializable
  headers?: Record<string, string>;
  signal?: AbortSignal;
}

export class RelayError extends Error {
  constructor(public status: number, public payload: unknown) {
    super(`relay ${status}: ${typeof payload === 'string' ? payload : JSON.stringify(payload)}`);
  }
}

// request issues a fetch against the relay with the configured auth header.
// Throws RelayError on non-2xx; returns parsed JSON otherwise.
export async function request<T>(path: string, opts: RequestOpts = {}): Promise<T> {
  const url = path.startsWith('http') ? path : `${RELAY_URL}${path}`;
  const headers: Record<string, string> = { ...(opts.headers ?? {}) };
  const mode = opts.auth ?? 'device';
  if (mode !== 'none') {
    const token = mode === 'device' ? await getDeviceToken() : await getSessionToken();
    if (token) headers.Authorization = `Bearer ${token}`;
  }
  let body: BodyInit | undefined;
  if (opts.body !== undefined) {
    headers['Content-Type'] = 'application/json';
    body = JSON.stringify(opts.body);
  }
  const res = await fetch(url, {
    method: opts.method ?? 'GET',
    headers,
    body,
    signal: opts.signal,
  });
  const text = await res.text();
  let payload: unknown = text;
  if (text) {
    try { payload = JSON.parse(text); } catch { /* leave as text */ }
  }
  if (!res.ok) throw new RelayError(res.status, payload);
  return payload as T;
}

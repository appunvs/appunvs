// Token storage with two backends:
//   - native (iOS/Android) → expo-secure-store (Keychain / Keystore)
//   - web                  → localStorage
// SecureStore is preferred everywhere it works; the web fallback acknowledges
// that browsers have no equivalent of Keychain and the relay is the source
// of truth for revocation anyway.
import * as SecureStore from 'expo-secure-store';
import { Platform } from 'react-native';

const SESSION_KEY = 'appunvs.session_token';
const DEVICE_KEY  = 'appunvs.device_token';
const USER_KEY    = 'appunvs.user_id';
const DEVICE_ID_KEY = 'appunvs.device_id';

const webStore = {
  async getItem(k: string): Promise<string | null> {
    if (typeof localStorage === 'undefined') return null;
    return localStorage.getItem(k);
  },
  async setItem(k: string, v: string) {
    if (typeof localStorage === 'undefined') return;
    localStorage.setItem(k, v);
  },
  async removeItem(k: string) {
    if (typeof localStorage === 'undefined') return;
    localStorage.removeItem(k);
  },
};

const store = Platform.OS === 'web' ? webStore : {
  getItem:    (k: string) => SecureStore.getItemAsync(k),
  setItem:    (k: string, v: string) => SecureStore.setItemAsync(k, v),
  removeItem: (k: string) => SecureStore.deleteItemAsync(k),
};

export async function getSessionToken(): Promise<string | null> {
  return store.getItem(SESSION_KEY);
}
export async function setSessionToken(t: string): Promise<void> {
  await store.setItem(SESSION_KEY, t);
}
export async function clearSessionToken(): Promise<void> {
  await store.removeItem(SESSION_KEY);
}
export async function getDeviceToken(): Promise<string | null> {
  return store.getItem(DEVICE_KEY);
}
export async function setDeviceToken(t: string): Promise<void> {
  await store.setItem(DEVICE_KEY, t);
}
export async function getUserID(): Promise<string | null> {
  return store.getItem(USER_KEY);
}
export async function setUserID(u: string): Promise<void> {
  await store.setItem(USER_KEY, u);
}
export async function getOrCreateDeviceID(): Promise<string> {
  const existing = await store.getItem(DEVICE_ID_KEY);
  if (existing) return existing;
  // crypto.randomUUID is available in Hermes (RN 0.74+) and modern web.
  const id = (globalThis as any).crypto?.randomUUID?.() ?? `dev_${Math.random().toString(36).slice(2)}`;
  await store.setItem(DEVICE_ID_KEY, id);
  return id;
}

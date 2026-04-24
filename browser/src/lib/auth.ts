// Client-side helpers around the relay's /auth/* endpoints.
// Persists session + device tokens in localStorage; no cookies, no server.
import { RELAY_BASE } from './config';
import { Platform } from './pb/wire';

const SESSION_KEY = 'appunvs.session_token';
const DEVICE_TOKEN_KEY = 'appunvs.token';
const DEVICE_ID_KEY = 'appunvs.device_id';
const USER_ID_KEY = 'appunvs.user_id';
const EMAIL_KEY = 'appunvs.email';

export interface SessionResponse {
  user_id: string;
  session_token: string;
}
export interface DeviceRegistration {
  token: string;
  user_id: string;
}

/** Current session token (empty if not logged in). */
export function sessionToken(): string {
  return typeof localStorage === 'undefined' ? '' : localStorage.getItem(SESSION_KEY) ?? '';
}

/** Current device token (empty until /auth/register succeeds). */
export function deviceToken(): string {
  return typeof localStorage === 'undefined' ? '' : localStorage.getItem(DEVICE_TOKEN_KEY) ?? '';
}

export function userId(): string {
  return typeof localStorage === 'undefined' ? '' : localStorage.getItem(USER_ID_KEY) ?? '';
}

export function email(): string {
  return typeof localStorage === 'undefined' ? '' : localStorage.getItem(EMAIL_KEY) ?? '';
}

export function deviceId(): string {
  return typeof localStorage === 'undefined' ? '' : localStorage.getItem(DEVICE_ID_KEY) ?? '';
}

export function setDeviceId(id: string): void {
  localStorage.setItem(DEVICE_ID_KEY, id);
}

export async function signup(emailIn: string, password: string): Promise<SessionResponse> {
  const resp = await fetch(`${RELAY_BASE}/auth/signup`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email: emailIn, password })
  });
  if (!resp.ok) throw new Error((await textErr(resp)) || `signup failed: ${resp.status}`);
  const body = (await resp.json()) as SessionResponse;
  localStorage.setItem(SESSION_KEY, body.session_token);
  localStorage.setItem(USER_ID_KEY, body.user_id);
  localStorage.setItem(EMAIL_KEY, emailIn);
  return body;
}

export async function login(emailIn: string, password: string): Promise<SessionResponse> {
  const resp = await fetch(`${RELAY_BASE}/auth/login`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email: emailIn, password })
  });
  if (!resp.ok) throw new Error((await textErr(resp)) || `login failed: ${resp.status}`);
  const body = (await resp.json()) as SessionResponse;
  localStorage.setItem(SESSION_KEY, body.session_token);
  localStorage.setItem(USER_ID_KEY, body.user_id);
  localStorage.setItem(EMAIL_KEY, emailIn);
  // A new session invalidates any previously-cached device token — force
  // re-registration on next boot so server + client stay in sync.
  localStorage.removeItem(DEVICE_TOKEN_KEY);
  return body;
}

export function logout(): void {
  localStorage.removeItem(SESSION_KEY);
  localStorage.removeItem(DEVICE_TOKEN_KEY);
  localStorage.removeItem(USER_ID_KEY);
  localStorage.removeItem(EMAIL_KEY);
  // Keep device_id: stable identity across logout/login cycles on the same
  // browser avoids spraying devices table entries.
}

/** Register this browser as a device for the logged-in user. */
export async function registerDevice(): Promise<DeviceRegistration> {
  const session = sessionToken();
  if (!session) throw new Error('not logged in');
  const id = deviceId();
  const resp = await fetch(`${RELAY_BASE}/auth/register`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      Authorization: `Bearer ${session}`
    },
    body: JSON.stringify({ device_id: id, platform: Platform.BROWSER })
  });
  if (!resp.ok) throw new Error((await textErr(resp)) || `register failed: ${resp.status}`);
  const body = (await resp.json()) as DeviceRegistration;
  localStorage.setItem(DEVICE_TOKEN_KEY, body.token);
  localStorage.setItem(USER_ID_KEY, body.user_id);
  return body;
}

async function textErr(resp: Response): Promise<string> {
  try {
    const body = (await resp.json()) as { error?: string };
    return body.error ?? '';
  } catch {
    return '';
  }
}

// Relay base URL, e.g. "http://localhost:8080".
// Override with VITE_RELAY_BASE at build/dev time.
export const RELAY_BASE: string =
  import.meta.env.VITE_RELAY_BASE ?? 'http://localhost:8080';

export function wsBase(): string {
  // Convert http(s) -> ws(s)
  if (RELAY_BASE.startsWith('https://')) return 'wss://' + RELAY_BASE.slice('https://'.length);
  if (RELAY_BASE.startsWith('http://')) return 'ws://' + RELAY_BASE.slice('http://'.length);
  return RELAY_BASE;
}

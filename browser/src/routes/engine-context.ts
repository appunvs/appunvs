import type { SyncEngine } from '$lib/sync/engine';

let current: SyncEngine | null = null;

export function setEngine(e: SyncEngine): void {
  current = e;
}

export function getEngine(): SyncEngine {
  if (!current) throw new Error('SyncEngine not initialized');
  return current;
}

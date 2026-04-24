// Typed wrappers around Tauri's invoke + event API.
// All IPC with Rust flows through this file so call sites stay pure TS.

import { invoke } from '@tauri-apps/api/core';
import { listen, type UnlistenFn } from '@tauri-apps/api/event';
import type { Record, SyncStatus, RecordsChangedEvent } from './types';

export async function writeRecord(id: string, data: string): Promise<void> {
  await invoke<void>('write_record', { id, data });
}

export async function deleteRecord(id: string): Promise<void> {
  await invoke<void>('delete_record', { id });
}

export async function queryRecords(): Promise<Record[]> {
  return await invoke<Record[]>('query_records');
}

export async function setRole(role: string): Promise<void> {
  await invoke<void>('set_role', { role });
}

export async function getSyncStatus(): Promise<SyncStatus> {
  return await invoke<SyncStatus>('get_sync_status');
}

// Event channel names. Keep in sync with Rust side.
export const EVT_MESSAGE = 'appunvs://message';
export const EVT_RECORDS_CHANGED = 'appunvs://records-changed';
export const EVT_CONN_STATE = 'appunvs://conn-state';

export function onRecordsChanged(cb: (e: RecordsChangedEvent) => void): Promise<UnlistenFn> {
  return listen<RecordsChangedEvent>(EVT_RECORDS_CHANGED, (ev) => cb(ev.payload));
}

export function onConnState(cb: (state: string) => void): Promise<UnlistenFn> {
  return listen<string>(EVT_CONN_STATE, (ev) => cb(ev.payload));
}

export function onMessage(cb: (raw: unknown) => void): Promise<UnlistenFn> {
  return listen<unknown>(EVT_MESSAGE, (ev) => cb(ev.payload));
}

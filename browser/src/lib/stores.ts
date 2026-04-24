import { writable, type Writable } from 'svelte/store';
import { Role } from './pb/wire';
import type { Record as DbRecord } from './db/records';

export type ConnState = 'disconnected' | 'connecting' | 'connected' | 'reconnecting';

export const connState: Writable<ConnState> = writable('disconnected');
export const role: Writable<Role> = writable(Role.BOTH);
export const records: Writable<DbRecord[]> = writable([]);
export const lastSeq: Writable<number> = writable(0);
export const deviceId: Writable<string> = writable('');
export const userId: Writable<string> = writable('');

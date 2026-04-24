import { writable } from 'svelte/store';
import type { ConnState, Record, Role } from './types';

export const connState = writable<ConnState>('idle');
export const role = writable<Role>('both');
export const records = writable<Record[]>([]);
export const lastSeq = writable<number>(0);

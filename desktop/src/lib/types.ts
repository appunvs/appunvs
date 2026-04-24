// Shared TypeScript types mirroring Rust command return shapes.

export type Role = 'provider' | 'connector' | 'both' | 'role_unspecified';

export type ConnState = 'idle' | 'connecting' | 'open' | 'closed';

export interface Record {
  id: string;
  data: string;
  seq: number;
  updated_at: number;
}

export interface SyncStatus {
  conn_state: string;
  role: string;
  last_seq: number;
}

export interface RecordsChangedEvent {
  op: 'upsert' | 'delete';
  record: Record;
}

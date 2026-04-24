// Hand-written TS mirrors of the HTTP request/response messages in
// shared/proto/appunvs.proto. Field names are snake_case and enum values are
// the short-lowercase forms used by the relay's protojson encoding
// (UseProtoNames=true, short enum values).
//
// This file only describes the HTTP wire shapes. The WS envelope and enums
// shared with it live in ./wire.ts.

import type { Platform } from './wire';

// ---------------------- Auth ----------------------

export interface Device {
  id: string;
  user_id: string;
  platform: Platform;
  created_at: number; // ms unix
  last_seen: number;  // ms unix; 0 when never seen
}

export interface MeResponse {
  user_id: string;
  email: string;
  created_at: number;
  devices: Device[];
}

// ---------------------- Dynamic schema ----------------------

// Column types as they appear on the wire. The proto enum ColumnType has
// UNSPECIFIED/TEXT/NUMBER/BOOL/JSON; the relay serialises these as the
// short lowercase names (without the `column_type_` prefix).
export const ColumnType = {
  UNSPECIFIED: 'column_type_unspecified',
  TEXT: 'text',
  NUMBER: 'number',
  BOOL: 'bool',
  JSON: 'json'
} as const;
export type ColumnType = (typeof ColumnType)[keyof typeof ColumnType];

export interface SchemaColumn {
  name: string;
  type: ColumnType;
  required: boolean;
  created_at: number;
}

export interface SchemaTable {
  name: string;
  created_at: number;
  columns: SchemaColumn[];
}

export interface SchemaCreateTableRequest {
  name: string;
}

export interface SchemaListTablesResponse {
  tables: SchemaTable[];
}

export interface SchemaAddColumnRequest {
  name: string;
  type: ColumnType;
  required: boolean;
}

// ---------------------- API keys ----------------------

export interface APIKeyCreateRequest {
  name: string;
}

export interface APIKeyCreateResponse {
  id: string;
  name: string;
  prefix: string;
  secret: string;
  created_at: number;
}

export interface APIKeySummary {
  id: string;
  name: string;
  prefix: string;
  created_at: number;
  last_used_at: number;
  revoked_at: number;
}

// GET /keys returns the bare array. The relay ships the plain list, not the
// wrapper message from the proto file — we mirror the actual wire shape here
// and note the divergence in api.ts.
export type APIKeyListResponse = APIKeySummary[];

// ---------------------- Billing ----------------------

export interface Plan {
  id: string;
  name: string;
  price_cents_monthly: number;
  messages_per_day: number;
  storage_bytes: number;
  max_devices: number;
  max_api_keys: number;
}

export interface BillingPlansResponse {
  plans: Plan[];
}

export interface BillingStatusResponse {
  plan: string;
  plan_name: string;
  status: string;
  period_start: number;
  period_end: number;
  messages_used: number;
  storage_bytes: number;
  limits: Plan;
}

export interface BillingCheckoutRequest {
  plan_id: string;
}

export interface BillingCheckoutResponse {
  url: string;
  mode: 'live' | 'mock';
}

// ---------------------- Error envelope ----------------------

export interface ErrorResponse {
  error: string;
}

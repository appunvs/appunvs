// Hand-written protojson mirror of shared/proto/appunvs.proto.
// On-the-wire enum values are short lowercase strings; field names are snake_case.

export const Role = {
  UNSPECIFIED: 'role_unspecified',
  PROVIDER: 'provider',
  CONNECTOR: 'connector',
  BOTH: 'both'
} as const;
export type Role = (typeof Role)[keyof typeof Role];

export const Op = {
  UNSPECIFIED: 'op_unspecified',
  UPSERT: 'upsert',
  DELETE: 'delete',
  TABLE_CREATE: 'table_create',
  TABLE_DELETE: 'table_delete',
  COLUMN_ADD: 'column_add',
  COLUMN_DELETE: 'column_delete',
  QUOTA_EXCEEDED: 'quota_exceeded'
} as const;
export type Op = (typeof Op)[keyof typeof Op];

export const Platform = {
  UNSPECIFIED: 'platform_unspecified',
  BROWSER: 'browser',
  DESKTOP: 'desktop',
  MOBILE: 'mobile'
} as const;
export type Platform = (typeof Platform)[keyof typeof Platform];

export interface Message {
  seq?: number;
  device_id: string;
  user_id: string;
  namespace: string;
  role: Role;
  op: Op;
  table: string;
  payload?: Record<string, unknown>;
  ts: number;
}

export interface RegisterRequest {
  device_id: string;
  platform: Platform;
}

export interface RegisterResponse {
  token: string;
  user_id: string;
}

function isValidRole(v: unknown): v is Role {
  return v === 'role_unspecified' || v === 'provider' || v === 'connector' || v === 'both';
}
function isValidOp(v: unknown): v is Op {
  return (
    v === 'op_unspecified' ||
    v === 'upsert' ||
    v === 'delete' ||
    v === 'table_create' ||
    v === 'table_delete' ||
    v === 'column_add' ||
    v === 'column_delete' ||
    v === 'quota_exceeded'
  );
}

/** Serialize a Message to canonical protojson. Matches the Go reference:
 *  default-valued scalars (empty strings, zero i64) are omitted. */
export function toJson(m: Message): string {
  const out: Record<string, unknown> = {};
  if (m.seq !== undefined && m.seq !== 0) out.seq = m.seq;
  if (m.device_id) out.device_id = m.device_id;
  if (m.user_id) out.user_id = m.user_id;
  if (m.namespace) out.namespace = m.namespace;
  if (m.role && m.role !== 'role_unspecified') out.role = m.role;
  if (m.op && m.op !== 'op_unspecified') out.op = m.op;
  if (m.table) out.table = m.table;
  if (m.payload !== undefined) out.payload = m.payload;
  if (m.ts !== undefined && m.ts !== 0) out.ts = m.ts;
  return JSON.stringify(out);
}

/** Parse a canonical protojson Message. Numeric fields tolerate string form
 *  (protojson renders int64 as string). */
export function fromJson(s: string): Message {
  const raw = JSON.parse(s) as Record<string, unknown>;

  const seqVal = raw.seq;
  const tsVal = raw.ts;
  const role = raw.role;
  const op = raw.op;

  if (!isValidRole(role)) throw new Error(`invalid role: ${String(role)}`);
  if (!isValidOp(op)) throw new Error(`invalid op: ${String(op)}`);

  const m: Message = {
    device_id: String(raw.device_id ?? ''),
    user_id: String(raw.user_id ?? ''),
    namespace: String(raw.namespace ?? ''),
    role,
    op,
    table: String(raw.table ?? ''),
    ts: numeric(tsVal) ?? 0
  };
  const seq = numeric(seqVal);
  if (seq !== undefined) m.seq = seq;
  if (raw.payload && typeof raw.payload === 'object') {
    m.payload = raw.payload as Record<string, unknown>;
  }
  return m;
}

function numeric(v: unknown): number | undefined {
  if (v === undefined || v === null) return undefined;
  if (typeof v === 'number') return v;
  if (typeof v === 'string' && v !== '') {
    const n = Number(v);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
}

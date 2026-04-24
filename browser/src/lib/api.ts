// Session-authenticated REST client for the relay's HTTP control plane.
// Every call reads the current session token out of localStorage via auth.ts
// and sets `Authorization: Bearer …`. Non-2xx responses are parsed into
// ApiError using the shared {error} envelope.
//
// Types mirror shared/proto/appunvs.proto via src/lib/pb/http.ts. Keeping the
// types in a separate file (rather than inline) makes drift checks and
// cross-module reuse easier.
import { RELAY_BASE } from './config';
import { sessionToken } from './auth';
import type {
  APIKeyCreateRequest,
  APIKeyCreateResponse,
  APIKeyListResponse,
  APIKeySummary,
  BillingCheckoutRequest,
  BillingCheckoutResponse,
  BillingPlansResponse,
  BillingStatusResponse,
  ColumnType,
  ErrorResponse,
  MeResponse,
  SchemaAddColumnRequest,
  SchemaColumn,
  SchemaCreateTableRequest,
  SchemaListTablesResponse,
  SchemaTable
} from './pb/http';

export class ApiError extends Error {
  readonly status: number;
  readonly body: string;
  constructor(status: number, message: string, body: string) {
    super(message);
    this.status = status;
    this.body = body;
  }
}

async function request<TResp>(
  method: string,
  path: string,
  body?: unknown,
  opts: { auth?: boolean; parse?: boolean } = {}
): Promise<TResp> {
  const auth = opts.auth ?? true;
  const parse = opts.parse ?? true;

  const headers: Record<string, string> = {};
  if (body !== undefined) headers['content-type'] = 'application/json';
  if (auth) {
    const token = sessionToken();
    if (!token) throw new ApiError(401, 'not logged in', '');
    headers['Authorization'] = `Bearer ${token}`;
  }

  const resp = await fetch(`${RELAY_BASE}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body)
  });

  if (!resp.ok) {
    const text = await resp.text();
    let msg = `${method} ${path} failed: ${resp.status}`;
    try {
      const parsed = JSON.parse(text) as ErrorResponse;
      if (parsed.error) msg = parsed.error;
    } catch {
      if (text) msg = text;
    }
    throw new ApiError(resp.status, msg, text);
  }

  if (!parse || resp.status === 204) return undefined as TResp;
  return (await resp.json()) as TResp;
}

// ---------------------- Auth ----------------------

export function me(): Promise<MeResponse> {
  return request<MeResponse>('GET', '/auth/me');
}

// ---------------------- Schema ----------------------

export function listTables(): Promise<SchemaListTablesResponse> {
  return request<SchemaListTablesResponse>('GET', '/schema/tables');
}

export function createTable(name: string): Promise<SchemaTable> {
  const req: SchemaCreateTableRequest = { name };
  return request<SchemaTable>('POST', '/schema/tables', req);
}

export function deleteTable(name: string): Promise<void> {
  return request<void>(
    'DELETE',
    `/schema/tables/${encodeURIComponent(name)}`,
    undefined,
    { parse: false }
  );
}

export function addColumn(
  table: string,
  column: { name: string; type: ColumnType; required: boolean }
): Promise<SchemaColumn> {
  const req: SchemaAddColumnRequest = column;
  return request<SchemaColumn>(
    'POST',
    `/schema/tables/${encodeURIComponent(table)}/columns`,
    req
  );
}

export function deleteColumn(table: string, column: string): Promise<void> {
  return request<void>(
    'DELETE',
    `/schema/tables/${encodeURIComponent(table)}/columns/${encodeURIComponent(column)}`,
    undefined,
    { parse: false }
  );
}

// ---------------------- API keys ----------------------

// The relay ships GET /keys as a bare array rather than the
// {keys:[...]} wrapper defined in APIKeyListResponse. We honour the wire
// shape here and expose a clean APIKeySummary[] to callers.
export function listKeys(): Promise<APIKeySummary[]> {
  return request<APIKeyListResponse>('GET', '/keys');
}

export function createKey(name: string): Promise<APIKeyCreateResponse> {
  const req: APIKeyCreateRequest = { name };
  return request<APIKeyCreateResponse>('POST', '/keys', req);
}

export function revokeKey(id: string): Promise<void> {
  return request<void>('DELETE', `/keys/${encodeURIComponent(id)}`, undefined, {
    parse: false
  });
}

// ---------------------- Billing ----------------------

export function listPlans(): Promise<BillingPlansResponse> {
  // Public endpoint, but sending a bearer token is harmless.
  return request<BillingPlansResponse>('GET', '/billing/plans', undefined, {
    auth: false
  });
}

export function billingStatus(): Promise<BillingStatusResponse> {
  return request<BillingStatusResponse>('GET', '/billing/status');
}

export function billingCheckout(planId: string): Promise<BillingCheckoutResponse> {
  const req: BillingCheckoutRequest = { plan_id: planId };
  return request<BillingCheckoutResponse>('POST', '/billing/checkout', req);
}

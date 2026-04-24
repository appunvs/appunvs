// Typed client for the relay's /box and /pair endpoints.  The shapes here
// MUST match shared/proto/appunvs.proto (BoxResponse, PairResponse,
// PairClaimResponse).  A drift test on the relay side asserts that for the
// Go mirrors; the TS mirrors are guarded by hand for now.
import { request } from './api';

export type RuntimeKind = 'rn_bundle' | 'unspecified';
export type PublishState = 'draft' | 'published' | 'archived' | 'unspecified';
export type BuildState = 'queued' | 'running' | 'succeeded' | 'failed' | 'unspecified';

export interface Box {
  box_id: string;
  namespace: string;
  provider_device_id: string;
  title: string;
  runtime: RuntimeKind;
  state: PublishState;
  current_version: string;
  created_at: number;
  updated_at: number;
}

export interface BundleRef {
  box_id: string;
  version: string;
  uri: string;
  content_hash: string;
  size_bytes: number;
  build_state: BuildState;
  build_log?: string;
  built_at: number;
  expires_at: number;
}

export interface BoxResponse {
  box: Box;
  current?: BundleRef;
}

export interface BoxListResponse {
  boxes: Box[];
}

export interface PairResponse {
  short_code: string;
  expires_at: number;
}

export interface PairClaimResponse {
  box_id: string;
  bundle?: BundleRef;
  namespace_token: string;
}

export const listBoxes = () => request<BoxListResponse>('/box');

export const createBox = (input: { title: string; runtime?: RuntimeKind }) =>
  request<BoxResponse>('/box', { method: 'POST', body: { title: input.title, runtime: input.runtime ?? 'rn_bundle' } });

export const getBox = (id: string) => request<BoxResponse>(`/box/${id}`);

export const publishBox = (id: string, source?: { entry_point?: string; files?: Record<string, string> }) =>
  request<BoxResponse>(`/box/${id}/publish`, { method: 'POST', body: source ?? {} });

export const archiveBox = (id: string) =>
  request<void>(`/box/${id}`, { method: 'DELETE' });

export const issuePair = (input: { box_id: string; ttl_sec?: number }) =>
  request<PairResponse>('/pair', { method: 'POST', body: input });

export const claimPair = (code: string) =>
  request<PairClaimResponse>(`/pair/${code}/claim`, { method: 'POST', body: {} });

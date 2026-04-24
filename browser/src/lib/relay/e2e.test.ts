// Cross-language E2E: drives the real Go relay with our TypeScript wire codec.
// Requires a relay + Redis reachable at APPUNVS_RELAY_BASE.
import { describe, it, expect, beforeAll } from 'vitest';
import WebSocket from 'ws';
import { fromJson, toJson, Role, Op, type Message } from '../pb/wire';

const BASE = process.env.APPUNVS_RELAY_BASE ?? 'http://localhost:8080';
const WS_BASE = BASE.replace(/^http/, 'ws');

async function registerDevice(deviceId: string, platform = 'browser') {
  const res = await fetch(`${BASE}/auth/register`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ device_id: deviceId, platform })
  });
  if (!res.ok) throw new Error(`register ${res.status}`);
  return (await res.json()) as { token: string; user_id: string };
}

async function dial(token: string, lastSeq?: number): Promise<WebSocket> {
  const url = new URL(`${WS_BASE}/ws`);
  url.searchParams.set('token', token);
  if (lastSeq !== undefined && lastSeq > 0) {
    url.searchParams.set('last_seq', String(lastSeq));
  }
  const ws = new WebSocket(url.toString());
  await new Promise<void>((resolve, reject) => {
    ws.once('open', () => resolve());
    ws.once('error', reject);
  });
  return ws;
}

function nextMessage(ws: WebSocket, timeoutMs = 3000): Promise<Message> {
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      ws.off('message', onMsg);
      reject(new Error(`timeout waiting for message after ${timeoutMs}ms`));
    }, timeoutMs);
    const onMsg = (data: WebSocket.RawData) => {
      clearTimeout(timer);
      resolve(fromJson(data.toString()));
    };
    ws.once('message', onMsg);
  });
}

let available = false;
let token = '';
let userId = '';

beforeAll(async () => {
  try {
    const r = await fetch(`${BASE}/health`);
    if (!r.ok) return;
    const reg = await registerDevice('ts-e2e-device');
    token = reg.token;
    userId = reg.user_id;
    available = true;
  } catch {
    available = false;
  }
});

describe('relay e2e (TS wire <-> Go relay)', () => {
  it('broadcasts a provider upsert back to the sender with a relay-assigned seq', async () => {
    if (!available) return;
    const ws = await dial(token);
    const payload = { id: 'r1', data: 'ts-wire-says-hi' };
    const msg: Message = {
      device_id: 'ts-e2e-device',
      user_id: userId,
      namespace: userId,
      role: Role.PROVIDER,
      op: Op.UPSERT,
      table: 'records',
      payload,
      ts: Date.now()
    };
    ws.send(toJson(msg));

    const echoed = await nextMessage(ws);
    expect(echoed.seq).toBeGreaterThan(0);
    expect(echoed.namespace).toBe(userId);
    expect(echoed.role).toBe(Role.PROVIDER);
    expect(echoed.op).toBe(Op.UPSERT);
    expect(echoed.payload).toEqual(payload);
    ws.close();
  });

  it('replays missed messages on reconnect via last_seq', async () => {
    if (!available) return;
    const ws = await dial(token);
    for (let i = 0; i < 3; i++) {
      ws.send(
        toJson({
          device_id: 'ts-e2e-device',
          user_id: userId,
          namespace: userId,
          role: Role.PROVIDER,
          op: Op.UPSERT,
          table: 'records',
          payload: { id: `catchup-${i}` },
          ts: Date.now()
        })
      );
    }
    const seenSeqs: number[] = [];
    for (let i = 0; i < 3; i++) {
      const m = await nextMessage(ws);
      if (m.seq !== undefined) seenSeqs.push(m.seq);
    }
    ws.close();
    const lastSeq = seenSeqs[seenSeqs.length - 1]!;

    // Publish one more from a different socket while the first one is offline.
    const other = await dial(token);
    other.send(
      toJson({
        device_id: 'ts-e2e-device',
        user_id: userId,
        namespace: userId,
        role: Role.PROVIDER,
        op: Op.UPSERT,
        table: 'records',
        payload: { id: 'while-offline' },
        ts: Date.now()
      })
    );
    const published = await nextMessage(other);
    expect(published.seq).toBe(lastSeq + 1);
    other.close();

    // Reconnect original "device" with last_seq = lastSeq → expect one replay.
    const back = await dial(token, lastSeq);
    const replayed = await nextMessage(back);
    expect(replayed.seq).toBe(lastSeq + 1);
    expect(replayed.payload).toEqual({ id: 'while-offline' });
    back.close();
  });
});

import { describe, it, expect } from 'vitest';
import { fromJson, toJson, Role, Op, type Message } from './wire';

describe('wire codec', () => {
  it('roundtrips a canonical protojson Message', () => {
    const src: Message = {
      seq: 1024,
      device_id: 'd1',
      user_id: 'u1',
      namespace: 'u1',
      role: Role.PROVIDER,
      op: Op.UPSERT,
      table: 'records',
      payload: { id: 'r1', data: 'hello' },
      ts: 1_714_000_000_000
    };
    const wire = toJson(src);
    const parsed = fromJson(wire);
    expect(parsed).toEqual(src);
  });

  it('omits seq when unset (client has not yet received a relay-assigned seq)', () => {
    const m: Message = {
      device_id: 'd1',
      user_id: 'u1',
      namespace: 'u1',
      role: Role.PROVIDER,
      op: Op.UPSERT,
      table: 'records',
      payload: { id: 'r1' },
      ts: 1
    };
    expect(JSON.parse(toJson(m))).not.toHaveProperty('seq');
  });

  it('serializes enums as short lowercase strings', () => {
    const m: Message = {
      device_id: 'd',
      user_id: 'u',
      namespace: 'u',
      role: Role.CONNECTOR,
      op: Op.DELETE,
      table: 't',
      ts: 0
    };
    const json = JSON.parse(toJson(m));
    expect(json.role).toBe('connector');
    expect(json.op).toBe('delete');
  });

  it('tolerates int64-as-string seq (protojson quirk)', () => {
    const wire = '{"seq":"42","device_id":"d","user_id":"u","namespace":"u","role":"provider","op":"upsert","table":"t","ts":"1"}';
    const m = fromJson(wire);
    expect(m.seq).toBe(42);
    expect(m.ts).toBe(1);
  });

  it('rejects unknown role', () => {
    const bad = '{"device_id":"d","user_id":"u","namespace":"u","role":"admin","op":"upsert","table":"t","ts":0}';
    expect(() => fromJson(bad)).toThrow(/invalid role/);
  });
});

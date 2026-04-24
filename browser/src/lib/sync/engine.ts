import { get } from 'svelte/store';
import * as recordsDb from '../db/records';
import { Op, Role, type Message } from '../pb/wire';
import type { RelayClient, RelayMessageEvent } from '../relay/client';
import { deviceId, lastSeq, role, userId } from '../stores';

const TABLE = 'records';

export class SyncEngine {
  constructor(private readonly relay: RelayClient) {
    relay.onMessage.addEventListener('message', (e) => {
      const ev = e as RelayMessageEvent;
      void this.handle(ev.message);
    });
  }

  setRole(r: Role): void {
    role.set(r);
  }

  private myRoleIncludesProvider(): boolean {
    const r = get(role);
    return r === Role.PROVIDER || r === Role.BOTH;
  }

  private myRoleIncludesConnector(): boolean {
    const r = get(role);
    return r === Role.CONNECTOR || r === Role.BOTH;
  }

  private async applyLocal(m: Message): Promise<void> {
    if (m.table !== TABLE) return;
    const payload = m.payload ?? {};
    const id = typeof payload.id === 'string' ? payload.id : '';
    if (!id) return;
    if (m.op === Op.UPSERT) {
      const data = typeof payload.data === 'string' ? payload.data : JSON.stringify(payload);
      await recordsDb.upsert(id, data, m.seq ?? 0);
    } else if (m.op === Op.DELETE) {
      await recordsDb.delete_(id);
    }
  }

  private async handle(m: Message): Promise<void> {
    // Seq gap detection: provider-sourced messages carry authoritative seq from relay.
    const prev = get(lastSeq);
    if (typeof m.seq === 'number' && m.seq > 0) {
      if (m.seq !== prev + 1 && prev !== 0) {
        // Gap — drop socket, reconnect will replay via last_seq.
        console.warn(`seq gap: expected ${prev + 1} got ${m.seq}; resetting`);
        this.relay.reset();
        return;
      }
      lastSeq.set(m.seq);
      this.relay.setLastSeq(m.seq);
    }

    // Ignore our own provider broadcasts — we already applied them locally.
    const myDevice = get(deviceId);
    if (m.device_id === myDevice && m.role === Role.PROVIDER) return;

    if (m.role === Role.PROVIDER) {
      // Authoritative change from a provider — apply locally.
      if (this.myRoleIncludesConnector() || this.myRoleIncludesProvider()) {
        await this.applyLocal(m);
      }
      return;
    }

    if (m.role === Role.CONNECTOR) {
      // A connector is asking providers to do something.
      if (this.myRoleIncludesProvider()) {
        await this.applyLocal(m);
        // Re-broadcast authoritatively as provider.
        const forward: Message = {
          device_id: myDevice,
          user_id: get(userId),
          namespace: get(userId),
          role: Role.PROVIDER,
          op: m.op,
          table: m.table,
          payload: m.payload,
          ts: Date.now()
        };
        this.relay.send(forward);
      }
      return;
    }

    // Unspecified / BOTH on the wire: ignore.
  }

  async addRecord(id: string, data: string): Promise<void> {
    const r = get(role);
    const my = get(deviceId);
    const uid = get(userId);
    const payload = { id, data };

    if (this.myRoleIncludesProvider()) {
      await recordsDb.upsert(id, data);
      const msg: Message = {
        device_id: my,
        user_id: uid,
        namespace: uid,
        role: Role.PROVIDER,
        op: Op.UPSERT,
        table: TABLE,
        payload,
        ts: Date.now()
      };
      this.relay.send(msg);
      return;
    }

    if (r === Role.CONNECTOR) {
      const msg: Message = {
        device_id: my,
        user_id: uid,
        namespace: uid,
        role: Role.CONNECTOR,
        op: Op.UPSERT,
        table: TABLE,
        payload,
        ts: Date.now()
      };
      this.relay.send(msg);
      return;
    }
  }

  async deleteRecord(id: string): Promise<void> {
    const r = get(role);
    const my = get(deviceId);
    const uid = get(userId);
    const payload = { id };

    if (this.myRoleIncludesProvider()) {
      await recordsDb.delete_(id);
      const msg: Message = {
        device_id: my,
        user_id: uid,
        namespace: uid,
        role: Role.PROVIDER,
        op: Op.DELETE,
        table: TABLE,
        payload,
        ts: Date.now()
      };
      this.relay.send(msg);
      return;
    }

    if (r === Role.CONNECTOR) {
      const msg: Message = {
        device_id: my,
        user_id: uid,
        namespace: uid,
        role: Role.CONNECTOR,
        op: Op.DELETE,
        table: TABLE,
        payload,
        ts: Date.now()
      };
      this.relay.send(msg);
      return;
    }
  }
}

import { writable, type Writable } from 'svelte/store';
import { wsBase } from '../config';
import { fromJson, toJson, type Message } from '../pb/wire';
import type { ConnState } from '../stores';

export interface RelayMessageEvent extends Event {
  readonly message: Message;
}

/**
 * RelayClient maintains a single WebSocket to the relay.
 *
 * - `state` is a svelte store you can subscribe to from UI code.
 * - `onMessage` is an EventTarget; listen with `onMessage.addEventListener('message', e => ...)`.
 *   Each event carries a `.message` property with the decoded Message.
 */
export class RelayClient {
  readonly state: Writable<ConnState> = writable('disconnected');
  readonly onMessage: EventTarget = new EventTarget();

  private ws: WebSocket | null = null;
  private token = '';
  private lastSeq = 0;
  private backoffMs = 1000;
  private readonly maxBackoffMs = 30_000;
  private stopped = true;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;

  connect(token: string, lastSeq: number): void {
    this.token = token;
    this.lastSeq = lastSeq;
    this.stopped = false;
    this.open();
  }

  /** Update last seen seq; used by reconnects to resume replay. */
  setLastSeq(seq: number): void {
    if (seq > this.lastSeq) this.lastSeq = seq;
  }

  /** Force-close the socket; the client will auto-reconnect unless stop() is called. */
  reset(): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      try {
        this.ws.close();
      } catch {
        // ignore
      }
    }
  }

  stop(): void {
    this.stopped = true;
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.ws) {
      try {
        this.ws.close();
      } catch {
        // ignore
      }
      this.ws = null;
    }
    this.state.set('disconnected');
  }

  send(m: Message): boolean {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      this.ws.send(toJson(m));
      return true;
    }
    return false;
  }

  private open(): void {
    if (this.stopped) return;
    this.state.set(this.backoffMs === 1000 ? 'connecting' : 'reconnecting');

    const url = `${wsBase()}/ws?token=${encodeURIComponent(this.token)}&last_seq=${this.lastSeq}`;
    let ws: WebSocket;
    try {
      ws = new WebSocket(url);
    } catch (e) {
      console.error('WebSocket constructor threw', e);
      this.scheduleReconnect();
      return;
    }
    this.ws = ws;

    ws.addEventListener('open', () => {
      this.backoffMs = 1000;
      this.state.set('connected');
    });

    ws.addEventListener('message', (ev: MessageEvent) => {
      const data = typeof ev.data === 'string' ? ev.data : '';
      if (!data) return;
      let msg: Message;
      try {
        msg = fromJson(data);
      } catch (e) {
        console.error('invalid wire message', e, data);
        return;
      }
      const evt = new Event('message') as RelayMessageEvent;
      Object.defineProperty(evt, 'message', { value: msg, enumerable: true });
      this.onMessage.dispatchEvent(evt);
    });

    ws.addEventListener('close', () => {
      this.ws = null;
      if (!this.stopped) this.scheduleReconnect();
      else this.state.set('disconnected');
    });

    ws.addEventListener('error', () => {
      // close will follow; don't double-schedule
    });
  }

  private scheduleReconnect(): void {
    if (this.stopped) return;
    this.state.set('reconnecting');
    const delay = this.backoffMs;
    this.backoffMs = Math.min(this.backoffMs * 2, this.maxBackoffMs);
    this.reconnectTimer = setTimeout(() => {
      this.reconnectTimer = null;
      this.open();
    }, delay);
  }
}

import { getDb } from './sqlite';

export interface Record {
  id: string;
  data: string;
  seq: number;
  updated_at: number;
}

type Listener = (snapshot: Record[]) => void;
const listeners = new Set<Listener>();

async function snapshot(): Promise<Record[]> {
  return queryAll();
}

async function notify(): Promise<void> {
  if (listeners.size === 0) return;
  const s = await snapshot();
  for (const l of listeners) {
    try {
      l(s);
    } catch (e) {
      console.error('record subscriber threw', e);
    }
  }
}

export async function upsert(
  id: string,
  data: string,
  seq?: number
): Promise<void> {
  const db = await getDb();
  const now = Date.now();
  await db.exec(
    `INSERT INTO records (id, data, seq, updated_at)
     VALUES (?, ?, ?, ?)
     ON CONFLICT(id) DO UPDATE SET
       data = excluded.data,
       seq = MAX(records.seq, excluded.seq),
       updated_at = excluded.updated_at`,
    [id, data, seq ?? 0, now]
  );
  await notify();
}

export async function remove(id: string): Promise<void> {
  const db = await getDb();
  await db.exec('DELETE FROM records WHERE id = ?', [id]);
  await notify();
}

export async function queryAll(): Promise<Record[]> {
  const db = await getDb();
  const rows = await db.query<Record>(
    'SELECT id, data, seq, updated_at FROM records ORDER BY updated_at DESC'
  );
  return rows.map((r) => ({
    id: String(r.id ?? ''),
    data: String(r.data ?? ''),
    seq: Number(r.seq ?? 0),
    updated_at: Number(r.updated_at ?? 0)
  }));
}

export async function maxSeq(): Promise<number> {
  const db = await getDb();
  const rows = await db.query<{ s: number | null }>(
    'SELECT COALESCE(MAX(seq), 0) AS s FROM records'
  );
  const s = rows[0]?.s;
  return typeof s === 'number' ? s : 0;
}

export function subscribe(fn: Listener): () => void {
  listeners.add(fn);
  // prime the subscriber with current state
  void snapshot().then((s) => {
    try {
      fn(s);
    } catch (e) {
      console.error('record subscriber prime threw', e);
    }
  });
  return () => {
    listeners.delete(fn);
  };
}

// Export under a friendlier name; `delete` is a reserved word.
export { remove as delete_ };

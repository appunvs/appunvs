// wa-sqlite wrapper.
// wa-sqlite is loaded via dynamic import so it stays out of the main chunk.
// We serialize all statements through an internal queue since wa-sqlite is
// not safe against concurrent statements on a single db handle.

type SQLiteAPI = {
  open_v2(path: string, flags?: number, zVfs?: string): Promise<number>;
  close(db: number): Promise<number>;
  prepare_v2(db: number, sqlPtr: number): Promise<{ stmt: number; sql: number } | null>;
  bind_collection(stmt: number, params: readonly unknown[]): number;
  step(stmt: number): Promise<number>;
  column_names(stmt: number): string[];
  row(stmt: number): unknown[];
  finalize(stmt: number): Promise<number>;
  exec(
    db: number,
    sql: string,
    callback?: (row: unknown[], cols: string[]) => void | Promise<void>
  ): Promise<number>;
  str_new(db: number, s: string): number;
  str_value(str: number): number;
  str_finish(str: number): void;
  vfs_register(vfs: unknown, makeDefault?: boolean): number;
};

// SQLite open flags (matching sqlite-constants.js values).
const SQLITE_OPEN_READWRITE = 0x00000002;
const SQLITE_OPEN_CREATE    = 0x00000004;
const SQLITE_ROW = 100;

export interface DbWrapper {
  exec(sql: string, params?: readonly unknown[]): Promise<void>;
  query<T = Record<string, unknown>>(
    sql: string,
    params?: readonly unknown[]
  ): Promise<T[]>;
}

let dbPromise: Promise<DbWrapper> | null = null;

const DB_NAME = 'appunvs.db';
// wa-sqlite ships several VFS backends. We use MemoryAsyncVFS — lives in
// RAM for the page lifetime, no persistence. IDBBatchAtomicVFS is the
// persistent option but requires wrapping work beyond this scaffold's
// scope; the SyncEngine rehydrates from the relay on reconnect anyway.
// Swap in IDBBatchAtomicVFS here when the persistent path is productized.
const VFS_NAME = 'memory-async';

export function getDb(): Promise<DbWrapper> {
  if (!dbPromise) dbPromise = init();
  return dbPromise;
}

async function init(): Promise<DbWrapper> {
  // Dynamic imports keep wa-sqlite out of the main bundle.
  // The Emscripten factory lives in dist/; the JS API wrapper lives in src/.
  const [waMod, apiMod, vfsMod] = await Promise.all([
    import(/* @vite-ignore */ 'wa-sqlite/dist/wa-sqlite-async.mjs'),
    import(/* @vite-ignore */ 'wa-sqlite/src/sqlite-api.js'),
    import(/* @vite-ignore */ 'wa-sqlite/src/examples/MemoryAsyncVFS.js')
  ]);

  const waFactory = (waMod as { default: (opts?: unknown) => Promise<unknown> }).default;
  const module = await waFactory();

  const sqlite = (apiMod as unknown as {
    Factory: (mod: unknown) => SQLiteAPI;
  }).Factory(module);

  const VFS = (vfsMod as { MemoryAsyncVFS: new () => unknown }).MemoryAsyncVFS;
  const vfs = new VFS();
  sqlite.vfs_register(vfs, true);

  // vfs_register(vfs, true) set it as default, so zVfs is implicit.
  const dbHandle = await sqlite.open_v2(
    DB_NAME,
    SQLITE_OPEN_READWRITE | SQLITE_OPEN_CREATE
  );

  // Serialized statement queue.
  let chain: Promise<unknown> = Promise.resolve();
  const serialize = <T>(fn: () => Promise<T>): Promise<T> => {
    const next = chain.then(fn, fn);
    chain = next.catch(() => undefined);
    return next;
  };

  // prepare_v2 takes a C-side string pointer, not a JS string. We wrap with
  // str_new / str_finish exactly like sqlite3.statements() does internally.
  const prepareOne = async (sql: string) => {
    const str = sqlite.str_new(dbHandle, sql);
    const prepared = await sqlite.prepare_v2(dbHandle, sqlite.str_value(str));
    return { prepared, finish: () => sqlite.str_finish(str) };
  };

  const runExec = async (sql: string, params?: readonly unknown[]): Promise<void> => {
    if (!params || params.length === 0) {
      await sqlite.exec(dbHandle, sql);
      return;
    }
    const { prepared, finish } = await prepareOne(sql);
    if (!prepared) {
      finish();
      return;
    }
    try {
      sqlite.bind_collection(prepared.stmt, params);
      await sqlite.step(prepared.stmt);
    } finally {
      await sqlite.finalize(prepared.stmt);
      finish();
    }
  };

  const runQuery = async <T>(
    sql: string,
    params?: readonly unknown[]
  ): Promise<T[]> => {
    const rows: T[] = [];
    if (!params || params.length === 0) {
      // Static SQL: exec with row callback.
      await sqlite.exec(dbHandle, sql, (row, cols) => {
        const obj: Record<string, unknown> = {};
        for (let i = 0; i < cols.length; i++) obj[cols[i]] = row[i];
        rows.push(obj as T);
      });
      return rows;
    }
    const { prepared, finish } = await prepareOne(sql);
    if (!prepared) {
      finish();
      return rows;
    }
    try {
      sqlite.bind_collection(prepared.stmt, params);
      const cols = sqlite.column_names(prepared.stmt);
      while ((await sqlite.step(prepared.stmt)) === SQLITE_ROW) {
        const r = sqlite.row(prepared.stmt);
        const obj: Record<string, unknown> = {};
        for (let i = 0; i < cols.length; i++) obj[cols[i]] = r[i];
        rows.push(obj as T);
      }
    } finally {
      await sqlite.finalize(prepared.stmt);
      finish();
    }
    return rows;
  };

  const wrapper: DbWrapper = {
    exec: (sql, params) => serialize(() => runExec(sql, params)),
    query: <T>(sql: string, params?: readonly unknown[]) =>
      serialize(() => runQuery<T>(sql, params))
  };

  await wrapper.exec(
    `CREATE TABLE IF NOT EXISTS records (
       id TEXT PRIMARY KEY,
       data TEXT,
       seq INTEGER DEFAULT 0,
       updated_at INTEGER
     );`
  );

  return wrapper;
}

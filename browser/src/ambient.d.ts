// wa-sqlite ships without TypeScript declarations. We dynamic-import it and
// cast to a minimal API surface in src/lib/db/sqlite.ts.
declare module 'wa-sqlite/dist/wa-sqlite-async.mjs';
declare module 'wa-sqlite/src/sqlite-api.js';
declare module 'wa-sqlite/src/examples/IDBBatchAtomicVFS.js';
declare module 'wa-sqlite/src/examples/MemoryAsyncVFS.js';

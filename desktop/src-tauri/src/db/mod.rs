//! SQLite access layer. Single `Connection` guarded by a `Mutex`; callers hold
//! it only long enough to run one statement to avoid contention with the tokio
//! runtime.

use std::path::Path;
use std::sync::{Arc, Mutex};

use anyhow::{Context, Result};
use rusqlite::Connection;

pub mod records;

pub use records::Record;

#[derive(Clone)]
pub struct Db {
    inner: Arc<Mutex<Connection>>,
}

impl Db {
    pub fn new(path: &Path) -> Result<Self> {
        if let Some(parent) = path.parent() {
            std::fs::create_dir_all(parent)
                .with_context(|| format!("create db parent: {}", parent.display()))?;
        }
        let conn = Connection::open(path)
            .with_context(|| format!("open sqlite: {}", path.display()))?;

        // WAL gives us better concurrency between Rust and forthcoming tauri
        // plugins / external readers.
        conn.pragma_update(None, "journal_mode", "WAL")?;
        conn.pragma_update(None, "synchronous", "NORMAL")?;

        conn.execute_batch(
            "CREATE TABLE IF NOT EXISTS records (
                id TEXT PRIMARY KEY,
                data TEXT,
                seq INTEGER DEFAULT 0,
                updated_at INTEGER
            );",
        )?;

        Ok(Self {
            inner: Arc::new(Mutex::new(conn)),
        })
    }

    pub fn conn(&self) -> Arc<Mutex<Connection>> {
        Arc::clone(&self.inner)
    }
}

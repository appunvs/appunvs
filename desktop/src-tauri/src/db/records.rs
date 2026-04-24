//! CRUD over the `records` table + helpers for applying broadcasts from relay.

use anyhow::{Context, Result};
use rusqlite::params;
use serde::Serialize;

use super::Db;
use crate::wire::{Message, Op};

#[derive(Debug, Clone, Serialize)]
pub struct Record {
    pub id: String,
    pub data: String,
    pub seq: i64,
    pub updated_at: i64,
}

impl Db {
    pub fn upsert(&self, id: &str, data: &str, seq: i64, updated_at: i64) -> Result<Record> {
        let conn = self.conn();
        let guard = conn.lock().expect("db mutex poisoned");
        guard.execute(
            "INSERT INTO records (id, data, seq, updated_at)
             VALUES (?1, ?2, ?3, ?4)
             ON CONFLICT(id) DO UPDATE SET
                data = excluded.data,
                seq = MAX(records.seq, excluded.seq),
                updated_at = excluded.updated_at",
            params![id, data, seq, updated_at],
        )?;
        Ok(Record {
            id: id.to_string(),
            data: data.to_string(),
            seq,
            updated_at,
        })
    }

    pub fn delete(&self, id: &str) -> Result<()> {
        let conn = self.conn();
        let guard = conn.lock().expect("db mutex poisoned");
        guard
            .execute("DELETE FROM records WHERE id = ?1", params![id])
            .context("delete record")?;
        Ok(())
    }

    pub fn query_all(&self) -> Result<Vec<Record>> {
        let conn = self.conn();
        let guard = conn.lock().expect("db mutex poisoned");
        let mut stmt = guard.prepare(
            "SELECT id, data, seq, updated_at FROM records ORDER BY updated_at ASC, id ASC",
        )?;
        let rows = stmt
            .query_map([], |row| {
                Ok(Record {
                    id: row.get::<_, String>(0)?,
                    data: row.get::<_, Option<String>>(1)?.unwrap_or_default(),
                    seq: row.get::<_, i64>(2)?,
                    updated_at: row.get::<_, Option<i64>>(3)?.unwrap_or(0),
                })
            })?
            .collect::<Result<Vec<_>, _>>()?;
        Ok(rows)
    }

    pub fn max_seq(&self) -> Result<i64> {
        let conn = self.conn();
        let guard = conn.lock().expect("db mutex poisoned");
        let n: i64 = guard
            .query_row("SELECT COALESCE(MAX(seq), 0) FROM records", [], |r| r.get(0))
            .unwrap_or(0);
        Ok(n)
    }

    /// Apply an inbound broadcast `Message` to local storage.
    /// Returns the affected record (for UI refresh) or `None` for deletes.
    pub fn apply_broadcast(&self, m: &Message) -> Result<BroadcastEffect> {
        let id = m
            .payload_id()
            .ok_or_else(|| anyhow::anyhow!("message payload missing id"))?
            .to_string();
        match m.op {
            Op::Upsert => {
                let data = m
                    .payload
                    .as_ref()
                    .and_then(|p| p.get("data"))
                    .and_then(|v| v.as_str())
                    .unwrap_or("")
                    .to_string();
                let rec = self.upsert(&id, &data, m.seq, m.ts)?;
                Ok(BroadcastEffect::Upsert(rec))
            }
            Op::Delete => {
                self.delete(&id)?;
                Ok(BroadcastEffect::Delete { id })
            }
            // Schema mutations and billing guardrails don't target the records
            // store — ignore them here; the UI layer observes them via its own
            // event handler once that lands.
            Op::TableCreate
            | Op::TableDelete
            | Op::ColumnAdd
            | Op::ColumnDelete
            | Op::QuotaExceeded => Ok(BroadcastEffect::Ignored),
            Op::OpUnspecified => Err(anyhow::anyhow!("unspecified op")),
        }
    }
}

#[derive(Debug, Clone)]
pub enum BroadcastEffect {
    Upsert(Record),
    Delete { id: String },
    // Non-records broadcast (schema change / quota guardrail). The caller
    // surfaces these via a different channel.
    Ignored,
}

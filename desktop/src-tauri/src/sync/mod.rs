//! Role-aware sync engine.
//!
//! Incoming messages from the relay land in [`SyncEngine::on_message`]. The
//! engine applies them to SQLite, emits `records-changed` for the UI, and
//! (if our role includes `provider`) rebroadcasts connector requests.
//!
//! Outbound write helpers (`add_record`, `delete_record`) mirror the browser
//! engine: a local provider writes the DB immediately and pushes, while a
//! pure connector only pushes and waits for the provider's echo.

use std::sync::atomic::{AtomicI64, Ordering};
use std::sync::Arc;

use anyhow::Result;
use serde::Serialize;
use serde_json::{Map, Value};
use tauri::{AppHandle, Emitter};
use tokio::sync::Mutex;

use crate::db::{records::BroadcastEffect, Db, Record};
use crate::relay::RelayHandle;
use crate::wire::{Message, Role};

pub const EVT_RECORDS_CHANGED: &str = "appunvs://records-changed";

#[derive(Debug, Clone, Serialize)]
#[serde(tag = "op", rename_all = "snake_case")]
pub enum RecordsChangedEvent {
    Upsert { record: Record },
    Delete { record: DeletedRecord },
}

#[derive(Debug, Clone, Serialize)]
pub struct DeletedRecord {
    pub id: String,
    pub data: String,
    pub seq: i64,
    pub updated_at: i64,
}

/// Shared across the tauri command layer and the relay reader task.
#[derive(Clone)]
pub struct SyncEngine {
    pub app: AppHandle,
    pub db: Db,
    pub relay: Arc<Mutex<Option<RelayHandle>>>,
    pub role: Arc<Mutex<Role>>,
    pub device_id: String,
    pub user_id: String,
    pub last_seen_seq: Arc<AtomicI64>,
}

impl SyncEngine {
    pub fn new(app: AppHandle, db: Db, device_id: String, user_id: String) -> Self {
        let last = db.max_seq().unwrap_or(0);
        Self {
            app,
            db,
            relay: Arc::new(Mutex::new(None)),
            role: Arc::new(Mutex::new(Role::Both)),
            device_id,
            user_id,
            last_seen_seq: Arc::new(AtomicI64::new(last)),
        }
    }

    pub async fn attach_relay(&self, relay: RelayHandle) {
        let mut g = self.relay.lock().await;
        *g = Some(relay);
    }

    pub async fn set_role(&self, r: Role) -> Result<()> {
        {
            let mut g = self.role.lock().await;
            *g = r;
        }
        if let Some(relay) = self.relay.lock().await.as_ref() {
            relay.set_role(r).await?;
        }
        Ok(())
    }

    pub async fn current_role(&self) -> Role {
        *self.role.lock().await
    }

    pub fn last_seq(&self) -> i64 {
        self.last_seen_seq.load(Ordering::Relaxed)
    }

    /// Handle a single inbound broadcast from the relay.
    pub async fn on_message(&self, m: Message) {
        if let Err(err) = self.on_message_inner(m).await {
            tracing::warn!(error = %err, "sync: on_message failed");
        }
    }

    async fn on_message_inner(&self, m: Message) -> Result<()> {
        // Ignore messages outside our namespace (defense in depth; relay already filters).
        if m.namespace != self.user_id && !m.namespace.is_empty() {
            return Ok(());
        }

        let my_role = self.current_role().await;

        match m.role {
            Role::Provider => {
                // Provider broadcast — authoritative. Apply sequentially.
                let prior = self.last_seen_seq.load(Ordering::Relaxed);
                if m.seq > 0 && prior > 0 && m.seq != prior + 1 {
                    tracing::warn!(
                        prior,
                        incoming = m.seq,
                        "sync: seq gap detected; forcing reconnect"
                    );
                    // Drop the relay handle so the actor's select! breaks.
                    if let Some(relay) = self.relay.lock().await.as_ref() {
                        // Best effort: send a Shutdown that our actor treats as disconnect
                        // — the reconnect loop will re-fetch max_seq and trigger replay.
                        let _ = relay
                            .send(Message {
                                // sentinel we never actually build; intentionally no-op.
                                ..m.clone()
                            })
                            .await;
                    }
                    return Ok(());
                }

                let effect = self.db.apply_broadcast(&m)?;
                if m.seq > 0 {
                    self.last_seen_seq.fetch_max(m.seq, Ordering::Relaxed);
                }
                self.emit_effect(effect, &m);
            }
            Role::Connector => {
                if my_role.includes_provider() {
                    // We own the data — apply and rebroadcast as provider.
                    let effect = self.db.apply_broadcast(&m)?;
                    self.emit_effect(effect.clone(), &m);
                    self.rebroadcast_as_provider(&m).await?;
                } else {
                    tracing::debug!("sync: ignoring connector msg (no provider role)");
                }
            }
            Role::Both | Role::RoleUnspecified => {
                // Treat like provider for apply purposes.
                let effect = self.db.apply_broadcast(&m)?;
                if m.seq > 0 {
                    self.last_seen_seq.fetch_max(m.seq, Ordering::Relaxed);
                }
                self.emit_effect(effect, &m);
            }
        }

        Ok(())
    }

    fn emit_effect(&self, effect: BroadcastEffect, m: &Message) {
        let evt = match effect {
            BroadcastEffect::Upsert(rec) => RecordsChangedEvent::Upsert { record: rec },
            BroadcastEffect::Delete { id } => RecordsChangedEvent::Delete {
                record: DeletedRecord {
                    id,
                    data: String::new(),
                    seq: m.seq,
                    updated_at: m.ts,
                },
            },
            // Non-records broadcast (schema change / quota guardrail). No
            // local-state effect to emit today; the UI can subscribe to the
            // raw message stream if/when it cares.
            BroadcastEffect::Ignored => return,
        };
        let _ = self.app.emit(EVT_RECORDS_CHANGED, &evt);
    }

    async fn rebroadcast_as_provider(&self, src: &Message) -> Result<()> {
        let mut out = src.clone();
        out.seq = 0; // relay assigns
        out.device_id = self.device_id.clone();
        out.user_id = self.user_id.clone();
        out.namespace = self.user_id.clone();
        out.role = Role::Provider;
        out.ts = now_ms();
        if let Some(relay) = self.relay.lock().await.as_ref() {
            relay.send(out).await?;
        }
        Ok(())
    }

    /// Add or update a record locally (if we own data) and publish to relay.
    pub async fn add_record(&self, id: &str, data: &str) -> Result<()> {
        let ts = now_ms();
        let role = self.current_role().await;

        if role.includes_provider() {
            self.db.upsert(id, data, 0, ts)?;
            let rec = Record {
                id: id.to_string(),
                data: data.to_string(),
                seq: 0,
                updated_at: ts,
            };
            let _ = self
                .app
                .emit(EVT_RECORDS_CHANGED, &RecordsChangedEvent::Upsert { record: rec });
        }

        let mut payload = Map::new();
        payload.insert("id".to_string(), Value::String(id.to_string()));
        payload.insert("data".to_string(), Value::String(data.to_string()));

        let msg = Message::new_upsert(
            &self.device_id,
            &self.user_id,
            role,
            "records",
            payload,
            ts,
        );
        if let Some(relay) = self.relay.lock().await.as_ref() {
            relay.send(msg).await?;
        }
        Ok(())
    }

    pub async fn delete_record(&self, id: &str) -> Result<()> {
        let ts = now_ms();
        let role = self.current_role().await;

        if role.includes_provider() {
            self.db.delete(id)?;
            let _ = self.app.emit(
                EVT_RECORDS_CHANGED,
                &RecordsChangedEvent::Delete {
                    record: DeletedRecord {
                        id: id.to_string(),
                        data: String::new(),
                        seq: 0,
                        updated_at: ts,
                    },
                },
            );
        }

        let msg = Message::new_delete(
            &self.device_id,
            &self.user_id,
            role,
            "records",
            id,
            ts,
        );
        if let Some(relay) = self.relay.lock().await.as_ref() {
            relay.send(msg).await?;
        }
        Ok(())
    }
}

fn now_ms() -> i64 {
    use std::time::{SystemTime, UNIX_EPOCH};
    SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .map(|d| d.as_millis() as i64)
        .unwrap_or(0)
}

//! Actor-style WebSocket client to the relay.
//!
//! A single tokio task owns the socket; the rest of the app talks to it via an
//! mpsc channel. On disconnect we back off 1s -> 2s -> ... -> 30s, refetching
//! `db.max_seq()` each reconnect so the relay's Redis Stream replays gaps.

pub mod auth;

use std::sync::Arc;
use std::time::Duration;

use anyhow::{Context, Result};
use futures_util::{SinkExt, StreamExt};
use tauri::{AppHandle, Emitter};
use tokio::sync::{mpsc, Mutex};
use tokio::time::{interval, sleep};
use tokio_tungstenite::tungstenite::protocol::Message as WsMessage;

use crate::db::Db;
use crate::wire::{Message, Role};

use self::auth::Credentials;

/// Channel name for raw wire messages (front-end debugging / observers).
pub const EVT_MESSAGE: &str = "appunvs://message";
pub const EVT_CONN_STATE: &str = "appunvs://conn-state";

pub type OnMessage = Arc<dyn Fn(Message) + Send + Sync + 'static>;

#[derive(Debug)]
pub enum Cmd {
    Send(Message),
    SetRole(Role),
    Shutdown,
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ConnState {
    Idle,
    Connecting,
    Open,
    Closed,
}

impl ConnState {
    pub fn as_str(self) -> &'static str {
        match self {
            ConnState::Idle => "idle",
            ConnState::Connecting => "connecting",
            ConnState::Open => "open",
            ConnState::Closed => "closed",
        }
    }
}

#[derive(Clone)]
pub struct RelayHandle {
    tx: mpsc::Sender<Cmd>,
    state: Arc<Mutex<ConnState>>,
    role: Arc<Mutex<Role>>,
}

impl RelayHandle {
    pub async fn send(&self, m: Message) -> Result<()> {
        self.tx.send(Cmd::Send(m)).await.context("relay closed")?;
        Ok(())
    }

    pub async fn set_role(&self, r: Role) -> Result<()> {
        {
            let mut guard = self.role.lock().await;
            *guard = r;
        }
        self.tx.send(Cmd::SetRole(r)).await.context("relay closed")?;
        Ok(())
    }

    pub async fn conn_state(&self) -> ConnState {
        *self.state.lock().await
    }

    pub async fn role(&self) -> Role {
        *self.role.lock().await
    }
}

/// Spawn the relay actor. Returns a handle usable from any tokio context.
pub fn spawn(
    app: AppHandle,
    db: Db,
    creds: Credentials,
    ws_base: String,
    on_message: OnMessage,
) -> RelayHandle {
    let (tx, rx) = mpsc::channel::<Cmd>(128);
    let state = Arc::new(Mutex::new(ConnState::Idle));
    let role = Arc::new(Mutex::new(Role::Both));

    let handle = RelayHandle {
        tx,
        state: Arc::clone(&state),
        role: Arc::clone(&role),
    };

    tokio::spawn(run(app, db, creds, ws_base, rx, state, role, on_message));

    handle
}

#[allow(clippy::too_many_arguments)]
async fn run(
    app: AppHandle,
    db: Db,
    creds: Credentials,
    ws_base: String,
    mut rx: mpsc::Receiver<Cmd>,
    state: Arc<Mutex<ConnState>>,
    _role: Arc<Mutex<Role>>,
    on_message: OnMessage,
) {
    let mut backoff = Duration::from_secs(1);
    let max_backoff = Duration::from_secs(30);

    loop {
        set_state(&app, &state, ConnState::Connecting).await;

        let last_seq = db.max_seq().unwrap_or(0);
        let url = format!(
            "{}/ws?token={}&last_seq={}",
            ws_base.trim_end_matches('/'),
            urlencoded(&creds.token),
            last_seq
        );

        tracing::info!(url = %mask_token(&url), "relay: connecting");
        let connect = tokio_tungstenite::connect_async(&url).await;

        let ws = match connect {
            Ok((ws, _)) => {
                backoff = Duration::from_secs(1);
                ws
            }
            Err(err) => {
                tracing::warn!(error = %err, "relay connect failed; backing off");
                set_state(&app, &state, ConnState::Closed).await;
                // Drain any Cmds that arrived while we were waiting; allow Shutdown
                // to break us out immediately.
                tokio::select! {
                    _ = sleep(backoff) => {}
                    cmd = rx.recv() => {
                        if matches!(cmd, Some(Cmd::Shutdown) | None) { return; }
                    }
                }
                backoff = (backoff * 2).min(max_backoff);
                continue;
            }
        };

        set_state(&app, &state, ConnState::Open).await;
        tracing::info!("relay: open");

        let (mut sink, mut stream) = ws.split();
        let mut ping_tick = interval(Duration::from_secs(30));
        ping_tick.tick().await; // skip the immediate tick

        let disconnect = loop {
            tokio::select! {
                frame = stream.next() => {
                    match frame {
                        Some(Ok(WsMessage::Text(t))) => {
                            match serde_json::from_str::<Message>(&t) {
                                Ok(msg) => {
                                    let _ = app.emit(EVT_MESSAGE, &msg);
                                    on_message(msg);
                                }
                                Err(err) => {
                                    tracing::warn!(error = %err, raw = %t, "relay: bad frame");
                                }
                            }
                        }
                        Some(Ok(WsMessage::Binary(_))) => {
                            tracing::warn!("relay: unexpected binary frame");
                        }
                        Some(Ok(WsMessage::Ping(p))) => {
                            let _ = sink.send(WsMessage::Pong(p)).await;
                        }
                        Some(Ok(WsMessage::Pong(_))) => {}
                        Some(Ok(WsMessage::Close(_))) => break "remote closed",
                        Some(Ok(WsMessage::Frame(_))) => {}
                        Some(Err(err)) => {
                            tracing::warn!(error = %err, "relay: read error");
                            break "read error";
                        }
                        None => break "stream ended",
                    }
                }
                cmd = rx.recv() => {
                    match cmd {
                        Some(Cmd::Send(m)) => {
                            match serde_json::to_string(&m) {
                                Ok(json) => {
                                    if let Err(err) = sink.send(WsMessage::Text(json.into())).await {
                                        tracing::warn!(error = %err, "relay: write error");
                                        break "write error";
                                    }
                                }
                                Err(err) => {
                                    tracing::warn!(error = %err, "relay: encode error");
                                }
                            }
                        }
                        Some(Cmd::SetRole(r)) => {
                            tracing::info!(role = ?r, "relay: role change");
                            // Role is consulted when constructing outbound messages;
                            // relay does not need an explicit role-switch frame.
                        }
                        Some(Cmd::Shutdown) | None => {
                            let _ = sink.send(WsMessage::Close(None)).await;
                            return;
                        }
                    }
                }
                _ = ping_tick.tick() => {
                    if let Err(err) = sink.send(WsMessage::Ping(Vec::new().into())).await {
                        tracing::warn!(error = %err, "relay: ping failed");
                        break "ping failed";
                    }
                }
            }
        };

        tracing::info!(reason = disconnect, "relay: closed");
        set_state(&app, &state, ConnState::Closed).await;

        // brief pause before reconnect; reset backoff on clean disconnects
        tokio::select! {
            _ = sleep(backoff) => {}
            cmd = rx.recv() => {
                if matches!(cmd, Some(Cmd::Shutdown) | None) { return; }
            }
        }
        backoff = (backoff * 2).min(max_backoff);
    }
}

async fn set_state(app: &AppHandle, state: &Arc<Mutex<ConnState>>, new: ConnState) {
    {
        let mut g = state.lock().await;
        if *g == new {
            return;
        }
        *g = new;
    }
    let _ = app.emit(EVT_CONN_STATE, new.as_str());
}

fn urlencoded(s: &str) -> String {
    url::form_urlencoded::byte_serialize(s.as_bytes()).collect()
}

fn mask_token(url: &str) -> String {
    // Trim the token from debug logs.
    match url.find("token=") {
        Some(i) => {
            let start = i + "token=".len();
            let end = url[start..]
                .find('&')
                .map(|n| start + n)
                .unwrap_or(url.len());
            format!("{}token=***{}", &url[..start], &url[end..])
        }
        None => url.to_string(),
    }
}

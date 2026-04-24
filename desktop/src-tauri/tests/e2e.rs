//! Cross-language end-to-end: drive the real Go relay using the Rust wire
//! codec.  Proves desktop and relay agree on the JSON shape.
//!
//! Requires a live relay at APPUNVS_RELAY_BASE (default http://localhost:8080)
//! and Redis behind it.  If the relay is unreachable the test prints a notice
//! and returns successfully so `cargo test` on a laptop without a running
//! relay stays green.

use std::time::Duration;

use appunvs_desktop_lib::wire::{Message, Op, RegisterResponse, Role};
use futures_util::{SinkExt, StreamExt};
use serde_json::{json, Map, Value};
use tokio::time::timeout;
use tokio_tungstenite::tungstenite::Message as WsMessage;

fn base() -> String {
    std::env::var("APPUNVS_RELAY_BASE").unwrap_or_else(|_| "http://localhost:8080".to_string())
}

fn ws_base() -> String {
    base().replacen("http", "ws", 1)
}

async fn relay_up() -> bool {
    reqwest::Client::new()
        .get(format!("{}/health", base()))
        .timeout(Duration::from_millis(500))
        .send()
        .await
        .map(|r| r.status().is_success())
        .unwrap_or(false)
}

async fn register(device_id: &str) -> RegisterResponse {
    reqwest::Client::new()
        .post(format!("{}/auth/register", base()))
        .json(&json!({"device_id": device_id, "platform": "desktop"}))
        .send()
        .await
        .expect("register")
        .json()
        .await
        .expect("decode register response")
}

async fn dial(
    token: &str,
    last_seq: Option<i64>,
) -> tokio_tungstenite::WebSocketStream<tokio_tungstenite::MaybeTlsStream<tokio::net::TcpStream>> {
    let mut url = format!("{}/ws?token={}", ws_base(), token);
    if let Some(s) = last_seq {
        url.push_str(&format!("&last_seq={}", s));
    }
    let (ws, _) = tokio_tungstenite::connect_async(&url).await.expect("ws dial");
    ws
}

async fn next_message<S>(
    ws: &mut tokio_tungstenite::WebSocketStream<S>,
) -> Option<Message>
where
    S: tokio::io::AsyncRead + tokio::io::AsyncWrite + Unpin,
{
    let frame = timeout(Duration::from_secs(3), ws.next()).await.ok()??;
    match frame.ok()? {
        WsMessage::Text(s) => serde_json::from_str(&s).ok(),
        _ => None,
    }
}

#[tokio::test]
async fn provider_upsert_echoes_with_seq() {
    if !relay_up().await {
        eprintln!("relay not reachable; skipping");
        return;
    }
    let reg = register("rs-e2e-device").await;
    let mut ws = dial(&reg.token, None).await;

    let mut payload = Map::new();
    payload.insert("id".into(), Value::String("r1".into()));
    payload.insert("data".into(), Value::String("rust-says-hi".into()));

    let msg = Message {
        seq: 0,
        device_id: "rs-e2e-device".into(),
        user_id: reg.user_id.clone(),
        namespace: reg.user_id.clone(),
        role: Role::Provider,
        op: Op::Upsert,
        table: "records".into(),
        payload: Some(payload.clone()),
        ts: 1_714_000_000_000,
    };

    ws.send(WsMessage::Text(serde_json::to_string(&msg).unwrap().into()))
        .await
        .expect("send");

    let echo = next_message(&mut ws).await.expect("no echo from relay");
    assert!(echo.seq > 0, "relay should assign seq, got {}", echo.seq);
    assert_eq!(echo.namespace, reg.user_id);
    assert_eq!(echo.role, Role::Provider);
    assert_eq!(echo.op, Op::Upsert);
    assert_eq!(echo.payload.as_ref().unwrap().get("id"), payload.get("id"));

    ws.close(None).await.ok();
}

#[tokio::test]
async fn catchup_on_reconnect_replays_missed_messages() {
    if !relay_up().await {
        eprintln!("relay not reachable; skipping");
        return;
    }
    let reg = register("rs-e2e-catchup").await;

    // First socket sends a message, records the seq, then closes.
    let mut ws = dial(&reg.token, None).await;
    ws.send(WsMessage::Text(
        serde_json::to_string(&Message::new_upsert(
            "rs-e2e-catchup",
            &reg.user_id,
            Role::Provider,
            "records",
            {
                let mut m = Map::new();
                m.insert("id".into(), Value::String("seen".into()));
                m
            },
            1,
        ))
        .unwrap()
        .into(),
    ))
    .await
    .unwrap();
    let first = next_message(&mut ws).await.expect("no echo");
    let last_seq = first.seq;
    assert!(last_seq > 0);
    ws.close(None).await.ok();

    // Second socket (same identity) publishes a message while the first is
    // "offline".  Its self-echo tells us the new seq.
    let mut other = dial(&reg.token, None).await;
    other
        .send(WsMessage::Text(
            serde_json::to_string(&Message::new_upsert(
                "rs-e2e-catchup",
                &reg.user_id,
                Role::Provider,
                "records",
                {
                    let mut m = Map::new();
                    m.insert("id".into(), Value::String("while-offline".into()));
                    m
                },
                2,
            ))
            .unwrap()
            .into(),
        ))
        .await
        .unwrap();
    let published = next_message(&mut other).await.expect("no echo");
    assert_eq!(published.seq, last_seq + 1);
    other.close(None).await.ok();

    // Reconnect with last_seq → expect replay.
    let mut back = dial(&reg.token, Some(last_seq)).await;
    let replayed = next_message(&mut back).await.expect("no replay");
    assert_eq!(replayed.seq, last_seq + 1);
    assert_eq!(
        replayed.payload.as_ref().unwrap().get("id"),
        Some(&Value::String("while-offline".into()))
    );
    back.close(None).await.ok();
}

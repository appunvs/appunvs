//! `#[tauri::command]` glue. Every command returns `Result<T, String>` so that
//! errors surface as a plain string on the front-end without needing a custom
//! error type in JS.

use serde::Serialize;
use tauri::State;

use crate::db::Record;
use crate::state::AppState;
use crate::wire::Role;

#[derive(Debug, Clone, Serialize)]
pub struct SyncStatus {
    pub conn_state: String,
    pub role: String,
    pub last_seq: i64,
}

#[tauri::command]
pub async fn write_record(
    state: State<'_, AppState>,
    id: String,
    data: String,
) -> Result<(), String> {
    state
        .sync
        .add_record(&id, &data)
        .await
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn delete_record(state: State<'_, AppState>, id: String) -> Result<(), String> {
    state
        .sync
        .delete_record(&id)
        .await
        .map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn query_records(state: State<'_, AppState>) -> Result<Vec<Record>, String> {
    state.db.query_all().map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn set_role(state: State<'_, AppState>, role: String) -> Result<(), String> {
    let r = Role::parse(&role).ok_or_else(|| format!("unknown role: {}", role))?;
    state.sync.set_role(r).await.map_err(|e| e.to_string())
}

#[tauri::command]
pub async fn get_sync_status(state: State<'_, AppState>) -> Result<SyncStatus, String> {
    let conn_state = match state.sync.relay.lock().await.as_ref() {
        Some(relay) => relay.conn_state().await.as_str().to_string(),
        None => "idle".to_string(),
    };
    let role = state.sync.current_role().await.as_wire().to_string();
    Ok(SyncStatus {
        conn_state,
        role,
        last_seq: state.sync.last_seq(),
    })
}

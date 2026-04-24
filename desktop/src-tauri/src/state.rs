//! App-wide state made available to every `#[tauri::command]` via `State<'_, AppState>`.

use crate::db::Db;
use crate::sync::SyncEngine;

#[derive(Clone)]
pub struct AppState {
    pub db: Db,
    pub sync: SyncEngine,
}

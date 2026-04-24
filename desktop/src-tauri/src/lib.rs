//! Library crate — `main.rs` re-exports `run()` from here so Tauri's mobile
//! target (which expects a library entry point) can also use it.

use std::sync::Arc;

use tauri::Manager;
use tracing_subscriber::EnvFilter;

pub mod commands;
pub mod config;
pub mod db;
pub mod relay;
pub mod state;
pub mod sync;
pub mod wire;

use crate::db::Db;
use crate::state::AppState;
use crate::sync::SyncEngine;

#[cfg_attr(mobile, tauri::mobile_entry_point)]
pub fn run() {
    let _ = tracing_subscriber::fmt()
        .with_env_filter(
            EnvFilter::try_from_env("APPUNVS_LOG")
                .unwrap_or_else(|_| EnvFilter::new("info,appunvs_desktop_lib=debug")),
        )
        .try_init();

    tauri::Builder::default()
        .setup(|app| {
            let handle = app.handle().clone();

            // Resolve app data dir (platform-specific). Fallback to a local dir if
            // the resolver fails (e.g. headless CI).
            let data_dir = app
                .path()
                .app_data_dir()
                .unwrap_or_else(|_| std::env::temp_dir().join("appunvs"));
            std::fs::create_dir_all(&data_dir).ok();

            let db = Db::new(&data_dir.join("appunvs.db"))?;

            // Heavy init (HTTP registration, WS connect) runs on the tokio runtime
            // Tauri already drives.
            tauri::async_runtime::spawn(async move {
                let relay_base = config::relay_base();

                let creds = match relay::auth::ensure_credentials(&data_dir, &relay_base).await {
                    Ok(c) => c,
                    Err(err) => {
                        tracing::error!(error = %err, "auth: could not register");
                        return;
                    }
                };

                let sync = SyncEngine::new(
                    handle.clone(),
                    db.clone(),
                    creds.device_id.clone(),
                    creds.user_id.clone(),
                );

                // Expose state to commands *before* the relay starts pumping events.
                handle.manage(AppState {
                    db: db.clone(),
                    sync: sync.clone(),
                });

                let ws_base = config::ws_base(&relay_base);
                let engine = sync.clone();
                let on_message: relay::OnMessage = Arc::new(move |m| {
                    let eng = engine.clone();
                    tauri::async_runtime::spawn(async move {
                        eng.on_message(m).await;
                    });
                });

                let relay_handle = relay::spawn(
                    handle.clone(),
                    db.clone(),
                    creds,
                    ws_base,
                    on_message,
                );

                sync.attach_relay(relay_handle).await;
            });

            Ok(())
        })
        .invoke_handler(tauri::generate_handler![
            commands::write_record,
            commands::delete_record,
            commands::query_records,
            commands::set_role,
            commands::get_sync_status,
        ])
        .run(tauri::generate_context!())
        .expect("error while running tauri application");
}

//! Runtime configuration + on-disk persistence for device_id / token.
//!
//! The token cache is stored under `app_data_dir()/config.json`. It is *not*
//! encrypted — we rely on OS user-level protection of the app data directory.
//! For a future hardening pass, move it to the OS keyring (e.g. `keyring` crate).

use std::fs;
use std::path::{Path, PathBuf};

use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use uuid::Uuid;

/// Relay HTTP base URL, e.g. `http://localhost:8080`.
/// Override via `APPUNVS_RELAY_BASE`.
pub fn relay_base() -> String {
    std::env::var("APPUNVS_RELAY_BASE").unwrap_or_else(|_| "http://localhost:8080".to_string())
}

/// Convert an `http(s)://` base URL to `ws(s)://`.
pub fn ws_base(http_base: &str) -> String {
    if let Some(rest) = http_base.strip_prefix("https://") {
        format!("wss://{}", rest)
    } else if let Some(rest) = http_base.strip_prefix("http://") {
        format!("ws://{}", rest)
    } else {
        http_base.to_string()
    }
}

#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct PersistedConfig {
    pub device_id: Option<String>,
    pub token: Option<String>,
    pub user_id: Option<String>,
}

impl PersistedConfig {
    pub fn path(data_dir: &Path) -> PathBuf {
        data_dir.join("config.json")
    }

    pub fn load(data_dir: &Path) -> Self {
        let p = Self::path(data_dir);
        match fs::read_to_string(&p) {
            Ok(s) => serde_json::from_str(&s).unwrap_or_default(),
            Err(_) => Self::default(),
        }
    }

    pub fn save(&self, data_dir: &Path) -> Result<()> {
        fs::create_dir_all(data_dir)
            .with_context(|| format!("create app data dir: {}", data_dir.display()))?;
        let p = Self::path(data_dir);
        let s = serde_json::to_string_pretty(self)?;
        fs::write(&p, s).with_context(|| format!("write config: {}", p.display()))?;
        Ok(())
    }

    /// Returns the persisted device_id, generating + saving one if missing.
    pub fn ensure_device_id(&mut self, data_dir: &Path) -> Result<String> {
        if let Some(id) = &self.device_id {
            return Ok(id.clone());
        }
        let id = format!("desktop_{}", Uuid::new_v4());
        self.device_id = Some(id.clone());
        self.save(data_dir)?;
        Ok(id)
    }
}

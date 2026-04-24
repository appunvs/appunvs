//! Account HTTP client used by the desktop app. Covers the four endpoints
//! that matter for bring-up:
//!
//! - POST /auth/signup   — create account, return session JWT
//! - POST /auth/login    — verify credentials, return session JWT
//! - POST /auth/register — persist this device under the session, return device JWT
//! - GET  /auth/me       — profile + device list
//!
//! The actual login UI is not wired yet (no Tauri command surface). Until the
//! UI lands, `ensure_credentials` only succeeds when the user has already
//! persisted a device token into relay_config.json via some other tool —
//! auto-registering without a session no longer works, the server rejects it.

use std::path::Path;

use anyhow::{Context, Result, anyhow};
use reqwest::Client;

use crate::config::PersistedConfig;
use crate::wire::{
    AuthCredentials, MeResponse, Platform, RegisterRequest, RegisterResponse, SessionResponse,
};

pub struct Credentials {
    pub device_id: String,
    pub user_id: String,
    pub token: String,
}

/// AccountClient is the thin HTTP wrapper. Constructed per operation so the
/// short-lived connection pool doesn't outlive the action; for high-frequency
/// calls hold a shared `reqwest::Client` and inject it instead.
pub struct AccountClient {
    base: String,
    client: Client,
}

impl AccountClient {
    pub fn new(base: &str) -> Result<Self> {
        let client = Client::builder().user_agent("appunvs-desktop/0.1").build()?;
        Ok(Self {
            base: base.trim_end_matches('/').to_string(),
            client,
        })
    }

    pub async fn signup(&self, email: &str, password: &str) -> Result<SessionResponse> {
        self.post_json::<_, SessionResponse>(
            "/auth/signup",
            &AuthCredentials {
                email: email.into(),
                password: password.into(),
            },
            None,
        )
        .await
    }

    pub async fn login(&self, email: &str, password: &str) -> Result<SessionResponse> {
        self.post_json::<_, SessionResponse>(
            "/auth/login",
            &AuthCredentials {
                email: email.into(),
                password: password.into(),
            },
            None,
        )
        .await
    }

    pub async fn register_device(
        &self,
        session_token: &str,
        device_id: &str,
        platform: Platform,
    ) -> Result<RegisterResponse> {
        self.post_json::<_, RegisterResponse>(
            "/auth/register",
            &RegisterRequest {
                device_id: device_id.to_string(),
                platform,
            },
            Some(session_token),
        )
        .await
    }

    pub async fn me(&self, session_token: &str) -> Result<MeResponse> {
        let url = format!("{}/auth/me", self.base);
        let resp = self
            .client
            .get(&url)
            .bearer_auth(session_token)
            .send()
            .await
            .with_context(|| format!("GET {}", url))?;
        if !resp.status().is_success() {
            let code = resp.status();
            let body = resp.text().await.unwrap_or_default();
            return Err(anyhow!("/auth/me {}: {}", code, body));
        }
        Ok(resp.json::<MeResponse>().await?)
    }

    async fn post_json<Req: serde::Serialize, Resp: serde::de::DeserializeOwned>(
        &self,
        path: &str,
        body: &Req,
        session_token: Option<&str>,
    ) -> Result<Resp> {
        let url = format!("{}{}", self.base, path);
        let mut req = self.client.post(&url).json(body);
        if let Some(tok) = session_token {
            req = req.bearer_auth(tok);
        }
        let resp = req
            .send()
            .await
            .with_context(|| format!("POST {}", url))?;
        if !resp.status().is_success() {
            let code = resp.status();
            let text = resp.text().await.unwrap_or_default();
            return Err(anyhow!("POST {} {}: {}", path, code, text));
        }
        Ok(resp.json::<Resp>().await?)
    }
}

/// Ensure we have a valid `(device_id, user_id, token)` triple from the
/// persisted config. Returns Err when no cached device token is present —
/// auto-registration against an unauthenticated relay is no longer possible
/// now that /auth/register requires a session JWT.
///
/// When the desktop login UI lands, the call site should instead:
///   1. Show a sign-in form.
///   2. Call AccountClient::signup / login to get a session token.
///   3. Call AccountClient::register_device to get a device token.
///   4. Persist both via PersistedConfig and start the RelayActor.
pub async fn ensure_credentials(data_dir: &Path, _relay_base: &str) -> Result<Credentials> {
    let mut cfg = PersistedConfig::load(data_dir);
    let device_id = cfg.ensure_device_id(data_dir)?;

    match (cfg.token.clone(), cfg.user_id.clone()) {
        (Some(token), Some(user_id)) => Ok(Credentials {
            device_id,
            user_id,
            token,
        }),
        _ => Err(anyhow!(
            "no cached session — desktop login UI not yet wired; \
             expected token+user_id in relay_config.json"
        )),
    }
}

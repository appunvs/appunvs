//! Hand-written protojson mirror of `shared/proto/appunvs.proto`.
//!
//! Wire format: canonical protojson with `UseProtoNames` enabled.
//! - Field names are snake_case (enforced via `#[serde(rename_all = "snake_case")]`).
//! - Enum values serialize as short lowercase strings (`"provider"`, `"upsert"`, ...).
//! - `seq` is omitted when zero so clients don't accidentally claim a sequence.

use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Role {
    RoleUnspecified,
    Provider,
    Connector,
    Both,
}

impl Role {
    pub fn as_wire(self) -> &'static str {
        match self {
            Role::RoleUnspecified => "role_unspecified",
            Role::Provider => "provider",
            Role::Connector => "connector",
            Role::Both => "both",
        }
    }

    pub fn parse(s: &str) -> Option<Self> {
        match s {
            "provider" => Some(Role::Provider),
            "connector" => Some(Role::Connector),
            "both" => Some(Role::Both),
            "role_unspecified" => Some(Role::RoleUnspecified),
            _ => None,
        }
    }

    pub fn includes_provider(self) -> bool {
        matches!(self, Role::Provider | Role::Both)
    }

    pub fn includes_connector(self) -> bool {
        matches!(self, Role::Connector | Role::Both)
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Op {
    OpUnspecified,
    Upsert,
    Delete,
    TableCreate,
    TableDelete,
    ColumnAdd,
    ColumnDelete,
    QuotaExceeded,
}

#[derive(Debug, Default, Clone, Copy, PartialEq, Eq, Serialize, Deserialize)]
#[serde(rename_all = "snake_case")]
pub enum Platform {
    #[default]
    PlatformUnspecified,
    Browser,
    Desktop,
    Mobile,
}

fn is_zero(v: &i64) -> bool {
    *v == 0
}
fn is_role_unspecified(r: &Role) -> bool {
    matches!(r, Role::RoleUnspecified)
}
fn is_op_unspecified(o: &Op) -> bool {
    matches!(o, Op::OpUnspecified)
}

/// Wire envelope. Matches `appunvs.v1.Message`.
///
/// Scalar fields with a default value are omitted on the wire (proto3 /
/// protojson convention). Parsing tolerates any subset of fields for forward
/// and backward compatibility with schema broadcasts that don't carry a
/// device_id, etc.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct Message {
    #[serde(default, skip_serializing_if = "is_zero")]
    pub seq: i64,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub device_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub user_id: String,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub namespace: String,
    #[serde(default = "default_role", skip_serializing_if = "is_role_unspecified")]
    pub role: Role,
    #[serde(default = "default_op", skip_serializing_if = "is_op_unspecified")]
    pub op: Op,
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub table: String,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub payload: Option<Map<String, Value>>,
    #[serde(default, skip_serializing_if = "is_zero")]
    pub ts: i64,
}

fn default_role() -> Role {
    Role::RoleUnspecified
}
fn default_op() -> Op {
    Op::OpUnspecified
}

impl Message {
    pub fn new_upsert(
        device_id: &str,
        user_id: &str,
        role: Role,
        table: &str,
        payload: Map<String, Value>,
        ts: i64,
    ) -> Self {
        Self {
            seq: 0,
            device_id: device_id.to_string(),
            user_id: user_id.to_string(),
            namespace: user_id.to_string(),
            role,
            op: Op::Upsert,
            table: table.to_string(),
            payload: Some(payload),
            ts,
        }
    }

    pub fn new_delete(
        device_id: &str,
        user_id: &str,
        role: Role,
        table: &str,
        id: &str,
        ts: i64,
    ) -> Self {
        let mut p = Map::new();
        p.insert("id".to_string(), Value::String(id.to_string()));
        Self {
            seq: 0,
            device_id: device_id.to_string(),
            user_id: user_id.to_string(),
            namespace: user_id.to_string(),
            role,
            op: Op::Delete,
            table: table.to_string(),
            payload: Some(p),
            ts,
        }
    }

    pub fn payload_id(&self) -> Option<&str> {
        self.payload.as_ref()?.get("id")?.as_str()
    }
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterRequest {
    pub device_id: String,
    pub platform: Platform,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RegisterResponse {
    pub token: String,
    pub user_id: String,
}

/// Payloads for /auth/signup and /auth/login — mirror appunvs.v1.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct AuthCredentials {
    pub email: String,
    pub password: String,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct SessionResponse {
    pub user_id: String,
    pub session_token: String,
}

/// Response shape for /auth/me.
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MeResponse {
    #[serde(default)]
    pub user_id: String,
    #[serde(default)]
    pub email: String,
    #[serde(default)]
    pub created_at: i64,
    #[serde(default)]
    pub devices: Vec<RemoteDevice>,
}

#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RemoteDevice {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub user_id: String,
    #[serde(default)]
    pub platform: Platform,
    #[serde(default)]
    pub created_at: i64,
    #[serde(default)]
    pub last_seen: i64,
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn role_serializes_as_lowercase_short_name() {
        let j = serde_json::to_string(&Role::Provider).unwrap();
        assert_eq!(j, "\"provider\"");
    }

    #[test]
    fn message_omits_seq_when_zero() {
        let m = Message {
            seq: 0,
            device_id: "d1".into(),
            user_id: "u1".into(),
            namespace: "u1".into(),
            role: Role::Provider,
            op: Op::Upsert,
            table: "records".into(),
            payload: None,
            ts: 1,
        };
        let j = serde_json::to_string(&m).unwrap();
        assert!(!j.contains("\"seq\""), "seq should be omitted when zero: {}", j);
    }
}

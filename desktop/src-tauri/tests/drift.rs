//! Drift check: the repo-wide golden fixture must roundtrip byte-identically
//! through the Rust wire codec.  If this breaks, the Rust implementation has
//! diverged from shared/proto/testdata and protocol interop with the other
//! clients is at risk.

use appunvs_desktop_lib::wire::Message;
use serde_json::{from_str, to_value, Value};

// include_str! path is resolved relative to this source file.  From
// desktop/src-tauri/tests/drift.rs → repo root → shared/proto/testdata.
const GOLDEN: &str = include_str!("../../../shared/proto/testdata/messages.json");

#[derive(serde::Deserialize)]
struct Case {
    name: String,
    message: Value,
}

fn sort_keys(v: &Value) -> Value {
    match v {
        Value::Object(map) => {
            let mut sorted = std::collections::BTreeMap::new();
            for (k, vv) in map {
                sorted.insert(k.clone(), sort_keys(vv));
            }
            Value::Object(sorted.into_iter().collect())
        }
        Value::Array(arr) => Value::Array(arr.iter().map(sort_keys).collect()),
        other => other.clone(),
    }
}

#[test]
fn all_golden_messages_roundtrip() {
    let cases: Vec<Case> = from_str(GOLDEN).expect("parse golden fixture");
    assert!(!cases.is_empty(), "golden fixture is empty");

    for c in cases {
        let parsed: Message = serde_json::from_value(c.message.clone())
            .unwrap_or_else(|e| panic!("case {} parse: {}", c.name, e));
        let produced = to_value(&parsed)
            .unwrap_or_else(|e| panic!("case {} serialize: {}", c.name, e));

        assert_eq!(
            sort_keys(&produced),
            sort_keys(&c.message),
            "drift in case {}",
            c.name
        );
    }
}

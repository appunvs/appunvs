# shared/proto

**Single source of truth** for the wire types shared by every appunvs
component (`relay/` and the host shells under `appunvs/{ios,android}/`).
Hand-written mirrors live alongside the generated output for languages
that prefer them; drift tests guard against divergence.

## Layout

Files are organized by **module**, not by project name — keeps the
schema reorganizable as the product grows and immune to project
renames.  Each file declares `package appunvs;` (no version suffix —
back-incompatible changes will land as a new package, e.g.
`appunvs2;`, when the time comes).

| File | Purpose |
| --- | --- |
| `common.proto`   | Cross-module enums (`Platform`) |
| `sync.proto`     | Data sync envelope: `Message`, `Role`, `Op`, `WsHandshakeParams` |
| `auth.proto`     | Account + device identity: signup / login / register / me |
| `schema.proto`   | User-defined dynamic schema: tables + columns + `ColumnType` |
| `apikey.proto`   | API key creation, listing, revocation |
| `billing.proto`  | Stripe-backed plan catalog + status + checkout |
| `box.proto`      | Stage pipeline: `Box`, `BundleRef`, publish flow, version-update broadcast |
| `pair.proto`     | One-shot pairing short codes (provider → connector) |
| `ai.proto`       | AI agent surface: turn request + streamed event frames |
| `error.proto`    | Generic error envelope for HTTP failures |

Cross-file imports are explicit: `pair.proto` imports `box.proto` for
`BundleRef`, `auth.proto` imports `common.proto` for `Platform`, etc.

## Wire format

On the wire we use **canonical protojson** (not binary protobuf). This
keeps the existing JSON protocol in `docs/protocol.md` working and lets
clients debug via standard JSON tools, while still giving every
language a typed codec.

If we ever need smaller frames, switch the WebSocket subprotocol from
`appunvs.json.v1` to `appunvs.proto.v1` and flip the codec; the message
shape is unchanged.

## JSON ↔ proto field mapping

protojson uses `lowerCamelCase` by default, but our wire uses
`snake_case`.  Every generated codec must be configured to use
**original field names**:

- Go (`protojson`): `MarshalOptions{UseProtoNames: true}`
- Swift (`swift-protobuf`): default for JSON output is original; verify
- Kotlin (`protoc-gen-kotlin`): `JsonFormat.printer().preservingProtoFieldNames()`

Enum values are serialized as their short name lowercased
(`ROLE_PROVIDER → "provider"`, `OP_UPSERT → "upsert"`, etc.).

## Regeneration

Prerequisites: `protoc` 25+, plus language plugins.

```
shared/proto/
  *.proto            # edit here
  buf.yaml           # buf config (lint + breaking-change rules)
  gen.sh             # one script, fan-out to every active target
```

Active targets:

| Component | Plugin | Output |
| --- | --- | --- |
| relay | hand-mirrored (no codegen) | `relay/internal/pb/` |

Future:

| Component | Plugin | Output |
| --- | --- | --- |
| appunvs/ios     | `swift-protobuf` | `appunvs/ios/Runtime/Generated/` |
| appunvs/android | `protoc-gen-kotlin` | `appunvs/android/app/src/main/java/com/appunvs/proto/` |

Run `./gen.sh` from this directory after any proto change, then
commit the generated output alongside the proto edit.

## Rules

1. Never hand-write wire types in any component without also adding a
   drift test against `testdata/messages.json`.
2. Adding a field is backwards compatible; renumbering or removing is
   not.  When a true break is needed, bump the proto package
   (`package appunvs2;`) and migrate consumers explicitly.
3. Enums always add new values at the end with the next integer; never
   reuse a discarded number.
4. `payload` (in `Message`) stays `google.protobuf.Struct`. Business
   tables are not part of the wire schema and must not be declared
   here — they live in the user-defined schema (see `schema.proto`).

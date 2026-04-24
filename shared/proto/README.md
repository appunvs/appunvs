# shared/proto

**Single source of truth** for the wire types shared by `relay/`, `mobile/`,
`browser/`, and `desktop/`. All generated code is committed under each
component to keep builds hermetic; do not import proto files directly.

## Wire format

On the wire we use **canonical protojson** (not binary protobuf). This keeps
the existing JSON protocol in `docs/protocol.md` working and lets browser
clients debug via devtools, while still giving every language a typed codec.

If we ever need smaller frames, switch the WebSocket subprotocol from
`appunvs.json.v1` to `appunvs.proto.v1` and flip the codec; the message shape
is unchanged.

## JSON ↔ proto field mapping

protojson uses `lowerCamelCase` by default, but our wire uses `snake_case`.
Every generated codec must be configured to use **original field names**:

- Go (`protojson`): `MarshalOptions{UseProtoNames: true}`
- TypeScript (`@bufbuild/protobuf`): `toJson({useProtoFieldName: true})`
- Dart (`protobuf`): `toProto3Json(typeRegistry: ..., fieldNameMode: .original)` via wrapper
- Rust (`prost` + `pbjson`): enable `preserve_proto_field_names`

Enum values are serialized as their short name lowercased:
`ROLE_PROVIDER → "provider"`, `OP_UPSERT → "upsert"`, etc. Each generator
needs a small adapter for this — see each component's `wire/` package.

## Regeneration

Prerequisites: `protoc` 25+, `buf` (optional), plus language plugins.

```
shared/proto/
  appunvs.proto          # edit here
  buf.yaml               # buf config (optional)
  gen.sh                 # one script, fan-out to every target
```

Targets:

| Component | Plugin | Output |
| --- | --- | --- |
| relay     | `protoc-gen-go`                      | `relay/internal/pb/` |
| browser   | `@bufbuild/protoc-gen-es`            | `browser/src/lib/pb/` |
| desktop   | `prost-build` via `build.rs`         | `desktop/src-tauri/src/pb/` |
| mobile    | `protoc-gen-dart`                    | `mobile/lib/pb/` |

Run `./gen.sh` from this directory after any proto change, then commit the
generated output alongside the proto edit.

## Rules

1. Never hand-write wire types in any component — always import from `pb/`.
2. Adding a field is backwards compatible; renumbering or removing is not.
   Bump the proto package (`appunvs.v2`) before breaking changes.
3. Enums always add new values at the end with the next integer.
4. `payload` stays `google.protobuf.Struct` — business tables are not part of
   the wire schema and must not be declared here.

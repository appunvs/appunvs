#!/usr/bin/env bash
# Regenerate wire code for every appunvs component from appunvs.proto.
# Run from repo root or from shared/proto/.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"
PROTO="$HERE/appunvs.proto"

echo "==> Go (relay)"
mkdir -p "$ROOT/relay/internal/pb"
protoc \
  --proto_path="$HERE" \
  --go_out="$ROOT/relay/internal/pb" \
  --go_opt=paths=source_relative \
  "$PROTO"

echo "==> TypeScript (browser)"
mkdir -p "$ROOT/browser/src/lib/pb"
# requires: npm i -D @bufbuild/protoc-gen-es @bufbuild/protobuf
protoc \
  --proto_path="$HERE" \
  --es_out="$ROOT/browser/src/lib/pb" \
  --es_opt=target=ts \
  "$PROTO"

echo "==> Dart (mobile)"
mkdir -p "$ROOT/mobile/lib/pb"
# requires: dart pub global activate protoc_plugin
protoc \
  --proto_path="$HERE" \
  --dart_out="$ROOT/mobile/lib/pb" \
  "$PROTO"

echo "==> Rust (desktop) — handled by desktop/src-tauri/build.rs via prost-build"

echo "done."

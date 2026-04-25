#!/usr/bin/env bash
# Regenerate wire code for every consumer that ships generated bindings.
# Run from repo root or from shared/proto/.
#
# As of the v0 native pivot, the only generated target is the relay's Go
# `pb` package.  iOS (swift-protobuf) and Android (protoc-gen-kotlin) get
# wired in once the native runtime workspace lands, at which point this
# script grows additional fan-out blocks.
set -euo pipefail

HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT="$(cd "$HERE/../.." && pwd)"

# Glob every .proto in this directory; protoc handles cross-file imports
# as long as --proto_path is set to the directory containing them all.
shopt -s nullglob
PROTOS=( "$HERE"/*.proto )
shopt -u nullglob
if [ ${#PROTOS[@]} -eq 0 ]; then
  echo "no .proto files in $HERE" >&2
  exit 1
fi

echo "==> Go (relay)"
mkdir -p "$ROOT/relay/internal/pb"
protoc \
  --proto_path="$HERE" \
  --go_out="$ROOT/relay/internal/pb" \
  --go_opt=paths=source_relative \
  "${PROTOS[@]}"

# Future targets — uncomment when the host shells consume generated
# bindings instead of the hand-written Codable / kotlinx-serialization
# mirrors they ship today (appunvs/ios/Runtime/Net/Models.swift,
# appunvs/android/app/src/main/java/com/appunvs/runtime/net/Models.kt):
#
# echo "==> Swift (appunvs/ios)"
# mkdir -p "$ROOT/appunvs/ios/Runtime/Generated"
# protoc --proto_path="$HERE" --swift_out="$ROOT/appunvs/ios/Runtime/Generated" "${PROTOS[@]}"
#
# echo "==> Kotlin (appunvs/android)"
# mkdir -p "$ROOT/appunvs/android/app/src/main/java/com/appunvs/proto"
# protoc --proto_path="$HERE" --kotlin_out="$ROOT/appunvs/android/app/src/main/java/com/appunvs/proto" "${PROTOS[@]}"

echo "done."

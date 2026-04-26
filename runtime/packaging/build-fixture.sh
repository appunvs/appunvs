#!/usr/bin/env bash
# build-fixture.sh — produces a static JS bundle from
# runtime/sandbox/fixture-rn/index.tsx for use by the D3.c.4 UI tests.
#
# Outputs:
#   runtime/build/fixture/ios/RuntimeRoot.jsbundle
#   runtime/build/fixture/android/RuntimeRoot.jsbundle
#
# Both files register a "RuntimeRoot" component via
# AppRegistry.registerComponent — RuntimeView.mm / RuntimeView.kt then
# load them as the moduleName.
#
# Run from the repo root or from runtime/.  Idempotent.
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/fixture"
mkdir -p "$OUT/ios" "$OUT/android"

ENTRY="sandbox/fixture-rn/index.tsx"
if [ ! -f "$ENTRY" ]; then
  echo "[build-fixture] missing $ENTRY (did you cd to runtime/?)" >&2
  exit 1
fi

echo "==> environment"
echo "  node: $(node --version 2>&1 || echo MISSING)"
echo "  npm:  $(npm --version 2>&1 || echo MISSING)"

if [ ! -d node_modules ]; then
  echo "==> npm install (first run)"
  npm install --no-audit --no-fund --legacy-peer-deps 2>&1 | tail -10
fi

# `--minify false` keeps the bundle readable in CI failure logs;
# bytes saved are negligible for a 50-line fixture.  `--dev false`
# disables Metro's dev-server warnings but still produces a single
# self-contained bundle.
for platform in ios android; do
  echo "==> bundling fixture for $platform"
  npx --no-install react-native bundle \
    --entry-file "$ENTRY" \
    --platform "$platform" \
    --dev false \
    --minify false \
    --bundle-output "$OUT/$platform/RuntimeRoot.jsbundle"
done

echo "==> built:"
ls -la "$OUT/ios/RuntimeRoot.jsbundle" "$OUT/android/RuntimeRoot.jsbundle"

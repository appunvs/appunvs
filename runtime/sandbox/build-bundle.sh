#!/usr/bin/env bash
# build-bundle.sh — runs inside the sandbox docker image (see Dockerfile).
#
# Inputs (bind-mounted at /work by the relay):
#   /work/src/                ← AI-generated TypeScript / JSX source tree
#   /work/entry.tsx           ← optional override; defaults to /work/src/index.tsx
#
# Outputs (written to /work):
#   /work/index.bundle        ← Hermes-ready JS bundle (UTF-8)
#   /work/index.hbc           ← Hermes bytecode (if HERMES_AVAILABLE=1)
#   /work/build.log           ← metro stderr/stdout
#
# Exit code:
#   0  → bundle built successfully
#   1  → AI source referenced a forbidden import (see metro.config.js)
#   2  → metro / hermes internal error
#
# The relay's `internal/sandbox` package consumes these via the docker
# run wrapper.  Failure modes propagate back into the publish pipeline
# as a `BuildState.failed` BundleRef with build_log attached.
set -euo pipefail

WORK=/work
ENTRY="${WORK}/entry.tsx"
if [ ! -f "$ENTRY" ]; then
  ENTRY="${WORK}/src/index.tsx"
fi

if [ ! -f "$ENTRY" ]; then
  echo "[sandbox] no entry file found (looked for /work/entry.tsx and /work/src/index.tsx)" >&2
  exit 1
fi

cd /sandbox

# Pin platform via SANDBOX_PLATFORM; default ios (the bundle is platform
# agnostic for almost all RN code, but reanimated and a few others have
# small platform-specific shims).
PLATFORM="${SANDBOX_PLATFORM:-ios}"

echo "[sandbox] entry: $ENTRY"
echo "[sandbox] platform: $PLATFORM"

# Run metro via the React Native CLI (already in node_modules from the
# Dockerfile npm install).
npx --no-install react-native bundle \
  --entry-file "$ENTRY" \
  --platform "$PLATFORM" \
  --dev false \
  --minify true \
  --bundle-output "${WORK}/index.bundle" \
  --config "/sandbox/metro.config.js" \
  2>&1 | tee "${WORK}/build.log"

# Optional: hermes pre-compile.  The hermes binary ships in
# react-native's node_modules; HERMES_AVAILABLE gates whether we run it.
if [ "${HERMES_AVAILABLE:-0}" = "1" ]; then
  HERMESC=$(node -e "console.log(require.resolve('react-native/sdks/hermesc/osx-bin/hermesc'))" 2>/dev/null || true)
  if [ -x "$HERMESC" ]; then
    "$HERMESC" -emit-binary -out "${WORK}/index.hbc" "${WORK}/index.bundle"
    echo "[sandbox] emitted index.hbc"
  fi
fi

echo "[sandbox] done"

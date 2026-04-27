#!/usr/bin/env bash
# build-sandbox.sh — builds the appunvs/sandbox docker image used by
# relay's DockerBuilder (relay/internal/sandbox/docker.go) for
# AI-bundle production.
#
# The image:
#   - Pre-installs runtime/package.json's dependencies (metro,
#     react-native, all Tier 1 native modules).
#   - Includes runtime/src/HostBridge.ts so the @appunvs/host metro
#     specifier resolves to the in-tree TypeScript surface.
#   - Drops AI source via bind-mount at /work; build-bundle.sh runs
#     metro and writes /work/index.bundle.
#
# Outputs (to local docker daemon):
#   appunvs/sandbox:${TAG:-latest}
#
# Optional env:
#   TAG=latest                # image tag suffix
#   PLATFORM=linux/amd64      # buildx target
#   PUSH=0                    # push to registry (1 to enable)
#   REGISTRY=                 # full registry prefix (e.g. ghcr.io/foo)
#                             # → image becomes $REGISTRY/sandbox:$TAG
#
# Run from the repo root or from anywhere; the script cds to runtime/.
set -euo pipefail

cd "$(dirname "$0")/.."

TAG="${TAG:-latest}"
NAME="${REGISTRY:+${REGISTRY%/}/}sandbox"
if [ -z "${REGISTRY:-}" ]; then
  NAME="appunvs/sandbox"
fi
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH="${PUSH:-0}"

if [ ! -f sandbox/Dockerfile ]; then
  echo "[build-sandbox] missing runtime/sandbox/Dockerfile" >&2
  exit 1
fi
if [ ! -f package-lock.json ]; then
  echo "[build-sandbox] missing runtime/package-lock.json — run \`npm install\` first" >&2
  exit 1
fi

echo "==> docker build $NAME:$TAG (context=runtime/)"

# Build context is runtime/ so the Dockerfile can COPY:
#   - package.json + package-lock.json (deps)
#   - src/ (HostBridge.ts)
#   - sandbox/metro.config.js + sandbox/build-bundle.sh
DOCKER_ARGS=(
  buildx build
  --platform "$PLATFORM"
  -f sandbox/Dockerfile
  -t "$NAME:$TAG"
)

if [ "$PUSH" = "1" ]; then
  DOCKER_ARGS+=(--push)
else
  # Without --push we need --load to make the image visible to the
  # local docker daemon (otherwise buildx writes to its OCI cache only).
  DOCKER_ARGS+=(--load)
fi

# Cache hooks: GitHub Actions sets ACTIONS_RUNTIME_TOKEN; if present we
# wire buildx to the gha cache.  Local runs skip caching.
if [ -n "${ACTIONS_RUNTIME_TOKEN:-}" ]; then
  DOCKER_ARGS+=(
    --cache-from type=gha,scope=appunvs-sandbox
    --cache-to   type=gha,scope=appunvs-sandbox,mode=max
  )
fi

DOCKER_ARGS+=(.)

docker "${DOCKER_ARGS[@]}"

echo "==> built $NAME:$TAG"

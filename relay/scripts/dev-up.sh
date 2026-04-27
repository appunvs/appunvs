#!/usr/bin/env bash
# dev-up.sh — bring up everything you need to dogfood the relay locally.
#
# What it does:
#   1. Starts a redis container if one named `appunvs-redis` isn't running.
#   2. Builds the appunvs/sandbox docker image if not present
#      (DockerBuilder fails fast at startup without it).
#   3. Runs the relay binary directly via `go run` with sane env.
#
# What it does NOT do:
#   - Fetch / set your AI API key.  Set APPUNVS_AI_API_KEY before running
#     (or accept the stub backend default — chats will echo).
#   - Configure HTTPS / public domain.  This script targets LAN-only
#     dogfood; the host app on a phone in the same wifi connects via
#     `http://<your-mac-ip>:8080`.
#
# Required env (or .env in the relay directory — autoloaded):
#   APPUNVS_AI_BACKEND     stub | anthropic | deepseek | volcengine | …
#   APPUNVS_AI_API_KEY     required when backend != stub
#   APPUNVS_AI_MODEL       optional override (each backend has a sane default)
#
# Usage:
#   bash relay/scripts/dev-up.sh                  # default everything
#   APPUNVS_AI_BACKEND=stub bash relay/scripts/dev-up.sh   # echo engine
#
# Stop everything:
#   docker rm -f appunvs-redis
#   (relay was foreground — Ctrl-C already killed it)
set -euo pipefail

cd "$(dirname "$0")/.."

# Load .env if present so secrets stay out of shell history.
if [ -f .env ]; then
  echo "[dev-up] loading .env"
  set -a
  # shellcheck disable=SC1091
  source .env
  set +a
fi

# --- 1. redis ----------------------------------------------------------------

if docker ps --format '{{.Names}}' | grep -qx appunvs-redis; then
  echo "[dev-up] redis already running (container appunvs-redis)"
else
  echo "[dev-up] starting redis…"
  docker run -d \
    --name appunvs-redis \
    -p 6379:6379 \
    --restart unless-stopped \
    redis:7-alpine \
    redis-server --save "" --appendonly no >/dev/null
fi

# --- 2. sandbox image --------------------------------------------------------

SANDBOX_TAG="${APPUNVS_SANDBOX_IMAGE:-appunvs/sandbox:latest}"

if docker image inspect "$SANDBOX_TAG" >/dev/null 2>&1; then
  echo "[dev-up] sandbox image $SANDBOX_TAG present"
else
  echo "[dev-up] sandbox image $SANDBOX_TAG missing — building…"
  bash ../runtime/packaging/build-sandbox.sh
fi

# --- 3. relay ----------------------------------------------------------------

# Defaults if user didn't set them.  stub backend means chats echo back —
# fine for first-time validation that the wire works; flip to anthropic
# (or any OpenAI-compat provider) once you have a key.
export APPUNVS_REDIS_ADDR="${APPUNVS_REDIS_ADDR:-localhost:6379}"
export APPUNVS_LISTEN="${APPUNVS_LISTEN:-:8080}"
export APPUNVS_LOG_LEVEL="${APPUNVS_LOG_LEVEL:-info}"
export APPUNVS_SANDBOX_BACKEND="${APPUNVS_SANDBOX_BACKEND:-docker}"
export APPUNVS_SANDBOX_IMAGE="$SANDBOX_TAG"
export APPUNVS_AI_BACKEND="${APPUNVS_AI_BACKEND:-stub}"

# AI key sanity nudge — relay would die on engine construction anyway,
# but a clear hint here saves you reading the trace.
if [ "$APPUNVS_AI_BACKEND" != "stub" ] && [ -z "${APPUNVS_AI_API_KEY:-}" ]; then
  echo "[dev-up] WARNING: APPUNVS_AI_BACKEND=$APPUNVS_AI_BACKEND but APPUNVS_AI_API_KEY unset" >&2
  echo "[dev-up]   set it in .env or export, or use APPUNVS_AI_BACKEND=stub for echo mode" >&2
fi

# Show resolved env (without leaking the key) so you know what relay
# will see.
echo "[dev-up] effective config:"
printf '  APPUNVS_AI_BACKEND      = %s\n'    "$APPUNVS_AI_BACKEND"
printf '  APPUNVS_AI_API_KEY      = %s\n'    "${APPUNVS_AI_API_KEY:+(set)}${APPUNVS_AI_API_KEY:-(unset)}"
printf '  APPUNVS_SANDBOX_BACKEND = %s\n'    "$APPUNVS_SANDBOX_BACKEND"
printf '  APPUNVS_SANDBOX_IMAGE   = %s\n'    "$APPUNVS_SANDBOX_IMAGE"
printf '  APPUNVS_REDIS_ADDR      = %s\n'    "$APPUNVS_REDIS_ADDR"
printf '  APPUNVS_LISTEN          = %s\n'    "$APPUNVS_LISTEN"

LAN_IP=$(ipconfig getifaddr en0 2>/dev/null || hostname -I 2>/dev/null | awk '{print $1}' || echo "<your-LAN-ip>")
echo "[dev-up] host app should connect to: http://${LAN_IP}${APPUNVS_LISTEN}"

echo "[dev-up] starting relay (Ctrl-C to stop)…"
exec go run ./cmd/server

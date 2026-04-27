# Sandbox image — installs the runtime SDK's npm dependencies (the
# curated module list from runtime/package.json's `dependencies`) so
# that AI-generated bundles can `import` them at build time.  AI source
# is mounted in at run time; metro is invoked with the externals config
# from `metro.config.js` so anything outside the allowlist fails fast.
#
# This image is the canonical build environment for ai-bundle production
# — relay's `internal/sandbox/` package shells out to it once per
# `Box.publish` request via DockerBuilder.
#
# BUILD CONTEXT: this Dockerfile expects context = `runtime/`, NOT
# `runtime/sandbox/`.  Reason: metro.config.js resolves `@appunvs/host`
# to `../src/HostBridge.ts`, and the install needs runtime/'s
# package.json + lockfile.  Build via:
#
#     docker build -f runtime/sandbox/Dockerfile -t appunvs/sandbox:latest runtime/
#
# Or run `runtime/packaging/build-sandbox.sh` which does this.
#
# Pinned to node 22 LTS to match runtime/package.json's `engines.node`.
FROM node:22-alpine

# Working dir well outside /tmp so AI source mounts don't collide with
# metro's own scratch dirs.  Layout mirrors a regular RN project:
#
#   /app/                ← image root, project / project-relative paths
#     package.json       ← runtime/package.json (deps, lockfile)
#     package-lock.json
#     node_modules/      ← npm install lands here
#     src/HostBridge.ts  ← from runtime/src/, used by @appunvs/host shim
#     sandbox/           ← from runtime/sandbox/
#       metro.config.js  ← resolves @appunvs/host -> ../src/HostBridge.ts
#       build-bundle.sh  ← entrypoint
WORKDIR /app

# --- Step 1: install deps (cached layer) ---
# Copy ONLY package.json + lockfile first so an unchanged dep set hits
# the npm-install layer from a previous build.
COPY package.json package-lock.json ./

# RN brings native build hooks (e.g. android codegen postinstall) that
# fail under alpine without Java/Android-SDK.  Disable them — we only
# need the JS surface for metro bundling.
ENV REACT_NATIVE_SKIP_CODEGEN=1
RUN npm install --no-audit --no-fund --ignore-scripts

# --- Step 2: copy the bits metro reads at build time ---
# runtime/src/ holds HostBridge.ts (resolves @appunvs/host) plus other
# shared TS surface AI bundles can `import {} from '@appunvs/host'`.
COPY src ./src

# Sandbox config + entry script.
COPY sandbox/metro.config.js sandbox/build-bundle.sh ./sandbox/
RUN chmod +x ./sandbox/build-bundle.sh

# Working dir at runtime is /app/sandbox so __dirname inside
# metro.config.js evaluates to /app/sandbox — the existing path
# arithmetic (../src/HostBridge.ts, ../node_modules) lines up.
WORKDIR /app/sandbox

# AI source is bind-mounted at /work at run time:
#   docker run --rm -v "$ai_src:/work" appunvs/sandbox /app/sandbox/build-bundle.sh
VOLUME ["/work"]

ENTRYPOINT ["/app/sandbox/build-bundle.sh"]

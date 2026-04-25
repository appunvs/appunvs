# Sandbox image — installs the runtime SDK's npm dependencies (the
# curated module list from runtime/package.json's `dependencies`) so
# that AI-generated bundles can `import` them at build time.  AI source
# is mounted in at run time; metro is invoked with the externals config
# from `metro.config.js` so anything outside the allowlist fails fast.
#
# This image is the canonical build environment for ai-bundle production
# — relay's `internal/sandbox/` package shells out to it once per
# `Box.publish` request.
#
# Pinned to node 22 LTS to match runtime/package.json's `engines.node`.
FROM node:22-alpine

# Pin a working dir well outside /tmp so AI source mounts don't collide
# with metro's own scratch dirs.
WORKDIR /sandbox

# Copy the runtime's package.json + lockfile (lockfile lands when the
# RN init's npm install actually runs in CI; until then we ship just
# package.json and let npm cache the resolution per build).
COPY package.json ./

# Resolved deps live in /sandbox/node_modules.  We do `npm install`
# (not `npm ci`) until the lockfile is committed.
RUN npm install --no-audit --no-fund

# Copy the metro config + bundle-build script.
COPY metro.config.js build-bundle.sh ./
RUN chmod +x build-bundle.sh

# AI source is bind-mounted at /work at run time:
#   docker run --rm -v "$ai_src:/work" appunvs/sandbox /sandbox/build-bundle.sh
VOLUME ["/work"]

ENTRYPOINT ["/sandbox/build-bundle.sh"]

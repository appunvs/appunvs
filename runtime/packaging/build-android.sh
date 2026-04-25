#!/usr/bin/env bash
# build-android.sh — produces runtime.aar, the artifact the host Android
# app (appunvs/android/) links to mount AI bundles inside its Stage tab.
#
# D2.a (this PR): builds the empty-shell library that exposes one method
# — `RuntimeSDK.hello()`.  Pure kotlin, no RN, no Hermes, no JNI.
# Proves the gradle → AAR → host-link chain works.
#
# D2.c will:
#   * link Hermes engine (libhermes.so per ABI: arm64-v8a, armeabi-v7a, x86_64)
#   * link React Native's C++ runtime (libreact_render*.so)
#   * link the curated Tier 1 native modules (see ../MODULES.md)
#   * widen the SDK surface to RuntimeView (loadBundle/reset)
#   * ship consumer ProGuard keep-rules for the JNI surface
#
# Output: runtime/build/android/runtime.aar
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/android"
mkdir -p "$OUT"

# Use the standalone gradle project under sdk/android/ — see its
# settings.gradle.kts for why this isn't a sibling of the dev-harness
# `runtime/android/` project.
(cd sdk/android && gradle :runtimesdk:assembleRelease --no-daemon)

AAR="sdk/android/runtimesdk/build/outputs/aar/runtimesdk-release.aar"
if [ ! -f "$AAR" ]; then
  echo "[build-android] expected AAR at $AAR but it's missing" >&2
  exit 1
fi

cp "$AAR" "$OUT/runtime.aar"

echo "==> built $OUT/runtime.aar"

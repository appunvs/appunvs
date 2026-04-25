#!/usr/bin/env bash
# build-android.sh — produces runtime.aar, the artifact the host Android
# app (appunvs/android/) links to mount AI bundles inside its Stage tab.
#
# D3.a (this PR): SDK module lives inside the RN init project at
# runtime/android/runtimesdk/ (was at runtime/sdk/android/runtimesdk/).
# RN init's settings.gradle requires npm install to materialize
# @react-native/gradle-plugin under node_modules/ before gradle can
# even parse settings.gradle.
#
# D3.b will start using react-android / hermes-android from this
# module, which is why the migration to the RN init project is the
# right home now.
#
# Output: runtime/build/android/runtime.aar
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/android"
mkdir -p "$OUT"

# 1. npm install — needed because runtime/android/settings.gradle pulls
#    @react-native/gradle-plugin from ../node_modules/.  Without this,
#    `gradle :runtimesdk:assembleRelease` errors during settings parse.
#    --no-audit + --no-fund speeds up CI; --legacy-peer-deps tolerates
#    RN's deeply-nested peer deps that strict npm 7+ would reject.
echo "==> npm install (for @react-native/gradle-plugin)"
npm install --no-audit --no-fund --legacy-peer-deps

# 2. assembleRelease the runtimesdk module.  app module isn't built —
#    it'd require an extra Hermes/JSC ABI download and we don't ship
#    the dev-harness app as a build artifact.
echo "==> gradle :runtimesdk:assembleRelease"
(cd android && gradle :runtimesdk:assembleRelease --no-daemon)

AAR="android/runtimesdk/build/outputs/aar/runtimesdk-release.aar"
if [ ! -f "$AAR" ]; then
  echo "[build-android] expected AAR at $AAR but it's missing" >&2
  exit 1
fi

cp "$AAR" "$OUT/runtime.aar"

echo "==> built $OUT/runtime.aar"

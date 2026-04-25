#!/usr/bin/env bash
# build-android.sh — produces runtime.aar, the artifact that the host
# Android app (appunvs/android/) links to gain JS/Hermes capability
# inside its Stage tab.
#
# What this should do (PR D2 lands the actual implementation):
#
#   1. Run gradle on runtime/android/ with a release variant that bundles
#        - Hermes engine (libhermes.so, per-ABI: arm64-v8a, armeabi-v7a,
#          x86_64)
#        - React Native's C++ runtime + JSI (libreact_render.so etc.)
#        - The curated Tier 1 native modules (see ../MODULES.md)
#        - The SubRuntime JNI bridge that exposes the Java/Kotlin
#          entry points to host code (`SubRuntime.spawn(bundleURL) ->
#          SubRuntimeView`)
#
#   2. ProGuard / R8 keep rules for the JNI surface so host shrinking
#      doesn't strip the bridge symbols
#
#   3. Copy the resulting runtime.aar to ./build/android/
#
# For PR D2-prep this script is a stub.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "build-android.sh: stub — PR D2 implements the real AAR build."
echo "Inputs: runtime/android/, runtime/src/, runtime/MODULES.md (tier 1)"
echo "Output: runtime/build/android/runtime.aar"
exit 0

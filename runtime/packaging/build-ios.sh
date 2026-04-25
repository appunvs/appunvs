#!/usr/bin/env bash
# build-ios.sh — produces RuntimeSDK.xcframework, the artifact that the
# host iOS app (appunvs/ios/) links to gain JS/Hermes capability inside
# its Stage tab.
#
# What this should do (PR D2 lands the actual implementation):
#
#   1. Build an XCFramework that bundles
#        - Hermes engine (libhermes.a, statically linked)
#        - React Native's C++ runtime + JSI
#        - The curated Tier 1 native modules (see ../MODULES.md), each
#          built as a static archive and registered with the SubRuntime
#          module registry
#        - The SubRuntime native module that exposes the C entry points
#          to host code (`SubRuntime.spawn(bundleURL) -> SubRuntimeView`)
#
#   2. Build for both device (arm64) and simulator (arm64 + x86_64)
#      slices, then `xcodebuild -create-xcframework` them together
#
#   3. Copy the resulting RuntimeSDK.xcframework to ./build/ios/
#
# For PR D2-prep this script is a stub — it documents the intent so the
# next PR has somewhere to land the real build steps.
set -euo pipefail

cd "$(dirname "$0")/.."

echo "build-ios.sh: stub — PR D2 implements the real XCFramework build."
echo "Inputs: runtime/ios/, runtime/src/, runtime/MODULES.md (tier 1)"
echo "Output: runtime/build/ios/RuntimeSDK.xcframework"
exit 0

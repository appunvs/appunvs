#!/usr/bin/env bash
# build-ios.sh — produces RuntimeSDK.xcframework, the artifact the host
# iOS app (appunvs/ios/) links to mount AI bundles inside its Stage tab.
#
# D2.a (this PR): builds the empty-shell framework that exposes one C
# function — `runtime_sdk_hello()`. No RN, no Hermes, no Pods.  Just
# proves the xcodebuild → xcframework → host-link chain works.
#
# D2.c will:
#   * link Hermes engine (libhermes.a, statically, both slices)
#   * link React Native's C++ runtime + JSI
#   * link the curated Tier 1 native modules (see ../MODULES.md)
#   * widen the SDK surface to RuntimeView (loadBundle:/reset)
#
# Output: runtime/build/ios/RuntimeSDK.xcframework
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/ios"
mkdir -p "$OUT"

# 1. Generate the framework xcodeproj from sdk/ios/SDK.yml.  We use
#    XcodeGen rather than committing the .xcodeproj because pbxproj
#    is hostile to manual edits and merge conflicts.
echo "==> xcodegen --version"
xcodegen --version || true

echo "==> xcodegen generate (sdk/ios/SDK.yml)"
(cd sdk/ios && xcodegen generate --spec SDK.yml --project .)

echo "==> generated project tree"
ls -la sdk/ios/
ls -la sdk/ios/RuntimeSDK.xcodeproj/ 2>&1 || echo "(xcodeproj missing)"

echo "==> available schemes"
xcodebuild -list -project sdk/ios/RuntimeSDK.xcodeproj || true

DEVICE_ARCHIVE="$OUT/RuntimeSDK-iphoneos.xcarchive"
SIM_ARCHIVE="$OUT/RuntimeSDK-iphonesimulator.xcarchive"

# 2. Archive for device (arm64).
xcodebuild archive \
  -project sdk/ios/RuntimeSDK.xcodeproj \
  -scheme RuntimeSDK \
  -configuration Release \
  -destination "generic/platform=iOS" \
  -archivePath "$DEVICE_ARCHIVE" \
  SKIP_INSTALL=NO \
  BUILD_LIBRARY_FOR_DISTRIBUTION=YES \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY=""

# 3. Archive for simulator (arm64 + x86_64 fat slice).
xcodebuild archive \
  -project sdk/ios/RuntimeSDK.xcodeproj \
  -scheme RuntimeSDK \
  -configuration Release \
  -destination "generic/platform=iOS Simulator" \
  -archivePath "$SIM_ARCHIVE" \
  SKIP_INSTALL=NO \
  BUILD_LIBRARY_FOR_DISTRIBUTION=YES \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY=""

# 4. Combine the two archives into one XCFramework.
rm -rf "$OUT/RuntimeSDK.xcframework"
xcodebuild -create-xcframework \
  -framework "$DEVICE_ARCHIVE/Products/Library/Frameworks/RuntimeSDK.framework" \
  -framework "$SIM_ARCHIVE/Products/Library/Frameworks/RuntimeSDK.framework" \
  -output "$OUT/RuntimeSDK.xcframework"

# 5. Tidy: drop the intermediate .xcarchive bundles.  Keep them around
#    if you're debugging by setting KEEP_ARCHIVES=1.
if [ "${KEEP_ARCHIVES:-0}" != "1" ]; then
  rm -rf "$DEVICE_ARCHIVE" "$SIM_ARCHIVE"
fi

echo "==> built $OUT/RuntimeSDK.xcframework"

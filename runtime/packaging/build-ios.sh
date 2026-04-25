#!/usr/bin/env bash
# build-ios.sh — produces RuntimeSDK.xcframework, the artifact the host
# iOS app (appunvs/ios/) links to mount AI bundles inside its Stage tab.
#
# D3.a (this PR): SDK lives inside the RN init project at runtime/ios/
# (was at runtime/sdk/ios/).  Empty-shell SDK still — D3.b adds the
# RN+Hermes deps that need pod install.  Today's build is independent
# of the RN init's xcworkspace; xcodebuild builds against the
# standalone SDK.xcodeproj only.
#
# Output: runtime/build/ios/RuntimeSDK.xcframework
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/ios"
mkdir -p "$OUT"

echo "==> xcodegen --version"
xcodegen --version || true

echo "==> xcodegen generate (ios/SDK.yml)"
(cd ios && xcodegen generate --spec SDK.yml --project .)

echo "==> available schemes"
xcodebuild -list -project ios/RuntimeSDK.xcodeproj || true

DEVICE_ARCHIVE="$OUT/RuntimeSDK-iphoneos.xcarchive"
SIM_ARCHIVE="$OUT/RuntimeSDK-iphonesimulator.xcarchive"

xcodebuild archive \
  -project ios/RuntimeSDK.xcodeproj \
  -scheme RuntimeSDK \
  -configuration Release \
  -destination "generic/platform=iOS" \
  -archivePath "$DEVICE_ARCHIVE" \
  SKIP_INSTALL=NO \
  BUILD_LIBRARY_FOR_DISTRIBUTION=YES \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY=""

xcodebuild archive \
  -project ios/RuntimeSDK.xcodeproj \
  -scheme RuntimeSDK \
  -configuration Release \
  -destination "generic/platform=iOS Simulator" \
  -archivePath "$SIM_ARCHIVE" \
  SKIP_INSTALL=NO \
  BUILD_LIBRARY_FOR_DISTRIBUTION=YES \
  CODE_SIGNING_ALLOWED=NO \
  CODE_SIGNING_REQUIRED=NO \
  CODE_SIGN_IDENTITY=""

rm -rf "$OUT/RuntimeSDK.xcframework"
xcodebuild -create-xcframework \
  -framework "$DEVICE_ARCHIVE/Products/Library/Frameworks/RuntimeSDK.framework" \
  -framework "$SIM_ARCHIVE/Products/Library/Frameworks/RuntimeSDK.framework" \
  -output "$OUT/RuntimeSDK.xcframework"

if [ "${KEEP_ARCHIVES:-0}" != "1" ]; then
  rm -rf "$DEVICE_ARCHIVE" "$SIM_ARCHIVE"
fi

echo "==> built $OUT/RuntimeSDK.xcframework"

#!/usr/bin/env bash
# build-ios.sh — produces RuntimeSDK.xcframework, the artifact the host
# iOS app (appunvs/ios/) links to mount AI bundles inside its Stage tab.
#
# D3.b (this PR): SDK now links React Native + Hermes via the same
# Podfile that backs the dev-harness app.  Build flow:
#
#   1. npm install          (resolves react-native, @react-native/...)
#   2. cd ios && pod install (materializes Hermes, React-Core, ReactCommon)
#   3. xcodegen generate     (produces SDK.xcodeproj using Pods xcconfig)
#   4. xcodebuild archive    (device + simulator)
#   5. xcodebuild -create-xcframework
#
# Output: runtime/build/ios/RuntimeSDK.xcframework
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/ios"
mkdir -p "$OUT"

echo "==> npm install (RN init's package.json)"
npm install --no-audit --no-fund --legacy-peer-deps

echo "==> pod install (cocoapods + RN pods)"
(cd ios && pod install --repo-update)

echo "==> xcodegen --version"
xcodegen --version || true

echo "==> xcodegen generate (ios/SDK.yml)"
(cd ios && xcodegen generate --spec SDK.yml --project .)

echo "==> available schemes"
xcodebuild -list -project ios/RuntimeSDK.xcodeproj || true

DEVICE_ARCHIVE="$OUT/RuntimeSDK-iphoneos.xcarchive"
SIM_ARCHIVE="$OUT/RuntimeSDK-iphonesimulator.xcarchive"

# Use -workspace so the SDK target finds Pods as well — the workspace
# pulls in Pods.xcodeproj which our .xcconfig references.  But since
# pod install only writes RuntimeSdk.xcworkspace (not SDK), build by
# project but with the Pods path setup via xcconfig (handled by
# configFiles: in SDK.yml).
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

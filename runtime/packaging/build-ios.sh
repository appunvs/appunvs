#!/usr/bin/env bash
# build-ios.sh — produces RuntimeSDK.xcframework, the artifact the host
# iOS app (appunvs/ios/) links to mount AI bundles inside its Stage tab.
#
# D3.b (this PR): SDK now links React Native + Hermes via the same
# Podfile that backs the dev-harness app.  Build flow:
#
#   1. npm install            (resolves react-native, @react-native/...)
#   2. xcodegen generate      (produces RuntimeSDK.xcodeproj — must
#                              exist before pod install can integrate)
#   3. cd ios && pod install  (materializes Hermes, React-Core, etc.,
#                              writes Pods-RuntimeSDK xcconfig that
#                              RuntimeSDK.xcodeproj's configFiles point at)
#   4. xcodebuild archive     (device + simulator slices)
#   5. xcodebuild -create-xcframework
#
# Output: runtime/build/ios/RuntimeSDK.xcframework
set -euo pipefail

cd "$(dirname "$0")/.."

OUT="build/ios"
mkdir -p "$OUT"

echo "==> environment"
echo "  node:   $(node --version 2>&1 || echo MISSING)"
echo "  npm:    $(npm --version 2>&1 || echo MISSING)"
echo "  pod:    $(pod --version 2>&1 || echo MISSING)"
echo "  xcodegen: $(xcodegen --version 2>&1 || echo MISSING)"

echo "==> npm install (RN init's package.json)"
npm install --no-audit --no-fund --legacy-peer-deps 2>&1 | tail -20

echo "==> xcodegen generate (ios/SDK.yml)"
(cd ios && xcodegen generate --spec SDK.yml --project .)

echo "==> pod install (cocoapods + RN pods, including new RuntimeSDK target)"
(cd ios && pod install --repo-update)

echo "==> available schemes"
xcodebuild -list -project ios/AppunvsRuntimeSDK.xcodeproj || true

DEVICE_ARCHIVE="$OUT/RuntimeSDK-iphoneos.xcarchive"
SIM_ARCHIVE="$OUT/RuntimeSDK-iphonesimulator.xcarchive"

xcodebuild archive \
  -project ios/AppunvsRuntimeSDK.xcodeproj \
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
  -project ios/AppunvsRuntimeSDK.xcodeproj \
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

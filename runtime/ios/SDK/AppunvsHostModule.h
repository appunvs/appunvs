// AppunvsHostModule — the bridge between AI bundles' `@appunvs/host`
// imports and the host app's capabilities.
//
// D3.e.1 (this PR): scaffolding only.  Exposes `sdkVersion` as a
// constant and an `echo` smoke-test method.  Real surfaces (identity,
// storage, network, publish) land in D3.e.{2,3,4,5}.
//
// Registered globally via RCT_EXPORT_MODULE — RN 0.85's bridgeless
// interop layer picks legacy modules up automatically as long as the
// class symbol is in the linked image.  Each RCTReactNativeFactory we
// stand up in RuntimeView.mm therefore gets one instance of this
// module by default.
//
// Per-instance identity (boxID etc.) is NOT here yet — D3.e.2 will
// thread it through loadBundleAtURL: into a per-host module instance.
#ifndef AppunvsHostModule_h
#define AppunvsHostModule_h

#import <React/RCTBridgeModule.h>

@class RuntimeBoxIdentity;

NS_ASSUME_NONNULL_BEGIN

@interface AppunvsHostModule : NSObject <RCTBridgeModule>

/// Stages identity into a static slot that the next module instance
/// will pick up at constantsToExport time.  RuntimeView calls this
/// BEFORE creating the RCTReactNativeFactory so the bridge's auto-
/// instantiated AppunvsHostModule sees the right values.  Per-process
/// (not per-bridge) — concurrent loadBundle calls would race; for D3.e.2
/// only one RuntimeView mounts at a time, so this is safe.
+ (void)setPendingIdentity:(nullable RuntimeBoxIdentity *)identity;

@end

NS_ASSUME_NONNULL_END

#endif /* AppunvsHostModule_h */

// AppunvsHostModule — the bridge between AI bundles' `@appunvs/host`
// imports and the host app's capabilities.
//
// D3.e.1: scaffolding (sdkVersion + echo smoke).
// D3.e.2: identity threaded through via a per-process pending slot.
// D3.e.3: storage backed by react-native-mmkv on the JS side.
// D3.e.{4,5} (this PR): network.request() + publish() — SDK side wires
// the bridge methods through to host-registered handler blocks.  SDK
// itself doesn't make HTTP calls or talk to the relay; the host's
// HTTPClient / publish flow does that, and registers a closure here at
// app launch via +registerRequestHandler: / +registerPublishHandler:.
//
// What this PR does NOT do:
//   - SubNetwork.subscribe (SSE) — separate architecture chunk.
//   - host shell wiring (registering the handlers in appunvs/ios) —
//     each host's responsibility, not SDK's.
#ifndef AppunvsHostModule_h
#define AppunvsHostModule_h

#import <Foundation/Foundation.h>

// Deliberately NOT importing <React/RCTBridgeModule.h> here.  The host
// shell consumes RuntimeSDK.xcframework as a brownfield binary and does
// NOT have React Native's headers in its include paths — pulling them
// into this public umbrella-imported header would break the host
// build.  The RCTBridgeModule conformance lives in a class extension
// inside AppunvsHostModule.mm; the host only needs the static class
// methods declared below.

@class RuntimeBoxIdentity;

NS_ASSUME_NONNULL_BEGIN

/// Callback the host invokes once a request has resolved.  `response`
/// is a JSON-able dict { status: int, headers: dict, body: string };
/// `error` is non-nil iff the host couldn't even attempt the call
/// (network down, handler unconfigured, etc.).  HTTP errors come back
/// as a non-nil response with non-2xx status.
typedef void (^AppunvsRequestCompletion)(NSDictionary *_Nullable response,
                                          NSError *_Nullable error);

/// Host-supplied handler called from JS via host().network.request().
/// Path is already prefixed with /box/{id}/ by the JS layer; host
/// fills in baseURL + auth.  Body is the request body string (JSON or
/// raw text), or nil for GET.
typedef void (^AppunvsRequestHandler)(NSString *method,
                                       NSString *path,
                                       NSString *_Nullable body,
                                       AppunvsRequestCompletion completion);

/// Same idea for publish: host-supplied handler completes with
/// { version: string, ok: bool }.
typedef void (^AppunvsPublishCompletion)(NSDictionary *_Nullable response,
                                          NSError *_Nullable error);

typedef void (^AppunvsPublishHandler)(NSString *_Nullable message,
                                       AppunvsPublishCompletion completion);

@interface AppunvsHostModule : NSObject

/// Stages identity into a static slot that the next module instance
/// will pick up at constantsToExport time.  RuntimeView calls this
/// BEFORE creating the RCTReactNativeFactory so the bridge's auto-
/// instantiated AppunvsHostModule sees the right values.  Per-process
/// (not per-bridge) — concurrent loadBundle calls would race; for D3.e.2
/// only one RuntimeView mounts at a time, so this is safe.
+ (void)setPendingIdentity:(nullable RuntimeBoxIdentity *)identity;

/// Hosts call this once at app launch to plug in their HTTP client.
/// Without a handler, host().network.request() rejects with
/// "no request handler".  Per-process: one handler covers all
/// RuntimeViews in the host.  Pass nil to clear (e.g. tests).
+ (void)registerRequestHandler:(nullable AppunvsRequestHandler)handler;

/// Hosts call this once at app launch to plug in the publish flow.
/// Without a handler, host().publish.publish() rejects.
+ (void)registerPublishHandler:(nullable AppunvsPublishHandler)handler;

@end

NS_ASSUME_NONNULL_END

#endif /* AppunvsHostModule_h */

// AppunvsHostModule — see AppunvsHostModule.h.
//
// D3.e.1 implementation: just the wiring + smoke-test `echo` method.
// Adding methods here is part of the SDK ABI — every method addition
// or signature change requires bumping `runtime/version.json`'s
// runtime.sdk_version, since AI bundles compiled against an older SDK
// must continue to run on a newer host (and vice versa within the
// supported window).
#import "AppunvsHostModule.h"
#import "RuntimeBoxIdentity.h"
#import <React/RCTBridgeModule.h>

// Class extension that adds RCTBridgeModule conformance to the public
// class declared in AppunvsHostModule.h.  The conformance lives here
// (not in the public header) because the host shell consumes
// RuntimeSDK.xcframework without React Native's headers — exposing the
// protocol publicly would force every host into a React-aware build
// graph just to call the static registration methods.
@interface AppunvsHostModule () <RCTBridgeModule>
@end

// Pending identity slot — RuntimeView calls +setPendingIdentity: before
// it creates the RCTReactNativeFactory; the next AppunvsHostModule
// instance reads from this slot at constantsToExport time.  Cleared on
// read so a subsequent module without an explicit setPendingIdentity:
// call sees the empty default.
static RuntimeBoxIdentity *_pendingIdentity = nil;

// Host-supplied handlers (per-process).  Hosts call
// +registerRequestHandler: / +registerPublishHandler: once at app
// launch.  D3.e.4/5: still nil until host wires them in.
static AppunvsRequestHandler _requestHandler = nil;
static AppunvsPublishHandler _publishHandler = nil;

@implementation AppunvsHostModule

// `RCT_EXPORT_MODULE()` registers this class with the React bridge as
// the native module named "AppunvsHost" — the JS side can reach it via
// `NativeModules.AppunvsHost`.  The macro emits a +(NSString*)moduleName
// and a +load that calls RCTRegisterModule().
RCT_EXPORT_MODULE(AppunvsHost)

+ (void)setPendingIdentity:(RuntimeBoxIdentity *)identity {
    _pendingIdentity = identity;
}

+ (void)registerRequestHandler:(AppunvsRequestHandler)handler {
    _requestHandler = handler;
}

+ (void)registerPublishHandler:(AppunvsPublishHandler)handler {
    _publishHandler = handler;
}

// Constants are exposed once, when the module is initialised, and
// available from JS as `NativeModules.AppunvsHost.sdkVersion` /
// `.identity`.  sdkVersion is the SDK ABI version (not host app, not
// bundle).  identity mirrors `BoxIdentity` in HostBridge.ts.
- (NSDictionary *)constantsToExport {
    RuntimeBoxIdentity *identity = _pendingIdentity;
    NSDictionary *identityDict = @{
        @"boxID":   identity.boxID   ?: @"",
        @"version": identity.version ?: @"",
        @"title":   identity.title   ?: @"",
    };
    return @{
        @"sdkVersion": @"0.1.0",
        @"identity":   identityDict,
    };
}

// `requiresMainQueueSetup` returning NO lets the bridge initialise this
// module on a background queue.  We don't touch UIKit during init, so
// background is fine.  Saves a small slice of host-app launch time.
+ (BOOL)requiresMainQueueSetup {
    return NO;
}

// Smoke-test method: `host()._echo(s)` round-trips a string through the
// bridge.  Kept around for D3.e.{4,5} CI smoke; an AI bundle wouldn't
// normally call it.
RCT_EXPORT_METHOD(echo:(NSString *)message
                  resolve:(RCTPromiseResolveBlock)resolve
                  reject:(RCTPromiseRejectBlock)reject) {
    if (message == nil) {
        reject(@"E_NIL_MESSAGE", @"echo: nil message", nil);
        return;
    }
    resolve(message);
}

// D3.e.4: network.request().  JS already prefixed `path` with /box/{id}/.
// Delegates to the host-registered handler — SDK doesn't make HTTP
// calls itself.  Rejects if the host hasn't registered a handler.
RCT_EXPORT_METHOD(request:(NSString *)method
                      path:(NSString *)path
                      body:(NSString *_Nullable)body
                  resolve:(RCTPromiseResolveBlock)resolve
                   reject:(RCTPromiseRejectBlock)reject) {
    AppunvsRequestHandler handler = _requestHandler;
    if (handler == nil) {
        reject(@"E_NO_REQUEST_HANDLER",
               @"host hasn't registered a network request handler "
               @"(see AppunvsHostModule.h +registerRequestHandler:)",
               nil);
        return;
    }
    handler(method, path, body, ^(NSDictionary *response, NSError *error) {
        if (error) {
            reject(@"E_REQUEST_FAILED",
                   error.localizedDescription ?: @"request failed",
                   error);
            return;
        }
        resolve(response ?: @{});
    });
}

// D3.e.5: publish().  Same pattern — host's relay client knows what
// publish means; SDK is a relay.
RCT_EXPORT_METHOD(publish:(NSString *_Nullable)message
                  resolve:(RCTPromiseResolveBlock)resolve
                   reject:(RCTPromiseRejectBlock)reject) {
    AppunvsPublishHandler handler = _publishHandler;
    if (handler == nil) {
        reject(@"E_NO_PUBLISH_HANDLER",
               @"host hasn't registered a publish handler "
               @"(see AppunvsHostModule.h +registerPublishHandler:)",
               nil);
        return;
    }
    handler(message, ^(NSDictionary *response, NSError *error) {
        if (error) {
            reject(@"E_PUBLISH_FAILED",
                   error.localizedDescription ?: @"publish failed",
                   error);
            return;
        }
        resolve(response ?: @{ @"version": @"", @"ok": @NO });
    });
}

@end

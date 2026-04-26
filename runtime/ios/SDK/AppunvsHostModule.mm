// AppunvsHostModule — see AppunvsHostModule.h.
//
// D3.e.1 implementation: just the wiring + smoke-test `echo` method.
// Adding methods here is part of the SDK ABI — every method addition
// or signature change requires bumping `runtime/version.json`'s
// runtime.sdk_version, since AI bundles compiled against an older SDK
// must continue to run on a newer host (and vice versa within the
// supported window).
#import "AppunvsHostModule.h"

@implementation AppunvsHostModule

// `RCT_EXPORT_MODULE()` registers this class with the React bridge as
// the native module named "AppunvsHost" — the JS side can reach it via
// `NativeModules.AppunvsHost`.  The macro emits a +(NSString*)moduleName
// and a +load that calls RCTRegisterModule().
RCT_EXPORT_MODULE(AppunvsHost)

// Constants are exposed once, when the module is initialised, and
// available from JS as `NativeModules.AppunvsHost.sdkVersion`.  This is
// the version of the SDK ABI itself, not of the host app or the bundle.
- (NSDictionary *)constantsToExport {
    return @{
        @"sdkVersion": @"0.1.0",
    };
}

// `requiresMainQueueSetup` returning NO lets the bridge initialise this
// module on a background queue.  We don't touch UIKit during init, so
// background is fine.  Saves a small slice of host-app launch time.
+ (BOOL)requiresMainQueueSetup {
    return NO;
}

// Smoke-test method: `host()._echo(s)` round-trips a string through the
// bridge.  D3.e.4 / D3.e.5 will replace this with real
// network.request / publish.publish methods; for now it lets us verify
// the JS→native→JS path works end to end.
RCT_EXPORT_METHOD(echo:(NSString *)message
                  resolve:(RCTPromiseResolveBlock)resolve
                  reject:(RCTPromiseRejectBlock)reject) {
    if (message == nil) {
        reject(@"E_NIL_MESSAGE", @"echo: nil message", nil);
        return;
    }
    resolve(message);
}

@end

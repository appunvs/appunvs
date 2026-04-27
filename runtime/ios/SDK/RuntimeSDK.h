// RuntimeSDK — public umbrella header.
//
// D2.a empty shell: exposes one C function (`runtime_sdk_hello`) so the
// host can link the framework, call into it, and prove the packaging
// pipeline (xcodebuild → xcframework → host link) works end-to-end.
//
// PR D2.c replaces this with the real surface:
//
//   @interface RuntimeView : UIView
//     - (instancetype)initWithFrame:(CGRect)frame;
//     - (void)loadBundleAtURL:(NSURL *)url
//                  completion:(void(^)(NSError * _Nullable))cb;
//     - (void)reset;
//   @end
//
// At that point the C surface widens to ObjC + a SwiftUI bridge layer.
#ifndef RuntimeSDK_h
#define RuntimeSDK_h

#import <Foundation/Foundation.h>

// NB: Xcode auto-generates RuntimeSDK_vers.c (from CURRENT_PROJECT_VERSION
// + MARKETING_VERSION) which defines RuntimeSDKVersionNumber and
// RuntimeSDKVersionString.  Don't redeclare them here — doing so makes
// the linker collide RuntimeSDK_vers.o against this header's user.

// Public surface: expose every header the host needs to import here so
// `@import RuntimeSDK;` (Swift `import RuntimeSDK`) gets all of them
// in one shot.  Without these #imports, Swift's `import RuntimeSDK`
// only sees what the umbrella exposes — the headers being copied to
// the framework's Headers/ dir isn't enough on its own.
#import <RuntimeSDK/RuntimeView.h>
#import <RuntimeSDK/AppunvsHostModule.h>
#import <RuntimeSDK/RuntimeBoxIdentity.h>

#ifdef __cplusplus
extern "C" {
#endif

/// Returns a NUL-terminated C string identifying this SDK build.
/// Lifetime: the returned pointer references static storage, callers
/// must NOT free it.
const char *runtime_sdk_hello(void);

#ifdef __cplusplus
}
#endif

#endif /* RuntimeSDK_h */

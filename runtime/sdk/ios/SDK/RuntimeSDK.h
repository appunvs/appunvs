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

//! Project version number for RuntimeSDK.
FOUNDATION_EXPORT double RuntimeSDKVersionNumber;

//! Project version string for RuntimeSDK.
FOUNDATION_EXPORT const unsigned char RuntimeSDKVersionString[];

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

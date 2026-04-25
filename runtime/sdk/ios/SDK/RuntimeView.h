// RuntimeView — public surface for mounting an AI-generated bundle
// inside the host's Stage tab.
//
// D2.c (this PR): placeholder UIView that just displays the loaded
// bundle URL as a label — proves the API shape end-to-end, callable
// from host SwiftUI / Compose, observable via loadBundle: + reset.
//
// D3 replaces the placeholder impl with the real React Native +
// Hermes mount.  The public surface here stays stable so host code
// doesn't move between D2.c and D3.
//
// Forward-declarations for D3:
//
//   - The placeholder will be swapped for an RCTHost-backed
//     React Fabric root view.
//   - loadBundleAtURL: will fetch the JS bundle, evaluate it under a
//     dedicated Hermes runtime, and mount the React tree inside this
//     view's bounds.
//   - reset will tear down the Hermes runtime and prepare for a
//     fresh bundle load (no cross-bundle JS state).
#ifndef RuntimeView_h
#define RuntimeView_h

#import <UIKit/UIKit.h>

NS_ASSUME_NONNULL_BEGIN

/// Block called when a loadBundleAtURL: completes (or fails).
/// `error` is nil on success.  Always invoked on the main thread.
typedef void (^RuntimeViewLoadCompletion)(NSError *_Nullable error);

@interface RuntimeView : UIView

/// The bundle URL currently displayed (or being loaded).  nil before
/// the first loadBundleAtURL: call.  KVO-observable.
@property (nonatomic, readonly, copy, nullable) NSURL *currentBundleURL;

/// Asks the runtime to fetch the bundle at `url` and mount its React
/// tree into this view's bounds.  Calling this while another bundle
/// is loaded resets first.  D2.c placeholder: just stores the URL
/// and displays it as text; D3 replaces with real Hermes mount.
- (void)loadBundleAtURL:(NSURL *)url
             completion:(nullable RuntimeViewLoadCompletion)completion;

/// Tears down the current bundle's runtime state.  Safe to call when
/// no bundle is loaded (no-op).
- (void)reset;

@end

NS_ASSUME_NONNULL_END

#endif /* RuntimeView_h */

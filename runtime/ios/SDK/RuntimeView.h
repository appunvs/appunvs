// RuntimeView — public surface for mounting an AI-generated bundle
// inside the host's Stage tab.
//
// D3.c (this PR): the placeholder UILabel was swapped for a real RN
// mount — `loadBundleAtURL:` now stands up an `RCTReactNativeFactory`
// (RN 0.85's brownfield entry point), fetches + evaluates the JS bundle
// under a dedicated Hermes runtime, and adds the resulting root view as
// a subview filling self.bounds.  `reset` releases the factory, which
// tears down the Hermes runtime so cross-bundle JS state can't leak.
//
// JS contract: the bundle MUST register a component named "RuntimeRoot"
// via `AppRegistry.registerComponent`.  Everything else is up to the
// bundle author.  Tier-1 native modules (gesture-handler etc.) come
// online in D3.d; the HostBridge plumbing in D3.e.
#ifndef RuntimeView_h
#define RuntimeView_h

#import <UIKit/UIKit.h>
#import "RuntimeBoxIdentity.h"

NS_ASSUME_NONNULL_BEGIN

/// Block called when a loadBundleAtURL: completes (or fails).
/// `error` is nil on success.  Always invoked on the main thread.
typedef void (^RuntimeViewLoadCompletion)(NSError *_Nullable error);

@interface RuntimeView : UIView

/// The bundle URL currently displayed (or being loaded).  nil before
/// the first loadBundleAtURL: call.  KVO-observable.
@property (nonatomic, readonly, copy, nullable) NSURL *currentBundleURL;

/// The Box identity currently mounted.  nil before the first
/// loadBundleAtURL:identity: call.
@property (nonatomic, readonly, strong, nullable) RuntimeBoxIdentity *currentIdentity;

/// Asks the runtime to fetch the bundle at `url` and mount its React
/// tree into this view's bounds.  `identity` is exposed to the JS
/// runtime as `host().identity` (boxID / version / title).  Calling
/// this while another bundle is loaded resets first.
- (void)loadBundleAtURL:(NSURL *)url
               identity:(RuntimeBoxIdentity *)identity
             completion:(nullable RuntimeViewLoadCompletion)completion;

/// Convenience overload that supplies an empty identity.  Provided so
/// existing host call sites compile against the new SDK without a
/// flag-day; new code should pass an explicit identity.
- (void)loadBundleAtURL:(NSURL *)url
             completion:(nullable RuntimeViewLoadCompletion)completion;

/// Tears down the current bundle's runtime state.  Safe to call when
/// no bundle is loaded (no-op).
- (void)reset;

@end

NS_ASSUME_NONNULL_END

#endif /* RuntimeView_h */

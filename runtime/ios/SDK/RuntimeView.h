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

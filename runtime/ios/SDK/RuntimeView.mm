// RuntimeView — D3.c implementation: real React Native mount.
//
// Owns one RCTReactNativeFactory per loadBundleAtURL: call.  The factory
// stands up a fresh Hermes runtime, evaluates the JS bundle at the given
// URL, and produces a UIView via its rootViewFactory; we add that view
// as a subview filling self.bounds.
//
// The factory is the single owner of all RN-side state for this bundle:
// tearing it down (reset / next loadBundleAtURL:) releases the Hermes
// runtime and the JSI<-->ObjC bridge, so cross-bundle JS state can't
// leak.  Each RuntimeView instance therefore runs an isolated runtime.
//
// JS contract: the bundle MUST register a component named "RuntimeRoot"
// via AppRegistry.registerComponent, otherwise rootViewFactory returns
// a view that just shows RN's red-box error overlay.  The fixture
// bundle in runtime/sandbox/fixture-rn/ does exactly this.
//
// This is deliberately the simplest viable mount: D3.d will plumb in
// the Tier-1 native modules (gesture-handler / reanimated / etc.),
// D3.e replaces the placeholder HostBridge with a real C++/JNI impl.
#import "RuntimeView.h"

#import <React/RCTBridge.h>
#import <React_RCTAppDelegate/RCTDefaultReactNativeFactoryDelegate.h>
#import <React_RCTAppDelegate/RCTReactNativeFactory.h>
#import <React_RCTAppDelegate/RCTRootViewFactory.h>
#import <ReactAppDependencyProvider/RCTAppDependencyProvider.h>

// Per-instance delegate that just hands the factory the URL we were
// asked to load.  RCTDefaultReactNativeFactoryDelegate handles the rest
// (turbo modules registration via dependencyProvider, default config).
@interface RuntimeViewFactoryDelegate : RCTDefaultReactNativeFactoryDelegate
@property (nonatomic, copy, nullable) NSURL *bundleURLOverride;
@end

@implementation RuntimeViewFactoryDelegate
- (NSURL *)bundleURL {
    return self.bundleURLOverride;
}
- (NSURL *)sourceURLForBridge:(RCTBridge *)bridge {
    return self.bundleURLOverride;
}
@end

@interface RuntimeView ()
@property (nonatomic, copy, nullable) NSURL *currentBundleURL;
@property (nonatomic, strong, nullable) RuntimeViewFactoryDelegate *rnDelegate;
@property (nonatomic, strong, nullable) RCTReactNativeFactory *rnFactory;
@property (nonatomic, weak,   nullable) UIView *rnRootView;
@end

@implementation RuntimeView

- (instancetype)initWithFrame:(CGRect)frame {
    if ((self = [super initWithFrame:frame])) {
        self.backgroundColor = [UIColor blackColor];
    }
    return self;
}

- (instancetype)initWithCoder:(NSCoder *)coder {
    if ((self = [super initWithCoder:coder])) {
        self.backgroundColor = [UIColor blackColor];
    }
    return self;
}

- (void)loadBundleAtURL:(NSURL *)url
             completion:(RuntimeViewLoadCompletion)completion {
    [self reset];

    self.currentBundleURL = url;

    RuntimeViewFactoryDelegate *delegate = [[RuntimeViewFactoryDelegate alloc] init];
    delegate.bundleURLOverride = url;
    delegate.dependencyProvider = [[RCTAppDependencyProvider alloc] init];
    self.rnDelegate = delegate;

    RCTReactNativeFactory *factory = [[RCTReactNativeFactory alloc] initWithDelegate:delegate];
    self.rnFactory = factory;

    UIView *rootView = [factory.rootViewFactory viewWithModuleName:@"RuntimeRoot"
                                                  initialProperties:@{}
                                                      launchOptions:nil];
    rootView.translatesAutoresizingMaskIntoConstraints = NO;
    [self addSubview:rootView];
    [NSLayoutConstraint activateConstraints:@[
        [rootView.topAnchor      constraintEqualToAnchor:self.topAnchor],
        [rootView.bottomAnchor   constraintEqualToAnchor:self.bottomAnchor],
        [rootView.leadingAnchor  constraintEqualToAnchor:self.leadingAnchor],
        [rootView.trailingAnchor constraintEqualToAnchor:self.trailingAnchor],
    ]];
    self.rnRootView = rootView;

    // RCTRootViewFactory loads + evaluates the bundle async on a JS
    // queue; the rootView returned shows a loading state until ready.
    // For the D3.c API contract we treat the moment after view-tree
    // attachment as "load complete" — bundle-fetch errors will surface
    // as RN's red-box overlay inside rootView, which is fine for now.
    // D3.e wires up a real progress / error callback path.
    if (completion) {
        dispatch_async(dispatch_get_main_queue(), ^{
            completion(nil);
        });
    }
}

- (void)reset {
    [self.rnRootView removeFromSuperview];
    self.rnRootView       = nil;
    self.rnFactory        = nil;
    self.rnDelegate       = nil;
    self.currentBundleURL = nil;
}

@end

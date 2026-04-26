// RuntimeBoxIdentity — read-only descriptor of the AI-generated Box
// the host has asked us to load.  Mirrors `BoxIdentity` in
// runtime/src/HostBridge.ts so AI bundles see exactly this shape via
// `host().identity`.
//
// Passed to RuntimeView's loadBundleAtURL:identity:completion: at load
// time.  The runtime SDK doesn't validate any of these fields — the
// host shell is responsible for sourcing them from its relay client
// (BoxWire.id / version / title).
#ifndef RuntimeBoxIdentity_h
#define RuntimeBoxIdentity_h

#import <Foundation/Foundation.h>

NS_ASSUME_NONNULL_BEGIN

@interface RuntimeBoxIdentity : NSObject

/// Stable per-Box id assigned by the relay.  Empty string on dev /
/// unbuilt drafts.
@property (nonatomic, readonly, copy) NSString *boxID;

/// Bundle version string (e.g. "v3").  Empty string on unbuilt drafts.
@property (nonatomic, readonly, copy) NSString *version;

/// Short title for display, mirrors BoxWire.title from the relay.
@property (nonatomic, readonly, copy) NSString *title;

- (instancetype)initWithBoxID:(NSString *)boxID
                      version:(NSString *)version
                        title:(NSString *)title NS_DESIGNATED_INITIALIZER;

- (instancetype)init NS_UNAVAILABLE;

@end

NS_ASSUME_NONNULL_END

#endif /* RuntimeBoxIdentity_h */

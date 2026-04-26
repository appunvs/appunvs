#import "RuntimeBoxIdentity.h"

@implementation RuntimeBoxIdentity

- (instancetype)initWithBoxID:(NSString *)boxID
                      version:(NSString *)version
                        title:(NSString *)title {
    if ((self = [super init])) {
        _boxID   = [boxID   copy] ?: @"";
        _version = [version copy] ?: @"";
        _title   = [title   copy] ?: @"";
    }
    return self;
}

@end

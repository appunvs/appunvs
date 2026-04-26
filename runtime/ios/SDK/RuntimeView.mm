// RuntimeView — D2.c placeholder implementation.
//
// Owns one centred UILabel that reflects the load state:
//
//   1. before loadBundleAtURL: → empty + "no bundle loaded"
//   2. during load              → "loading <url>..."
//   3. after load                → "loaded <url>"
//   4. after reset               → "(reset; no bundle)"
//
// Lets the host smoke-test the API + view-hierarchy embedding without
// needing the RN brownfield work D3 brings in.
#import "RuntimeView.h"

@interface RuntimeView ()
@property (nonatomic, copy, nullable) NSURL *currentBundleURL;
@property (nonatomic, strong) UILabel *statusLabel;
@end

@implementation RuntimeView

- (instancetype)initWithFrame:(CGRect)frame {
    if ((self = [super initWithFrame:frame])) {
        [self setUp];
    }
    return self;
}

- (instancetype)initWithCoder:(NSCoder *)coder {
    if ((self = [super initWithCoder:coder])) {
        [self setUp];
    }
    return self;
}

- (void)setUp {
    self.backgroundColor = [UIColor blackColor];

    UILabel *label = [[UILabel alloc] init];
    label.translatesAutoresizingMaskIntoConstraints = NO;
    label.textAlignment = NSTextAlignmentCenter;
    label.numberOfLines = 0;
    label.font = [UIFont monospacedSystemFontOfSize:13.0
                                            weight:UIFontWeightRegular];
    label.textColor = [UIColor lightGrayColor];
    label.text = @"no bundle loaded";

    [self addSubview:label];
    [NSLayoutConstraint activateConstraints:@[
        [label.centerXAnchor constraintEqualToAnchor:self.centerXAnchor],
        [label.centerYAnchor constraintEqualToAnchor:self.centerYAnchor],
        [label.leadingAnchor  constraintGreaterThanOrEqualToAnchor:self.leadingAnchor  constant:16.0],
        [label.trailingAnchor constraintLessThanOrEqualToAnchor:self.trailingAnchor    constant:-16.0],
    ]];

    self.statusLabel = label;
}

- (void)loadBundleAtURL:(NSURL *)url
             completion:(RuntimeViewLoadCompletion)completion {
    [self reset];

    self.currentBundleURL = url;
    self.statusLabel.text = [NSString stringWithFormat:@"loading\n%@", url.absoluteString];

    // D2.c placeholder: just display the URL.  Simulate "load
    // complete" on the next runloop turn so the API contract
    // (completion callback) holds.
    dispatch_async(dispatch_get_main_queue(), ^{
        self.statusLabel.text = [NSString stringWithFormat:@"loaded\n%@\n\n(D3 replaces this with a real React Native render)", url.absoluteString];
        if (completion) {
            completion(nil);
        }
    });
}

- (void)reset {
    self.currentBundleURL = nil;
    self.statusLabel.text = @"(reset; no bundle)";
}

@end

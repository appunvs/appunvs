#import "RuntimeSDK.h"

// RuntimeSDKVersionNumber + RuntimeSDKVersionString are defined by
// Xcode's auto-generated RuntimeSDK_vers.c — defining them here too
// produces "duplicate symbol" linker errors.

const char *runtime_sdk_hello(void) {
    return "hello from runtime SDK (D2.a empty shell)";
}

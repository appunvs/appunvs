// Net config — relay base URL.  Hardcoded to localhost for dev; later
// pulls from a build-time setting injected via xcconfig.  Override at
// runtime by setting `APPUNVS_RELAY_URL` in the scheme's environment
// variables — useful when pointing the simulator at a host other than
// the dev machine.
import Foundation

enum NetConfig {
    static var relayBaseURL: URL {
        if let raw = ProcessInfo.processInfo.environment["APPUNVS_RELAY_URL"],
           let url = URL(string: raw) {
            return url
        }
        return URL(string: "http://localhost:8080")!
    }
}

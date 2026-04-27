// RuntimeBridgeWiring — registers the SDK's AppunvsHostModule callbacks
// against this host shell's HTTPClient and publish flow.
//
// Called from SignedInRoot.wireAndLoad() once we have a live, auth'd
// HTTPClient.  Without these registrations, AI bundles loaded into a
// RuntimeView would call host().network.request() / .publish() and get
// "host hasn't registered a ... handler" errors.
//
// What this wires:
//   - request handler -> HTTPClient.raw (auth-aware, follows token rotation)
//   - publish handler -> stub returning { version: "", ok: false }
//                        (relay's publish endpoint not yet defined; the
//                         contract is wired so AI bundles get a real
//                         response shape rather than a "no handler"
//                         rejection — when the relay surface lands,
//                         swap this for the real call).
//
// What's NOT wired here (deliberately):
//   - SubNetwork.subscribe (SSE) — needs RCTEventEmitter migration on
//     the SDK side + a generic SSE consumer; tracked as the only D3.e
//     remainder.
//
// Registrations are per-process; calling register(http:) again replaces
// the previous handlers (e.g., on sign-out + re-sign-in the new
// HTTPClient takes over).
import Foundation
import RuntimeSDK

@MainActor
enum RuntimeBridgeWiring {

    static func register(http: HTTPClient) {
        registerRequest(http: http)
        registerPublish()
    }

    // MARK: - request

    private static func registerRequest(http: HTTPClient) {
        AppunvsHostModule.registerRequestHandler { method, path, body, completion in
            // The handler block runs on the React bridge's queue.  We can't
            // `await` directly there, so spin a Task that awaits the actor
            // and delivers via the completion callback.
            Task {
                do {
                    let result = try await http.raw(
                        method: method,
                        path: path,
                        body: body,
                    )
                    completion([
                        "status":  result.status,
                        "headers": result.headers,
                        "body":    result.body,
                    ], nil)
                } catch {
                    completion(nil, error as NSError)
                }
            }
        }
    }

    // MARK: - publish

    private static func registerPublish() {
        AppunvsHostModule.registerPublishHandler { _, completion in
            // No relay-side publish endpoint defined yet (BoxAPI doesn't
            // expose one).  AI bundles calling publish() get an explicit
            // ok=false response so they can surface "publish unavailable"
            // — better than a "no handler" rejection because the JS side
            // can branch on .ok.  When the relay endpoint lands, replace
            // with a real /box/{id}/publish call.
            completion([
                "version": "",
                "ok": false,
            ], nil)
        }
    }
}

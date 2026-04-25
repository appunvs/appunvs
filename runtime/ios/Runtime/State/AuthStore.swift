// AuthStore — owns the local view of the user's auth state.
//
// Lifecycle:
//   1. boot()       loads persisted device token (if any), drives gate
//   2. signup/login mints a session token, then immediately calls
//                  /auth/register to swap it for a device token
//   3. signOut()    clears Keychain + memory
//
// We keep a stable per-install `device_id` in UserDefaults — the relay
// uses it as the key for /auth/register so the same simulator instance
// keeps reusing one device row instead of registering a new one each
// launch.  The token itself lives in Keychain.
import Foundation
import SwiftUI

@MainActor
final class AuthStore: ObservableObject {
    enum Phase: Equatable {
        case bootstrapping
        case signedOut
        case signedIn(userID: String)
    }

    @Published private(set) var phase: Phase = .bootstrapping
    @Published private(set) var me: MeResponse?
    @Published var lastError: String?

    /// Currently active token surfaced to network clients via closure.
    /// During `signup` / `login` it briefly holds the **session** token
    /// (so /auth/register works) and is then replaced by the **device**
    /// token for the rest of the session.
    private var token: String?

    private let http: HTTPClient
    private let auth: AuthAPI

    init() {
        // The HTTPClient captures `self` weakly via the closure so token
        // rotations are picked up by in-flight calls without re-injecting.
        var tokenRef: (() -> String?)?
        let provider: @Sendable () -> String? = { tokenRef?() }
        self.http = HTTPClient(tokenProvider: provider)
        self.auth = AuthAPI(http: self.http)
        tokenRef = { [weak self] in self?.token }
    }

    /// Shared client used by Box / AI stores so they share the same
    /// token rotation surface.
    var sharedHTTP: HTTPClient { http }

    /// SSE client tied to the same token closure as `sharedHTTP`.
    func aiClient() -> AISSEClient {
        AISSEClient(tokenProvider: { [weak self] in self?.token })
    }

    // MARK: - Lifecycle

    func boot() async {
        if let saved = Keychain.get(Keys.deviceToken),
           let userID = Keychain.get(Keys.userID) {
            token = saved
            phase = .signedIn(userID: userID)
            // Pre-fetch /auth/me with the *session* token in mind — but
            // /auth/me requires a session token, so we skip it on cold
            // boot when we only have a device token.  ProfileView stays
            // empty until the next login, which is fine for v1.
            return
        }
        phase = .signedOut
    }

    func signup(email: String, password: String) async {
        await runAuth(label: "signup") {
            try await self.auth.signup(email: email, password: password)
        }
    }

    func login(email: String, password: String) async {
        await runAuth(label: "login") {
            try await self.auth.login(email: email, password: password)
        }
    }

    func signOut() {
        token = nil
        me = nil
        Keychain.remove(Keys.deviceToken)
        Keychain.remove(Keys.userID)
        phase = .signedOut
    }

    // MARK: - private

    private func runAuth(label: String, op: () async throws -> SessionResponse) async {
        lastError = nil
        do {
            let session = try await op()
            // 1. Hold the session token transiently so /auth/register
            //    accepts it.
            token = session.sessionToken
            // 2. Capture profile while the session token is still hot.
            //    Failure here is non-fatal — we proceed to /register.
            self.me = try? await auth.me()
            // 3. Swap to a device token for the rest of the session.
            let dev = try await auth.registerDevice(
                deviceID: stableDeviceID(),
                platform: "mobile"
            )
            token = dev.token
            Keychain.set(dev.token, for: Keys.deviceToken)
            Keychain.set(session.userID, for: Keys.userID)
            phase = .signedIn(userID: session.userID)
        } catch let err as HTTPError {
            lastError = err.errorDescription ?? "\(label) failed"
            token = nil
        } catch {
            lastError = "\(label) failed: \(error.localizedDescription)"
            token = nil
        }
    }

    /// A per-install identifier reused across launches so the relay
    /// keeps one device row per simulator/device.  Generated lazily on
    /// first use; persists in UserDefaults (not Keychain — losing this
    /// is harmless, the relay just creates a new device row).
    private func stableDeviceID() -> String {
        let key = Keys.deviceID
        if let existing = UserDefaults.standard.string(forKey: key) {
            return existing
        }
        let id = "d_" + UUID().uuidString.replacingOccurrences(of: "-", with: "").lowercased()
        UserDefaults.standard.set(id, forKey: key)
        return id
    }

    private enum Keys {
        static let deviceToken = "appunvs.deviceToken"
        static let userID      = "appunvs.userID"
        static let deviceID    = "appunvs.deviceID"
    }
}

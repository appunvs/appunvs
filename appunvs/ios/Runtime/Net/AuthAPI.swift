// AuthAPI — typed wrappers for the relay's /auth/* endpoints.
//
//   POST /auth/signup     public   -> session token
//   POST /auth/login      public   -> session token
//   POST /auth/register   session  -> device token
//   GET  /auth/me         session  -> profile + devices
//
// The two-step (session -> device) shape lives here intentionally:
// callers normally invoke `signup(...)` / `login(...)`, then immediately
// `registerDevice(sessionToken:...)` to get the longer-lived device
// token used by everything else (box / pair / ai/turn).
import Foundation

struct AuthAPI {
    let http: HTTPClient

    func signup(email: String, password: String) async throws -> SessionResponse {
        try await http.send(
            "/auth/signup",
            method: .post,
            body: SignupRequest(email: email, password: password)
        )
    }

    func login(email: String, password: String) async throws -> SessionResponse {
        try await http.send(
            "/auth/login",
            method: .post,
            body: LoginRequest(email: email, password: password)
        )
    }

    /// Calls /auth/register with the **session** token in scope (caller
    /// must arrange for `tokenProvider` to surface it for this call).
    /// Returns a long-lived device token used by box/pair/ai endpoints.
    func registerDevice(deviceID: String, platform: String) async throws -> RegisterResponse {
        try await http.send(
            "/auth/register",
            method: .post,
            body: RegisterRequest(deviceID: deviceID, platform: platform)
        )
    }

    func me() async throws -> MeResponse {
        try await http.get("/auth/me")
    }
}

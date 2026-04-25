// HTTPClient — typed wrapper around URLSession + async/await.
//
// One client per process, picks up the auth token via a closure so
// the `AuthStore`'s token rotations are visible to in-flight calls
// without re-injecting per call.
//
// Errors surface as `HTTPError` carrying status code + parsed
// `ErrorResponse.error` when the server returned that shape.
import Foundation

enum HTTPError: Error, LocalizedError {
    case status(Int, String)
    case decoding(Error)
    case transport(Error)
    case noData

    var errorDescription: String? {
        switch self {
        case .status(let code, let msg): return "HTTP \(code): \(msg)"
        case .decoding(let err):         return "Decode failed: \(err)"
        case .transport(let err):        return "Network: \(err)"
        case .noData:                    return "Empty response body"
        }
    }
}

enum HTTPMethod: String {
    case get  = "GET"
    case post = "POST"
    case put  = "PUT"
    case del  = "DELETE"
}

actor HTTPClient {
    private let baseURL: URL
    private let session: URLSession
    private let tokenProvider: @Sendable () -> String?

    init(
        baseURL: URL = NetConfig.relayBaseURL,
        session: URLSession = .shared,
        tokenProvider: @escaping @Sendable () -> String?
    ) {
        self.baseURL = baseURL
        self.session = session
        self.tokenProvider = tokenProvider
    }

    // MARK: - GET

    func get<R: Decodable>(_ path: String) async throws -> R {
        try await request(path, method: .get, body: Optional<EmptyBody>.none)
    }

    // MARK: - POST / DELETE etc.

    func send<B: Encodable, R: Decodable>(
        _ path: String,
        method: HTTPMethod = .post,
        body: B? = nil
    ) async throws -> R {
        try await request(path, method: method, body: body)
    }

    /// Variant for endpoints that return 204 / empty body.
    func sendNoContent<B: Encodable>(
        _ path: String,
        method: HTTPMethod = .post,
        body: B? = nil
    ) async throws {
        let _: EmptyBody = try await request(path, method: method, body: body, allowEmpty: true)
    }

    // MARK: - core

    private func request<B: Encodable, R: Decodable>(
        _ path: String,
        method: HTTPMethod,
        body: B?,
        allowEmpty: Bool = false
    ) async throws -> R {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = method.rawValue
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = tokenProvider() {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let body {
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = try JSONEncoder().encode(body)
        }

        let data: Data
        let resp: URLResponse
        do {
            (data, resp) = try await session.data(for: req)
        } catch {
            throw HTTPError.transport(error)
        }
        guard let http = resp as? HTTPURLResponse else {
            throw HTTPError.noData
        }
        guard (200..<300).contains(http.statusCode) else {
            let msg = (try? JSONDecoder().decode(ErrorResponse.self, from: data))?.error
                ?? String(data: data, encoding: .utf8)
                ?? "(no body)"
            throw HTTPError.status(http.statusCode, msg)
        }
        if allowEmpty && data.isEmpty {
            // Caller said empty body is OK; manufacture a default-constructed
            // EmptyBody so the generic return type is satisfied.
            if R.self == EmptyBody.self {
                return EmptyBody() as! R
            }
        }
        do {
            return try JSONDecoder().decode(R.self, from: data)
        } catch {
            throw HTTPError.decoding(error)
        }
    }
}

/// Marker type for endpoints with no request body / no response body.
struct EmptyBody: Codable, Sendable {}

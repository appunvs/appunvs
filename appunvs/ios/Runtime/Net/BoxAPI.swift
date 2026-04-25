// BoxAPI — typed wrappers for /box and /pair.  All routes require a
// device token (see BoxDeps / PairingDeps in relay/internal/handler).
import Foundation

struct BoxAPI {
    let http: HTTPClient

    func list() async throws -> [BoxWire] {
        let resp: BoxListResponse = try await http.get("/box")
        return resp.boxes ?? []
    }

    func create(title: String, runtime: String = "rn_bundle") async throws -> BoxResponse {
        try await http.send(
            "/box",
            method: .post,
            body: BoxCreateRequest(title: title, runtime: runtime)
        )
    }

    func get(id: String) async throws -> BoxResponse {
        try await http.get("/box/\(id)")
    }

    func archive(id: String) async throws {
        try await http.sendNoContent(
            "/box/\(id)",
            method: .del,
            body: Optional<EmptyBody>.none
        )
    }
}

struct PairAPI {
    let http: HTTPClient

    func issue(boxID: String, ttlSec: Int = 300) async throws -> PairResponse {
        try await http.send(
            "/pair",
            method: .post,
            body: PairRequestBody(boxID: boxID, ttlSec: ttlSec)
        )
    }

    func claim(code: String, deviceID: String, platform: String) async throws -> PairClaimResponse {
        try await http.send(
            "/pair/\(code)/claim",
            method: .post,
            body: PairClaimRequest(deviceID: deviceID, platform: platform)
        )
    }
}

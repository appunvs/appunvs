// Models — Codable mirrors of `shared/proto/{auth,box,pair,common}.proto`.
//
// We hand-mirror rather than generate via swift-protobuf because:
//   (a) the wire is canonical protojson, not binary protobuf — Codable
//       handles JSON natively and the field-name mapping is trivial
//       (`CodingKeys` with snake_case strings)
//   (b) zero codegen step keeps the iOS build hermetic
//
// If a shape drifts from the proto, the relay's `internal/pb` drift
// test still catches it on the relay side; we update both halves
// together when the schema changes.
import Foundation

// MARK: - Common

enum Platform: String, Codable {
    case unspecified = "unspecified"
    case browser, desktop, mobile
}

// MARK: - Auth

struct SignupRequest: Encodable {
    let email: String
    let password: String
}

struct LoginRequest: Encodable {
    let email: String
    let password: String
}

struct SessionResponse: Decodable {
    let userID: String
    let sessionToken: String

    enum CodingKeys: String, CodingKey {
        case userID       = "user_id"
        case sessionToken = "session_token"
    }
}

struct RegisterRequest: Encodable {
    let deviceID: String
    let platform: String

    enum CodingKeys: String, CodingKey {
        case deviceID = "device_id"
        case platform
    }
}

struct RegisterResponse: Decodable {
    let token: String
    let userID: String

    enum CodingKeys: String, CodingKey {
        case token
        case userID = "user_id"
    }
}

struct DeviceInfo: Decodable, Identifiable {
    let id: String
    let userID: String
    let platform: String
    let createdAt: Int64
    let lastSeen: Int64

    enum CodingKeys: String, CodingKey {
        case id
        case userID    = "user_id"
        case platform
        case createdAt = "created_at"
        case lastSeen  = "last_seen"
    }
}

struct MeResponse: Decodable {
    let userID: String
    let email: String
    let createdAt: Int64
    let devices: [DeviceInfo]?

    enum CodingKeys: String, CodingKey {
        case userID    = "user_id"
        case email
        case createdAt = "created_at"
        case devices
    }
}

// MARK: - Box / BundleRef

enum BoxStateWire: String, Codable {
    case unspecified = "unspecified"
    case draft, published, archived
}

enum BuildStateWire: String, Codable {
    case unspecified = "unspecified"
    case queued, running, succeeded, failed
}

enum RuntimeKindWire: String, Codable {
    case unspecified = "unspecified"
    case rnBundle = "rn_bundle"
}

struct BoxWire: Decodable, Identifiable, Hashable {
    var id: String { boxID }
    let boxID: String
    let namespace: String
    let providerDeviceID: String
    let title: String
    let runtime: RuntimeKindWire
    let state: BoxStateWire
    let currentVersion: String
    let createdAt: Int64
    let updatedAt: Int64

    enum CodingKeys: String, CodingKey {
        case boxID            = "box_id"
        case namespace
        case providerDeviceID = "provider_device_id"
        case title
        case runtime
        case state
        case currentVersion   = "current_version"
        case createdAt        = "created_at"
        case updatedAt        = "updated_at"
    }
}

struct BundleRef: Decodable, Hashable {
    let boxID: String
    let version: String
    let uri: String
    let contentHash: String
    let sizeBytes: Int64
    let buildState: BuildStateWire
    let buildLog: String?
    let builtAt: Int64
    let expiresAt: Int64

    enum CodingKeys: String, CodingKey {
        case boxID       = "box_id"
        case version
        case uri
        case contentHash = "content_hash"
        case sizeBytes   = "size_bytes"
        case buildState  = "build_state"
        case buildLog    = "build_log"
        case builtAt     = "built_at"
        case expiresAt   = "expires_at"
    }
}

struct BoxResponse: Decodable {
    let box: BoxWire
    let current: BundleRef?
}

struct BoxListResponse: Decodable {
    let boxes: [BoxWire]?
}

struct BoxCreateRequest: Encodable {
    let title: String
    let runtime: String
}

// MARK: - Pair

struct PairRequestBody: Encodable {
    let boxID: String
    let ttlSec: Int

    enum CodingKeys: String, CodingKey {
        case boxID  = "box_id"
        case ttlSec = "ttl_sec"
    }
}

struct PairResponse: Decodable {
    let shortCode: String
    let expiresAt: Int64

    enum CodingKeys: String, CodingKey {
        case shortCode = "short_code"
        case expiresAt = "expires_at"
    }
}

struct PairClaimRequest: Encodable {
    let deviceID: String
    let platform: String

    enum CodingKeys: String, CodingKey {
        case deviceID = "device_id"
        case platform
    }
}

struct PairClaimResponse: Decodable {
    let boxID: String
    let bundle: BundleRef?
    let namespaceToken: String

    enum CodingKeys: String, CodingKey {
        case boxID          = "box_id"
        case bundle
        case namespaceToken = "namespace_token"
    }
}

// MARK: - Generic

struct ErrorResponse: Decodable {
    let error: String
}

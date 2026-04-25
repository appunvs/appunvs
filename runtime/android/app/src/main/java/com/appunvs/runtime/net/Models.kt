// Models — Kotlinx-serialization mirrors of `shared/proto/*.proto`.
//
// Hand-mirrored (no protoc-jvm step) for the same reasons as the iOS
// side: protojson is plain JSON, snake_case mapping is trivial, zero
// codegen step keeps the Gradle build hermetic.  When a shape drifts,
// the relay's `internal/pb` drift test catches it on the relay side
// and we update both halves together.
package com.appunvs.runtime.net

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// MARK: - Auth

@Serializable
data class SignupRequest(
    val email: String,
    val password: String,
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
)

@Serializable
data class SessionResponse(
    @SerialName("user_id")       val userID: String,
    @SerialName("session_token") val sessionToken: String,
)

@Serializable
data class RegisterRequest(
    @SerialName("device_id") val deviceID: String,
    val platform: String,
)

@Serializable
data class RegisterResponse(
    val token: String,
    @SerialName("user_id") val userID: String,
)

@Serializable
data class DeviceInfo(
    val id: String,
    @SerialName("user_id")    val userID: String,
    val platform: String,
    @SerialName("created_at") val createdAt: Long,
    @SerialName("last_seen")  val lastSeen: Long,
)

@Serializable
data class MeResponse(
    @SerialName("user_id")    val userID: String,
    val email: String,
    @SerialName("created_at") val createdAt: Long,
    val devices: List<DeviceInfo> = emptyList(),
)

// MARK: - Box

@Serializable
data class BoxWire(
    @SerialName("box_id")              val boxID: String,
    val namespace: String,
    @SerialName("provider_device_id")  val providerDeviceID: String,
    val title: String,
    val runtime: String,
    val state: String,
    @SerialName("current_version")     val currentVersion: String,
    @SerialName("created_at")          val createdAt: Long,
    @SerialName("updated_at")          val updatedAt: Long,
)

@Serializable
data class BundleRef(
    @SerialName("box_id")       val boxID: String,
    val version: String,
    val uri: String,
    @SerialName("content_hash") val contentHash: String,
    @SerialName("size_bytes")   val sizeBytes: Long,
    @SerialName("build_state")  val buildState: String,
    @SerialName("build_log")    val buildLog: String? = null,
    @SerialName("built_at")     val builtAt: Long,
    @SerialName("expires_at")   val expiresAt: Long,
)

@Serializable
data class BoxResponse(
    val box: BoxWire,
    val current: BundleRef? = null,
)

@Serializable
data class BoxListResponse(
    val boxes: List<BoxWire> = emptyList(),
)

@Serializable
data class BoxCreateRequest(
    val title: String,
    val runtime: String = "rn_bundle",
)

// MARK: - Pair

@Serializable
data class PairRequestBody(
    @SerialName("box_id")  val boxID: String,
    @SerialName("ttl_sec") val ttlSec: Int,
)

@Serializable
data class PairResponse(
    @SerialName("short_code") val shortCode: String,
    @SerialName("expires_at") val expiresAt: Long,
)

@Serializable
data class PairClaimRequest(
    @SerialName("device_id") val deviceID: String,
    val platform: String,
)

@Serializable
data class PairClaimResponse(
    @SerialName("box_id")          val boxID: String,
    val bundle: BundleRef? = null,
    @SerialName("namespace_token") val namespaceToken: String,
)

// MARK: - AI

@Serializable
data class AITurnRequest(
    @SerialName("box_id") val boxID: String,
    val text: String,
)

// MARK: - Generic

@Serializable
data class ErrorResponse(
    val error: String,
)

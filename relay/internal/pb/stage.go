// Stage-related types: Box, BundleRef, pairing request/response, version
// update events, and AI chat stream frames. These mirror the messages added
// to shared/proto/appunvs.proto alongside the original Message envelope.
//
// Canonical on-the-wire form is protojson with UseProtoNames=true and enum
// short lowercase values ("publish_state_published" -> "published",
// "build_state_succeeded" -> "succeeded").
package pb

import "strings"

// PublishState mirrors appunvs.v1.PublishState.
type PublishState int32

const (
	PublishStateUnspecified PublishState = 0
	PublishStateDraft       PublishState = 1
	PublishStatePublished   PublishState = 2
	PublishStateArchived    PublishState = 3
)

func (p PublishState) String() string {
	switch p {
	case PublishStateDraft:
		return "draft"
	case PublishStatePublished:
		return "published"
	case PublishStateArchived:
		return "archived"
	default:
		return "unspecified"
	}
}

// ParsePublishState accepts short or fully qualified names.
func ParsePublishState(s string) PublishState {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "PUBLISH_STATE_")) {
	case "draft":
		return PublishStateDraft
	case "published":
		return PublishStatePublished
	case "archived":
		return PublishStateArchived
	default:
		return PublishStateUnspecified
	}
}

// BuildState mirrors appunvs.v1.BuildState.
type BuildState int32

const (
	BuildStateUnspecified BuildState = 0
	BuildStateQueued      BuildState = 1
	BuildStateRunning     BuildState = 2
	BuildStateSucceeded   BuildState = 3
	BuildStateFailed      BuildState = 4
)

func (b BuildState) String() string {
	switch b {
	case BuildStateQueued:
		return "queued"
	case BuildStateRunning:
		return "running"
	case BuildStateSucceeded:
		return "succeeded"
	case BuildStateFailed:
		return "failed"
	default:
		return "unspecified"
	}
}

// ParseBuildState accepts short or fully qualified names.
func ParseBuildState(s string) BuildState {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "BUILD_STATE_")) {
	case "queued":
		return BuildStateQueued
	case "running":
		return BuildStateRunning
	case "succeeded":
		return BuildStateSucceeded
	case "failed":
		return BuildStateFailed
	default:
		return BuildStateUnspecified
	}
}

// RuntimeKind mirrors appunvs.v1.RuntimeKind.
type RuntimeKind int32

const (
	RuntimeKindUnspecified RuntimeKind = 0
	RuntimeKindRNBundle    RuntimeKind = 1
)

func (r RuntimeKind) String() string {
	switch r {
	case RuntimeKindRNBundle:
		return "rn_bundle"
	default:
		return "unspecified"
	}
}

// ParseRuntimeKind accepts short or fully qualified names.
func ParseRuntimeKind(s string) RuntimeKind {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "RUNTIME_KIND_")) {
	case "rn_bundle":
		return RuntimeKindRNBundle
	default:
		return RuntimeKindUnspecified
	}
}

// Box mirrors appunvs.v1.Box.
type Box struct {
	BoxID            string       `json:"box_id,omitempty"`
	Namespace        string       `json:"namespace,omitempty"`
	ProviderDeviceID string       `json:"provider_device_id,omitempty"`
	Title            string       `json:"title,omitempty"`
	Runtime          RuntimeKind  `json:"runtime,omitempty"`
	State            PublishState `json:"state,omitempty"`
	CurrentVersion   string       `json:"current_version,omitempty"`
	CreatedAt        int64        `json:"created_at,omitempty"`
	UpdatedAt        int64        `json:"updated_at,omitempty"`
}

// BundleRef mirrors appunvs.v1.BundleRef.
type BundleRef struct {
	BoxID       string     `json:"box_id,omitempty"`
	Version     string     `json:"version,omitempty"`
	URI         string     `json:"uri,omitempty"`
	ContentHash string     `json:"content_hash,omitempty"`
	SizeBytes   int64      `json:"size_bytes,omitempty"`
	BuildState  BuildState `json:"build_state,omitempty"`
	BuildLog    string     `json:"build_log,omitempty"`
	BuiltAt     int64      `json:"built_at,omitempty"`
	ExpiresAt   int64      `json:"expires_at,omitempty"`
}

// BoxCreateRequest mirrors appunvs.v1.BoxCreateRequest.
type BoxCreateRequest struct {
	Title   string      `json:"title,omitempty"`
	Runtime RuntimeKind `json:"runtime,omitempty"`
}

// BoxResponse mirrors appunvs.v1.BoxResponse.
type BoxResponse struct {
	Box     Box        `json:"box"`
	Current *BundleRef `json:"current,omitempty"`
}

// BoxListResponse mirrors appunvs.v1.BoxListResponse.
type BoxListResponse struct {
	Boxes []Box `json:"boxes"`
}

// PairRequest mirrors appunvs.v1.PairRequest.
type PairRequest struct {
	BoxID  string `json:"box_id,omitempty"`
	TTLSec int32  `json:"ttl_sec,omitempty"`
}

// PairResponse mirrors appunvs.v1.PairResponse.
type PairResponse struct {
	ShortCode string `json:"short_code,omitempty"`
	ExpiresAt int64  `json:"expires_at,omitempty"`
}

// PairClaimRequest mirrors appunvs.v1.PairClaimRequest.
type PairClaimRequest struct {
	DeviceID string   `json:"device_id,omitempty"`
	Platform Platform `json:"platform,omitempty"`
}

// PairClaimResponse mirrors appunvs.v1.PairClaimResponse.
type PairClaimResponse struct {
	BoxID          string     `json:"box_id,omitempty"`
	Bundle         *BundleRef `json:"bundle,omitempty"`
	NamespaceToken string     `json:"namespace_token,omitempty"`
}

// BoxVersionUpdate mirrors appunvs.v1.BoxVersionUpdate.
type BoxVersionUpdate struct {
	BoxID  string     `json:"box_id,omitempty"`
	Bundle *BundleRef `json:"bundle,omitempty"`
}

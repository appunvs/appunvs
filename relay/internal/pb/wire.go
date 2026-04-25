// Package pb contains hand-written Go types that mirror the protobuf
// definitions in shared/proto/*.proto.
//
// TODO: shared/proto/*.proto is the canonical source of truth for
// these types. If the proto changes, update this package by hand (or, later,
// by running `buf generate`).  For this task we intentionally avoid protoc
// so the relay has zero build-time codegen dependencies.
package pb

import "strings"

// Platform mirrors appunvs.Platform.
type Platform int32

const (
	PlatformUnspecified Platform = 0
	PlatformBrowser     Platform = 1
	PlatformDesktop     Platform = 2
	PlatformMobile      Platform = 3
)

// String returns the canonical protojson short lowercase name.
func (p Platform) String() string {
	switch p {
	case PlatformBrowser:
		return "browser"
	case PlatformDesktop:
		return "desktop"
	case PlatformMobile:
		return "mobile"
	default:
		return "unspecified"
	}
}

// ParsePlatform accepts either the short lowercase ("browser") or the
// fully qualified ("PLATFORM_BROWSER") spelling.
func ParsePlatform(s string) Platform {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "PLATFORM_")) {
	case "browser":
		return PlatformBrowser
	case "desktop":
		return PlatformDesktop
	case "mobile":
		return PlatformMobile
	default:
		return PlatformUnspecified
	}
}

// Role mirrors appunvs.Role.
type Role int32

const (
	RoleUnspecified Role = 0
	RoleProvider    Role = 1
	RoleConnector   Role = 2
	RoleBoth        Role = 3
)

func (r Role) String() string {
	switch r {
	case RoleProvider:
		return "provider"
	case RoleConnector:
		return "connector"
	case RoleBoth:
		return "both"
	default:
		return "unspecified"
	}
}

// ParseRole accepts either short or fully qualified role names.
func ParseRole(s string) Role {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "ROLE_")) {
	case "provider":
		return RoleProvider
	case "connector":
		return RoleConnector
	case "both":
		return RoleBoth
	default:
		return RoleUnspecified
	}
}

// IsProvider returns true if the role participates in provider fanout.
func (r Role) IsProvider() bool { return r == RoleProvider || r == RoleBoth }

// IsConnector returns true if the role participates in connector fanout.
func (r Role) IsConnector() bool { return r == RoleConnector || r == RoleBoth }

// Op mirrors appunvs.Op.
type Op int32

const (
	OpUnspecified Op = 0
	OpUpsert      Op = 1
	OpDelete      Op = 2
	// Schema change ops carry metadata in payload and target the reserved
	// "_schema" table.  The relay broadcasts them to every device owned by
	// the user whose schema changed.
	OpTableCreate  Op = 3
	OpTableDelete  Op = 4
	OpColumnAdd    Op = 5
	OpColumnDelete Op = 6
	// OpQuotaExceeded is sent back to a provider whose message was dropped
	// because the daily messages quota for their plan was hit. The client
	// stays connected; retrying on the next UTC day (or after upgrading)
	// resumes normal flow. Block 4 added this.
	OpQuotaExceeded Op = 7
)

func (o Op) String() string {
	switch o {
	case OpUpsert:
		return "upsert"
	case OpDelete:
		return "delete"
	case OpTableCreate:
		return "table_create"
	case OpTableDelete:
		return "table_delete"
	case OpColumnAdd:
		return "column_add"
	case OpColumnDelete:
		return "column_delete"
	case OpQuotaExceeded:
		return "quota_exceeded"
	default:
		return "unspecified"
	}
}

// ParseOp accepts either short or fully qualified op names.
func ParseOp(s string) Op {
	switch strings.ToLower(strings.TrimPrefix(strings.ToUpper(s), "OP_")) {
	case "upsert":
		return OpUpsert
	case "delete":
		return OpDelete
	case "table_create":
		return OpTableCreate
	case "table_delete":
		return OpTableDelete
	case "column_add":
		return OpColumnAdd
	case "column_delete":
		return OpColumnDelete
	case "quota_exceeded":
		return OpQuotaExceeded
	default:
		return OpUnspecified
	}
}

// Message mirrors appunvs.Message.
// Payload is kept as raw JSON bytes because the relay treats it as opaque.
type Message struct {
	Seq       int64  `json:"seq,omitempty"`
	DeviceID  string `json:"device_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Role      Role   `json:"role,omitempty"`
	Op        Op     `json:"op,omitempty"`
	Table     string `json:"table,omitempty"`
	// Payload is the raw JSON object for google.protobuf.Struct. The relay
	// never introspects it.
	Payload []byte `json:"payload,omitempty"`
	TS      int64  `json:"ts,omitempty"`
}

// RegisterRequest mirrors appunvs.RegisterRequest (HTTP body).
type RegisterRequest struct {
	DeviceID string `json:"device_id"`
	Platform string `json:"platform"`
}

// RegisterResponse mirrors appunvs.RegisterResponse.
type RegisterResponse struct {
	Token  string `json:"token"`
	UserID string `json:"user_id"`
}

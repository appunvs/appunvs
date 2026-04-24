package pb

import (
	"encoding/json"
	"fmt"
)

// ---------------------- Box ----------------------

type boxWire struct {
	BoxID            string `json:"box_id,omitempty"`
	Namespace        string `json:"namespace,omitempty"`
	ProviderDeviceID string `json:"provider_device_id,omitempty"`
	Title            string `json:"title,omitempty"`
	Runtime          string `json:"runtime,omitempty"`
	State            string `json:"state,omitempty"`
	CurrentVersion   string `json:"current_version,omitempty"`
	CreatedAt        int64  `json:"created_at,omitempty"`
	UpdatedAt        int64  `json:"updated_at,omitempty"`
}

// MarshalJSON emits canonical protojson for Box.
func (b Box) MarshalJSON() ([]byte, error) {
	w := boxWire{
		BoxID:            b.BoxID,
		Namespace:        b.Namespace,
		ProviderDeviceID: b.ProviderDeviceID,
		Title:            b.Title,
		CurrentVersion:   b.CurrentVersion,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
	}
	if b.Runtime != RuntimeKindUnspecified {
		w.Runtime = b.Runtime.String()
	}
	if b.State != PublishStateUnspecified {
		w.State = b.State.String()
	}
	return json.Marshal(w)
}

// UnmarshalJSON parses canonical protojson for Box.
func (b *Box) UnmarshalJSON(raw []byte) error {
	var w boxWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("pb: decode Box: %w", err)
	}
	b.BoxID = w.BoxID
	b.Namespace = w.Namespace
	b.ProviderDeviceID = w.ProviderDeviceID
	b.Title = w.Title
	b.Runtime = ParseRuntimeKind(w.Runtime)
	b.State = ParsePublishState(w.State)
	b.CurrentVersion = w.CurrentVersion
	b.CreatedAt = w.CreatedAt
	b.UpdatedAt = w.UpdatedAt
	return nil
}

// ---------------------- BundleRef ----------------------

type bundleRefWire struct {
	BoxID       string `json:"box_id,omitempty"`
	Version     string `json:"version,omitempty"`
	URI         string `json:"uri,omitempty"`
	ContentHash string `json:"content_hash,omitempty"`
	SizeBytes   int64  `json:"size_bytes,omitempty"`
	BuildState  string `json:"build_state,omitempty"`
	BuildLog    string `json:"build_log,omitempty"`
	BuiltAt     int64  `json:"built_at,omitempty"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
}

// MarshalJSON emits canonical protojson for BundleRef.
func (b BundleRef) MarshalJSON() ([]byte, error) {
	w := bundleRefWire{
		BoxID:       b.BoxID,
		Version:     b.Version,
		URI:         b.URI,
		ContentHash: b.ContentHash,
		SizeBytes:   b.SizeBytes,
		BuildLog:    b.BuildLog,
		BuiltAt:     b.BuiltAt,
		ExpiresAt:   b.ExpiresAt,
	}
	if b.BuildState != BuildStateUnspecified {
		w.BuildState = b.BuildState.String()
	}
	return json.Marshal(w)
}

// UnmarshalJSON parses canonical protojson for BundleRef.
func (b *BundleRef) UnmarshalJSON(raw []byte) error {
	var w bundleRefWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("pb: decode BundleRef: %w", err)
	}
	b.BoxID = w.BoxID
	b.Version = w.Version
	b.URI = w.URI
	b.ContentHash = w.ContentHash
	b.SizeBytes = w.SizeBytes
	b.BuildState = ParseBuildState(w.BuildState)
	b.BuildLog = w.BuildLog
	b.BuiltAt = w.BuiltAt
	b.ExpiresAt = w.ExpiresAt
	return nil
}

// ---------------------- BoxCreateRequest ----------------------

type boxCreateRequestWire struct {
	Title   string `json:"title,omitempty"`
	Runtime string `json:"runtime,omitempty"`
}

func (b BoxCreateRequest) MarshalJSON() ([]byte, error) {
	w := boxCreateRequestWire{Title: b.Title}
	if b.Runtime != RuntimeKindUnspecified {
		w.Runtime = b.Runtime.String()
	}
	return json.Marshal(w)
}

func (b *BoxCreateRequest) UnmarshalJSON(raw []byte) error {
	var w boxCreateRequestWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("pb: decode BoxCreateRequest: %w", err)
	}
	b.Title = w.Title
	b.Runtime = ParseRuntimeKind(w.Runtime)
	return nil
}

// ---------------------- PairClaimRequest ----------------------

type pairClaimRequestWire struct {
	DeviceID string `json:"device_id,omitempty"`
	Platform string `json:"platform,omitempty"`
}

func (p PairClaimRequest) MarshalJSON() ([]byte, error) {
	w := pairClaimRequestWire{DeviceID: p.DeviceID}
	if p.Platform != PlatformUnspecified {
		w.Platform = p.Platform.String()
	}
	return json.Marshal(w)
}

func (p *PairClaimRequest) UnmarshalJSON(raw []byte) error {
	var w pairClaimRequestWire
	if err := json.Unmarshal(raw, &w); err != nil {
		return fmt.Errorf("pb: decode PairClaimRequest: %w", err)
	}
	p.DeviceID = w.DeviceID
	p.Platform = ParsePlatform(w.Platform)
	return nil
}

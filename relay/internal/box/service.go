// Package box wires the persistence (store.Boxes), build (sandbox.Builder),
// and storage (artifact.Store) collaborators into the use-cases the HTTP
// handlers and the AI agent's publish tool both rely on.
//
// Keep this layer policy-only: no transport, no auth, no JSON.  The handler
// package upcasts user input to the typed methods here, and downcasts the
// returned domain types back to wire shapes via the pb package.
package box

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// SignedURLTTL is how long a freshly minted bundle URL stays valid for the
// runner to fetch.  Connectors should re-request via /pair or /box/:id
// after expiry rather than caching the URL.
const SignedURLTTL = 30 * time.Minute

// Service exposes the high-level box use-cases.
type Service struct {
	Boxes     *store.Boxes
	Sandbox   sandbox.Builder
	Artifact  artifact.Store
	Workspace *workspace.Store
	// Events is the per-namespace fanout BuildAndPublish notifies on
	// success.  May be nil — tests that don't care about the
	// notification surface leave it unset; production wires a
	// NewEvents() in cmd/server/main.go.
	Events *Events
}

// New returns a Service wired with all four collaborators.  `ws` may be
// nil for tests that want to pass source directly through `sandbox.Source`
// without going through the git-backed workspace layer.  `events` may
// also be nil — see Service.Events.
func New(boxes *store.Boxes, builder sandbox.Builder, store artifact.Store, ws *workspace.Store, events *Events) *Service {
	return &Service{Boxes: boxes, Sandbox: builder, Artifact: store, Workspace: ws, Events: events}
}

// Create creates a new draft box owned by providerDeviceID inside namespace.
// Returns the persisted Box (with an assigned id and timestamps).
func (s *Service) Create(ctx context.Context, namespace, providerDeviceID, title string, runtime pb.RuntimeKind) (store.Box, error) {
	if runtime == pb.RuntimeKindUnspecified {
		runtime = pb.RuntimeKindRNBundle
	}
	id, err := newID("box_")
	if err != nil {
		return store.Box{}, err
	}
	now := time.Now().UnixMilli()
	box := store.Box{
		ID:               id,
		Namespace:        namespace,
		ProviderDeviceID: providerDeviceID,
		Title:            title,
		Runtime:          runtime,
		State:            pb.PublishStateDraft,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := s.Boxes.Create(ctx, box); err != nil {
		return store.Box{}, err
	}
	return box, nil
}

// Get returns a single box plus, when present, its current bundle.
func (s *Service) Get(ctx context.Context, namespace, id string) (store.Box, *store.Bundle, error) {
	b, err := s.Boxes.Get(ctx, namespace, id)
	if err != nil {
		return store.Box{}, nil, err
	}
	if b.CurrentVersion == "" {
		return b, nil, nil
	}
	bundle, err := s.Boxes.GetBundle(ctx, b.ID, b.CurrentVersion)
	if err != nil {
		// A missing current bundle is consistent state during build / GC
		// races.  Treat as "no current" rather than an error.
		if errors.Is(err, store.ErrBoxNotFound) {
			return b, nil, nil
		}
		return store.Box{}, nil, err
	}
	return b, &bundle, nil
}

// List returns every box owned by namespace, most recent first.
func (s *Service) List(ctx context.Context, namespace string) ([]store.Box, error) {
	return s.Boxes.List(ctx, namespace)
}

// BuildAndPublish runs source through the configured Builder, uploads the
// resulting bundle bytes to Artifact, records a Bundle row, sets the box's
// current_version, and flips the PublishState to PUBLISHED.  Returns the
// freshly persisted Bundle.
//
// When a Workspace is configured and `src.Files` is non-empty, the files
// are committed to the box's git repo first; the build then runs off the
// workspace snapshot.  When `src.Files` is empty, the snapshot alone is
// used — this is the "re-publish current HEAD" shape.
//
// This is the canonical "publish" path called both by the HTTP handler
// (POST /box/:id/publish) and by the AI agent's publish_box tool.
func (s *Service) BuildAndPublish(ctx context.Context, namespace string, src sandbox.Source) (store.Bundle, error) {
	box, err := s.Boxes.Get(ctx, namespace, src.BoxID)
	if err != nil {
		return store.Bundle{}, err
	}
	if box.State == pb.PublishStateArchived {
		return store.Bundle{}, fmt.Errorf("box %s is archived", box.ID)
	}
	if src.Version == "" {
		src.Version, err = newVersion()
		if err != nil {
			return store.Bundle{}, err
		}
	}

	// Persist incoming files into the git workspace, then replace src.Files
	// with the snapshot at HEAD so the builder sees exactly what's versioned.
	if s.Workspace != nil {
		repo, err := s.Workspace.Open(box.ID)
		if err != nil {
			return store.Bundle{}, fmt.Errorf("workspace open: %w", err)
		}
		if len(src.Files) > 0 {
			ops := make([]workspace.WriteOp, 0, len(src.Files))
			for path, content := range src.Files {
				ops = append(ops, workspace.WriteOp{Path: path, Content: content})
			}
			if _, err := repo.Commit(ctx, ops,
				fmt.Sprintf("publish %s", src.Version),
				"appunvs", "agent@appunvs"); err != nil {
				return store.Bundle{}, fmt.Errorf("workspace commit: %w", err)
			}
		}
		snap, err := repo.Snapshot(ctx)
		if err != nil {
			return store.Bundle{}, fmt.Errorf("workspace snapshot: %w", err)
		}
		src.Files = snap
	}

	// Mark the bundle as building before invoking the (potentially slow)
	// builder so concurrent reads see a non-final state.
	now := time.Now().UnixMilli()
	pending := store.Bundle{
		BoxID:      box.ID,
		Version:    src.Version,
		BuildState: pb.BuildStateRunning,
		BuiltAt:    now,
	}
	if err := s.Boxes.PutBundle(ctx, pending); err != nil {
		return store.Bundle{}, err
	}

	res, buildErr := s.Sandbox.Build(ctx, src)
	if buildErr != nil {
		failed := pending
		failed.BuildState = pb.BuildStateFailed
		failed.BuildLog = truncateLog(buildErr.Error())
		failed.BuiltAt = time.Now().UnixMilli()
		_ = s.Boxes.PutBundle(ctx, failed)
		return failed, buildErr
	}

	obj, err := s.Artifact.Put(ctx, bytes.NewReader(res.Bytes))
	if err != nil {
		failed := pending
		failed.BuildState = pb.BuildStateFailed
		failed.BuildLog = truncateLog("artifact upload: " + err.Error())
		failed.BuiltAt = time.Now().UnixMilli()
		_ = s.Boxes.PutBundle(ctx, failed)
		return failed, err
	}
	url, expires, err := s.Artifact.SignURL(ctx, obj.Hash, SignedURLTTL)
	if err != nil {
		return store.Bundle{}, err
	}

	bundle := store.Bundle{
		BoxID:       box.ID,
		Version:     src.Version,
		URI:         url,
		ContentHash: obj.Hash,
		SizeBytes:   obj.SizeBytes,
		BuildState:  pb.BuildStateSucceeded,
		BuildLog:    truncateLog(res.Log),
		BuiltAt:     time.Now().UnixMilli(),
		ExpiresAt:   expires.UnixMilli(),
	}
	if err := s.Boxes.PutBundle(ctx, bundle); err != nil {
		return store.Bundle{}, err
	}
	if err := s.Boxes.SetCurrentVersion(ctx, namespace, box.ID, src.Version); err != nil {
		return store.Bundle{}, err
	}
	if box.State != pb.PublishStatePublished {
		if err := s.Boxes.SetState(ctx, namespace, box.ID, pb.PublishStatePublished); err != nil {
			return store.Bundle{}, err
		}
	}
	// Notify subscribed hosts that a new bundle is available.  Done
	// AFTER all persistence so a host that immediately calls
	// boxes.list/get sees the new current_version.
	if s.Events != nil {
		s.Events.Publish(namespace, Event{
			Type:        EventBundleReady,
			BoxID:       bundle.BoxID,
			Version:     bundle.Version,
			URI:         bundle.URI,
			ContentHash: bundle.ContentHash,
			SizeBytes:   bundle.SizeBytes,
		})
	}
	return bundle, nil
}

// Archive flips a box to ARCHIVED so no new pairings or publishes are
// accepted.  Existing bundles remain queryable for forensic / billing
// purposes.
func (s *Service) Archive(ctx context.Context, namespace, id string) error {
	return s.Boxes.SetState(ctx, namespace, id, pb.PublishStateArchived)
}

// ToPB packs a store.Box (and optional store.Bundle) into the wire types.
func ToPB(b store.Box, current *store.Bundle) pb.BoxResponse {
	out := pb.BoxResponse{Box: pb.Box{
		BoxID:            b.ID,
		Namespace:        b.Namespace,
		ProviderDeviceID: b.ProviderDeviceID,
		Title:            b.Title,
		Runtime:          b.Runtime,
		State:            b.State,
		CurrentVersion:   b.CurrentVersion,
		CreatedAt:        b.CreatedAt,
		UpdatedAt:        b.UpdatedAt,
	}}
	if current != nil {
		out.Current = &pb.BundleRef{
			BoxID:       current.BoxID,
			Version:     current.Version,
			URI:         current.URI,
			ContentHash: current.ContentHash,
			SizeBytes:   current.SizeBytes,
			BuildState:  current.BuildState,
			BuildLog:    current.BuildLog,
			BuiltAt:     current.BuiltAt,
			ExpiresAt:   current.ExpiresAt,
		}
	}
	return out
}

// BundleToPB converts a store.Bundle to its wire shape.
func BundleToPB(b store.Bundle) pb.BundleRef {
	return pb.BundleRef{
		BoxID:       b.BoxID,
		Version:     b.Version,
		URI:         b.URI,
		ContentHash: b.ContentHash,
		SizeBytes:   b.SizeBytes,
		BuildState:  b.BuildState,
		BuildLog:    b.BuildLog,
		BuiltAt:     b.BuiltAt,
		ExpiresAt:   b.ExpiresAt,
	}
}

func newID(prefix string) (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}

func newVersion() (string, error) {
	var b [6]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d-%s", time.Now().Unix(), hex.EncodeToString(b[:])), nil
}

func truncateLog(s string) string {
	const max = 4096
	if len(s) <= max {
		return s
	}
	return "...[truncated]\n" + s[len(s)-max:]
}

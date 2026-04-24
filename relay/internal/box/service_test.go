package box_test

import (
	"context"
	"strings"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/workspace"
)

// TestBuildAndPublishRoundTrip drives the full publish path against the
// LocalStub builder + LocalFS artifact store + SQLite in :memory:.  This is
// the regression guard for the v1 wiring; once a real Metro builder lands,
// the same scenario must still pass with the pluggable Builder swapped in.
func TestBuildAndPublishRoundTrip(t *testing.T) {
	ctx := context.Background()

	st, err := store.Open(ctx, t.TempDir()+"/relay.db")
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	defer func() { _ = st.Close() }()

	// users(namespace) row is required by FK constraints downstream of the
	// schema (app_tables references users) — boxes table itself doesn't
	// FK to users so we can use any namespace string here, but seeding one
	// keeps the test honest about the multi-tenant model.
	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_test", "t@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	art, err := artifact.NewLocalFS(t.TempDir(), "http://localhost:8080/_artifacts")
	if err != nil {
		t.Fatalf("artifact: %v", err)
	}
	svc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, nil)

	created, err := svc.Create(ctx, "u_test", "dev_a", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.State != pb.PublishStateDraft {
		t.Fatalf("new box state = %s, want draft", created.State)
	}

	bundle, err := svc.BuildAndPublish(ctx, "u_test", sandbox.Source{
		BoxID:      created.ID,
		EntryPoint: "index.tsx",
		Files: map[string][]byte{
			"index.tsx": []byte("export default function App(){return null}\n"),
		},
	})
	if err != nil {
		t.Fatalf("publish: %v", err)
	}
	if bundle.BuildState != pb.BuildStateSucceeded {
		t.Fatalf("build_state = %s, want succeeded; log=%s", bundle.BuildState, bundle.BuildLog)
	}
	if !strings.HasPrefix(bundle.ContentHash, "sha256:") {
		t.Fatalf("content_hash %q must start with sha256:", bundle.ContentHash)
	}
	if bundle.URI == "" {
		t.Fatalf("missing URI")
	}

	got, current, err := svc.Get(ctx, "u_test", created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != pb.PublishStatePublished {
		t.Fatalf("state after publish = %s, want published", got.State)
	}
	if got.CurrentVersion != bundle.Version {
		t.Fatalf("current_version mismatch: %s vs %s", got.CurrentVersion, bundle.Version)
	}
	if current == nil || current.URI != bundle.URI {
		t.Fatalf("current bundle not echoed back")
	}

	// Cross-namespace probe must NOT find the box.
	if _, _, err := svc.Get(ctx, "u_other", created.ID); err == nil {
		t.Fatalf("cross-namespace lookup unexpectedly succeeded")
	}
}

// TestPublishWithWorkspace verifies that when a git workspace is wired in,
// publish first commits the supplied files, then rebuilds from the
// workspace snapshot.  The second publish (passing no files) must still
// produce a bundle from the previously committed HEAD.
func TestPublishWithWorkspace(t *testing.T) {
	ctx := context.Background()

	st, err := store.Open(ctx, t.TempDir()+"/relay.db")
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	defer func() { _ = st.Close() }()

	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_ws", "ws@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed: %v", err)
	}

	art, err := artifact.NewLocalFS(t.TempDir(), "http://localhost:8080/_artifacts")
	if err != nil {
		t.Fatalf("artifact: %v", err)
	}
	ws, err := workspace.NewStore(workspace.Config{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	svc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, ws)

	b, err := svc.Create(ctx, "u_ws", "dev_ws", "demo-ws", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// First publish: supply source files; workspace commits them.
	bundle1, err := svc.BuildAndPublish(ctx, "u_ws", sandbox.Source{
		BoxID:      b.ID,
		EntryPoint: "index.tsx",
		Files: map[string][]byte{
			"index.tsx": []byte("export const v = 1\n"),
			"util.ts":   []byte("export const u = 'a'\n"),
		},
	})
	if err != nil {
		t.Fatalf("publish 1: %v", err)
	}
	if bundle1.BuildState != pb.BuildStateSucceeded {
		t.Fatalf("build_state 1 = %s", bundle1.BuildState)
	}

	// Second publish: no files — builder should still see the previously
	// committed snapshot via workspace.
	bundle2, err := svc.BuildAndPublish(ctx, "u_ws", sandbox.Source{
		BoxID:      b.ID,
		EntryPoint: "index.tsx",
	})
	if err != nil {
		t.Fatalf("publish 2: %v", err)
	}
	if bundle2.BuildState != pb.BuildStateSucceeded {
		t.Fatalf("build_state 2 = %s; log=%s", bundle2.BuildState, bundle2.BuildLog)
	}
	if bundle2.Version == bundle1.Version {
		t.Fatalf("expected a fresh version on re-publish")
	}
	if bundle2.SizeBytes == 0 {
		t.Fatalf("bundle 2 from workspace snapshot should not be empty")
	}
}

package box_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/appunvs/appunvs/relay/internal/artifact"
	"github.com/appunvs/appunvs/relay/internal/box"
	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/sandbox"
	"github.com/appunvs/appunvs/relay/internal/store"
	"github.com/appunvs/appunvs/relay/internal/usage"
)

func itoa(n int) string       { return strconv.Itoa(n) }
func nowMillis() int64        { return time.Now().UnixMilli() }

// newGatedService spins up a box.Service with a real SQLite store
// behind it and a quota gate wired against the requested plan.  The
// rig is the same shape as TestBuildAndPublishRoundTrip; the new
// twist is the Quota / PlanFor injection.
func newGatedService(t *testing.T, plan usage.Plan) (*box.Service, *store.Store) {
	t.Helper()
	ctx := context.Background()

	st, err := store.Open(ctx, t.TempDir()+"/relay.db")
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if _, err := st.DB.ExecContext(ctx,
		`INSERT INTO users(id, email, password_hash, created_at) VALUES(?,?,?,?)`,
		"u_test", "t@example.com", "x", int64(1)); err != nil {
		t.Fatalf("seed user: %v", err)
	}

	art, err := artifact.NewLocalFS(t.TempDir(), "http://localhost:8080/_artifacts")
	if err != nil {
		t.Fatalf("artifact: %v", err)
	}
	svc := box.New(st.Boxes(), sandbox.NewLocalStub(), art, nil, nil)
	svc.Quota = usage.NewQuota(st.Turns(), st.Boxes(), st.Boxes())
	svc.PlanFor = func(_ string) usage.Plan { return plan }
	return svc, st
}

// TestCreate_StorageGate_AllowsBelowCap — fresh user can create up
// to ActiveBoxes boxes without tripping anything.
func TestCreate_StorageGate_AllowsBelowCap(t *testing.T) {
	svc, _ := newGatedService(t, usage.Plans[usage.PlanFree])
	ctx := context.Background()

	// Free has ActiveBoxes=1; first create succeeds.
	if _, err := svc.Create(ctx, "u_test", "dev_a", "first", pb.RuntimeKindRNBundle); err != nil {
		t.Fatalf("first create: %v", err)
	}
}

// TestCreate_StorageGate_DeniesAtCap — Free user with 1 active box
// (the cap) gets ErrPlanExhausted on the second create.
func TestCreate_StorageGate_DeniesAtCap(t *testing.T) {
	svc, _ := newGatedService(t, usage.Plans[usage.PlanFree])
	ctx := context.Background()

	// First box succeeds.
	if _, err := svc.Create(ctx, "u_test", "dev_a", "first", pb.RuntimeKindRNBundle); err != nil {
		t.Fatalf("first create: %v", err)
	}
	// Second create hits the storage gate.
	_, err := svc.Create(ctx, "u_test", "dev_a", "second", pb.RuntimeKindRNBundle)
	if err == nil {
		t.Fatal("expected ErrPlanExhausted on second create at cap")
	}
	pe, ok := box.AsPlanExhausted(err)
	if !ok {
		t.Fatalf("err = %v, want ErrPlanExhausted", err)
	}
	if pe.Window != string(usage.WindowStorage) {
		t.Errorf("Window = %q, want %q", pe.Window, usage.WindowStorage)
	}
	if pe.Plan != "free" {
		t.Errorf("Plan = %q, want free", pe.Plan)
	}
	// Storage cap is point-in-time — RetryAfter is meaningless (user
	// has to free a slot, not wait), so 0 is the right value.
	if pe.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, want 0 for storage cap", pe.RetryAfter)
	}
}

// TestCreate_StorageGate_MaxPlusUnlimited — Max+ has -1 ActiveBoxes,
// so create never trips even at high counts.
func TestCreate_StorageGate_MaxPlusUnlimited(t *testing.T) {
	svc, _ := newGatedService(t, usage.Plans[usage.PlanMaxPlus])
	ctx := context.Background()
	for i := 0; i < 10; i++ {
		if _, err := svc.Create(ctx, "u_test", "dev_a", "many", pb.RuntimeKindRNBundle); err != nil {
			t.Fatalf("create #%d: %v", i, err)
		}
	}
}

// TestCreate_NoQuota_GateDisabled — leaving Quota nil keeps the
// pre-Phase-C behavior so dev / CI / dogfood paths don't break.
func TestCreate_NoQuota_GateDisabled(t *testing.T) {
	svc, _ := newGatedService(t, usage.Plans[usage.PlanFree])
	svc.Quota = nil // disable the gate after the rig built it
	ctx := context.Background()

	for i := 0; i < 5; i++ { // way past Free's ActiveBoxes=1
		if _, err := svc.Create(ctx, "u_test", "dev_a", "ungated", pb.RuntimeKindRNBundle); err != nil {
			t.Fatalf("create #%d: %v", i, err)
		}
	}
}

// TestBuildAndPublish_SandboxGate_DeniesAtCap — Free user who's
// already done SandboxRunsPerMonth builds gets ErrPlanExhausted.
//
// The fixture seeds bundle rows directly (not via BuildAndPublish)
// to skip the full docker round-trip — the gate's job is to count
// bundle rows since month start, so seeding bundles is the same
// state from its perspective.
func TestBuildAndPublish_SandboxGate_DeniesAtCap(t *testing.T) {
	svc, st := newGatedService(t, usage.Plans[usage.PlanFree])
	ctx := context.Background()

	// Create a box and seed Free.SandboxRunsPerMonth bundle rows
	// against it to exhaust the sandbox quota before we try to
	// publish again.
	created, err := svc.Create(ctx, "u_test", "dev_a", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	plan := usage.Plans[usage.PlanFree]
	for i := 0; i < plan.SandboxRunsPerMonth; i++ {
		if err := st.Boxes().PutBundle(ctx, store.Bundle{
			BoxID:      created.ID,
			Version:    "seed-" + itoa(i),
			BuildState: pb.BuildStateSucceeded,
			BuiltAt:    1, // any non-zero past timestamp; must be >= month_start though
		}); err != nil {
			t.Fatalf("seed bundle %d: %v", i, err)
		}
	}
	// Re-seed with a recent timestamp so it falls within the
	// rolling-month window the gate queries.
	for i := 0; i < plan.SandboxRunsPerMonth; i++ {
		if err := st.Boxes().PutBundle(ctx, store.Bundle{
			BoxID:      created.ID,
			Version:    "seed-" + itoa(i),
			BuildState: pb.BuildStateSucceeded,
			BuiltAt:    nowMillis(),
		}); err != nil {
			t.Fatalf("re-seed bundle %d: %v", i, err)
		}
	}

	_, err = svc.BuildAndPublish(ctx, "u_test", sandbox.Source{
		BoxID:      created.ID,
		EntryPoint: "index.tsx",
	})
	if err == nil {
		t.Fatal("expected ErrPlanExhausted on (cap+1)th publish")
	}
	pe, ok := box.AsPlanExhausted(err)
	if !ok {
		t.Fatalf("err = %v, want ErrPlanExhausted", err)
	}
	if pe.Window != string(usage.WindowSandbox) {
		t.Errorf("Window = %q, want %q", pe.Window, usage.WindowSandbox)
	}
	if pe.RetryAfter <= 0 {
		t.Errorf("RetryAfter = %v, want > 0 (next-month reset)", pe.RetryAfter)
	}
}

// TestBuildAndPublish_SandboxGate_AllowsBelowCap — Free user under
// SandboxRunsPerMonth gets to build normally.
func TestBuildAndPublish_SandboxGate_AllowsBelowCap(t *testing.T) {
	svc, _ := newGatedService(t, usage.Plans[usage.PlanFree])
	ctx := context.Background()

	created, err := svc.Create(ctx, "u_test", "dev_a", "demo", pb.RuntimeKindRNBundle)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	// First publish — well under Free's SandboxRunsPerMonth=5.
	if _, err := svc.BuildAndPublish(ctx, "u_test", sandbox.Source{
		BoxID:      created.ID,
		EntryPoint: "index.tsx",
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}
}

// TestAsPlanExhausted_UnwrapsThroughErrorf — callers that wrap the
// error (eg. fmt.Errorf("foo: %w", err)) still get the structured
// extraction.  Belt + suspenders for the AS chain.
func TestAsPlanExhausted_UnwrapsThroughErrorf(t *testing.T) {
	wrapped := errors.Join(box.ErrPlanExhausted{Window: "sandbox", Plan: "pro"})
	pe, ok := box.AsPlanExhausted(wrapped)
	if !ok {
		t.Fatalf("AsPlanExhausted didn't unwrap: %v", wrapped)
	}
	if pe.Window != "sandbox" || pe.Plan != "pro" {
		t.Errorf("got %+v, want Window=sandbox Plan=pro", pe)
	}
}

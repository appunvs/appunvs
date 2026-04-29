// Integration test for Turns.CountByNamespace — exercises the SQL JOIN
// against app_boxes through a real SQLite store, complementing the
// stub-based tests in internal/usage/quota_test.go (which guard the
// math but can't catch a typo in the query).
package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/appunvs/appunvs/relay/internal/pb"
	"github.com/appunvs/appunvs/relay/internal/store"
)

// seedTurn writes one ai_turns row attached to box.  CreatedAt is
// caller-supplied so the test controls the windowing.
func seedTurn(t *testing.T, st *store.Store, boxID, id string, createdAt time.Time) {
	t.Helper()
	if err := st.Turns().Insert(context.Background(), store.Turn{
		ID:         id,
		BoxID:      boxID,
		UserText:   "hi",
		Messages:   "[]",
		CreatedAt:  createdAt.UnixMilli(),
		StopReason: "end_turn",
	}); err != nil {
		t.Fatalf("seed turn %s: %v", id, err)
	}
}

func TestTurnsCountByNamespace_FiltersByTimeAndOwner(t *testing.T) {
	st := newStore(t)
	ctx := context.Background()

	alice := mkUser(t, st, "alice@example.com")
	bob := mkUser(t, st, "bob@example.com")

	// Each user owns one box.  ai_turns rows under each box should
	// only count toward that user's namespace.
	if err := st.Boxes().Create(ctx, store.Box{
		ID: "box-a", Namespace: alice, ProviderDeviceID: "dev-a",
		Title: "alice", Runtime: pb.RuntimeKindRNBundle,
	}); err != nil {
		t.Fatalf("create box-a: %v", err)
	}
	if err := st.Boxes().Create(ctx, store.Box{
		ID: "box-b", Namespace: bob, ProviderDeviceID: "dev-b",
		Title: "bob", Runtime: pb.RuntimeKindRNBundle,
	}); err != nil {
		t.Fatalf("create box-b: %v", err)
	}

	now := time.Now().UTC()
	// Alice: 3 recent turns + 1 stale turn (10 days old).
	seedTurn(t, st, "box-a", "a1", now.Add(-1*time.Hour))
	seedTurn(t, st, "box-a", "a2", now.Add(-2*time.Hour))
	seedTurn(t, st, "box-a", "a3", now.Add(-4*24*time.Hour))
	seedTurn(t, st, "box-a", "a4-stale", now.Add(-10*24*time.Hour))
	// Bob: 2 recent turns.
	seedTurn(t, st, "box-b", "b1", now.Add(-3*time.Hour))
	seedTurn(t, st, "box-b", "b2", now.Add(-5*time.Hour))

	turns := st.Turns()
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()

	gotAlice, err := turns.CountByNamespace(ctx, alice, since5d)
	if err != nil {
		t.Fatalf("count alice: %v", err)
	}
	if gotAlice != 3 {
		t.Errorf("alice 5d count = %d, want 3 (1h + 2h + 4d; 10d stale excluded)", gotAlice)
	}

	gotBob, err := turns.CountByNamespace(ctx, bob, since5d)
	if err != nil {
		t.Fatalf("count bob: %v", err)
	}
	if gotBob != 2 {
		t.Errorf("bob 5d count = %d, want 2", gotBob)
	}

	// Wider window should pick up the stale turn for alice.
	since30d := now.Add(-30 * 24 * time.Hour).UnixMilli()
	gotAliceMonth, err := turns.CountByNamespace(ctx, alice, since30d)
	if err != nil {
		t.Fatalf("count alice month: %v", err)
	}
	if gotAliceMonth != 4 {
		t.Errorf("alice 30d count = %d, want 4", gotAliceMonth)
	}
}

func TestTurnsCountByNamespace_ReturnsZeroForEmptyNamespace(t *testing.T) {
	st := newStore(t)
	got, err := st.Turns().CountByNamespace(context.Background(), "no-such-user", 0)
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if got != 0 {
		t.Errorf("got %d, want 0 for unknown namespace", got)
	}
}

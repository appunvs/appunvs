package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/appunvs/appunvs/relay/internal/store"
)

// newStore opens a fresh SQLite file in a temp dir, runs migrations, and
// returns the hydrated Store. Billing migration seeds the three canonical
// plans as part of Open().
func newStore(t *testing.T) *store.Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "billing_test.db")
	st, err := store.Open(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("store open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// mkUser inserts a minimal user row so subscriptions' FK is satisfied.
// Callers pass whatever email/id they like.
func mkUser(t *testing.T, st *store.Store, email string) string {
	t.Helper()
	u, err := st.Users().Create(context.Background(), email, "hunter22")
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	return u.ID
}

// Seeded plans must match the CanonicalPlans slice exactly.
func TestPlansSeeded(t *testing.T) {
	st := newStore(t)
	plans, err := st.Plans().List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(plans) != len(store.CanonicalPlans) {
		t.Fatalf("seeded plans=%d, want %d", len(plans), len(store.CanonicalPlans))
	}
	for _, want := range store.CanonicalPlans {
		got, err := st.Plans().Get(context.Background(), want.ID)
		if err != nil {
			t.Fatalf("get %s: %v", want.ID, err)
		}
		if got == nil {
			t.Fatalf("plan %s missing from DB", want.ID)
		}
		if *got != want {
			t.Fatalf("plan %s mismatch: got=%+v want=%+v", want.ID, *got, want)
		}
	}
	// Unknown plan returns nil, not an error.
	if got, err := st.Plans().Get(context.Background(), "mystery"); err != nil || got != nil {
		t.Fatalf("get unknown: got=%+v err=%v", got, err)
	}
}

// EnsureFree is idempotent and does NOT clobber an upgraded plan.
func TestEnsureFreeIdempotent(t *testing.T) {
	st := newStore(t)
	uid := mkUser(t, st, "ensure@x.com")

	if err := st.Subscriptions().EnsureFree(context.Background(), uid); err != nil {
		t.Fatalf("ensure #1: %v", err)
	}
	sub, err := st.Subscriptions().Get(context.Background(), uid)
	if err != nil || sub == nil {
		t.Fatalf("get after ensure: sub=%+v err=%v", sub, err)
	}
	if sub.PlanID != "free" {
		t.Fatalf("plan_id=%s, want free", sub.PlanID)
	}

	// Upgrade out-of-band, then re-run EnsureFree — must NOT downgrade.
	if err := st.Subscriptions().SetPlan(context.Background(), uid, "pro", "active",
		time.Now().UTC(), time.Now().UTC().AddDate(0, 1, 0), "cus_x", "sub_x"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	if err := st.Subscriptions().EnsureFree(context.Background(), uid); err != nil {
		t.Fatalf("ensure #2: %v", err)
	}
	sub2, _ := st.Subscriptions().Get(context.Background(), uid)
	if sub2.PlanID != "pro" {
		t.Fatalf("EnsureFree clobbered pro -> %s", sub2.PlanID)
	}
}

// Usage increment + query across days.
func TestUsageIncrement(t *testing.T) {
	st := newStore(t)
	uid := mkUser(t, st, "usage@x.com")

	today := store.DayKey(time.Now())
	yesterday := store.DayKey(time.Now().Add(-24 * time.Hour))

	// Increment today twice, yesterday once.
	if err := st.Usage().Increment(context.Background(), uid, today, 1); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := st.Usage().Increment(context.Background(), uid, today, 3); err != nil {
		t.Fatalf("increment: %v", err)
	}
	if err := st.Usage().Increment(context.Background(), uid, yesterday, 5); err != nil {
		t.Fatalf("increment yesterday: %v", err)
	}

	if n, _ := st.Usage().MessagesForDay(context.Background(), uid, today); n != 4 {
		t.Fatalf("today = %d, want 4", n)
	}
	if n, _ := st.Usage().MessagesForDay(context.Background(), uid, yesterday); n != 5 {
		t.Fatalf("yesterday = %d, want 5", n)
	}
	sum, err := st.Usage().MessagesForPeriod(context.Background(), uid, yesterday, today)
	if err != nil {
		t.Fatalf("period: %v", err)
	}
	if sum != 9 {
		t.Fatalf("period sum = %d, want 9", sum)
	}
}

// Stripe-customer lookup finds a subscription by customer id.
func TestFindByStripeCustomer(t *testing.T) {
	st := newStore(t)
	uid := mkUser(t, st, "stripe@x.com")
	if err := st.Subscriptions().SetPlan(context.Background(), uid, "pro", "active",
		time.Now().UTC(), time.Now().UTC().AddDate(0, 1, 0), "cus_abc", "sub_abc"); err != nil {
		t.Fatalf("setplan: %v", err)
	}
	got, err := st.Subscriptions().FindByStripeCustomer(context.Background(), "cus_abc")
	if err != nil || got == nil {
		t.Fatalf("lookup: got=%+v err=%v", got, err)
	}
	if got.UserID != uid {
		t.Fatalf("user_id=%s, want %s", got.UserID, uid)
	}
	// Empty string -> nil, no error.
	g2, err := st.Subscriptions().FindByStripeCustomer(context.Background(), "")
	if err != nil || g2 != nil {
		t.Fatalf("empty: got=%+v err=%v", g2, err)
	}
}

// FindCanonicalPlan lets code hand out plan metadata without a DB trip.
func TestFindCanonicalPlan(t *testing.T) {
	if p := store.FindCanonicalPlan("free"); p == nil || p.ID != "free" {
		t.Fatalf("free: %+v", p)
	}
	if p := store.FindCanonicalPlan("doesnotexist"); p != nil {
		t.Fatalf("unknown returned non-nil: %+v", p)
	}
}

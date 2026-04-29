package usage

import (
	"context"
	"testing"
	"time"
)

// stubCounter is a deterministic turnCounter.  Each call records its
// `since` argument so tests can assert on the windowing math without
// needing SQLite.
type stubCounter struct {
	// Map of namespace → sinceMillis → row count.  The test seeds
	// expected windows; any miss returns 0 (an unseeded window
	// signals an assertion bug, not a real-world condition).
	counts map[string]map[int64]int64
	// Calls records every (namespace, since) pair the gate asked for.
	calls []call
}

type call struct {
	ns    string
	since int64
}

func (s *stubCounter) CountByNamespace(_ context.Context, namespace string, since int64) (int64, error) {
	s.calls = append(s.calls, call{ns: namespace, since: since})
	if m, ok := s.counts[namespace]; ok {
		return m[since], nil
	}
	return 0, nil
}

// fixedNow returns a clock-injection helper pinned to the same instant
// every Quota operation; lets us pre-compute window boundaries.
func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func TestPlansHaveAllRequiredTiers(t *testing.T) {
	required := []PlanID{PlanFree, PlanPro, PlanMax, PlanMaxPlus, PlanBYOK}
	for _, id := range required {
		p, ok := Plans[id]
		if !ok {
			t.Errorf("plan %q missing from registry", id)
			continue
		}
		if p.ID != id {
			t.Errorf("plan %q ID field disagrees: got %q", id, p.ID)
		}
		if p.Label == "" {
			t.Errorf("plan %q missing Label", id)
		}
	}
}

func TestPlanFor_FallsBackToFreeOnUnknown(t *testing.T) {
	// Stale users.plan pointing at a renamed tier should not break the
	// gate; PlanFor returns Free as the conservative default.
	p := PlanFor(PlanID("tier-that-doesnt-exist"))
	if p.ID != PlanFree {
		t.Errorf("PlanFor unknown should return Free, got %q", p.ID)
	}
}

func TestPlanFor_KnownReturnsItself(t *testing.T) {
	for id := range Plans {
		if PlanFor(id).ID != id {
			t.Errorf("PlanFor(%q).ID = %q, want self", id, PlanFor(id).ID)
		}
	}
}

func TestPro_TurnsPer5dCapEnforced(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	stub := &stubCounter{counts: map[string]map[int64]int64{
		"alice": {
			since5d:    int64(plan.TurnsPer5d), // exactly at cap
			monthStart: int64(plan.TurnsPer5d),
		},
	}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	dec, err := q.Check(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny when 5d count == cap")
	}
	if dec.LimitedBy != Window5d {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, Window5d)
	}
	if dec.RetryAfter == 0 {
		t.Error("RetryAfter should be set when denied")
	}
}

func TestPro_BelowCapAllowed(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	stub := &stubCounter{counts: map[string]map[int64]int64{
		"alice": {
			since5d:    int64(plan.TurnsPer5d) - 1,
			monthStart: int64(plan.TurnsPerMonth) - 1,
		},
	}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	dec, err := q.Check(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("expected allow, got deny: limited_by=%q", dec.LimitedBy)
	}
	if dec.RetryAfter != 0 {
		t.Errorf("RetryAfter on allow should be zero, got %v", dec.RetryAfter)
	}
}

func TestMonthlyCapTrips_When5dUnderButMonthOver(t *testing.T) {
	// User stayed under their 5d cap but accumulated enough across
	// multiple 5d buckets to hit the monthly fallback.  Should deny
	// with LimitedBy=monthly.
	now := time.Date(2026, 4, 28, 23, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	stub := &stubCounter{counts: map[string]map[int64]int64{
		"alice": {
			since5d:    50,
			monthStart: int64(plan.TurnsPerMonth),
		},
	}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	dec, err := q.Check(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny on monthly cap")
	}
	if dec.LimitedBy != WindowMonthly {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, WindowMonthly)
	}
	// Retry-After should land somewhere between "minutes" and "5 days".
	if dec.RetryAfter <= 0 || dec.RetryAfter > 5*24*time.Hour {
		t.Errorf("RetryAfter = %v, expected within (0, 5d]", dec.RetryAfter)
	}
}

func TestMaxPlus_NoFiveDayCap_OnlyMonthly(t *testing.T) {
	// Max+ has TurnsPer5d = -1 (unlimited).  A user 9999 turns in 5
	// days should still be allowed if they're under the monthly cap.
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanMaxPlus]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	stub := &stubCounter{counts: map[string]map[int64]int64{
		"power": {
			since5d:    9999, // would trip every other tier's 5d cap
			monthStart: int64(plan.TurnsPerMonth) - 1,
		},
	}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	dec, err := q.Check(context.Background(), "power", plan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("Max+ should allow when only monthly window applies, got deny limited_by=%q", dec.LimitedBy)
	}
}

func TestBYOK_NeverDeniedByTurnGate(t *testing.T) {
	// BYOK has -1 on both windows.  Even an absurd usage count shouldn't
	// deny.  (Sandbox / publish quotas are separate and not in scope here.)
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanBYOK]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	stub := &stubCounter{counts: map[string]map[int64]int64{
		"hacker": {since5d: 1_000_000, monthStart: 1_000_000},
	}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	dec, err := q.Check(context.Background(), "hacker", plan)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !dec.Allowed {
		t.Errorf("BYOK should never be turn-gated; got deny limited_by=%q", dec.LimitedBy)
	}
}

func TestUsed_CallsBothWindows(t *testing.T) {
	// Confirm the exact (namespace, since) pairs the gate asks for so
	// the SQL the store sees stays predictable across refactors.
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	stub := &stubCounter{counts: map[string]map[int64]int64{}}
	q := &Quota{turns: stub, now: fixedNow(now)}

	if _, err := q.Used(context.Background(), "alice"); err != nil {
		t.Fatalf("Used: %v", err)
	}
	if got := len(stub.calls); got != 2 {
		t.Fatalf("want 2 store calls (5d + month), got %d", got)
	}

	want5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	wantMonth := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	got5d := stub.calls[0]
	gotMonth := stub.calls[1]
	if got5d.ns != "alice" || got5d.since != want5d {
		t.Errorf("first call = %+v, want ns=alice since=%d", got5d, want5d)
	}
	if gotMonth.ns != "alice" || gotMonth.since != wantMonth {
		t.Errorf("second call = %+v, want ns=alice since=%d", gotMonth, wantMonth)
	}
}

func TestUsed_RejectsEmptyNamespace(t *testing.T) {
	q := NewQuota(&stubCounter{})
	if _, err := q.Used(context.Background(), ""); err == nil {
		t.Error("expected error on empty namespace")
	}
}

func TestStartOfMonth_RoundsToFirstUTC(t *testing.T) {
	mid := time.Date(2026, 4, 15, 23, 59, 59, 999_999_999, time.UTC)
	got := startOfMonth(mid)
	want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("startOfMonth(%v) = %v, want %v", mid, got, want)
	}
}

func TestNextMonth_HandlesYearWrap(t *testing.T) {
	dec := time.Date(2026, 12, 20, 10, 0, 0, 0, time.UTC)
	got := nextMonth(dec)
	want := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("nextMonth(%v) = %v, want %v", dec, got, want)
	}
}

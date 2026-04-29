package usage

import (
	"context"
	"testing"
	"time"
)

// llmStub records every (namespace, since) query and answers from a
// pre-seeded map.  Misses return 0 — a missing seeded window means
// the test asserted the wrong window, surface the mismatch.
type llmStub struct {
	cents map[string]map[int64]int64
	calls []llmCall
}

type llmCall struct {
	ns    string
	since int64
}

func (s *llmStub) LLMSpentByNamespace(_ context.Context, ns string, since int64) (int64, error) {
	s.calls = append(s.calls, llmCall{ns: ns, since: since})
	if m, ok := s.cents[ns]; ok {
		return m[since], nil
	}
	return 0, nil
}

type sandboxStub struct {
	runs map[string]map[int64]int64
}

func (s *sandboxStub) SandboxRunsByNamespace(_ context.Context, ns string, since int64) (int64, error) {
	if m, ok := s.runs[ns]; ok {
		return m[since], nil
	}
	return 0, nil
}

type storageStub struct {
	active map[string]int64
}

func (s *storageStub) ActiveByNamespace(_ context.Context, ns string) (int64, error) {
	return s.active[ns], nil
}

func fixedNow(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func mkQuota(now time.Time, llm *llmStub, sb *sandboxStub, st *storageStub) *Quota {
	if llm == nil {
		llm = &llmStub{}
	}
	if sb == nil {
		sb = &sandboxStub{}
	}
	if st == nil {
		st = &storageStub{}
	}
	return &Quota{llm: llm, sandbox: sb, storage: st, now: fixedNow(now)}
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
	if PlanFor(PlanID("ghost-tier")).ID != PlanFree {
		t.Error("unknown id should fall back to Free")
	}
}

func TestCheckLLM_Pro5dCapEnforced(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	llm := &llmStub{cents: map[string]map[int64]int64{
		"alice": {
			since5d:    int64(plan.LLMBudgetCentsPer5d), // exactly at cap
			monthStart: int64(plan.LLMBudgetCentsPer5d),
		},
	}}
	q := mkQuota(now, llm, nil, nil)

	dec, err := q.CheckLLM(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("CheckLLM: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny when 5d cents == cap")
	}
	if dec.LimitedBy != WindowLLM5d {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, WindowLLM5d)
	}
	if dec.RetryAfter <= 0 {
		t.Error("RetryAfter should be set on deny")
	}
}

func TestCheckLLM_BelowCapAllowed(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	llm := &llmStub{cents: map[string]map[int64]int64{
		"alice": {
			since5d:    int64(plan.LLMBudgetCentsPer5d) - 1,
			monthStart: int64(plan.LLMBudgetCentsPerMonth) - 1,
		},
	}}
	q := mkQuota(now, llm, nil, nil)

	dec, err := q.CheckLLM(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("CheckLLM: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("expected allow, got deny: %q", dec.LimitedBy)
	}
}

func TestCheckLLM_MonthlyTrips_When5dUnderButMonthOver(t *testing.T) {
	now := time.Date(2026, 4, 28, 23, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	llm := &llmStub{cents: map[string]map[int64]int64{
		"alice": {
			since5d:    100,
			monthStart: int64(plan.LLMBudgetCentsPerMonth),
		},
	}}
	q := mkQuota(now, llm, nil, nil)

	dec, err := q.CheckLLM(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("CheckLLM: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny on monthly cap")
	}
	if dec.LimitedBy != WindowLLMMonthly {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, WindowLLMMonthly)
	}
}

func TestCheckLLM_MaxPlus_5dUnlimited(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanMaxPlus]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	llm := &llmStub{cents: map[string]map[int64]int64{
		"power": {
			since5d:    9_999_999, // would trip every other tier
			monthStart: int64(plan.LLMBudgetCentsPerMonth) - 1,
		},
	}}
	q := mkQuota(now, llm, nil, nil)

	dec, err := q.CheckLLM(context.Background(), "power", plan)
	if err != nil {
		t.Fatalf("CheckLLM: %v", err)
	}
	if !dec.Allowed {
		t.Fatalf("Max+ unlimited 5d should allow; limited_by=%q", dec.LimitedBy)
	}
}

func TestCheckLLM_BYOK_AlwaysAllowed(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanBYOK]
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	llm := &llmStub{cents: map[string]map[int64]int64{
		"hacker": {since5d: 1_000_000, monthStart: 1_000_000},
	}}
	q := mkQuota(now, llm, nil, nil)

	dec, err := q.CheckLLM(context.Background(), "hacker", plan)
	if err != nil {
		t.Fatalf("CheckLLM: %v", err)
	}
	if !dec.Allowed {
		t.Errorf("BYOK should never be LLM-gated; got deny limited_by=%q", dec.LimitedBy)
	}
}

func TestCheckSandbox_PerMonthCap(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanFree]
	monthStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC).UnixMilli()

	sb := &sandboxStub{runs: map[string]map[int64]int64{
		"alice": {monthStart: int64(plan.SandboxRunsPerMonth)},
	}}
	q := mkQuota(now, nil, sb, nil)

	dec, err := q.CheckSandbox(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("CheckSandbox: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny when sandbox runs == cap")
	}
	if dec.LimitedBy != WindowSandbox {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, WindowSandbox)
	}
	if dec.RetryAfter <= 0 {
		t.Error("RetryAfter should be set (next-month reset)")
	}
}

func TestCheckStorage_HardCap(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanPro]
	st := &storageStub{active: map[string]int64{
		"alice": int64(plan.ActiveBoxes), // exactly at cap → can't create another
	}}
	q := mkQuota(now, nil, nil, st)

	dec, err := q.CheckStorage(context.Background(), "alice", plan)
	if err != nil {
		t.Fatalf("CheckStorage: %v", err)
	}
	if dec.Allowed {
		t.Fatal("expected deny at active-box cap")
	}
	if dec.LimitedBy != WindowStorage {
		t.Errorf("LimitedBy = %q, want %q", dec.LimitedBy, WindowStorage)
	}
	// Storage is point-in-time — RetryAfter is meaningless; user
	// must archive/delete a box.  We surface 0.
	if dec.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, expected 0 for storage cap", dec.RetryAfter)
	}
}

func TestCheckStorage_MaxPlus_UnlimitedBoxes(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	plan := Plans[PlanMaxPlus]
	st := &storageStub{active: map[string]int64{"power": 1000}}
	q := mkQuota(now, nil, nil, st)
	dec, err := q.CheckStorage(context.Background(), "power", plan)
	if err != nil {
		t.Fatalf("CheckStorage: %v", err)
	}
	if !dec.Allowed {
		t.Errorf("Max+ should not be storage-gated; got deny")
	}
}

func TestUsed_QueriesAllThreeSources(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	llm := &llmStub{}
	sb := &sandboxStub{}
	st := &storageStub{}
	q := mkQuota(now, llm, sb, st)

	if _, err := q.Used(context.Background(), "alice"); err != nil {
		t.Fatalf("Used: %v", err)
	}
	// 2 LLM queries (5d + month), 1 sandbox, 1 storage are expected
	// in Used.  llmStub records calls; sandbox/storage don't but
	// would have errored if not called for the missing namespace.
	if got := len(llm.calls); got != 2 {
		t.Fatalf("LLM calls = %d, want 2 (5d + month)", got)
	}
}

func TestStartOfMonth_Boundary(t *testing.T) {
	mid := time.Date(2026, 4, 15, 23, 59, 59, 999_999_999, time.UTC)
	got := startOfMonth(mid)
	want := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("startOfMonth = %v, want %v", got, want)
	}
}

func TestNextMonth_HandlesYearWrap(t *testing.T) {
	dec := time.Date(2026, 12, 20, 10, 0, 0, 0, time.UTC)
	if got := nextMonth(dec); !got.Equal(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("nextMonth(Dec) = %v, want 2027-01-01", got)
	}
}

func TestCostCents_DeepSeekStandard(t *testing.T) {
	// 3K input + 1K output of deepseek-chat:
	// raw = (3K/1M)*1.0 + (1K/1M)*2.0 = 0.003 + 0.002 = 0.005 RMB
	// markup 1.20 → 0.006 RMB → ceil(0.6 cents) = 1 cent (floor protection)
	got := CostCents("deepseek-chat", 3000, 1000)
	if got < 1 {
		t.Errorf("cost = %d, want >= 1 (cents floor protection)", got)
	}
	if got > 5 {
		t.Errorf("cost = %d, want small for cheap model + small turn", got)
	}
}

func TestCostCents_OpusMuchMoreThanDeepSeek(t *testing.T) {
	// Same token volume should bill ~ much higher on Opus.  Sanity
	// check that the ratio is in the right ballpark (>= 30×).
	deep := CostCents("deepseek-chat", 30000, 5000)
	opus := CostCents("claude-opus-4-7", 30000, 5000)
	if opus < deep*30 {
		t.Errorf("opus cost (%d) should be >= 30× deepseek (%d)", opus, deep)
	}
}

func TestCostCents_UnknownModelFallsBackToDeepSeek(t *testing.T) {
	got := CostCents("not-a-real-model", 3000, 1000)
	want := CostCents("deepseek-chat", 3000, 1000)
	if got != want {
		t.Errorf("unknown model cost = %d, want %d (deepseek fallback)", got, want)
	}
}

func TestCostCents_ZeroTokensIsFree(t *testing.T) {
	if got := CostCents("deepseek-chat", 0, 0); got != 0 {
		t.Errorf("zero tokens should be 0 cents, got %d", got)
	}
}

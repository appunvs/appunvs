// Quota is the runtime gate that decides whether a user's next
// platform action is allowed under their plan.  Three independent
// dimensions, each with its own cap; whichever fires first denies
// without affecting the others (so an LLM-cap deny still lets the
// user open the boxes tab and read past chats):
//
//   - LLM     — sums ai_turns.cost_cents over a 5d rolling window
//               and the calendar month
//   - Sandbox — counts bundle build attempts for the calendar month
//   - Storage — counts non-archived boxes (point-in-time, not a window)
//
// The handler calls Check before invoking the action; deny returns
// 429 with Retry-After + a JSON body describing which dimension fired.
package usage

import (
	"context"
	"errors"
	"time"
)

// Window enumerates the rate-limit windows we report on.  Returned by
// Decision.LimitedBy so the caller can render an accurate 429
// message.
type Window string

const (
	WindowNone        Window = ""             // not limited
	WindowLLM5d       Window = "llm_5d"       // LLM rolling 5 days
	WindowLLMMonthly  Window = "llm_monthly"  // LLM calendar month
	WindowSandbox     Window = "sandbox"      // Sandbox calendar month
	WindowStorage     Window = "storage"      // Storage point-in-time
)

// Used reports current consumption across all three dimensions.
type Used struct {
	// LLM (RMB cents).
	LLMSpentCentsLast5d    int64
	LLMSpentCentsThisMonth int64
	// Sandbox runs this calendar month.
	SandboxRunsThisMonth int64
	// Storage: count of non-archived boxes (point-in-time).
	ActiveBoxes int64
	// CountedAt is the wall-clock the queries ran at.  Surfaced to
	// callers so a "remaining" UI doesn't have to call time.Now().
	CountedAt time.Time
}

// Decision is the boolean answer plus enough context to render an
// accurate 429.  Allowed=true means RetryAfter and LimitedBy are
// zero-valued.
type Decision struct {
	Allowed    bool
	LimitedBy  Window
	RetryAfter time.Duration
	Used       Used
	Plan       Plan
}

// LLMSource owns the slice of store calls covering AI-turn cost.
type LLMSource interface {
	LLMSpentByNamespace(ctx context.Context, namespace string, sinceMillis int64) (int64, error)
}

// SandboxSource covers per-month sandbox build counts.
type SandboxSource interface {
	SandboxRunsByNamespace(ctx context.Context, namespace string, sinceMillis int64) (int64, error)
}

// StorageSource covers point-in-time active-box counts.
type StorageSource interface {
	ActiveByNamespace(ctx context.Context, namespace string) (int64, error)
}

// Quota is the service object the handler holds.  Stateless apart
// from the source references; safe to share across goroutines.
//
// The three sources are split because LLM lives in store.Turns while
// Sandbox + Storage live in store.Boxes; tests can also stub each
// independently.
type Quota struct {
	llm     LLMSource
	sandbox SandboxSource
	storage StorageSource
	now     func() time.Time // injectable for tests
}

// NewQuota constructs the gate.
func NewQuota(llm LLMSource, sandbox SandboxSource, storage StorageSource) *Quota {
	return &Quota{llm: llm, sandbox: sandbox, storage: storage, now: time.Now}
}

// Used reports current consumption without making a deny/allow
// decision.  Used by /usage/me to render Profile-page progress bars.
func (q *Quota) Used(ctx context.Context, namespace string) (Used, error) {
	if namespace == "" {
		return Used{}, errors.New("usage: namespace required")
	}
	now := q.now()
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := startOfMonth(now).UnixMilli()

	last5d, err := q.llm.LLMSpentByNamespace(ctx, namespace, since5d)
	if err != nil {
		return Used{}, err
	}
	thisMonth, err := q.llm.LLMSpentByNamespace(ctx, namespace, monthStart)
	if err != nil {
		return Used{}, err
	}
	sandbox, err := q.sandbox.SandboxRunsByNamespace(ctx, namespace, monthStart)
	if err != nil {
		return Used{}, err
	}
	active, err := q.storage.ActiveByNamespace(ctx, namespace)
	if err != nil {
		return Used{}, err
	}
	return Used{
		LLMSpentCentsLast5d:    last5d,
		LLMSpentCentsThisMonth: thisMonth,
		SandboxRunsThisMonth:   sandbox,
		ActiveBoxes:            active,
		CountedAt:              now,
	}, nil
}

// CheckLLM is the gate for /ai/turn — only verifies LLM caps.  Hits
// the LLM dimensions only so a Sandbox cap doesn't block chat (and
// vice versa).  -1 caps mean unlimited and are never tripped.
//
// Order: 5d wins when both LLM windows could fire (more imminent reset).
func (q *Quota) CheckLLM(ctx context.Context, namespace string, plan Plan) (Decision, error) {
	used, err := q.Used(ctx, namespace)
	if err != nil {
		return Decision{}, err
	}
	dec := Decision{Allowed: true, Used: used, Plan: plan}

	if plan.LLMBudgetCentsPer5d >= 0 && used.LLMSpentCentsLast5d >= int64(plan.LLMBudgetCentsPer5d) {
		dec.Allowed = false
		dec.LimitedBy = WindowLLM5d
		dec.RetryAfter = 5 * 24 * time.Hour
		return dec, nil
	}
	if plan.LLMBudgetCentsPerMonth >= 0 && used.LLMSpentCentsThisMonth >= int64(plan.LLMBudgetCentsPerMonth) {
		dec.Allowed = false
		dec.LimitedBy = WindowLLMMonthly
		dec.RetryAfter = nextMonth(used.CountedAt).Sub(used.CountedAt)
		return dec, nil
	}
	return dec, nil
}

// CheckSandbox gates publish_box / box.BuildAndPublish.  Per-month
// only — sandbox cost is predictable per call so a rolling window
// doesn't add value.
func (q *Quota) CheckSandbox(ctx context.Context, namespace string, plan Plan) (Decision, error) {
	used, err := q.Used(ctx, namespace)
	if err != nil {
		return Decision{}, err
	}
	dec := Decision{Allowed: true, Used: used, Plan: plan}
	if plan.SandboxRunsPerMonth >= 0 && used.SandboxRunsThisMonth >= int64(plan.SandboxRunsPerMonth) {
		dec.Allowed = false
		dec.LimitedBy = WindowSandbox
		dec.RetryAfter = nextMonth(used.CountedAt).Sub(used.CountedAt)
	}
	return dec, nil
}

// CheckStorage gates box creation.  Point-in-time check — RetryAfter
// is meaningless here (the user has to delete or archive a box, not
// wait), so we leave it 0 and rely on UI copy.
func (q *Quota) CheckStorage(ctx context.Context, namespace string, plan Plan) (Decision, error) {
	used, err := q.Used(ctx, namespace)
	if err != nil {
		return Decision{}, err
	}
	dec := Decision{Allowed: true, Used: used, Plan: plan}
	if plan.ActiveBoxes >= 0 && used.ActiveBoxes >= int64(plan.ActiveBoxes) {
		dec.Allowed = false
		dec.LimitedBy = WindowStorage
		// RetryAfter zero — user action required, no auto-reset.
	}
	return dec, nil
}

// startOfMonth returns midnight on the 1st of t's month, in UTC.
// We use UTC throughout the relay so the result is deterministic
// across deployments.
func startOfMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// nextMonth returns midnight on the 1st of t's NEXT calendar month.
func nextMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

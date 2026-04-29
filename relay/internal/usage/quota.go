// Quota is the runtime gate that decides whether a user's next /ai/turn
// call is allowed under their plan.  It runs two queries against
// store.Turns — one for the rolling 5-day window, one for the calendar
// month — and reports the count plus a Decision describing which
// window (if any) is exhausted.
//
// The handler calls Check before invoking the engine; a deny returns
// HTTP 429 with a Retry-After header derived from Decision.RetryAfter.
package usage

import (
	"context"
	"errors"
	"time"
)

// Window enumerates the named rate-limit windows.  Returned by
// Decision.LimitedBy so the caller can render an accurate 429 message
// ("you hit your weekly cap" vs "you hit your monthly cap").
type Window string

const (
	WindowNone    Window = ""        // not limited
	Window5d      Window = "5d"      // rolling 5 days
	WindowMonthly Window = "monthly" // current calendar month
)

// Used reports current consumption across both windows.  Token / publish
// counts are not populated here — those are independent quotas tracked
// elsewhere.
type Used struct {
	Last5d    int64
	ThisMonth int64
	// CountedAt is the wall-clock the queries ran at; quota math uses
	// the same instant for the two SQL counts so a turn that lands
	// between them can't be double-counted.  Surfaced to callers so a
	// "remaining" UI doesn't have to call time.Now() itself.
	CountedAt time.Time
}

// Decision is the boolean answer plus enough context to render an
// accurate 429.  Allowed=true means RetryAfter and LimitedBy are
// zero-valued.
type Decision struct {
	Allowed    bool
	LimitedBy  Window
	RetryAfter time.Duration // how long until the limiting window resets
	Used       Used
	Plan       Plan
}

// turnCounter is the slice of store.Turns we depend on.  Defined as an
// interface so quota_test.go can plug in a stub without spinning up
// SQLite for every assertion.
type turnCounter interface {
	CountByNamespace(ctx context.Context, namespace string, sinceMillis int64) (int64, error)
}

// Quota is the service object the handler holds.  Stateless apart from
// the store reference; safe to share across goroutines.
type Quota struct {
	turns turnCounter
	now   func() time.Time // injectable for tests
}

// NewQuota constructs the gate.  Pass store.Turns (or any turnCounter)
// — the indirection is only for tests.
func NewQuota(turns turnCounter) *Quota {
	return &Quota{turns: turns, now: time.Now}
}

// Used reports current consumption without making a deny/allow decision.
// Used by the Profile / Boxes UI to render "本周 N/M · 本月 N/M".
func (q *Quota) Used(ctx context.Context, namespace string) (Used, error) {
	if namespace == "" {
		return Used{}, errors.New("usage: namespace required")
	}
	now := q.now()
	since5d := now.Add(-5 * 24 * time.Hour).UnixMilli()
	monthStart := startOfMonth(now).UnixMilli()

	last5d, err := q.turns.CountByNamespace(ctx, namespace, since5d)
	if err != nil {
		return Used{}, err
	}
	thisMonth, err := q.turns.CountByNamespace(ctx, namespace, monthStart)
	if err != nil {
		return Used{}, err
	}
	return Used{
		Last5d:    last5d,
		ThisMonth: thisMonth,
		CountedAt: now,
	}, nil
}

// Check runs Used and applies the plan's caps.  The first window to
// trip determines LimitedBy + RetryAfter.  Both windows being open
// returns Allowed=true.  -1 caps mean "unlimited" and are never tripped.
//
// Order matters when both could fire: 5d wins (it's the more granular
// signal and the more imminent reset).
func (q *Quota) Check(ctx context.Context, namespace string, plan Plan) (Decision, error) {
	used, err := q.Used(ctx, namespace)
	if err != nil {
		return Decision{}, err
	}
	dec := Decision{Allowed: true, Used: used, Plan: plan}

	if plan.TurnsPer5d >= 0 && used.Last5d >= int64(plan.TurnsPer5d) {
		dec.Allowed = false
		dec.LimitedBy = Window5d
		// Reset = oldest turn-in-window's time + 5d.  We don't have
		// that timestamp here without an extra query; fall back to a
		// conservative "5d from now" hint.  The handler can refine
		// this later via a separate "earliest turn in window" query
		// if we decide the imprecise hint is bad UX.
		dec.RetryAfter = 5 * 24 * time.Hour
		return dec, nil
	}
	if plan.TurnsPerMonth >= 0 && used.ThisMonth >= int64(plan.TurnsPerMonth) {
		dec.Allowed = false
		dec.LimitedBy = WindowMonthly
		dec.RetryAfter = nextMonth(used.CountedAt).Sub(used.CountedAt)
		return dec, nil
	}
	return dec, nil
}

// startOfMonth returns midnight on the 1st of t's month, in t's
// location.  We use UTC throughout the relay so the result is
// deterministic across deployments.
func startOfMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// nextMonth returns midnight on the 1st of t's NEXT calendar month.
// Used to compute Retry-After when the monthly cap fires.
func nextMonth(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, time.UTC)
}

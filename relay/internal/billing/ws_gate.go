package billing

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/store"
)

// QuotaGate is the narrow interface /ws depends on. It's satisfied by
// *Gate below; tests can implement it directly.
type QuotaGate interface {
	CheckAndIncrement(ctx context.Context, userID string, bytes int64) error
}

// Gate enforces messages_per_day quotas synchronously against SQLite.
// This is Option B from the Block 4 spec: simple, exact, and adequate
// up to the write throughput of a single SQLite file. For higher scale,
// swap in a Redis-backed implementation of QuotaGate that buffers
// increments and flushes periodically — the interface stays the same.
//
// TODO(scale): Redis-backed Tick + 60s flush to usage_daily. Keeps the
// same QuotaGate shape; only the internals change.
type Gate struct {
	store *store.Store
	log   *zap.Logger
}

// NewGate builds a SQLite-backed QuotaGate.
func NewGate(st *store.Store, log *zap.Logger) *Gate {
	return &Gate{store: st, log: log}
}

// CheckAndIncrement returns ErrQuotaExceeded when the user has already
// hit their daily limit; otherwise bumps the daily counters (messages by
// one, bytes by the caller-supplied delta) and returns nil.
//
// This is intentionally a read-then-write without row locking. Two racing
// requests at the exact boundary could each see count=limit-1, both
// increment, and land count=limit+1. In exchange we avoid a transaction
// per broadcast. The quota is a soft cap; a one-message overshoot is
// acceptable.
func (g *Gate) CheckAndIncrement(ctx context.Context, userID string, bytes int64) error {
	sub, err := g.store.Subscriptions().Get(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		// Self-heal: every authenticated user should have at least a free
		// subscription. Missing rows are a bug, not a reason to drop the
		// message.
		if err := g.store.Subscriptions().EnsureFree(ctx, userID); err != nil {
			return err
		}
		sub, err = g.store.Subscriptions().Get(ctx, userID)
		if err != nil {
			return err
		}
		if sub == nil {
			return errors.New("subscription missing after heal")
		}
	}
	plan, err := g.store.Plans().Get(ctx, sub.PlanID)
	if err != nil {
		return err
	}
	if plan == nil {
		if cp := store.FindCanonicalPlan(sub.PlanID); cp != nil {
			plan = cp
		} else {
			return errors.New("plan missing")
		}
	}

	day := store.DayKey(time.Now())
	used, err := g.store.Usage().MessagesForDay(ctx, userID, day)
	if err != nil {
		return err
	}
	if used >= plan.MessagesPerDay {
		return ErrQuotaExceeded
	}
	return g.store.Usage().IncrementBoth(ctx, userID, day, 1, bytes)
}

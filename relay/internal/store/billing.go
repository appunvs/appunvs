// Package store — billing.go owns the plans/subscriptions/usage DAOs and
// the migration that seeds the three canonical plans.
//
// Block 4 of the product pivot lives here. We picked schema_version = 4
// per the coordination note that reserved 2 and 3 for Blocks 2/3. Never
// renumber a shipped migration.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Plan mirrors a row in the `plans` table and the const table in code.
// It's deliberately a value type so tests and the frontend can consume
// an identical copy without surgery.
type Plan struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	PriceCentsMonthly int64  `json:"price_cents_monthly"`
	MessagesPerDay    int64  `json:"messages_per_day"`
	StorageBytes      int64  `json:"storage_bytes"`
	MaxDevices        int64  `json:"max_devices"`
	MaxAPIKeys        int64  `json:"max_api_keys"`
}

// CanonicalPlans is the source of truth the migration seeds into SQLite.
// Tests (and the future frontend bundle) can import this slice directly so
// there's a single authoritative definition of what each tier allows.
// CanonicalPlans is the seed list for the `plans` table.  Prices are in
// RMB *fen* (¥1 = 100 fen), matching `price_cents_monthly`'s int64 shape
// (the column name predates the currency choice — see docs/billing.md
// once that lands).  The Chinese coding-tool market caps Pro at roughly
// ¥30–50 and Max at ¥150–250; these values stay inside that band:
//
//   Free  — acquisition / tryout.  Heavy caps; watermarked bundles.
//   Pro   — ¥39/mo.  The mainline single-seat tier.
//   Max   — ¥199/mo.  Heavy personal use; priority queue will land later.
//
// Existing rows in `plans` are NOT overwritten by the seed (the migration
// uses INSERT OR IGNORE).  Operators who change a shipped plan must run
// a separate data migration; shipping-time tweaks are safe.
var CanonicalPlans = []Plan{
	{
		ID:                "free",
		Name:              "Free",
		PriceCentsMonthly: 0,
		MessagesPerDay:    1_000,
		StorageBytes:      10 * 1024 * 1024, // 10 MiB
		MaxDevices:        3,
		MaxAPIKeys:        2,
	},
	{
		ID:                "pro",
		Name:              "Pro",
		PriceCentsMonthly: 3_900, // ¥39
		MessagesPerDay:    100_000,
		StorageBytes:      1 * 1024 * 1024 * 1024, // 1 GiB
		MaxDevices:        10,
		MaxAPIKeys:        10,
	},
	{
		// Max targets heavy personal users (Cursor Ultra style): higher
		// ceilings than Pro across every dimension, still single-seat.
		ID:                "max",
		Name:              "Max",
		PriceCentsMonthly: 19_900, // ¥199
		MessagesPerDay:    500_000,
		StorageBytes:      5 * 1024 * 1024 * 1024, // 5 GiB
		MaxDevices:        25,
		MaxAPIKeys:        25,
	},
}

// FindCanonicalPlan returns the canonical in-code plan for id, or nil when
// no such id exists. Useful for defaults without a DB roundtrip.
func FindCanonicalPlan(id string) *Plan {
	for i := range CanonicalPlans {
		if CanonicalPlans[i].ID == id {
			p := CanonicalPlans[i]
			return &p
		}
	}
	return nil
}

// Plans is the DAO for the plans table.
type Plans struct{ s *Store }

// Plans returns the plans DAO bound to this store.
func (s *Store) Plans() *Plans { return &Plans{s: s} }

// List returns every plan in the DB, sorted by price ascending.
func (p *Plans) List(ctx context.Context) ([]Plan, error) {
	rows, err := p.s.DB.QueryContext(ctx,
		`SELECT id, name, price_cents_monthly, messages_per_day, storage_bytes, max_devices, max_api_keys
		   FROM plans
		   ORDER BY price_cents_monthly ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []Plan
	for rows.Next() {
		var pl Plan
		if err := rows.Scan(&pl.ID, &pl.Name, &pl.PriceCentsMonthly, &pl.MessagesPerDay,
			&pl.StorageBytes, &pl.MaxDevices, &pl.MaxAPIKeys); err != nil {
			return nil, err
		}
		out = append(out, pl)
	}
	return out, rows.Err()
}

// Get fetches a single plan by id, returning nil when not found.
func (p *Plans) Get(ctx context.Context, id string) (*Plan, error) {
	var pl Plan
	err := p.s.DB.QueryRowContext(ctx,
		`SELECT id, name, price_cents_monthly, messages_per_day, storage_bytes, max_devices, max_api_keys
		   FROM plans WHERE id = ?`, id,
	).Scan(&pl.ID, &pl.Name, &pl.PriceCentsMonthly, &pl.MessagesPerDay,
		&pl.StorageBytes, &pl.MaxDevices, &pl.MaxAPIKeys)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get plan %s: %w", id, err)
	}
	return &pl, nil
}

// Subscription is the hydrated subscriptions row for a user.
type Subscription struct {
	UserID               string
	PlanID               string
	StripeCustomerID     string
	StripeSubscriptionID string
	Status               string
	CurrentPeriodStart   time.Time
	CurrentPeriodEnd     time.Time
	UpdatedAt            time.Time
}

// Subscriptions is the DAO for the subscriptions table.
type Subscriptions struct{ s *Store }

// Subscriptions returns the subscriptions DAO bound to this store.
func (s *Store) Subscriptions() *Subscriptions { return &Subscriptions{s: s} }

// EnsureFree makes sure a user has at least a free-tier subscription. It is
// idempotent — calling it twice in a row is a no-op.
//
// Semantics:
//   - if the user has no subscriptions row, insert one pointing at `free`
//     with a monthly window starting now.
//   - if the user already has a subscription (any plan), leave it alone.
//
// Signup calls this; webhook upgrades call `Upsert` instead.
func (subs *Subscriptions) EnsureFree(ctx context.Context, userID string) error {
	now := time.Now().UTC()
	periodStart := now
	periodEnd := now.AddDate(0, 1, 0)
	_, err := subs.s.DB.ExecContext(ctx, `
INSERT INTO subscriptions
    (user_id, plan_id, status, current_period_start, current_period_end, updated_at)
VALUES (?, 'free', 'active', ?, ?, ?)
ON CONFLICT(user_id) DO NOTHING`,
		userID, periodStart.UnixMilli(), periodEnd.UnixMilli(), now.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("ensure free sub: %w", err)
	}
	return nil
}

// Get returns the subscription row for userID, or nil if no row exists.
func (subs *Subscriptions) Get(ctx context.Context, userID string) (*Subscription, error) {
	var (
		sub      Subscription
		cust     sql.NullString
		stripeID sql.NullString
		ps, pe   int64
		ua       int64
	)
	err := subs.s.DB.QueryRowContext(ctx, `
SELECT user_id, plan_id, stripe_customer_id, stripe_subscription_id, status,
       current_period_start, current_period_end, updated_at
  FROM subscriptions WHERE user_id = ?`, userID,
	).Scan(&sub.UserID, &sub.PlanID, &cust, &stripeID, &sub.Status, &ps, &pe, &ua)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get subscription: %w", err)
	}
	if cust.Valid {
		sub.StripeCustomerID = cust.String
	}
	if stripeID.Valid {
		sub.StripeSubscriptionID = stripeID.String
	}
	sub.CurrentPeriodStart = time.UnixMilli(ps).UTC()
	sub.CurrentPeriodEnd = time.UnixMilli(pe).UTC()
	sub.UpdatedAt = time.UnixMilli(ua).UTC()
	return &sub, nil
}

// Upsert writes the subscription row. Intended for webhook handlers after
// a successful checkout or a status change from Stripe.
func (subs *Subscriptions) Upsert(ctx context.Context, s Subscription) error {
	now := time.Now().UTC()
	s.UpdatedAt = now
	var cust, stripeID any
	if s.StripeCustomerID != "" {
		cust = s.StripeCustomerID
	}
	if s.StripeSubscriptionID != "" {
		stripeID = s.StripeSubscriptionID
	}
	_, err := subs.s.DB.ExecContext(ctx, `
INSERT INTO subscriptions
    (user_id, plan_id, stripe_customer_id, stripe_subscription_id, status,
     current_period_start, current_period_end, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    plan_id = excluded.plan_id,
    stripe_customer_id = COALESCE(excluded.stripe_customer_id, subscriptions.stripe_customer_id),
    stripe_subscription_id = COALESCE(excluded.stripe_subscription_id, subscriptions.stripe_subscription_id),
    status = excluded.status,
    current_period_start = excluded.current_period_start,
    current_period_end = excluded.current_period_end,
    updated_at = excluded.updated_at`,
		s.UserID, s.PlanID, cust, stripeID, s.Status,
		s.CurrentPeriodStart.UnixMilli(), s.CurrentPeriodEnd.UnixMilli(),
		now.UnixMilli(),
	)
	if err != nil {
		return fmt.Errorf("upsert subscription: %w", err)
	}
	return nil
}

// SetPlan updates plan_id + status + period + stripe refs without changing
// user_id. Simpler signature for webhook paths that already know the user.
func (subs *Subscriptions) SetPlan(ctx context.Context, userID, planID, status string,
	periodStart, periodEnd time.Time, customerID, stripeSubID string,
) error {
	return subs.Upsert(ctx, Subscription{
		UserID:               userID,
		PlanID:               planID,
		StripeCustomerID:     customerID,
		StripeSubscriptionID: stripeSubID,
		Status:               status,
		CurrentPeriodStart:   periodStart,
		CurrentPeriodEnd:     periodEnd,
	})
}

// FindByStripeCustomer returns the subscription matching a stripe customer
// id; nil if none. Used by webhook dispatch when the event only carries the
// stripe ids, not our user id.
func (subs *Subscriptions) FindByStripeCustomer(ctx context.Context, customerID string) (*Subscription, error) {
	if customerID == "" {
		return nil, nil
	}
	var (
		sub      Subscription
		cust     sql.NullString
		stripeID sql.NullString
		ps, pe   int64
		ua       int64
	)
	err := subs.s.DB.QueryRowContext(ctx, `
SELECT user_id, plan_id, stripe_customer_id, stripe_subscription_id, status,
       current_period_start, current_period_end, updated_at
  FROM subscriptions WHERE stripe_customer_id = ?`, customerID,
	).Scan(&sub.UserID, &sub.PlanID, &cust, &stripeID, &sub.Status, &ps, &pe, &ua)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find by customer: %w", err)
	}
	if cust.Valid {
		sub.StripeCustomerID = cust.String
	}
	if stripeID.Valid {
		sub.StripeSubscriptionID = stripeID.String
	}
	sub.CurrentPeriodStart = time.UnixMilli(ps).UTC()
	sub.CurrentPeriodEnd = time.UnixMilli(pe).UTC()
	sub.UpdatedAt = time.UnixMilli(ua).UTC()
	return &sub, nil
}

// Usage is the DAO for the usage_daily table.
type Usage struct{ s *Store }

// Usage returns the usage DAO bound to this store.
func (s *Store) Usage() *Usage { return &Usage{s: s} }

// DayKey converts a time to YYYYMMDD as an int. UTC is the canonical clock
// here; reset-at-midnight-UTC gives predictable windows across regions.
func DayKey(t time.Time) int {
	t = t.UTC()
	y, m, d := t.Date()
	return y*10000 + int(m)*100 + d
}

// Increment bumps the daily message counter by n (typically 1). Inserts the
// row if missing.
func (u *Usage) Increment(ctx context.Context, userID string, day int, n int64) error {
	return u.IncrementBoth(ctx, userID, day, n, 0)
}

// IncrementBoth bumps both messages and bytes for the day.  Either can be 0;
// passing (n=1, bytes=0) is identical to Increment.
func (u *Usage) IncrementBoth(ctx context.Context, userID string, day int, messages, bytes int64) error {
	_, err := u.s.DB.ExecContext(ctx, `
INSERT INTO usage_daily(user_id, day, messages, bytes) VALUES (?, ?, ?, ?)
ON CONFLICT(user_id, day) DO UPDATE SET
    messages = messages + excluded.messages,
    bytes    = bytes    + excluded.bytes`,
		userID, day, messages, bytes,
	)
	if err != nil {
		return fmt.Errorf("increment usage: %w", err)
	}
	return nil
}

// MessagesForDay returns the counter for (user, day), or 0 if no row.
func (u *Usage) MessagesForDay(ctx context.Context, userID string, day int) (int64, error) {
	var n int64
	err := u.s.DB.QueryRowContext(ctx,
		`SELECT messages FROM usage_daily WHERE user_id = ? AND day = ?`,
		userID, day,
	).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get usage: %w", err)
	}
	return n, nil
}

// MessagesForPeriod sums counters across [startDay, endDay] inclusive.
func (u *Usage) MessagesForPeriod(ctx context.Context, userID string, startDay, endDay int) (int64, error) {
	var n sql.NullInt64
	err := u.s.DB.QueryRowContext(ctx,
		`SELECT SUM(messages) FROM usage_daily WHERE user_id = ? AND day BETWEEN ? AND ?`,
		userID, startDay, endDay,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("sum usage: %w", err)
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

// BytesForPeriod sums payload bytes observed across [startDay, endDay].
// This is "bytes transferred through the relay this period" — an honest
// proxy for storage since the relay itself persists only Redis Stream
// (24h retention). Clients hold the authoritative data at rest.
func (u *Usage) BytesForPeriod(ctx context.Context, userID string, startDay, endDay int) (int64, error) {
	var n sql.NullInt64
	err := u.s.DB.QueryRowContext(ctx,
		`SELECT SUM(bytes) FROM usage_daily WHERE user_id = ? AND day BETWEEN ? AND ?`,
		userID, startDay, endDay,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("sum bytes: %w", err)
	}
	if !n.Valid {
		return 0, nil
	}
	return n.Int64, nil
}

// init registers the Block 4 migrations. Kept in a package init so we don't
// have to edit store.go every time a new block lands; the migrations slice
// is still read in strict version order at startup.
func init() {
	migrations = append(migrations,
		migration{
			version: 4,
			sql:     billingMigrationSQL() + seedPlansSQL(),
		},
		// v5: add bytes counter to usage_daily so /billing/status can report
		// "bytes transferred this period". Honest to what we can measure;
		// the relay does not persist payload bodies (Redis Stream is 24h).
		migration{
			version: 5,
			sql:     `ALTER TABLE usage_daily ADD COLUMN bytes INTEGER NOT NULL DEFAULT 0;`,
		},
	)
}

func billingMigrationSQL() string {
	return `
CREATE TABLE plans (
    id                   TEXT PRIMARY KEY,
    name                 TEXT NOT NULL,
    price_cents_monthly  INTEGER NOT NULL,
    messages_per_day     INTEGER NOT NULL,
    storage_bytes        INTEGER NOT NULL,
    max_devices          INTEGER NOT NULL,
    max_api_keys         INTEGER NOT NULL
);

CREATE TABLE subscriptions (
    user_id                 TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    plan_id                 TEXT NOT NULL REFERENCES plans(id),
    stripe_customer_id      TEXT,
    stripe_subscription_id  TEXT,
    status                  TEXT NOT NULL,
    current_period_start    INTEGER NOT NULL,
    current_period_end      INTEGER NOT NULL,
    updated_at              INTEGER NOT NULL
);
CREATE INDEX idx_subs_stripe_customer ON subscriptions(stripe_customer_id);

CREATE TABLE usage_daily (
    user_id     TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day         INTEGER NOT NULL,
    messages    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, day)
);
`
}

func seedPlansSQL() string {
	s := ""
	for _, p := range CanonicalPlans {
		s += fmt.Sprintf(
			"INSERT OR IGNORE INTO plans(id, name, price_cents_monthly, messages_per_day, storage_bytes, max_devices, max_api_keys) VALUES ('%s', '%s', %d, %d, %d, %d, %d);\n",
			p.ID, p.Name, p.PriceCentsMonthly, p.MessagesPerDay, p.StorageBytes, p.MaxDevices, p.MaxAPIKeys,
		)
	}
	return s
}

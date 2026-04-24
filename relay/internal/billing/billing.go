// Package billing owns plans, subscriptions, Stripe checkout/webhook, and
// the quota gate that the /ws handler calls before broadcasting a provider
// message.
//
// Mock vs real mode
//
// If STRIPE_SECRET_KEY (or the config key billing.stripe_secret_key) is
// empty at startup, the Service runs in mock mode:
//
//   - POST /billing/checkout returns
//     {"url":"https://stripe.mock/checkout/<plan_id>?uid=<user_id>"}
//   - POST /billing/webhook accepts an unsigned body carrying the magic
//     header "X-Mock-Upgrade: <plan_id>" and a "X-Mock-User: <user_id>"
//     header. It immediately upserts the user's subscription to <plan_id>
//     with status=active and a fresh monthly window.
//
// Mock mode is intended for local dev and tests. Any deploy that actually
// charges customers must set billing.stripe_secret_key and
// billing.stripe_webhook_secret.
package billing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/checkout/session"
	"github.com/stripe/stripe-go/v79/webhook"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/store"
)

// ErrQuotaExceeded is returned by QuotaGate.CheckAndIncrement when the
// user's current-period counter has reached the plan's messages_per_day
// limit. Callers should drop the message and inform the client.
var ErrQuotaExceeded = errors.New("messages_per_day quota exceeded")

// ErrUnknownPlan is returned when a requested plan id doesn't exist.
var ErrUnknownPlan = errors.New("unknown plan")

// Service bundles plans + Stripe + quota state behind a single struct so
// handlers and the ws gate get a consistent view of billing.
type Service struct {
	Store               *store.Store
	Log                 *zap.Logger
	StripeSecretKey     string
	StripeWebhookSecret string
	CheckoutSuccessURL  string
	CheckoutCancelURL   string
}

// Mode reports whether the service runs against real Stripe or is stubbed.
func (s *Service) Mode() string {
	if s.StripeSecretKey == "" {
		return "mock"
	}
	return "real"
}

// New builds a Service and, if running in real mode, configures the
// process-global stripe.Key.
func New(st *store.Store, log *zap.Logger, secret, webhookSecret, successURL, cancelURL string) *Service {
	if secret != "" {
		stripe.Key = secret
	}
	return &Service{
		Store:               st,
		Log:                 log,
		StripeSecretKey:     secret,
		StripeWebhookSecret: webhookSecret,
		CheckoutSuccessURL:  successURL,
		CheckoutCancelURL:   cancelURL,
	}
}

// CheckoutResult is what POST /billing/checkout returns.
type CheckoutResult struct {
	URL  string `json:"url"`
	Mode string `json:"mode"` // "mock" | "real"
}

// CreateCheckout starts a Stripe Checkout session for userID + planID.
// In mock mode it just returns a deterministic synthetic URL.
func (s *Service) CreateCheckout(ctx context.Context, userID, planID string) (*CheckoutResult, error) {
	plan, err := s.Store.Plans().Get(ctx, planID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		return nil, ErrUnknownPlan
	}
	if s.Mode() == "mock" {
		return &CheckoutResult{
			URL:  fmt.Sprintf("https://stripe.mock/checkout/%s?uid=%s", plan.ID, userID),
			Mode: "mock",
		}, nil
	}
	// Real Stripe session: a subscription with a one-line inline price.
	// Production deployments would likely pre-create Prices in Stripe and
	// reference them by id; we mint an inline price for simplicity.
	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		SuccessURL:        stripe.String(s.CheckoutSuccessURL),
		CancelURL:         stripe.String(s.CheckoutCancelURL),
		ClientReferenceID: stripe.String(userID),
		LineItems: []*stripe.CheckoutSessionLineItemParams{{
			PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
				Currency: stripe.String("usd"),
				Recurring: &stripe.CheckoutSessionLineItemPriceDataRecurringParams{
					Interval: stripe.String("month"),
				},
				ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
					Name: stripe.String("appunvs " + plan.Name),
				},
				UnitAmount: stripe.Int64(plan.PriceCentsMonthly),
			},
			Quantity: stripe.Int64(1),
		}},
		Metadata: map[string]string{
			"plan_id": plan.ID,
			"user_id": userID,
		},
	}
	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("stripe checkout: %w", err)
	}
	return &CheckoutResult{URL: sess.URL, Mode: "real"}, nil
}

// Status is the hydrated response for GET /billing/status.
type Status struct {
	Plan          string      `json:"plan"`
	PlanName      string      `json:"plan_name"`
	Status        string      `json:"status"`
	PeriodStart   int64       `json:"period_start"` // unix millis
	PeriodEnd     int64       `json:"period_end"`
	MessagesUsed  int64       `json:"messages_used"`
	StorageBytes  int64       `json:"storage_bytes"`
	Limits        store.Plan  `json:"limits"`
}

// LoadStatus fetches the current subscription + usage for userID. If no
// subscription row exists (should be rare — signup auto-seeds it) it heals
// by calling EnsureFree first.
func (s *Service) LoadStatus(ctx context.Context, userID string) (*Status, error) {
	sub, err := s.Store.Subscriptions().Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		// Self-heal: seed a free subscription on-demand.
		if err := s.Store.Subscriptions().EnsureFree(ctx, userID); err != nil {
			return nil, err
		}
		sub, err = s.Store.Subscriptions().Get(ctx, userID)
		if err != nil {
			return nil, err
		}
		if sub == nil {
			return nil, errors.New("subscription ensure failed")
		}
	}
	plan, err := s.Store.Plans().Get(ctx, sub.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil {
		// Plan was removed from the DB but the subscription row still
		// references it. Fall back to the in-code canonical table so the
		// dashboard doesn't 500.
		cp := store.FindCanonicalPlan(sub.PlanID)
		if cp == nil {
			return nil, fmt.Errorf("plan %s missing", sub.PlanID)
		}
		plan = cp
	}
	// messages_used: day counter (resets at UTC midnight).
	day := store.DayKey(time.Now())
	used, err := s.Store.Usage().MessagesForDay(ctx, userID, day)
	if err != nil {
		return nil, err
	}
	// storage_bytes: sum payload bytes across the current billing period.
	// The relay is not a storage service — Redis Stream holds messages for
	// 24h — so this is "bytes transferred this period" rather than "bytes
	// at rest". It's the honest measurement given where payloads actually
	// live (on the clients).
	startDay := store.DayKey(sub.CurrentPeriodStart)
	endDay := store.DayKey(sub.CurrentPeriodEnd)
	if endDay < startDay {
		endDay = startDay
	}
	bytesUsed, err := s.Store.Usage().BytesForPeriod(ctx, userID, startDay, endDay)
	if err != nil {
		return nil, err
	}
	return &Status{
		Plan:         sub.PlanID,
		PlanName:     plan.Name,
		Status:       sub.Status,
		PeriodStart:  sub.CurrentPeriodStart.UnixMilli(),
		PeriodEnd:    sub.CurrentPeriodEnd.UnixMilli(),
		MessagesUsed: used,
		StorageBytes: bytesUsed,
		Limits:       *plan,
	}, nil
}

// ApplyWebhookUpgrade persists a subscription change from either a real
// Stripe event or the mock X-Mock-Upgrade shortcut. It's the single write
// path for /billing/webhook.
func (s *Service) ApplyWebhookUpgrade(ctx context.Context, userID, planID, status string, periodStart, periodEnd time.Time, stripeCustomer, stripeSubID string) error {
	plan, err := s.Store.Plans().Get(ctx, planID)
	if err != nil {
		return err
	}
	if plan == nil {
		return ErrUnknownPlan
	}
	if periodStart.IsZero() {
		periodStart = time.Now().UTC()
	}
	if periodEnd.IsZero() {
		periodEnd = periodStart.AddDate(0, 1, 0)
	}
	if status == "" {
		status = "active"
	}
	return s.Store.Subscriptions().SetPlan(ctx, userID, planID, status, periodStart, periodEnd, stripeCustomer, stripeSubID)
}

// VerifyStripeSignature wraps webhook.ConstructEvent so callers don't have
// to import the Stripe SDK directly. Returns the decoded event.
func (s *Service) VerifyStripeSignature(payload []byte, sigHeader string) (*stripe.Event, error) {
	if s.StripeWebhookSecret == "" {
		return nil, errors.New("webhook secret not configured")
	}
	ev, err := webhook.ConstructEvent(payload, sigHeader, s.StripeWebhookSecret)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

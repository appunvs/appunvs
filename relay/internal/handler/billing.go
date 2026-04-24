// Package handler — billing.go wires the /billing/* HTTP surface.
//
// Route layout (matches README):
//
//	GET  /billing/plans      public
//	GET  /billing/status     session-authenticated
//	POST /billing/checkout   session-authenticated
//	POST /billing/webhook    public (Stripe hits it; signature verified inline)
package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/billing"
)

// BillingDeps bundles the collaborators the billing handlers need.
// Named "BillingDeps" (not "Deps") to avoid clashing with the /ws Deps.
type BillingDeps struct {
	Signer  *auth.Signer
	Billing *billing.Service
	Log     *zap.Logger
}

// BillingPlans returns the seeded plan table. Public route — the dashboard
// hits it pre-login to render the pricing page.
func BillingPlans(d BillingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		plans, err := d.Billing.Store.Plans().List(c.Request.Context())
		if err != nil {
			d.Log.Error("billing: list plans", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"plans": plans})
	}
}

// BillingStatus returns the logged-in user's plan + usage snapshot.
func BillingStatus(d BillingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractSession(c, d.Signer)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		st, err := d.Billing.LoadStatus(c.Request.Context(), claims.UserID)
		if err != nil {
			d.Log.Error("billing: status", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, st)
	}
}

// BillingCheckout creates (or simulates) a Stripe Checkout session.
func BillingCheckout(d BillingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := extractSession(c, d.Signer)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
			return
		}
		var body struct {
			PlanID string `json:"plan_id"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.PlanID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
			return
		}
		res, err := d.Billing.CreateCheckout(c.Request.Context(), claims.UserID, body.PlanID)
		if err != nil {
			if errors.Is(err, billing.ErrUnknownPlan) {
				c.JSON(http.StatusBadRequest, gin.H{"error": "unknown plan"})
				return
			}
			d.Log.Error("billing: checkout", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, res)
	}
}

// BillingWebhook handles the Stripe webhook delivery or the mock-mode
// X-Mock-Upgrade shortcut.
//
// Real mode: verifies Stripe-Signature against billing.stripe_webhook_secret
// via webhook.ConstructEvent (constant-time HMAC compare). Dispatches on
// event type:
//
//   - checkout.session.completed   → provision subscription
//   - customer.subscription.updated → refresh plan + period + status
//   - customer.subscription.deleted → downgrade to free, status=canceled
//
// Mock mode: if billing.stripe_secret_key is empty and the request carries
// X-Mock-Upgrade: <plan_id> + X-Mock-User: <user_id>, immediately apply
// the upgrade. No signature required — mock mode is dev/test only.
func BillingWebhook(d BillingDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		payload, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
			return
		}

		if d.Billing.Mode() == "mock" {
			plan := c.GetHeader("X-Mock-Upgrade")
			uid := c.GetHeader("X-Mock-User")
			if plan == "" || uid == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "mock webhook requires X-Mock-Upgrade + X-Mock-User"})
				return
			}
			if err := d.Billing.ApplyWebhookUpgrade(c.Request.Context(), uid, plan, "active",
				time.Time{}, time.Time{}, "mock_cus_"+uid, "mock_sub_"+uid); err != nil {
				if errors.Is(err, billing.ErrUnknownPlan) {
					c.JSON(http.StatusBadRequest, gin.H{"error": "unknown plan"})
					return
				}
				d.Log.Error("billing: mock upgrade", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
			c.JSON(http.StatusOK, gin.H{"ok": true, "mode": "mock"})
			return
		}

		// Real mode: HMAC-verify the payload.
		sig := c.GetHeader("Stripe-Signature")
		ev, err := d.Billing.VerifyStripeSignature(payload, sig)
		if err != nil {
			d.Log.Info("billing: bad signature", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad signature"})
			return
		}

		switch ev.Type {
		case "checkout.session.completed":
			// The session's metadata.plan_id + client_reference_id carry
			// the user id we stamped in CreateCheckout.
			var sess struct {
				ClientReferenceID string            `json:"client_reference_id"`
				Customer          string            `json:"customer"`
				Subscription      string            `json:"subscription"`
				Metadata          map[string]string `json:"metadata"`
			}
			if err := json.Unmarshal(ev.Data.Raw, &sess); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad event payload"})
				return
			}
			planID := sess.Metadata["plan_id"]
			userID := sess.ClientReferenceID
			if userID == "" {
				userID = sess.Metadata["user_id"]
			}
			if userID == "" || planID == "" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "missing metadata"})
				return
			}
			if err := d.Billing.ApplyWebhookUpgrade(c.Request.Context(), userID, planID, "active",
				time.Time{}, time.Time{}, sess.Customer, sess.Subscription); err != nil {
				d.Log.Error("billing: apply checkout", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
		case "customer.subscription.updated":
			var sub struct {
				Customer           string            `json:"customer"`
				ID                 string            `json:"id"`
				Status             string            `json:"status"`
				CurrentPeriodStart int64             `json:"current_period_start"`
				CurrentPeriodEnd   int64             `json:"current_period_end"`
				Metadata           map[string]string `json:"metadata"`
			}
			if err := json.Unmarshal(ev.Data.Raw, &sub); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad event payload"})
				return
			}
			existing, err := d.Billing.Store.Subscriptions().FindByStripeCustomer(c.Request.Context(), sub.Customer)
			if err != nil {
				d.Log.Error("billing: lookup customer", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
			if existing == nil {
				d.Log.Info("billing: subscription.updated for unknown customer", zap.String("customer", sub.Customer))
				c.JSON(http.StatusOK, gin.H{"ok": true, "note": "unknown customer"})
				return
			}
			planID := existing.PlanID
			if p := sub.Metadata["plan_id"]; p != "" {
				planID = p
			}
			periodStart := time.Unix(sub.CurrentPeriodStart, 0).UTC()
			periodEnd := time.Unix(sub.CurrentPeriodEnd, 0).UTC()
			if err := d.Billing.ApplyWebhookUpgrade(c.Request.Context(), existing.UserID, planID, sub.Status,
				periodStart, periodEnd, sub.Customer, sub.ID); err != nil {
				d.Log.Error("billing: apply sub.updated", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
		case "customer.subscription.deleted":
			var sub struct {
				Customer string `json:"customer"`
				ID       string `json:"id"`
			}
			if err := json.Unmarshal(ev.Data.Raw, &sub); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "bad event payload"})
				return
			}
			existing, err := d.Billing.Store.Subscriptions().FindByStripeCustomer(c.Request.Context(), sub.Customer)
			if err != nil {
				d.Log.Error("billing: lookup customer (deleted)", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
			if existing == nil {
				c.JSON(http.StatusOK, gin.H{"ok": true, "note": "unknown customer"})
				return
			}
			now := time.Now().UTC()
			if err := d.Billing.ApplyWebhookUpgrade(c.Request.Context(), existing.UserID, "free", "canceled",
				now, now.AddDate(0, 1, 0), sub.Customer, sub.ID); err != nil {
				d.Log.Error("billing: apply sub.deleted", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}
		default:
			// Ignore unhandled event types but 200 so Stripe stops retrying.
			d.Log.Info("billing: ignoring webhook event", zap.String("type", string(ev.Type)))
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	}
}

// extractSession is defined in ws-sibling handlers; duplicate a minimal
// version here to avoid leaking auth-package internals. Kept in sync
// with auth.extractSession (the package-internal counterpart).
func extractSession(c *gin.Context, signer *auth.Signer) (*auth.Claims, error) {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return nil, errors.New("missing Authorization bearer token")
	}
	claims, err := signer.Verify(strings.TrimPrefix(h, "Bearer "), auth.TokenSession)
	if err != nil {
		return nil, errors.New("invalid session token")
	}
	return claims, nil
}

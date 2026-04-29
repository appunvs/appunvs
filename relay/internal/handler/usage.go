// Package handler — GET /usage/me reports the caller's current plan
// quota state.  Returned shape is a stable contract for the host
// shell's Profile screen ("本周 N/M · 本月 N/M") and any future status
// UI; bump version on any breaking change.
//
// This endpoint is independent of cfg.Pricing.Enabled — the quota
// gate may be off (dogfood) but the UI still wants to render
// consumption numbers.  Auth is the same device-JWT gate /ai/turn
// uses; ownership is implicit (you only ever read your own usage).
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/usage"
)

// UsageDeps groups everything GET /usage/me depends on.  Quota and
// PlanFor are required (unlike AIDeps where they're optional) — this
// route's whole purpose is to surface consumption.
type UsageDeps struct {
	Signer  *auth.Signer
	Quota   *usage.Quota
	PlanFor func(userID string) usage.Plan
	Log     *zap.Logger
}

// RegisterUsageRoutes wires GET /usage/me.  The route requires a
// device JWT; the response is the caller's own usage and never
// reveals other users' state.
func RegisterUsageRoutes(r gin.IRouter, d UsageDeps) {
	r.GET("/usage/me", usageMe(d))
}

// usageMeResponse is the JSON payload.  Numbers are int64 because
// SQLite's COUNT(*) returns int64 and the JSON encoder follows suit;
// host-shell decoders should mirror.
type usageMeResponse struct {
	Plan   usagePlanField   `json:"plan"`
	Used   usageUsedField   `json:"used"`
	Limits usageLimitsField `json:"limits"`
}

type usagePlanField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type usageUsedField struct {
	// Last5d is the rolling-5-day turn count (the gate's primary
	// window).  Always present even when Plan.TurnsPer5d == -1
	// (Max+ / BYOK) so the UI can render a "lifetime burst" bar.
	Last5d int64 `json:"last_5d"`
	// ThisMonth is the calendar-month turn count.
	ThisMonth int64 `json:"this_month"`
}

type usageLimitsField struct {
	// -1 sentinel propagates straight through to the client.  The
	// host-shell decoder treats negative values as "unlimited" and
	// renders them as "—" or hides the bar entirely.
	TurnsPer5d        int `json:"turns_per_5d"`
	TurnsPerMonth     int `json:"turns_per_month"`
	ActiveBoxes       int `json:"active_boxes"`
	PublishesPerMonth int `json:"publishes_per_month"`
}

func usageMe(d UsageDeps) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := requireDevice(c, d.Signer)
		if !ok {
			return
		}
		plan := d.PlanFor(claims.UserID)
		used, err := d.Quota.Used(c.Request.Context(), claims.UserID)
		if err != nil {
			d.Log.Error("usage.me: lookup", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
			return
		}
		c.JSON(http.StatusOK, usageMeResponse{
			Plan: usagePlanField{
				ID:    string(plan.ID),
				Label: plan.Label,
			},
			Used: usageUsedField{
				Last5d:    used.Last5d,
				ThisMonth: used.ThisMonth,
			},
			Limits: usageLimitsField{
				TurnsPer5d:        plan.TurnsPer5d,
				TurnsPerMonth:     plan.TurnsPerMonth,
				ActiveBoxes:       plan.ActiveBoxes,
				PublishesPerMonth: plan.PublishesPerMonth,
			},
		})
	}
}

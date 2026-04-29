// Package handler — GET /usage/me reports the caller's current plan
// quota state across the three v3 dimensions: LLM (RMB cents),
// Sandbox (build runs / month), Storage (active boxes).
//
// Returned shape is a stable contract for the host shell's Profile
// screen progress bars and any future status UI; bump version on any
// breaking change.
//
// Independent of cfg.Pricing.Enabled — the gate may be off (dogfood)
// but the UI still wants to render consumption numbers.  Auth is the
// same device-JWT gate /ai/turn uses; ownership is implicit.
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/auth"
	"github.com/appunvs/appunvs/relay/internal/usage"
)

// UsageDeps groups everything GET /usage/me depends on.
type UsageDeps struct {
	Signer  *auth.Signer
	Quota   *usage.Quota
	PlanFor func(userID string) usage.Plan
	Log     *zap.Logger
}

// RegisterUsageRoutes wires GET /usage/me.
func RegisterUsageRoutes(r gin.IRouter, d UsageDeps) {
	r.GET("/usage/me", usageMe(d))
}

// usageMeResponse is the JSON payload.  Three top-level dimensions
// — LLM / Sandbox / Storage — each with a `used` and a `limit`.
//
// Limits of -1 mean unlimited (Max+ unlimited 5d window, BYOK both
// LLM windows, Max+ unlimited boxes); host-shell decoders treat
// negative values as "no cap" and render "—" or hide the bar.
type usageMeResponse struct {
	Plan    usagePlanField    `json:"plan"`
	LLM     usageLLMField     `json:"llm"`
	Sandbox usageSandboxField `json:"sandbox"`
	Storage usageStorageField `json:"storage"`
}

type usagePlanField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// usageLLMField surfaces token cost — both windows side by side, in
// RMB cents.  UI will typically show 5d as the primary bar and month
// as a secondary line ("本月 ¥X / ¥Y").
type usageLLMField struct {
	UsedCentsLast5d    int64 `json:"used_cents_last_5d"`
	UsedCentsThisMonth int64 `json:"used_cents_this_month"`
	BudgetCentsPer5d   int   `json:"budget_cents_per_5d"`
	BudgetCentsPerMonth int  `json:"budget_cents_per_month"`
}

type usageSandboxField struct {
	UsedThisMonth int64 `json:"used_this_month"`
	PerMonth      int   `json:"per_month"`
}

type usageStorageField struct {
	Active      int64 `json:"active"`
	ActiveLimit int   `json:"active_limit"`
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
			LLM: usageLLMField{
				UsedCentsLast5d:     used.LLMSpentCentsLast5d,
				UsedCentsThisMonth:  used.LLMSpentCentsThisMonth,
				BudgetCentsPer5d:    plan.LLMBudgetCentsPer5d,
				BudgetCentsPerMonth: plan.LLMBudgetCentsPerMonth,
			},
			Sandbox: usageSandboxField{
				UsedThisMonth: used.SandboxRunsThisMonth,
				PerMonth:      plan.SandboxRunsPerMonth,
			},
			Storage: usageStorageField{
				Active:      used.ActiveBoxes,
				ActiveLimit: plan.ActiveBoxes,
			},
		})
	}
}

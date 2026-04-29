// Package box — plan-quota gates on the Sandbox / Storage dimensions.
//
// LLM gating lives in the AI handler (handler/ai.go) since it's
// per-turn and the engine layer never sees the quota service
// directly.  Sandbox and Storage gates fire *inside the service*
// instead of at the HTTP layer because they have to apply to both
// HTTP callers and the AI agent's publish_box tool — wiring at the
// service level catches both call paths with one diff.
//
// The gate is optional (nil Quota or nil PlanFor disables it) so
// dev / CI / dogfood paths don't bounce off Free-tier caps that were
// designed for production.
package box

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appunvs/appunvs/relay/internal/usage"
)

// ErrPlanExhausted is returned by Service.Create / BuildAndPublish
// when the caller's plan-quota gate denies the action.  Callers map
// this to HTTP 429 (handlers) or a structured tool result frame
// (publish_box tool) so the client can render the right message.
type ErrPlanExhausted struct {
	// Window names which dimension fired ("sandbox" or "storage").
	Window string
	// Plan is the caller's tier id at the time of the deny.
	Plan string
	// RetryAfter is how long until the limiting window resets.
	// Zero for storage caps — the user has to free up a slot, not wait.
	RetryAfter time.Duration
}

// Error implements error.
func (e ErrPlanExhausted) Error() string {
	return fmt.Sprintf("plan_exhausted: %s (plan=%s)", e.Window, e.Plan)
}

// AsPlanExhausted is a helper for callers that need to extract the
// structured fields from a wrapped error.  Returns (zero, false) when
// the error chain doesn't contain an ErrPlanExhausted.
func AsPlanExhausted(err error) (ErrPlanExhausted, bool) {
	var target ErrPlanExhausted
	if errors.As(err, &target) {
		return target, true
	}
	return ErrPlanExhausted{}, false
}

// gateSandbox runs CheckSandbox if the service has a quota wired.
// Returns nil to allow, ErrPlanExhausted to deny, any other error on
// internal failure (treat as deny on the safe side).
func (s *Service) gateSandbox(ctx context.Context, namespace string) error {
	if s.Quota == nil || s.PlanFor == nil {
		return nil
	}
	plan := s.PlanFor(namespace)
	dec, err := s.Quota.CheckSandbox(ctx, namespace, plan)
	if err != nil {
		return fmt.Errorf("quota check sandbox: %w", err)
	}
	if dec.Allowed {
		return nil
	}
	return ErrPlanExhausted{
		Window:     string(usage.WindowSandbox),
		Plan:       string(plan.ID),
		RetryAfter: dec.RetryAfter,
	}
}

// gateStorage runs CheckStorage if the service has a quota wired.
func (s *Service) gateStorage(ctx context.Context, namespace string) error {
	if s.Quota == nil || s.PlanFor == nil {
		return nil
	}
	plan := s.PlanFor(namespace)
	dec, err := s.Quota.CheckStorage(ctx, namespace, plan)
	if err != nil {
		return fmt.Errorf("quota check storage: %w", err)
	}
	if dec.Allowed {
		return nil
	}
	return ErrPlanExhausted{
		Window:     string(usage.WindowStorage),
		Plan:       string(plan.ID),
		RetryAfter: dec.RetryAfter,
	}
}

// Package usage tracks per-user platform consumption against
// subscription tier caps.
//
// v3 adopts a three-quantum model — each subscription tier exposes
// independent caps on three orthogonal cost lines:
//
//   - LLM    — token cost, billed in RMB cents (real cost × markup)
//   - Sandbox — number of sandbox build runs (each publish_box → 1)
//   - Storage — number of active boxes (proxy for object-store + git
//     workspace footprint)
//
// Each line is gated independently: hitting the LLM cap pauses /ai/turn
// without blocking sandbox builds, and so on.  Whichever fires first
// returns its own 429 hint so the client can render the right message.
//
// LLM is metered by real cost so a 100K-token Opus turn deducts ~30×
// the budget of a 3K-token DeepSeek turn — see pricing.go for the
// reference price table and markup.  No fixed-integer multiplier per
// model: token amplification + cross-model variance is captured by
// the same math.
//
// BYOK: users with their own provider key bypass the LLM cap (LLM
// cost lands on their account, not ours), but still consume Sandbox /
// Storage which are platform-side resources.  See docs/pricing-strategy.md.
package usage

// PlanID is the canonical id used in the users table once Phase D
// wires Stripe entitlement.
type PlanID string

const (
	PlanFree    PlanID = "free"
	PlanPro     PlanID = "pro"
	PlanMax     PlanID = "max"
	PlanMaxPlus PlanID = "max_plus"
	PlanBYOK    PlanID = "byok"
)

// Plan captures the quota knobs for one subscription tier.  -1 means
// "unlimited for this dimension"; 0 is reserved (would block every
// request, so use -1 instead when you want no cap).
//
// Numbers mirror docs/pricing-strategy.md.  Adjusting them post-dogfood
// is a code change here, not a schema change.
type Plan struct {
	ID    PlanID
	Label string

	// LLM (RMB cents).  Two windows so a single burst can't drain
	// the month and a determined user can't saturate every 5d
	// bucket.  Whichever fires first pauses /ai/turn until that
	// window resets.
	LLMBudgetCentsPer5d    int
	LLMBudgetCentsPerMonth int

	// Sandbox: number of sandbox build runs (every publish_box) per
	// calendar month.  Per-month only — builds are predictable
	// per-call (~¥0.10) and don't need a rolling window.
	SandboxRunsPerMonth int

	// Storage: maximum active (non-archived) boxes the user may own
	// at any one time.  Hard cap, not a count over a window.
	ActiveBoxes int
}

// Plans is the authoritative tier registry.  Keep entries in price
// order so iteration in any UI surface (status page, Profile screen)
// renders top-to-bottom from cheapest.
//
// LLM caps:
//   - Pro ¥30/month: ¥15 included LLM budget at 5d window primary
//     (¥3 / day) + ¥30/month fallback.  At DeepSeek prices this is
//     ~¥15 / 0.05 = ~300 turn; at Opus it's ~¥15 / 0.7 ≈ 20 turn.
//     Markup-inclusive cost.
//   - BYOK uses -1 sentinel for LLM (cost lands on user's own key).
//
// Sandbox / Storage:
//   - Free: 5 builds, 1 box (a-ha moment available; can't bulk-publish)
//   - Pro: 50 builds, 5 boxes
//   - Max: 200 builds, 20 boxes
//   - Max+: 500 builds, unlimited boxes
//   - BYOK: same Sandbox / Storage as Pro
var Plans = map[PlanID]Plan{
	PlanFree: {
		ID:                     PlanFree,
		Label:                  "Free",
		LLMBudgetCentsPer5d:    50,  // ¥0.50 — a few real turns to taste
		LLMBudgetCentsPerMonth: 100, // ¥1.00
		SandboxRunsPerMonth:    5,
		ActiveBoxes:            1,
	},
	PlanPro: {
		ID:                     PlanPro,
		Label:                  "Pro",
		LLMBudgetCentsPer5d:    1500, // ¥15
		LLMBudgetCentsPerMonth: 3000, // ¥30
		SandboxRunsPerMonth:    50,
		ActiveBoxes:            5,
	},
	PlanMax: {
		ID:                     PlanMax,
		Label:                  "Max",
		LLMBudgetCentsPer5d:    4000, // ¥40
		LLMBudgetCentsPerMonth: 8000, // ¥80
		SandboxRunsPerMonth:    200,
		ActiveBoxes:            20,
	},
	PlanMaxPlus: {
		ID:                     PlanMaxPlus,
		Label:                  "Max+",
		LLMBudgetCentsPer5d:    -1,    // unlimited 5d — only month gates
		LLMBudgetCentsPerMonth: 20000, // ¥200
		SandboxRunsPerMonth:    500,
		ActiveBoxes:            -1,
	},
	PlanBYOK: {
		ID:    PlanBYOK,
		Label: "BYOK",
		// LLM-side caps don't apply — engine routes to user's own
		// API key and skips the LLM cap.  We still keep Sandbox /
		// Storage on Pro-level numbers to recover platform cost.
		LLMBudgetCentsPer5d:    -1,
		LLMBudgetCentsPerMonth: -1,
		SandboxRunsPerMonth:    50,
		ActiveBoxes:            5,
	},
}

// PlanFor returns the Plan struct for id, falling back to Free when
// id isn't recognized — keeps the gate behavior conservative if a
// stale users.plan value points at a tier we've since renamed.
func PlanFor(id PlanID) Plan {
	if p, ok := Plans[id]; ok {
		return p
	}
	return Plans[PlanFree]
}

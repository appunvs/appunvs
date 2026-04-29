// Package usage tracks per-user AI turn consumption against subscription
// quotas.
//
// v0 implements two rolling windows mirroring docs/pricing-strategy.md:
//
//   - 5d rolling: primary cap, prevents a single burst from emptying the
//     month's allowance and stranding the user for weeks
//   - Calendar month: fallback cap, stops a determined user from
//     saturating every 5d window
//
// Whichever window fires first pauses the user's /ai/turn calls until
// that window resets.  Plans live in code (Plan); the user → plan
// binding is owned by store.Users post-Phase D (Stripe webhook).  For
// now everyone is treated as Free unless caller passes an explicit
// override — keeps Phase A test-only without touching billing.
//
// BYOK users bypass turn quotas entirely (their LLM cost is on them);
// they still consume sandbox / publish quotas — those land in a
// separate Phase B work item.
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
// Numbers mirror docs/pricing-strategy.md v0 draft.  Adjusting them
// post-dogfood is a code change here, not a schema change.
type Plan struct {
	ID                PlanID
	Label             string // display name, used by the per-tier 429 hint
	TurnsPer5d        int    // primary rate-limit window
	TurnsPerMonth     int    // fallback safety net
	ActiveBoxes       int    // independent hard cap; not enforced in Phase A
	PublishesPerMonth int    // independent hard cap; not enforced in Phase A
}

// Plans is the authoritative tier registry.  Keep entries in price
// order so iteration in any UI surface (status page, Profile screen)
// renders top-to-bottom from cheapest.
var Plans = map[PlanID]Plan{
	PlanFree: {
		ID:                PlanFree,
		Label:             "Free",
		TurnsPer5d:        20,
		TurnsPerMonth:     50,
		ActiveBoxes:       1,
		PublishesPerMonth: 5,
	},
	PlanPro: {
		ID:                PlanPro,
		Label:             "Pro",
		TurnsPer5d:        200,
		TurnsPerMonth:     500,
		ActiveBoxes:       5,
		PublishesPerMonth: 50,
	},
	PlanMax: {
		ID:                PlanMax,
		Label:             "Max",
		TurnsPer5d:        800,
		TurnsPerMonth:     2000,
		ActiveBoxes:       20,
		PublishesPerMonth: 200,
	},
	PlanMaxPlus: {
		ID:                PlanMaxPlus,
		Label:             "Max+",
		TurnsPer5d:        -1, // unlimited 5d — the doc says only month gates Max+
		TurnsPerMonth:     5000,
		ActiveBoxes:       -1,
		PublishesPerMonth: 500,
	},
	PlanBYOK: {
		ID:    PlanBYOK,
		Label: "BYOK",
		// LLM-side quotas don't apply to BYOK users — the engine layer
		// short-circuits the gate when User.HasOwnKey is true.  We
		// still keep the struct populated with sentinel -1s so a future
		// refactor doesn't accidentally treat zero as "no quota".
		TurnsPer5d:        -1,
		TurnsPerMonth:     -1,
		ActiveBoxes:       5,
		PublishesPerMonth: 50,
	},
}

// PlanFor returns the Plan struct for id, falling back to Free when id
// isn't recognized — keeps the gate behavior conservative if a stale
// users.plan value points at a tier we've since renamed.
func PlanFor(id PlanID) Plan {
	if p, ok := Plans[id]; ok {
		return p
	}
	return Plans[PlanFree]
}

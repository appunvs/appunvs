// Pricing — model token-cost reference table.
//
// Each entry maps a model id (the same string callers pass on
// /ai/turn's `model` field) to its provider's headline pricing.
// Numbers are in RMB per 1M tokens, taken from each provider's public
// pricing page; refresh quarterly or when a provider announces a price
// change.  Out-of-table models fall back to the deepseek-chat row so
// missing-entry bugs don't silently bill at zero.
//
// Cost is computed at turn-completion time and persisted on the
// ai_turns row alongside tokens_in / tokens_out, so historical
// quota math stays stable even if the table changes later.
package usage

import (
	"math"
)

// ModelPrice captures one model's per-token prices.  Input and output
// tokens are billed differently by every provider; output is always
// the more expensive direction.
type ModelPrice struct {
	// InRMBPer1M is RMB charged for 1,000,000 input tokens.
	InRMBPer1M float64
	// OutRMBPer1M is RMB charged for 1,000,000 output tokens.
	OutRMBPer1M float64
}

// ModelPricing is the authoritative table.  Update on quarterly
// review or when a provider announces a price change.  Keys MUST
// match the model ids the engines emit (see engines' default Model
// + the host-shell ModelCatalog).
var ModelPricing = map[string]ModelPrice{
	// DeepSeek (RMB-native pricing on api.deepseek.com)
	"deepseek-chat":     {InRMBPer1M: 1.0, OutRMBPer1M: 2.0},
	"deepseek-v4-pro":   {InRMBPer1M: 2.0, OutRMBPer1M: 8.0},

	// OpenAI (USD prices × 7.2 ≈ RMB at 2026 rates; refresh on FX swing)
	"gpt-4o":       {InRMBPer1M: 18.0, OutRMBPer1M: 72.0},
	"gpt-5.5":      {InRMBPer1M: 25.0, OutRMBPer1M: 100.0},
	"gpt-5.4-mini": {InRMBPer1M: 3.0, OutRMBPer1M: 12.0},

	// Anthropic
	"claude-sonnet-4-0": {InRMBPer1M: 22.0, OutRMBPer1M: 110.0},
	"claude-sonnet-4-6": {InRMBPer1M: 22.0, OutRMBPer1M: 110.0},
	"claude-opus-4-1":   {InRMBPer1M: 110.0, OutRMBPer1M: 550.0},
	"claude-opus-4-7":   {InRMBPer1M: 110.0, OutRMBPer1M: 550.0},

	// Google Gemini
	"gemini-2.5-pro": {InRMBPer1M: 9.0, OutRMBPer1M: 72.0},

	// Chinese providers (all RMB-native)
	"kimi-k2.6":         {InRMBPer1M: 1.5, OutRMBPer1M: 6.0},
	"glm-4.7":           {InRMBPer1M: 0.5, OutRMBPer1M: 1.5},
	"qwen3-coder-next":  {InRMBPer1M: 4.0, OutRMBPer1M: 16.0},
	"MiniMax-M2.7":      {InRMBPer1M: 8.0, OutRMBPer1M: 24.0},
}

// Markup is the fraction of platform fee added on top of raw token
// cost.  0.20 = 20%; covers sandbox / relay / storage / support
// burdens that aren't billed separately.
//
// Kept as a single number so quarterly margin tuning is one diff.
const Markup = 0.20

// CostCents returns the RMB-cents cost of one turn's token usage,
// including platform markup.  Unknown model ids fall back to the
// deepseek-chat row so we don't silently bill at zero on a typo.
//
// Always returns >= 1 cent for non-empty inputs; we don't want
// successful turns to free-ride on rounding.
func CostCents(model string, tokensIn, tokensOut int64) int64 {
	if tokensIn <= 0 && tokensOut <= 0 {
		return 0
	}
	p, ok := ModelPricing[model]
	if !ok {
		p = ModelPricing["deepseek-chat"]
	}
	rmb := (float64(tokensIn)/1_000_000.0)*p.InRMBPer1M +
		(float64(tokensOut)/1_000_000.0)*p.OutRMBPer1M
	rmb *= 1.0 + Markup
	cents := int64(math.Ceil(rmb * 100))
	if cents < 1 {
		cents = 1
	}
	return cents
}

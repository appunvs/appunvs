// Provider registry — curated defaults for the OpenAI-compatible endpoints
// the engine supports out of the box.  Adding a new provider is one struct
// literal here; the engine code is provider-agnostic and reaches for these
// values only when a user sets `Config.Provider` by id.
//
// Scope of this file:
//   - DeepSeek (default) · api.deepseek.com
//   - Volcengine Ark (火山方舟) · ark.cn-beijing.volces.com
//   - Moonshot (Kimi)   · api.moonshot.cn
//   - Zhipu (GLM)       · open.bigmodel.cn
//   - Dashscope (Qwen)  · dashscope.aliyuncs.com
//
// Any other OpenAI-compatible endpoint is still reachable by leaving
// `Provider` empty and setting `BaseURL` + `Model` explicitly — the
// registry is a convenience, not a gate.
package ai

import (
	"errors"
	"fmt"
	"sort"
)

// Provider describes one OpenAI-compatible LLM backend.  All fields are
// defaults; any of `BaseURL` / `ModelChat` / `ModelReason` on a caller's
// `Config` override these without touching the registry.
type Provider struct {
	ID          string // lookup key: "deepseek", "volcengine", ...
	Name        string // human display name
	BaseURL     string
	ModelChat   string // default general-purpose chat model (empty if the provider has no useful public default)
	ModelReason string // optional reasoning / thinking variant
	DocsURL     string
	Note        string // any pitfall worth surfacing in config help
	EnvAPIKey   string // conventional env var name users will already have set (e.g. DEEPSEEK_API_KEY)
}

// Providers is the canonical registry.  Keys match what a user passes to
// `Config.Provider` (and what `ai.backend` in config.yaml accepts).  Keep
// lowercase ASCII for the id; case-sensitive matching on purpose to
// avoid silent typos.
var Providers = map[string]Provider{
	"deepseek": {
		ID:          "deepseek",
		Name:        "DeepSeek",
		BaseURL:     "https://api.deepseek.com/v1",
		ModelChat:   "deepseek-chat",
		ModelReason: "deepseek-reasoner",
		DocsURL:     "https://api-docs.deepseek.com",
		EnvAPIKey:   "DEEPSEEK_API_KEY",
	},
	"volcengine": {
		ID:      "volcengine",
		Name:    "Volcengine Ark (火山方舟)",
		BaseURL: "https://ark.cn-beijing.volces.com/api/v3",
		// Ark routes via per-tenant "接入点 / endpoint" IDs (ep-YYYYMMDD-xyz)
		// the caller must register in the console.  No useful chat default —
		// set APPUNVS_AI_MODEL to the endpoint id after creating it.
		ModelChat: "",
		DocsURL:   "https://www.volcengine.com/docs/82379/1330310",
		Note:      "Ark uses per-account endpoint IDs (ep-…). Create an endpoint in the console and set Model to that id.",
		EnvAPIKey: "ARK_API_KEY",
	},
	"moonshot": {
		ID:          "moonshot",
		Name:        "Moonshot (Kimi)",
		BaseURL:     "https://api.moonshot.cn/v1",
		ModelChat:   "kimi-k2-turbo-preview", // long-context, tool_call friendly
		ModelReason: "",
		DocsURL:     "https://platform.moonshot.cn/docs",
		EnvAPIKey:   "MOONSHOT_API_KEY",
	},
	"zhipu": {
		ID:          "zhipu",
		Name:        "Zhipu GLM",
		BaseURL:     "https://open.bigmodel.cn/api/paas/v4",
		ModelChat:   "glm-4.6",
		ModelReason: "glm-4.6", // unified model; pass `"thinking": {"type": "enabled"}` when wanted
		DocsURL:     "https://open.bigmodel.cn/dev/api",
		EnvAPIKey:   "ZHIPU_API_KEY",
	},
	"dashscope": {
		ID:        "dashscope",
		Name:      "Dashscope (Alibaba Qwen)",
		BaseURL:   "https://dashscope.aliyuncs.com/compatible-mode/v1",
		ModelChat: "qwen3-coder-plus",
		DocsURL:   "https://help.aliyun.com/zh/model-studio",
		EnvAPIKey: "DASHSCOPE_API_KEY",
	},
}

// ErrUnknownProvider is returned by Resolve when the id isn't registered.
// Callers may surface this as a config error without leaking the full map.
var ErrUnknownProvider = errors.New("ai: unknown provider id")

// Resolve returns the Provider for id, or ErrUnknownProvider.  Intended
// for config loaders that want to fail fast at startup rather than at
// first turn.
func Resolve(id string) (Provider, error) {
	p, ok := Providers[id]
	if !ok {
		return Provider{}, fmt.Errorf("%w: %q (known: %v)", ErrUnknownProvider, id, knownProviderIDs())
	}
	return p, nil
}

// knownProviderIDs returns a stable-sorted slice of ids for error messages.
func knownProviderIDs() []string {
	ids := make([]string, 0, len(Providers))
	for id := range Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

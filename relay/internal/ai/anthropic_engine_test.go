package ai_test

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
)

// TestNewAnthropicEngineRequiresAPIKey: no key, no engine.  Mirrors the
// OpenAI side — fail loud at startup rather than at first turn.
func TestNewAnthropicEngineRequiresAPIKey(t *testing.T) {
	_, err := ai.NewAnthropicEngine(ai.AnthropicConfig{}, nil, nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when APIKey missing")
	}
	if !strings.Contains(err.Error(), "APIKey") {
		t.Fatalf("error %q should mention APIKey", err)
	}
}

// TestNewAnthropicEngineDefaultsModel: caller doesn't have to pick a
// model — engine picks the most-capable default.  Useful for first-run
// smoke without scrolling the SDK's model id list.
func TestNewAnthropicEngineDefaultsModel(t *testing.T) {
	eng, err := ai.NewAnthropicEngine(ai.AnthropicConfig{
		APIKey: "sk-test",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if eng == nil {
		t.Fatal("engine is nil")
	}
}

// TestNewAnthropicEngineHonorsExplicitModel: caller can override the
// default; explicit Model wins.
func TestNewAnthropicEngineHonorsExplicitModel(t *testing.T) {
	_, err := ai.NewAnthropicEngine(ai.AnthropicConfig{
		APIKey: "sk-test",
		Model:  "claude-3-5-haiku-latest",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
}

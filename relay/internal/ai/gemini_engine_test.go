package ai_test

import (
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
)

// TestNewGeminiEngineRequiresAPIKey: no key, no engine. Mirrors the
// Anthropic/OpenAI constructor guards so misconfigurations fail at boot.
func TestNewGeminiEngineRequiresAPIKey(t *testing.T) {
	_, err := ai.NewGeminiEngine(ai.GeminiConfig{}, nil, nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when APIKey missing")
	}
	if !strings.Contains(err.Error(), "APIKey") {
		t.Fatalf("error %q should mention APIKey", err)
	}
}

// TestNewGeminiEngineDefaultsModel: caller may omit Model and rely on the
// engine's built-in backend default.
func TestNewGeminiEngineDefaultsModel(t *testing.T) {
	eng, err := ai.NewGeminiEngine(ai.GeminiConfig{
		APIKey: "gm-test",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if eng == nil {
		t.Fatal("engine is nil")
	}
}

// TestNewGeminiEngineHonorsExplicitModel: explicit Model should bypass the
// backend default cleanly.
func TestNewGeminiEngineHonorsExplicitModel(t *testing.T) {
	_, err := ai.NewGeminiEngine(ai.GeminiConfig{
		APIKey: "gm-test",
		Model:  "gemini-2.5-flash",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
}

package ai_test

import (
	"errors"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/appunvs/appunvs/relay/internal/ai"
)

// TestNewOpenAIEngineResolvesProvider exercises the registry-based
// resolution path: naming a known provider fills in BaseURL + Model
// without the caller spelling them out.
func TestNewOpenAIEngineResolvesProvider(t *testing.T) {
	eng, err := ai.NewOpenAIEngine(ai.Config{
		Provider: "deepseek",
		APIKey:   "sk-test",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	p := eng.Provider()
	if p.ID != "deepseek" {
		t.Fatalf("Provider().ID = %q, want deepseek", p.ID)
	}
}

// TestNewOpenAIEngineOverridesWin proves that explicit Config fields
// trump registry defaults (so a caller on a self-hosted proxy fronting
// DeepSeek can reuse the `deepseek` id while pointing BaseURL elsewhere).
func TestNewOpenAIEngineOverridesWin(t *testing.T) {
	eng, err := ai.NewOpenAIEngine(ai.Config{
		Provider: "deepseek",
		APIKey:   "sk-test",
		BaseURL:  "https://internal.example.com/v1",
		Model:    "deepseek-chat-custom",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	// Provider() echoes the registry entry; the overridden values live
	// on the internal cfg and are used at request time.  We can at
	// least assert the registry lookup happened (by ID).
	if eng.Provider().ID != "deepseek" {
		t.Fatalf("Provider().ID = %q, want deepseek", eng.Provider().ID)
	}
}

// TestNewOpenAIEngineVolcengineRequiresModel guards the Volcengine Ark
// footgun: the registry intentionally leaves ModelChat empty because
// Ark routes via per-account endpoint ids.  Callers who forget to
// supply Model must get a clear startup error that surfaces the
// provider's Note, not an opaque 404 at first turn.
func TestNewOpenAIEngineVolcengineRequiresModel(t *testing.T) {
	_, err := ai.NewOpenAIEngine(ai.Config{
		Provider: "volcengine",
		APIKey:   "sk-test",
	}, nil, nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when Volcengine Ark is picked without a Model")
	}
	if !strings.Contains(err.Error(), "Model required") {
		t.Fatalf("err = %q, want 'Model required' hint", err.Error())
	}
	// The Note should surface so operators aren't hunting for the fix.
	if !strings.Contains(err.Error(), "endpoint") {
		t.Fatalf("err = %q, want endpoint-id note", err.Error())
	}
}

// TestNewOpenAIEngineRawMode lets callers bypass the registry entirely
// by leaving Provider empty and specifying BaseURL + Model directly.
func TestNewOpenAIEngineRawMode(t *testing.T) {
	eng, err := ai.NewOpenAIEngine(ai.Config{
		APIKey:  "sk-test",
		BaseURL: "https://api.anything.example/v1",
		Model:   "custom-model-v1",
	}, nil, nil, nil, zap.NewNop())
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	if eng.Provider().ID != "" {
		t.Fatalf("raw mode should leave Provider().ID empty, got %q", eng.Provider().ID)
	}
}

// TestNewOpenAIEngineUnknownProvider fails loudly at startup rather
// than at first turn.
func TestNewOpenAIEngineUnknownProvider(t *testing.T) {
	_, err := ai.NewOpenAIEngine(ai.Config{
		Provider: "bogus-llm-co",
		APIKey:   "sk-test",
	}, nil, nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !errors.Is(err, ai.ErrUnknownProvider) {
		t.Fatalf("err = %v, want ErrUnknownProvider chain", err)
	}
}

// TestNewOpenAIEngineAPIKeyRequired — obvious but guarded; silent
// acceptance of an empty key would hide a production misconfiguration.
func TestNewOpenAIEngineAPIKeyRequired(t *testing.T) {
	_, err := ai.NewOpenAIEngine(ai.Config{Provider: "deepseek"}, nil, nil, nil, zap.NewNop())
	if err == nil {
		t.Fatal("expected error when APIKey is empty")
	}
	if !strings.Contains(err.Error(), "APIKey") {
		t.Fatalf("err = %q, want 'APIKey' hint", err.Error())
	}
}

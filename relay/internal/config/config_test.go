package config_test

import (
	"testing"

	"github.com/appunvs/appunvs/relay/internal/config"
)

// TestLoadLeavesAIModelUnsetByDefault ensures backend-specific defaults are
// applied by the engine layer instead of being shadowed by a global model id.
func TestLoadLeavesAIModelUnsetByDefault(t *testing.T) {
	cfg, err := config.Load("definitely-missing-config.yaml")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.AI.Model != "" {
		t.Fatalf("AI.Model = %q, want empty to allow backend defaults", cfg.AI.Model)
	}
}

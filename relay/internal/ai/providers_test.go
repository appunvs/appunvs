package ai_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/appunvs/appunvs/relay/internal/ai"
)

// TestProviderRegistryShape locks the minimum contract every registered
// provider must satisfy.  A regression here means someone added a provider
// entry without filling in the required fields — callers would then get
// opaque errors at runtime instead of startup.
func TestProviderRegistryShape(t *testing.T) {
	if len(ai.Providers) == 0 {
		t.Fatal("Providers registry is empty")
	}
	for id, p := range ai.Providers {
		if p.ID != id {
			t.Errorf("provider key %q disagrees with struct ID %q", id, p.ID)
		}
		if p.Name == "" {
			t.Errorf("provider %q missing Name", id)
		}
		if !strings.HasPrefix(p.BaseURL, "https://") {
			t.Errorf("provider %q BaseURL %q must be HTTPS", id, p.BaseURL)
		}
		if p.EnvAPIKey == "" {
			t.Errorf("provider %q missing EnvAPIKey convention", id)
		}
		if p.DocsURL != "" && !strings.HasPrefix(p.DocsURL, "https://") {
			t.Errorf("provider %q DocsURL %q must be HTTPS or empty", id, p.DocsURL)
		}
	}
}

// TestResolveKnown returns the registered entry for every id.
func TestResolveKnown(t *testing.T) {
	for id := range ai.Providers {
		got, err := ai.Resolve(id)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", id, err)
		}
		if got.ID != id {
			t.Errorf("Resolve(%q).ID = %q", id, got.ID)
		}
	}
}

// TestResolveUnknown surfaces ErrUnknownProvider and includes a hint
// listing known ids.
func TestResolveUnknown(t *testing.T) {
	_, err := ai.Resolve("not-a-provider-42")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !errors.Is(err, ai.ErrUnknownProvider) {
		t.Fatalf("err = %v, want ErrUnknownProvider chain", err)
	}
	// The error message should at least name one real provider so the
	// operator knows their typo is close to a real option.
	if !strings.Contains(err.Error(), "deepseek") {
		t.Errorf("err %q should list known ids (contains 'deepseek')", err.Error())
	}
}

// TestRequiredProvidersPresent guards the five launch providers.  Adding
// a provider is fine; removing one of these five is a ship-blocking
// change that must be discussed.
func TestRequiredProvidersPresent(t *testing.T) {
	required := []string{"deepseek", "volcengine", "moonshot", "zhipu", "dashscope"}
	for _, id := range required {
		if _, ok := ai.Providers[id]; !ok {
			t.Errorf("required provider %q missing from registry", id)
		}
	}
}

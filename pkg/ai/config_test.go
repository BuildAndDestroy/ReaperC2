package ai

import (
	"os"
	"testing"
)

func TestNormalizeProviderID(t *testing.T) {
	tests := map[string]string{
		"OpenAI":     ProviderOpenAI,
		"chatgpt":    ProviderOpenAI,
		"claude":     ProviderAnthropic,
		"local":      ProviderOllama,
		"  ollama ":  ProviderOllama,
		"unknown":    "unknown",
	}
	for in, want := range tests {
		if got := normalizeProviderID(in); got != want {
			t.Errorf("normalizeProviderID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveProviderRequiresConfigured(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_API_KEY", "")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "")
	t.Setenv("REAPER_AI_ANTHROPIC_API_KEY", "")
	t.Setenv("REAPER_AI_OLLAMA_ENABLED", "0")
	t.Setenv("REAPER_AI_OLLAMA_MODEL", "")

	_, err := ResolveProvider(ProviderOpenAI, "")
	if err == nil {
		t.Fatal("expected error when OpenAI not configured")
	}
}

func TestOllamaConfiguredWithExplicitEnable(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_OLLAMA_ENABLED", "1")
	t.Setenv("REAPER_AI_OLLAMA_MODEL", "")

	for _, p := range loadAllProviders() {
		if p.ID == ProviderOllama && !p.Configured {
			t.Fatal("expected Ollama configured when REAPER_AI_OLLAMA_ENABLED=1")
		}
	}
}

func TestDefaultProviderIDPrefersEnv(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_DEFAULT_PROVIDER", "anthropic")
	t.Setenv("REAPER_AI_ANTHROPIC_API_KEY", "test-key")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "other-key")

	if got := DefaultProviderID(); got != ProviderAnthropic {
		t.Fatalf("DefaultProviderID() = %q, want %q", got, ProviderAnthropic)
	}
	os.Unsetenv("REAPER_AI_ANTHROPIC_API_KEY")
}

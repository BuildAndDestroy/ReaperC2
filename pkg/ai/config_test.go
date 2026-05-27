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
		"bedrock":    ProviderBedrock,
		"aws":        ProviderBedrock,
		"azure":      ProviderFoundry,
		"foundry":    ProviderFoundry,
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
	t.Setenv("REAPER_AI_BEDROCK_ENABLED", "0")
	t.Setenv("REAPER_AI_BEDROCK_REGION", "")

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

func TestBedrockConfiguredWithIAMKeys(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-east-1")
	t.Setenv("REAPER_AI_BEDROCK_API_KEY", "")
	t.Setenv("REAPER_AI_BEDROCK_ACCESS_KEY_ID", "AKIATEST")
	t.Setenv("REAPER_AI_BEDROCK_SECRET_ACCESS_KEY", "secret")
	t.Setenv("REAPER_AI_BEDROCK_MODELS", "us.anthropic.claude-sonnet-4-6")

	for _, p := range loadAllProviders() {
		if p.ID == ProviderBedrock && !p.Configured {
			t.Fatal("expected Bedrock configured with region and IAM keys")
		}
	}
	models := EnabledModels()
	if len(models) < 1 || models[0].Provider != ProviderBedrock {
		t.Fatalf("EnabledModels() = %+v", models)
	}
}

func TestBedrockConfiguredWithAPIKey(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-east-1")
	t.Setenv("REAPER_AI_BEDROCK_API_KEY", "bedrock-api-key-test")
	t.Setenv("REAPER_AI_BEDROCK_ACCESS_KEY_ID", "")
	t.Setenv("REAPER_AI_BEDROCK_SECRET_ACCESS_KEY", "")
	t.Setenv("REAPER_AI_BEDROCK_MODELS", "us.anthropic.claude-sonnet-4-6")

	for _, p := range loadAllProviders() {
		if p.ID == ProviderBedrock && !p.Configured {
			t.Fatal("expected Bedrock configured with API key")
		}
	}
}

func TestFoundryConfiguredWithEndpointAndKey(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_FOUNDRY_API_KEY", "azure-key")
	t.Setenv("REAPER_AI_FOUNDRY_API_URL", "https://myresource.openai.azure.com")

	var found bool
	for _, p := range loadAllProviders() {
		if p.ID == ProviderFoundry {
			found = true
			if !p.Configured {
				t.Fatal("expected Foundry configured with endpoint and key")
			}
			if p.APIURL != "https://myresource.openai.azure.com/openai/v1" {
				t.Fatalf("APIURL = %q", p.APIURL)
			}
		}
	}
	if !found {
		t.Fatal("foundry provider missing from loadAllProviders")
	}
}

func TestBedrockConfiguredWithIAMExplicit(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_ENABLED", "1")
	t.Setenv("REAPER_AI_BEDROCK_REGION", "us-west-2")
	t.Setenv("REAPER_AI_BEDROCK_USE_IAM", "1")
	t.Setenv("REAPER_AI_BEDROCK_ACCESS_KEY_ID", "")
	t.Setenv("REAPER_AI_BEDROCK_SECRET_ACCESS_KEY", "")
	t.Setenv("REAPER_AI_BEDROCK_MODELS", "amazon.nova-lite-v1:0")

	for _, p := range loadAllProviders() {
		if p.ID == ProviderBedrock && !p.Configured {
			t.Fatal("expected Bedrock configured with IAM on EKS-style setup")
		}
	}
}

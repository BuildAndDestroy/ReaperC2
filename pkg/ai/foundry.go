package ai

import (
	"context"
	"os"
	"strings"
	"time"
)

func foundryAPIKeyFromEnv() string {
	for _, key := range []string{
		"REAPER_AI_FOUNDRY_API_KEY",
		"AZURE_OPENAI_API_KEY",
		"AZURE_AI_INFERENCE_KEY",
		"AZURE_INFERENCE_CREDENTIAL",
	} {
		if k := strings.TrimSpace(os.Getenv(key)); k != "" {
			return k
		}
	}
	return ""
}

func foundryAPIURLFromEnv() string {
	for _, key := range []string{
		"REAPER_AI_FOUNDRY_API_URL",
		"AZURE_OPENAI_ENDPOINT",
		"AZURE_AI_FOUNDRY_ENDPOINT",
		"AZURE_AI_PROJECT_ENDPOINT",
	} {
		if u := strings.TrimSpace(os.Getenv(key)); u != "" {
			return normalizeFoundryAPIURL(u)
		}
	}
	return ""
}

// normalizeFoundryAPIURL maps common Azure resource bases to the OpenAI v1 base used for discovery and chat.
func normalizeFoundryAPIURL(raw string) string {
	u := strings.TrimRight(strings.TrimSpace(raw), "/")
	if u == "" {
		return u
	}
	lower := strings.ToLower(u)
	if strings.HasSuffix(lower, "/openai/v1") {
		return u
	}
	if strings.HasSuffix(lower, "/openai") {
		return u + "/v1"
	}
	if strings.Contains(lower, ".openai.azure.com") || strings.Contains(lower, ".services.ai.azure.com") || strings.Contains(lower, "ai.azure.com") {
		return u + "/openai/v1"
	}
	return u
}

func foundryDiscoverEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("REAPER_AI_FOUNDRY_DISCOVER")); v != "" {
		return envBoolDefault("REAPER_AI_FOUNDRY_DISCOVER", false)
	}
	return foundryAPIKeyFromEnv() != "" && foundryAPIURLFromEnv() != "" && aiEnabled()
}

func foundryUseAPIKeyHeader() bool {
	if v := strings.TrimSpace(os.Getenv("REAPER_AI_FOUNDRY_USE_API_KEY_HEADER")); v != "" {
		return envBoolDefault("REAPER_AI_FOUNDRY_USE_API_KEY_HEADER", false)
	}
	// Legacy deployment-scoped Azure OpenAI endpoints (non-v1 base) expect api-key.
	url := strings.ToLower(foundryAPIURLFromEnv())
	return url != "" && !strings.HasSuffix(url, "/openai/v1")
}

func foundryModelsDiscoveredOrConfigured() []string {
	var discovered []string
	if foundryDiscoverEnabled() {
		p := foundrySettings(maxTokensFromEnv())
		if p.APIKey != "" && p.APIURL != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if names, err := discoverOpenAIModels(ctx, p.APIURL, p.APIKey, foundryUseAPIKeyHeader()); err == nil {
				discovered = names
			}
		}
	}
	if extra := modelNamesFromEnvOnly("REAPER_AI_FOUNDRY_MODELS", "REAPER_AI_FOUNDRY_MODEL", ""); len(extra) > 0 {
		discovered = append(discovered, extra...)
	}
	discovered = dedupeStrings(discovered)
	if len(discovered) == 0 {
		return nil
	}
	return mergePreferredModels(preferredFoundryModels, discovered)
}

func foundryModelNamesMerged() []string {
	if extra := modelNamesFromEnvOnly("REAPER_AI_FOUNDRY_MODELS", "REAPER_AI_FOUNDRY_MODEL", ""); len(extra) > 0 {
		return mergePreferredModels(preferredFoundryModels, extra)
	}
	names := foundryModelsDiscoveredOrConfigured()
	if len(names) == 0 && foundryAPIKeyFromEnv() != "" && foundryAPIURLFromEnv() != "" && aiEnabled() {
		return defaultFoundryModels()
	}
	return names
}

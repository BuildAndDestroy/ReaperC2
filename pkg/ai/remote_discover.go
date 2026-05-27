package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// discoverOpenAIModels lists chat-capable models from GET {apiURL}/models.
func discoverOpenAIModels(ctx context.Context, apiURL, apiKey string, useAPIKeyHeader bool) ([]string, error) {
	base := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if base == "" || apiKey == "" {
		return nil, fmt.Errorf("OpenAI discovery: missing API URL or key")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/models", nil)
	if err != nil {
		return nil, err
	}
	setOpenAICompatAuth(req, apiKey, useAPIKeyHeader)

	client := &http.Client{Timeout: 12 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenAI models HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse OpenAI models: %w", err)
	}
	var names []string
	for _, m := range parsed.Data {
		if id := strings.TrimSpace(m.ID); id != "" && openaiChatModel(id) {
			names = append(names, id)
		}
	}
	return mergePreferredModels(preferredOpenAIModels, dedupeStrings(names)), nil
}

// discoverAnthropicModels lists models from GET {apiURL}/models (paginated).
func discoverAnthropicModels(ctx context.Context, apiURL, apiKey string) ([]string, error) {
	base := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if base == "" || apiKey == "" {
		return nil, fmt.Errorf("Anthropic discovery: missing API URL or key")
	}
	client := &http.Client{Timeout: 12 * time.Second}
	var names []string
	afterID := ""
	for page := 0; page < 20; page++ {
		url := base + "/models?limit=100"
		if afterID != "" {
			url += "&after_id=" + afterID
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		res, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
		res.Body.Close()
		if err != nil {
			return nil, err
		}
		if res.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Anthropic models HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
		}
		var parsed struct {
			Data    []struct {
				ID   string `json:"id"`
				Type string `json:"type"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("parse Anthropic models: %w", err)
		}
		for _, m := range parsed.Data {
			if m.Type != "" && m.Type != "model" {
				continue
			}
			if id := strings.TrimSpace(m.ID); id != "" {
				names = append(names, id)
			}
		}
		if !parsed.HasMore || parsed.LastID == "" {
			break
		}
		afterID = parsed.LastID
	}
	return mergePreferredModels(preferredAnthropicModels, dedupeStrings(names)), nil
}

func openaiChatModel(id string) bool {
	lower := strings.ToLower(id)
	if strings.Contains(lower, "embed") {
		return false
	}
	for _, skip := range []string{
		"dall-e", "whisper", "tts", "audio", "realtime", "moderation",
		"transcribe", "search-api", "sora", "davinci", "babbage", "curie", "ada",
	} {
		if strings.Contains(lower, skip) {
			return false
		}
	}
	if strings.HasPrefix(lower, "gpt-") {
		return true
	}
	if strings.HasPrefix(lower, "chatgpt-") {
		return true
	}
	if len(lower) >= 2 && lower[0] == 'o' && lower[1] >= '0' && lower[1] <= '9' {
		return true
	}
	return false
}

func envBoolDefault(key string, defaultOn bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return defaultOn
	}
	return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
}

func openaiAPIKeyFromEnv() string {
	if k := strings.TrimSpace(os.Getenv("REAPER_AI_OPENAI_API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("REAPER_AI_API_KEY"))
}

func openaiDiscoverEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("REAPER_AI_OPENAI_DISCOVER")); v != "" {
		return envBoolDefault("REAPER_AI_OPENAI_DISCOVER", false)
	}
	return openaiAPIKeyFromEnv() != "" && aiEnabled()
}

func anthropicDiscoverEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_DISCOVER")); v != "" {
		return envBoolDefault("REAPER_AI_ANTHROPIC_DISCOVER", false)
	}
	key := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_API_KEY"))
	return key != "" && aiEnabled()
}

func openaiModelsDiscoveredOrConfigured() []string {
	var discovered []string
	if openaiDiscoverEnabled() {
		p := openAISettings(maxTokensFromEnv())
		if p.APIKey != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if names, err := discoverOpenAIModels(ctx, p.APIURL, p.APIKey, false); err == nil {
				discovered = names
			}
		}
	}
	if extra := modelNamesFromEnvOnly("REAPER_AI_OPENAI_MODELS", "REAPER_AI_OPENAI_MODEL", "REAPER_AI_MODEL"); len(extra) > 0 {
		discovered = append(discovered, extra...)
	}
	discovered = dedupeStrings(discovered)
	if len(discovered) == 0 {
		return nil
	}
	return mergePreferredModels(preferredOpenAIModels, discovered)
}

func openaiModelNamesMerged() []string {
	if extra := modelNamesFromEnvOnly("REAPER_AI_OPENAI_MODELS", "REAPER_AI_OPENAI_MODEL", "REAPER_AI_MODEL"); len(extra) > 0 {
		return mergePreferredModels(preferredOpenAIModels, extra)
	}
	names := openaiModelsDiscoveredOrConfigured()
	if len(names) == 0 && openaiAPIKeyFromEnv() != "" && aiEnabled() {
		return defaultOpenAIModels()
	}
	return names
}

func anthropicModelsDiscoveredOrConfigured() []string {
	var discovered []string
	if anthropicDiscoverEnabled() {
		p := anthropicSettings(maxTokensFromEnv())
		if p.APIKey != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
			defer cancel()
			if names, err := discoverAnthropicModels(ctx, p.APIURL, p.APIKey); err == nil {
				discovered = names
			}
		}
	}
	if extra := modelNamesFromEnvOnly("REAPER_AI_ANTHROPIC_MODELS", "REAPER_AI_ANTHROPIC_MODEL", ""); len(extra) > 0 {
		discovered = append(discovered, extra...)
	}
	discovered = dedupeStrings(discovered)
	if len(discovered) == 0 {
		return nil
	}
	return mergePreferredModels(preferredAnthropicModels, discovered)
}

func anthropicModelNamesMerged() []string {
	if extra := modelNamesFromEnvOnly("REAPER_AI_ANTHROPIC_MODELS", "REAPER_AI_ANTHROPIC_MODEL", ""); len(extra) > 0 {
		return mergePreferredModels(preferredAnthropicModels, extra)
	}
	names := anthropicModelsDiscoveredOrConfigured()
	key := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_API_KEY"))
	if len(names) == 0 && key != "" && aiEnabled() {
		return defaultAnthropicModels()
	}
	return names
}

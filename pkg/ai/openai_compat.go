package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// chatOpenAICompatible calls POST {base}/chat/completions (OpenAI, Ollama, and compatibles).
func chatOpenAICompatible(ctx context.Context, cfg ProviderSettings, system string, messages []Message) (string, error) {
	if useAzureOpenAICompatMaxCompletion(cfg) && azureFoundryChatIncompatibleModel(cfg.Model) {
		return "", fmt.Errorf("%s: deployment %q is not available on the OpenAI-compatible Chat Completions API (Azure returns api_not_supported for Claude on this path). Use a GPT deployment here, or use catalog ids `bedrock:…` / `anthropic:…` for Claude", cfg.Label, cfg.Model)
	}

	apiMessages := []Message{{Role: "system", Content: system}}
	apiMessages = append(apiMessages, messages...)

	body := map[string]interface{}{
		"model":    cfg.Model,
		"messages": apiMessages,
	}
	// Azure AI Foundry / OpenAI v1: GPT-5.x and some SKUs reject max_tokens; require max_completion_tokens.
	// Also match by URL so mis-tagged env (OpenAI vars pointing at *.openai.azure.com) still works.
	if useAzureOpenAICompatMaxCompletion(cfg) {
		body["max_completion_tokens"] = cfg.MaxTokens
		// Azure GPT-5.x rejects non-default temperature (e.g. 0.4); only 1 is supported.
		body["temperature"] = 1.0
	} else {
		body["max_tokens"] = cfg.MaxTokens
		body["temperature"] = 0.4
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.APIURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	useAPIKey := cfg.ID == ProviderFoundry && foundryUseAPIKeyHeader()
	setOpenAICompatAuth(req, cfg.APIKey, useAPIKey)

	client := &http.Client{Timeout: 120 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", providerHTTPError(cfg.Label, res.StatusCode, b)
	}

	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return "", fmt.Errorf("parse %s response: %w", cfg.Label, err)
	}
	if parsed.Error != nil && parsed.Error.Message != "" {
		return "", fmt.Errorf("%s: %s", cfg.Label, parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("%s: empty response", cfg.Label)
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), nil
}

// useAzureOpenAICompatMaxCompletion is true for Azure AI Foundry / Azure OpenAI inference
// (chat completions require max_completion_tokens for GPT-5.x and newer SKUs).
func useAzureOpenAICompatMaxCompletion(cfg ProviderSettings) bool {
	if cfg.ID == ProviderFoundry {
		return true
	}
	u := strings.ToLower(cfg.APIURL)
	return strings.Contains(u, ".openai.azure.com") || strings.Contains(u, ".services.ai.azure.com")
}

// azureFoundryChatIncompatibleModel is true for Claude / Opus deployments that Azure does not
// route through POST …/openai/v1/chat/completions (ReaperC2 only supports that path for Foundry).
func azureFoundryChatIncompatibleModel(model string) bool {
	m := strings.ToLower(strings.TrimSpace(model))
	if m == "" {
		return false
	}
	if strings.Contains(m, "claude") || strings.Contains(m, "anthropic") {
		return true
	}
	if strings.Contains(m, "opus-4") || strings.Contains(m, "opus-3") {
		return true
	}
	return false
}

func setOpenAICompatAuth(req *http.Request, apiKey string, useAPIKeyHeader bool) {
	if apiKey == "" {
		return
	}
	if useAPIKeyHeader {
		req.Header.Set("api-key", apiKey)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
}

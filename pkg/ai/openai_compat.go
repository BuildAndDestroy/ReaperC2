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
			FinishReason string `json:"finish_reason"`
			Message      struct {
				Content json.RawMessage `json:"content"`
				Refusal *string         `json:"refusal"`
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
	choice := parsed.Choices[0]
	if choice.Message.Refusal != nil {
		if s := strings.TrimSpace(*choice.Message.Refusal); s != "" {
			return "", fmt.Errorf("%s: model refused: %s", cfg.Label, s)
		}
	}
	content, err := extractOpenAIMessageContent(choice.Message.Content)
	if err != nil {
		return "", fmt.Errorf("parse %s response content: %w", cfg.Label, err)
	}
	if content == "" {
		return "", fmt.Errorf("%s: %s", cfg.Label, openAICompatEmptyReplyError(choice.FinishReason, useAzureOpenAICompatMaxCompletion(cfg)))
	}
	return content, nil
}

// extractOpenAIMessageContent handles OpenAI-compatible message content as a string or
// an array of {type,text} parts (Azure GPT-5 / newer SKUs may return the latter).
func extractOpenAIMessageContent(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return strings.TrimSpace(s), nil
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &parts); err == nil {
		var out []string
		for _, p := range parts {
			if t := strings.TrimSpace(p.Text); t != "" {
				out = append(out, t)
			}
		}
		return strings.TrimSpace(strings.Join(out, "\n")), nil
	}
	return "", fmt.Errorf("unexpected content shape")
}

func openAICompatEmptyReplyError(finishReason string, azureReasoning bool) string {
	switch strings.ToLower(strings.TrimSpace(finishReason)) {
	case "length":
		if azureReasoning {
			return "model returned no visible text (output token budget exhausted; reasoning tokens count toward max_completion_tokens). Increase REAPER_AI_MAX_TOKENS or retry with a shorter prompt"
		}
		return "model returned no visible text (output token budget exhausted). Increase REAPER_AI_MAX_TOKENS or retry with a shorter prompt"
	case "content_filter":
		return "response blocked by content filter"
	default:
		return "empty message content"
	}
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

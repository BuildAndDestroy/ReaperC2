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
	apiMessages := []Message{{Role: "system", Content: system}}
	apiMessages = append(apiMessages, messages...)

	body := map[string]interface{}{
		"model":       cfg.Model,
		"messages":    apiMessages,
		"max_tokens":  cfg.MaxTokens,
		"temperature": 0.4,
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
	if cfg.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	}

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

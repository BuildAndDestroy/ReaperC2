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

// discoverOllamaModels returns chat model names from the Ollama /api/tags endpoint.
func discoverOllamaModels(ctx context.Context, apiV1URL string) ([]string, error) {
	base := ollamaTagsBaseURL(apiV1URL)
	if base == "" {
		return nil, fmt.Errorf("empty Ollama API URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama tags HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var parsed struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("parse Ollama tags: %w", err)
	}
	var names []string
	for _, m := range parsed.Models {
		name := strings.TrimSpace(m.Name)
		if name == "" || ollamaEmbeddingModel(name) {
			continue
		}
		names = append(names, name)
	}
	return dedupeStrings(names), nil
}

func ollamaTagsBaseURL(apiV1URL string) string {
	u := strings.TrimRight(strings.TrimSpace(apiV1URL), "/")
	if strings.HasSuffix(u, "/v1") {
		return strings.TrimSuffix(u, "/v1")
	}
	return u
}

func ollamaEmbeddingModel(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "embed")
}

func ollamaDiscoverEnabled() bool {
	if v := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_DISCOVER")); v != "" {
		return v == "1" || strings.EqualFold(v, "true") || strings.EqualFold(v, "yes")
	}
	return strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_ENABLED")) == "1"
}

func ollamaModelsDiscoveredOrConfigured() []string {
	var names []string
	if ollamaDiscoverEnabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		url := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_API_URL"))
		if url == "" {
			url = "http://127.0.0.1:11434/v1"
		}
		if discovered, err := discoverOllamaModels(ctx, url); err == nil {
			names = append(names, discovered...)
		}
	}
	if extra := modelNamesFromEnvOnly("REAPER_AI_OLLAMA_MODELS", "REAPER_AI_OLLAMA_MODEL", ""); len(extra) > 0 {
		names = append(names, extra...)
	}
	return dedupeStrings(names)
}

func ollamaModelNamesMerged() []string {
	names := ollamaModelsDiscoveredOrConfigured()
	if len(names) == 0 && strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_ENABLED")) == "1" {
		return []string{"llama3.2"}
	}
	return names
}

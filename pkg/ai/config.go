package ai

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Provider IDs for Operator AI.
const (
	ProviderOpenAI    = "openai"
	ProviderAnthropic = "anthropic"
	ProviderOllama    = "ollama"
)

// ProviderSettings is one configured LLM backend.
type ProviderSettings struct {
	ID         string
	Label      string
	APIURL     string
	APIKey     string
	Model      string
	MaxTokens  int
	Configured bool
}

// ProviderInfo is returned to the UI (no secrets).
type ProviderInfo struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Configured  bool   `json:"configured"`
	Model       string `json:"model"`
	APIURL      string `json:"api_url"`
	RequiresKey bool   `json:"requires_key"`
}

// Catalog lists all supported providers and whether each is configured.
func Catalog() []ProviderInfo {
	defaults := loadAllProviders()
	out := make([]ProviderInfo, 0, len(defaults))
	for _, p := range defaults {
		out = append(out, ProviderInfo{
			ID:          p.ID,
			Label:       p.Label,
			Configured:  p.Configured,
			Model:       p.Model,
			APIURL:      p.APIURL,
			RequiresKey: p.ID == ProviderOpenAI || p.ID == ProviderAnthropic,
		})
	}
	return out
}

// AnyConfigured reports whether at least one provider can be used.
func AnyConfigured() bool {
	for _, p := range Catalog() {
		if p.Configured {
			return true
		}
	}
	return false
}

// DefaultProviderID returns the configured default or the first configured provider.
func DefaultProviderID() string {
	if id := normalizeProviderID(os.Getenv("REAPER_AI_DEFAULT_PROVIDER")); id != "" {
		for _, p := range Catalog() {
			if p.ID == id && p.Configured {
				return id
			}
		}
	}
	for _, p := range Catalog() {
		if p.Configured {
			return p.ID
		}
	}
	return ProviderOpenAI
}

// ResolveProvider returns settings for providerID, optionally overriding the model name.
func ResolveProvider(providerID, modelOverride string) (ProviderSettings, error) {
	id := normalizeProviderID(providerID)
	if id == "" {
		id = DefaultProviderID()
	}
	for _, p := range loadAllProviders() {
		if p.ID != id {
			continue
		}
		if !p.Configured {
			return ProviderSettings{}, fmt.Errorf("provider %q is not configured on this server", id)
		}
		if m := strings.TrimSpace(modelOverride); m != "" {
			p.Model = m
		}
		return p, nil
	}
	return ProviderSettings{}, fmt.Errorf("unknown provider %q", providerID)
}

func loadAllProviders() []ProviderSettings {
	maxTok := maxTokensFromEnv()
	return []ProviderSettings{
		openAISettings(maxTok),
		anthropicSettings(maxTok),
		ollamaSettings(maxTok),
	}
}

func openAISettings(maxTok int) ProviderSettings {
	key := strings.TrimSpace(os.Getenv("REAPER_AI_OPENAI_API_KEY"))
	if key == "" {
		key = strings.TrimSpace(os.Getenv("REAPER_AI_API_KEY")) // legacy
	}
	url := strings.TrimSpace(os.Getenv("REAPER_AI_OPENAI_API_URL"))
	if url == "" {
		url = strings.TrimSpace(os.Getenv("REAPER_AI_API_URL"))
	}
	if url == "" {
		url = "https://api.openai.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("REAPER_AI_OPENAI_MODEL"))
	if model == "" {
		model = strings.TrimSpace(os.Getenv("REAPER_AI_MODEL"))
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	return ProviderSettings{
		ID:         ProviderOpenAI,
		Label:      "OpenAI",
		APIURL:     strings.TrimRight(url, "/"),
		APIKey:     key,
		Model:      model,
		MaxTokens:  maxTok,
		Configured: key != "" && aiEnabled(),
	}
}

func anthropicSettings(maxTok int) ProviderSettings {
	key := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_API_KEY"))
	url := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_API_URL"))
	if url == "" {
		url = "https://api.anthropic.com/v1"
	}
	model := strings.TrimSpace(os.Getenv("REAPER_AI_ANTHROPIC_MODEL"))
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return ProviderSettings{
		ID:         ProviderAnthropic,
		Label:      "Anthropic",
		APIURL:     strings.TrimRight(url, "/"),
		APIKey:     key,
		Model:      model,
		MaxTokens:  maxTok,
		Configured: key != "" && aiEnabled(),
	}
}

func ollamaSettings(maxTok int) ProviderSettings {
	url := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_API_URL"))
	if url == "" {
		url = "http://127.0.0.1:11434/v1"
	}
	model := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_MODEL"))
	key := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_API_KEY"))
	// Configured when a model is set, or operator explicitly enables Ollama without a model (use server default tag).
	explicit := strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_ENABLED")) == "1"
	configured := (model != "" || explicit) && aiEnabled()
	if configured && model == "" {
		model = "llama3.2"
	}
	return ProviderSettings{
		ID:         ProviderOllama,
		Label:      "Ollama",
		APIURL:     strings.TrimRight(url, "/"),
		APIKey:     key,
		Model:      model,
		MaxTokens:  maxTok,
		Configured: configured,
	}
}

func maxTokensFromEnv() int {
	maxTok := 2048
	if s := strings.TrimSpace(os.Getenv("REAPER_AI_MAX_TOKENS")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 128000 {
			maxTok = n
		}
	}
	return maxTok
}

func aiEnabled() bool {
	return strings.TrimSpace(os.Getenv("REAPER_AI_ENABLED")) != "0"
}

func normalizeProviderID(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case ProviderOpenAI, "chatgpt", "gpt":
		return ProviderOpenAI
	case ProviderAnthropic, "claude":
		return ProviderAnthropic
	case ProviderOllama, "local":
		return ProviderOllama
	default:
		return strings.ToLower(strings.TrimSpace(s))
	}
}

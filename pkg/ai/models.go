package ai

import (
	"fmt"
	"os"
	"strings"
)

// ModelAuto is the UI/server sentinel for automatic model selection.
const ModelAuto = "auto"

// ModelOption is one selectable model in the Operator AI UI.
type ModelOption struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

// EnabledModels returns configured provider/model pairs (excludes Auto).
func EnabledModels() []ModelOption {
	if unified := strings.TrimSpace(os.Getenv("REAPER_AI_MODELS")); unified != "" {
		return filterEnabled(parseUnifiedModelList(unified))
	}
	providers := loadAllProviders()
	var out []ModelOption
	for _, p := range providers {
		if !p.Configured {
			continue
		}
		for _, name := range providerModelNames(p.ID) {
			out = append(out, modelOption(p, name))
		}
	}
	return out
}

// DefaultModelID returns the default selection for Auto (env or first enabled model).
func DefaultModelID() string {
	if id := strings.TrimSpace(os.Getenv("REAPER_AI_DEFAULT_MODEL")); id != "" {
		if id == ModelAuto {
			return ModelAuto
		}
		if _, err := resolveModelID(id); err == nil {
			return normalizeModelID(id)
		}
	}
	return ModelAuto
}

// ResolveModel returns provider settings for a catalog model id or Auto.
func ResolveModel(modelID string) (ProviderSettings, error) {
	id := strings.TrimSpace(modelID)
	if id == "" || strings.EqualFold(id, ModelAuto) {
		return resolveAutoModel()
	}
	choice, err := resolveModelID(id)
	if err != nil {
		return ProviderSettings{}, err
	}
	return choice, nil
}

func resolveAutoModel() (ProviderSettings, error) {
	def := strings.TrimSpace(os.Getenv("REAPER_AI_DEFAULT_MODEL"))
	if def != "" && !strings.EqualFold(def, ModelAuto) {
		if cfg, err := resolveModelID(def); err == nil {
			return cfg, nil
		}
	}
	if pid := DefaultProviderID(); pid != "" {
		for _, m := range EnabledModels() {
			if m.Provider == pid {
				return resolveModelID(m.ID)
			}
		}
	}
	models := EnabledModels()
	if len(models) == 0 {
		return ProviderSettings{}, fmt.Errorf("no AI models are configured")
	}
	return resolveModelID(models[0].ID)
}

func resolveModelID(modelID string) (ProviderSettings, error) {
	id := normalizeModelID(modelID)
	for _, m := range EnabledModels() {
		if m.ID == id {
			return providerSettingsFor(m.Provider, m.Model)
		}
	}
	// Legacy: bare model name when unique across enabled providers.
	if !strings.Contains(id, ":") {
		var matches []ModelOption
		for _, m := range EnabledModels() {
			if m.Model == id {
				matches = append(matches, m)
			}
		}
		if len(matches) == 1 {
			return providerSettingsFor(matches[0].Provider, matches[0].Model)
		}
	}
	return ProviderSettings{}, fmt.Errorf("unknown or disabled model %q", modelID)
}

func providerSettingsFor(providerID, model string) (ProviderSettings, error) {
	return ResolveProvider(providerID, model)
}

func normalizeModelID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || strings.EqualFold(id, ModelAuto) {
		return ModelAuto
	}
	if strings.Contains(id, ":") {
		parts := strings.SplitN(id, ":", 2)
		return normalizeProviderID(parts[0]) + ":" + strings.TrimSpace(parts[1])
	}
	return id
}

func modelOption(p ProviderSettings, model string) ModelOption {
	model = strings.TrimSpace(model)
	return ModelOption{
		ID:       p.ID + ":" + model,
		Label:    p.Label + " · " + model,
		Provider: p.ID,
		Model:    model,
	}
}

func providerModelNames(providerID string) []string {
	switch providerID {
	case ProviderOpenAI:
		return modelNamesFromEnv("REAPER_AI_OPENAI_MODELS", "REAPER_AI_OPENAI_MODEL", "REAPER_AI_MODEL")
	case ProviderAnthropic:
		return modelNamesFromEnv("REAPER_AI_ANTHROPIC_MODELS", "REAPER_AI_ANTHROPIC_MODEL", "")
	case ProviderOllama:
		return modelNamesFromEnv("REAPER_AI_OLLAMA_MODELS", "REAPER_AI_OLLAMA_MODEL", "")
	default:
		return nil
	}
}

func modelNamesFromEnv(listKey, singleKey, legacySingleKey string) []string {
	if s := strings.TrimSpace(os.Getenv(listKey)); s != "" {
		return dedupeStrings(splitCSV(s))
	}
	if s := strings.TrimSpace(os.Getenv(singleKey)); s != "" {
		return []string{s}
	}
	if legacySingleKey != "" {
		if s := strings.TrimSpace(os.Getenv(legacySingleKey)); s != "" {
			return []string{s}
		}
	}
	// Provider defaults when configured but no model env set.
	switch singleKey {
	case "REAPER_AI_OPENAI_MODEL":
		return []string{"gpt-4o-mini"}
	case "REAPER_AI_ANTHROPIC_MODEL":
		return []string{"claude-sonnet-4-20250514"}
	case "REAPER_AI_OLLAMA_MODEL":
		if strings.TrimSpace(os.Getenv("REAPER_AI_OLLAMA_ENABLED")) == "1" {
			return []string{"llama3.2"}
		}
	}
	return nil
}

func parseUnifiedModelList(s string) []ModelOption {
	var out []ModelOption
	providers := map[string]ProviderSettings{}
	for _, p := range loadAllProviders() {
		providers[p.ID] = p
	}
	for _, part := range splitCSV(s) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pid, model, ok := strings.Cut(part, ":")
		if !ok {
			continue
		}
		pid = normalizeProviderID(pid)
		p, ok := providers[pid]
		if !ok {
			continue
		}
		out = append(out, modelOption(p, model))
	}
	return dedupeModelOptions(out)
}

func filterEnabled(in []ModelOption) []ModelOption {
	configured := map[string]bool{}
	for _, p := range loadAllProviders() {
		configured[p.ID] = p.Configured
	}
	var out []ModelOption
	for _, m := range in {
		if configured[m.Provider] {
			out = append(out, m)
		}
	}
	return dedupeModelOptions(out)
}

func splitCSV(s string) []string {
	s = strings.ReplaceAll(s, "\n", ",")
	var parts []string
	for _, p := range strings.Split(s, ",") {
		if t := strings.TrimSpace(p); t != "" {
			parts = append(parts, t)
		}
	}
	return parts
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func dedupeModelOptions(in []ModelOption) []ModelOption {
	seen := map[string]bool{}
	var out []ModelOption
	for _, m := range in {
		if seen[m.ID] {
			continue
		}
		seen[m.ID] = true
		out = append(out, m)
	}
	return out
}

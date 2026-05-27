package ai

import "strings"

// Curated latest models shown in the Operator AI dropdown when discovery is off,
// and merged ahead of live API results so flagship IDs appear even before an account
// lists them (actual chat still requires model access on the provider).
var (
	preferredOpenAIModels = []string{
		"gpt-5.5",
		"gpt-5.5-pro",
		"gpt-4.1",
		"gpt-4o-mini",
		"gpt-4o",
	}
	preferredAnthropicModels = []string{
		"claude-opus-4-7",
		"claude-sonnet-4-6",
		"claude-haiku-4-5-20251001",
	}
	// Base foundation model IDs; resolved to inference profiles (e.g. us.anthropic.*) at runtime.
	preferredBedrockModels = []string{
		"anthropic.claude-opus-4-7",
		"anthropic.claude-sonnet-4-6",
		"anthropic.claude-haiku-4-5-20251001-v1:0",
		"amazon.nova-lite-v1:0",
	}
	preferredFoundryModels = []string{
		"gpt-5.5",
		"gpt-4.1",
		"gpt-4o",
	}
)

func defaultOpenAIModels() []string {
	return append([]string(nil), preferredOpenAIModels...)
}

func defaultAnthropicModels() []string {
	return append([]string(nil), preferredAnthropicModels...)
}

func defaultBedrockModels() []string {
	return append([]string(nil), preferredBedrockModels...)
}

func defaultFoundryModels() []string {
	return append([]string(nil), preferredFoundryModels...)
}

// mergePreferredModels puts preferred IDs first, then any discovered names not already listed.
func mergePreferredModels(preferred, discovered []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, name := range preferred {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, name := range discovered {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

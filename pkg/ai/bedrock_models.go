package ai

import (
	"os"
	"strings"
)

// bedrockConverseModelID returns the modelId for Bedrock Converse. Newer Anthropic Claude
// models on Bedrock require a cross-Region inference profile (e.g. us.anthropic.claude-sonnet-4-6)
// instead of the bare foundation model ID.
func bedrockConverseModelID(modelID string) string {
	modelID = strings.TrimSpace(modelID)
	if modelID == "" || bedrockAlreadyInferenceProfile(modelID) {
		return modelID
	}
	lower := strings.ToLower(modelID)
	// Claude Fable 5 uses a global inference profile on Bedrock Converse (not us./eu.*).
	if strings.Contains(lower, "claude-fable-5") && strings.HasPrefix(lower, "anthropic.") {
		return "global." + modelID
	}
	if !bedrockRequiresInferenceProfile(modelID) {
		return modelID
	}
	return bedrockInferenceProfilePrefix() + "." + modelID
}

func bedrockAlreadyInferenceProfile(modelID string) bool {
	lower := strings.ToLower(modelID)
	if strings.HasPrefix(lower, "arn:") {
		return true
	}
	for _, prefix := range []string{"us.", "eu.", "global.", "jp.", "au.", "apac."} {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

func bedrockRequiresInferenceProfile(modelID string) bool {
	lower := strings.ToLower(strings.TrimSpace(modelID))
	if !strings.HasPrefix(lower, "anthropic.claude-") {
		return false
	}
	// Dateless Claude 4.x (Opus/Sonnet 4.6+) require inference profiles for on-demand Converse.
	if strings.Contains(lower, "claude-opus-4-") || strings.Contains(lower, "claude-sonnet-4-") {
		return true
	}
	// Claude 3.7 / 3.5 v2 also reject direct on-demand invocation.
	if strings.Contains(lower, "claude-3-7-") {
		return true
	}
	if strings.Contains(lower, "claude-3-5-sonnet-20241022") {
		return true
	}
	return false
}

// bedrockInferenceProfilePrefix selects the geo inference profile prefix for the configured region.
// Override with REAPER_AI_BEDROCK_INFERENCE_PREFIX=us|eu|jp|au|global
func bedrockInferenceProfilePrefix() string {
	if p := strings.TrimSpace(os.Getenv("REAPER_AI_BEDROCK_INFERENCE_PREFIX")); p != "" {
		return strings.Trim(strings.ToLower(p), ".")
	}
	region := strings.ToLower(bedrockRegionFromEnv())
	switch {
	case strings.HasPrefix(region, "eu-"):
		return "eu"
	case region == "ap-northeast-1" || strings.HasPrefix(region, "ap-northeast-"):
		return "jp"
	case strings.HasPrefix(region, "ap-"):
		return "global"
	case strings.HasPrefix(region, "au-"):
		return "au"
	default:
		return "us"
	}
}

func resolveBedrockModelIDs(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, bedrockConverseModelID(n))
		}
	}
	return dedupeStrings(out)
}

func bedrockModelNamesMerged() []string {
	names := modelNamesFromEnv("REAPER_AI_BEDROCK_MODELS", "REAPER_AI_BEDROCK_MODEL", "")
	if len(names) == 0 {
		if bedrockRegionFromEnv() != "" && bedrockCanAuthenticate() {
			names = defaultBedrockModels()
		}
	}
	return resolveBedrockModelIDs(names)
}

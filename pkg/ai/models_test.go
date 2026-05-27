package ai

import (
	"testing"
)

func TestEnabledModelsMultiplePerProvider(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_MODELS", "")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "key")
	t.Setenv("REAPER_AI_OPENAI_MODELS", "gpt-4o-mini, gpt-4o")
	t.Setenv("REAPER_AI_ANTHROPIC_API_KEY", "")
	t.Setenv("REAPER_AI_OLLAMA_ENABLED", "0")

	models := EnabledModels()
	if len(models) != 5 {
		t.Fatalf("EnabledModels() len = %d, want 5 (preferred + env): %+v", len(models), models)
	}
	if models[0].ID != "openai:gpt-5.5" {
		t.Fatalf("first model id = %q, want openai:gpt-5.5", models[0].ID)
	}
}

func TestResolveModelAutoUsesDefaultModelEnv(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "key")
	t.Setenv("REAPER_AI_OPENAI_MODELS", "gpt-4o-mini,gpt-4o")
	t.Setenv("REAPER_AI_DEFAULT_MODEL", "openai:gpt-4o")

	cfg, err := ResolveModel(ModelAuto)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Model != "gpt-4o" {
		t.Fatalf("model = %q, want gpt-4o", cfg.Model)
	}
}

func TestUnifiedModelsList(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_MODELS", "openai:gpt-4o, ollama:llama3.2")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "key")
	t.Setenv("REAPER_AI_OLLAMA_ENABLED", "1")
	t.Setenv("REAPER_AI_OLLAMA_DISCOVER", "0")
	t.Setenv("REAPER_AI_OPENAI_DISCOVER", "0")
	t.Setenv("REAPER_AI_ANTHROPIC_DISCOVER", "0")
	t.Setenv("REAPER_AI_FOUNDRY_DISCOVER", "0")

	models := EnabledModels()
	if len(models) != 2 {
		t.Fatalf("EnabledModels() len = %d, want 2", len(models))
	}
}

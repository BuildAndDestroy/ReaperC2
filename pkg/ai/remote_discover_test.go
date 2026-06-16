package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIChatModelFilter(t *testing.T) {
	tests := []struct {
		id   string
		want bool
	}{
		{"gpt-4.1", true},
		{"gpt-5.5", true},
		{"o3-mini", true},
		{"text-embedding-3-small", false},
		{"dall-e-3", false},
		{"whisper-1", false},
	}
	for _, tc := range tests {
		if got := openaiChatModel(tc.id); got != tc.want {
			t.Errorf("openaiChatModel(%q) = %v, want %v", tc.id, got, tc.want)
		}
	}
}

func TestDiscoverOpenAIModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{
				{"id": "gpt-4.1"},
				{"id": "text-embedding-3-small"},
				{"id": "gpt-5.5"},
			},
		})
	}))
	defer srv.Close()

	names, err := discoverOpenAIModels(context.Background(), srv.URL+"/v1", "test-key", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) < 2 || names[0] != "gpt-5.5" {
		t.Fatalf("names = %#v", names)
	}
}

func TestDiscoverAnthropicModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []map[string]string{
				{"id": "claude-sonnet-4-6", "type": "model"},
				{"id": "claude-opus-4-7", "type": "model"},
			},
			"has_more": false,
		})
	}))
	defer srv.Close()

	names, err := discoverAnthropicModels(context.Background(), srv.URL+"/v1", "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 4 || names[0] != "claude-fable-5" {
		t.Fatalf("names = %#v", names)
	}
}

func TestOpenAIModelNamesMergedDiscoverOffUsesDefaults(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_OPENAI_API_KEY", "key")
	t.Setenv("REAPER_AI_OPENAI_DISCOVER", "0")
	t.Setenv("REAPER_AI_OPENAI_MODELS", "")
	t.Setenv("REAPER_AI_OPENAI_MODEL", "")
	t.Setenv("REAPER_AI_MODEL", "")

	names := openaiModelNamesMerged()
	if len(names) != 5 || names[0] != "gpt-5.5" {
		t.Fatalf("names = %#v", names)
	}
}

func TestAnthropicModelNamesMergedDiscoverOffUsesDefaults(t *testing.T) {
	t.Setenv("REAPER_AI_ENABLED", "1")
	t.Setenv("REAPER_AI_ANTHROPIC_API_KEY", "key")
	t.Setenv("REAPER_AI_ANTHROPIC_DISCOVER", "0")
	t.Setenv("REAPER_AI_ANTHROPIC_MODELS", "")
	t.Setenv("REAPER_AI_ANTHROPIC_MODEL", "")

	names := anthropicModelNamesMerged()
	if len(names) != 4 || names[0] != "claude-fable-5" {
		t.Fatalf("names = %#v", names)
	}
}

package ai

import "testing"

func TestOllamaTagsBaseURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"http://host.docker.internal:11434/v1", "http://host.docker.internal:11434"},
		{"http://127.0.0.1:11434/v1/", "http://127.0.0.1:11434"},
		{"http://ollama:11434", "http://ollama:11434"},
	}
	for _, tc := range tests {
		if got := ollamaTagsBaseURL(tc.in); got != tc.want {
			t.Errorf("ollamaTagsBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestOllamaEmbeddingModel(t *testing.T) {
	if !ollamaEmbeddingModel("mxbai-embed-large:latest") {
		t.Fatal("expected embed model")
	}
	if ollamaEmbeddingModel("llama3.2:latest") {
		t.Fatal("llama3.2 should not be embed")
	}
}

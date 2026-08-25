package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractOpenAIMessageContent_string(t *testing.T) {
	got, err := extractOpenAIMessageContent(json.RawMessage(`"hello world"`))
	if err != nil || got != "hello world" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestExtractOpenAIMessageContent_array(t *testing.T) {
	got, err := extractOpenAIMessageContent(json.RawMessage(`[{"type":"text","text":"line one"},{"type":"text","text":"line two"}]`))
	if err != nil || got != "line one\nline two" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestExtractOpenAIMessageContent_null(t *testing.T) {
	got, err := extractOpenAIMessageContent(json.RawMessage(`null`))
	if err != nil || got != "" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestOpenAICompatEmptyReplyError_length(t *testing.T) {
	msg := openAICompatEmptyReplyError("length", true)
	if msg == "" || !strings.Contains(msg, "REAPER_AI_MAX_TOKENS") {
		t.Fatalf("unexpected message: %q", msg)
	}
}

func TestChatOpenAICompatible_parsesArrayContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"finish_reason": "stop",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": []map[string]string{{"type": "text", "text": "Azure reply"}},
					},
				},
			},
		})
	}))
	defer srv.Close()

	cfg := ProviderSettings{
		ID:        ProviderOpenAI,
		Label:     "OpenAI",
		APIURL:    srv.URL,
		APIKey:    "test",
		Model:     "gpt-4o",
		MaxTokens: 256,
	}
	reply, err := chatOpenAICompatible(context.Background(), cfg, "sys", []Message{{Role: "user", Content: "hi"}})
	if err != nil || reply != "Azure reply" {
		t.Fatalf("reply=%q err=%v", reply, err)
	}
}

func TestChatOpenAICompatible_emptyContentIsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"finish_reason": "length",
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
					},
				},
			},
		})
	}))
	defer srv.Close()

	cfg := ProviderSettings{
		ID:        ProviderFoundry,
		Label:     "Azure AI Foundry",
		APIURL:    srv.URL,
		APIKey:    "test",
		Model:     "gpt-5.5",
		MaxTokens: 2048,
	}
	_, err := chatOpenAICompatible(context.Background(), cfg, "sys", []Message{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

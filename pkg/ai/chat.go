package ai

import (
	"context"
	"fmt"
	"strings"
)

// Message is one chat turn for the provider API.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResult is a completed model reply.
type ChatResult struct {
	Reply    string
	Provider string
	Model    string
}

// Chat routes to the selected catalog model (or Auto) and provider backend.
func Chat(ctx context.Context, modelID, systemExtra string, messages []Message) (ChatResult, error) {
	if !AnyConfigured() {
		return ChatResult{}, fmt.Errorf("AI assistant is not configured (set provider API keys, enable Ollama, configure AWS Bedrock, or Azure AI Foundry)")
	}
	cfg, err := ResolveModel(modelID)
	if err != nil {
		return ChatResult{}, err
	}
	system := buildSystemContent(systemExtra)
	msgs, err := normalizeMessages(messages)
	if err != nil {
		return ChatResult{}, err
	}

	var reply string
	switch cfg.ID {
	case ProviderAnthropic:
		reply, err = chatAnthropic(ctx, cfg, system, msgs)
	case ProviderBedrock:
		reply, err = chatBedrock(ctx, cfg, system, msgs)
	default:
		// OpenAI and Ollama both use OpenAI-compatible /chat/completions.
		reply, err = chatOpenAICompatible(ctx, cfg, system, msgs)
	}
	if err != nil {
		return ChatResult{}, err
	}
	return ChatResult{
		Reply:    reply,
		Provider: cfg.ID,
		Model:    cfg.Model,
	}, nil
}

func buildSystemContent(systemExtra string) string {
	return strings.TrimSpace(SystemPrompt() + "\n\n" + systemExtra)
}

func normalizeMessages(messages []Message) ([]Message, error) {
	var apiMessages []Message
	for _, m := range messages {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		apiMessages = append(apiMessages, Message{Role: role, Content: content})
	}
	if len(apiMessages) == 0 {
		return nil, fmt.Errorf("message required")
	}
	return apiMessages, nil
}

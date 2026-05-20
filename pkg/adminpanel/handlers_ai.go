package adminpanel

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"ReaperC2/pkg/ai"
	"ReaperC2/pkg/dbconnections"
)

const maxAIChatMessages = 24
const maxAIChatMessageRunes = 12000

func (s *Server) handleAIPage(w http.ResponseWriter, r *http.Request) {
	user, role, ok := s.requireHTMLAuth(w, r)
	if !ok {
		return
	}
	eng, ok := s.requireActiveEngagement(w, r, user, role)
	if !ok {
		return
	}
	body := `
<h1>Operator AI</h1>
<p class="muted cmd-page-lead">Red team assistant for <strong>` + eng.Name + `</strong>. Use the <strong>Operator AI</strong> panel on the right edge of any page — click the tab on the screen edge or the button below to open it.</p>
<p><button type="button" class="btn" id="ai-page-open-drawer">Open Operator AI panel</button>
<a class="btn btn-secondary" href="/commands" style="margin-left:.5rem">Back to Commands</a></p>
<script>
document.getElementById('ai-page-open-drawer').onclick = function() {
  if (window.reaperOpenAIDrawer) window.reaperOpenAIDrawer();
};
if (window.reaperOpenAIDrawer) window.reaperOpenAIDrawer();
</script>`
	s.writeAppPage(w, user, role, "ai", "Operator AI", body, eng)
}

func (s *Server) handleAPIAIStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if _, _, ok := s.sessionUser(r); !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"configured":       ai.AnyConfigured(),
		"models":           ai.EnabledModels(),
		"default_model_id": ai.DefaultModelID(),
		"providers":        ai.Catalog(),
		"default_provider": ai.DefaultProviderID(),
	})
}

func (s *Server) handleAPIAIChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	user, role, ok := s.sessionUser(r)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	eng, ok := s.engagementForAPI(w, r, user, role)
	if !ok {
		return
	}
	if !ai.AnyConfigured() {
		jsonError(w, http.StatusServiceUnavailable, "AI assistant not configured (configure OpenAI, Anthropic, or Ollama)")
		return
	}

	var req struct {
		Provider string       `json:"provider"`
		Model    string       `json:"model"`
		Messages []ai.Message `json:"messages"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 512<<10)).Decode(&req); err != nil {
		jsonError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	msgs, err := normalizeAIChatMessages(req.Messages)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 130*time.Second)
	defer cancel()
	extra, err := ai.BuildEngagementContext(ctx, eng)
	if err != nil {
		log.Printf("admin: ai engagement context: %v", err)
		jsonError(w, http.StatusInternalServerError, "failed to build engagement context")
		return
	}

	modelID := strings.TrimSpace(req.Model)
	if modelID == "" {
		modelID = ai.ModelAuto
	} else if !strings.Contains(modelID, ":") && !strings.EqualFold(modelID, ai.ModelAuto) {
		if p := strings.TrimSpace(req.Provider); p != "" {
			modelID = p + ":" + modelID
		}
	}
	result, err := ai.Chat(ctx, modelID, extra, msgs)
	if err != nil {
		log.Printf("admin: ai chat: %v", err)
		jsonError(w, http.StatusBadGateway, err.Error())
		return
	}

	lastUser := msgs[len(msgs)-1].Content
	if aerr := dbconnections.InsertAuditLog(ctx, user, dbconnections.AuditActionAIChat, aiChatAuditDetails(
		result.Provider, result.Model, modelID, lastUser, result.Reply,
	), eng.ID.Hex()); aerr != nil {
		log.Printf("admin: audit ai chat: %v", aerr)
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"reply":    result.Reply,
		"provider": result.Provider,
		"model":    result.Model,
	})
}

func normalizeAIChatMessages(in []ai.Message) ([]ai.Message, error) {
	if len(in) == 0 {
		return nil, fmt.Errorf("messages required")
	}
	if len(in) > maxAIChatMessages {
		in = in[len(in)-maxAIChatMessages:]
	}
	out := make([]ai.Message, 0, len(in))
	for _, m := range in {
		role := strings.ToLower(strings.TrimSpace(m.Role))
		if role != "user" && role != "assistant" {
			continue
		}
		content := strings.TrimSpace(m.Content)
		if content == "" {
			continue
		}
		if utf8.RuneCountInString(content) > maxAIChatMessageRunes {
			content = truncateRunesAdmin(content, maxAIChatMessageRunes)
		}
		out = append(out, ai.Message{Role: role, Content: content})
	}
	if len(out) == 0 || out[len(out)-1].Role != "user" {
		return nil, fmt.Errorf("last message must be from user")
	}
	return out, nil
}

func truncateRunesAdmin(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

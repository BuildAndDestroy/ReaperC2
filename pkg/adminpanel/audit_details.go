package adminpanel

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
)

const (
	auditDetailsJSONMaxRunes   = 360
	auditDetailsAIChatMaxRunes = 8000
	maxAuditAIStoredRunes      = 32000
)

// formatAuditLogDetails renders a human-readable Details cell (JSON fallback for other actions).
func formatAuditLogDetails(e dbconnections.AuditLogEntry) string {
	if e.Action == dbconnections.AuditActionAIChat {
		return truncateAuditDetailRunes(formatAIChatAuditDetails(e.Details), auditDetailsAIChatMaxRunes)
	}
	if e.Details == nil {
		return ""
	}
	b, _ := json.Marshal(e.Details)
	return truncateAuditDetailRunes(string(b), auditDetailsJSONMaxRunes)
}

func formatAIChatAuditDetails(d bson.M) string {
	if d == nil {
		return ""
	}
	user, _ := d["user_message"].(string)
	reply, _ := d["assistant_reply"].(string)
	// Legacy rows stored only metadata.
	if user == "" && reply == "" {
		if u, ok := d["user_prompt"].(string); ok {
			user = u
		}
		b, _ := json.Marshal(d)
		return string(b)
	}
	var b strings.Builder
	if p, _ := d["provider"].(string); p != "" {
		b.WriteString("Provider: ")
		b.WriteString(p)
		if m, _ := d["model"].(string); m != "" {
			b.WriteString(" · ")
			b.WriteString(m)
		}
		if mid, _ := d["model_id"].(string); mid != "" {
			b.WriteString(" (")
			b.WriteString(mid)
			b.WriteString(")")
		}
		b.WriteString("\n\n")
	}
	if user != "" {
		b.WriteString("User:\n")
		b.WriteString(user)
		b.WriteString("\n\n")
	}
	if reply != "" {
		b.WriteString("Assistant:\n")
		b.WriteString(reply)
	}
	return strings.TrimSpace(b.String())
}

func truncateAuditDetailRunes(s string, max int) string {
	if max < 1 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

func truncateForAuditStorage(s string, max int) string {
	return truncateAuditDetailRunes(s, max)
}

func aiChatAuditDetails(provider, model, modelID, userMsg, reply string) bson.M {
	return bson.M{
		"provider":         provider,
		"model":            model,
		"model_id":         modelID,
		"user_message":     truncateForAuditStorage(userMsg, maxAuditAIStoredRunes),
		"assistant_reply":  truncateForAuditStorage(reply, maxAuditAIStoredRunes),
		"user_chars":       utf8.RuneCountInString(userMsg),
		"reply_chars":      utf8.RuneCountInString(reply),
	}
}

package adminpanel

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	auditDetailsGenericMaxRunes = 8000
	auditDetailsAIChatMaxRunes  = 8000
	maxAuditAIStoredRunes       = 32000
)

// preferredAuditDetailKeys lists detail map keys shown first when present; any other keys follow in sort order.
var preferredAuditDetailKeys = []string{
	"client_id",
	"engagement_id",
	"profile_id",
	"profile_name",
	"connection_type",
	"heartbeat_interval_sec",
	"profile_saved_ok",
	"command",
	"length",
	"output_preview",
	"output_bytes",
	"commands",
	"format",
	"redact",
	"scope",
	"rows",
	"entry_count",
	"chat_count",
	"audit_rows",
	"data_rows",
	"chat_rows",
	"new_username",
	"new_role",
	"target_username",
	"operators",
}

// formatAuditLogDetails renders a human-readable Details cell (JSON fallback for other actions).
func formatAuditLogDetails(e dbconnections.AuditLogEntry) string {
	if e.Action == dbconnections.AuditActionAIChat {
		return truncateAuditDetailRunes(formatAIChatAuditDetails(e.Details), auditDetailsAIChatMaxRunes)
	}
	if e.Details == nil || len(e.Details) == 0 {
		return ""
	}
	s := formatGenericAuditDetails(e.Details)
	if strings.TrimSpace(s) == "" {
		b, _ := json.Marshal(e.Details)
		s = string(b)
	}
	return truncateAuditDetailRunes(s, auditDetailsGenericMaxRunes)
}

func sortedAuditDetailKeys(m bson.M) []string {
	prefSet := make(map[string]struct{}, len(preferredAuditDetailKeys))
	for _, k := range preferredAuditDetailKeys {
		prefSet[k] = struct{}{}
	}
	var pref, rest []string
	for _, k := range preferredAuditDetailKeys {
		if _, ok := m[k]; ok {
			pref = append(pref, k)
		}
	}
	for k := range m {
		if _, ok := prefSet[k]; ok {
			continue
		}
		rest = append(rest, k)
	}
	sort.Strings(rest)
	return append(pref, rest...)
}

func auditDetailKeyLabel(k string) string {
	parts := strings.Split(k, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		low := strings.ToLower(p)
		switch low {
		case "id":
			parts[i] = "ID"
		default:
			parts[i] = strings.ToUpper(low[:1]) + low[1:]
		}
	}
	return strings.Join(parts, " ")
}

func formatGenericAuditDetails(m bson.M) string {
	keys := sortedAuditDetailKeys(m)
	var blocks []string
	for _, k := range keys {
		label := auditDetailKeyLabel(k) + ":"
		body := strings.TrimSpace(formatAuditDetailValue(m[k]))
		if body == "" {
			continue
		}
		if strings.Contains(body, "\n") || len(body) > 96 {
			blocks = append(blocks, label+"\n"+body)
		} else {
			blocks = append(blocks, label+" "+body)
		}
	}
	return strings.TrimSpace(strings.Join(blocks, "\n\n"))
}

func formatAuditDetailValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		if t {
			return "true"
		}
		return "false"
	case int:
		return fmt.Sprintf("%d", t)
	case int32:
		return fmt.Sprintf("%d", t)
	case int64:
		return fmt.Sprintf("%d", t)
	case uint32:
		return fmt.Sprintf("%d", t)
	case uint64:
		return fmt.Sprintf("%d", t)
	case float64:
		return fmt.Sprintf("%g", t)
	case float32:
		return fmt.Sprintf("%g", t)
	case []byte:
		return string(t)
	case primitive.ObjectID:
		return t.Hex()
	case primitive.DateTime:
		return t.Time().UTC().Format(timeRFC3339NanoTrim)
	case primitive.Timestamp:
		return fmt.Sprintf("(%d,%d)", t.T, t.I)
	case bson.M:
		return indentLines(formatGenericAuditDetails(t), "  ")
	case map[string]interface{}:
		return indentLines(formatGenericAuditDetails(bson.M(t)), "  ")
	case primitive.D:
		mm := bson.M{}
		for _, e := range t {
			mm[e.Key] = e.Value
		}
		return indentLines(formatGenericAuditDetails(mm), "  ")
	case []interface{}:
		return formatAuditDetailSlice(t)
	case []string:
		items := make([]interface{}, len(t))
		for i := range t {
			items[i] = t[i]
		}
		return formatAuditDetailSlice(items)
	case primitive.A:
		return formatAuditDetailSlice([]interface{}(t))
	default:
		b, err := json.MarshalIndent(t, "", "  ")
		if err != nil {
			return fmt.Sprint(t)
		}
		return strings.TrimSpace(string(b))
	}
}

const timeRFC3339NanoTrim = "2006-01-02T15:04:05.999999999Z07:00"

func indentLines(s, pfx string) string {
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		if lines[i] != "" {
			lines[i] = pfx + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func formatAuditDetailSlice(items []interface{}) string {
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	for i, it := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(fmt.Sprintf("%d. ", i+1))
		sub := strings.TrimSpace(formatAuditDetailValue(it))
		if strings.Contains(sub, "\n") {
			b.WriteString("\n")
			b.WriteString(indentLines(sub, "   "))
		} else {
			b.WriteString(sub)
		}
	}
	return b.String()
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

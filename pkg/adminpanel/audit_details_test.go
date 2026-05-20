package adminpanel

import (
	"strings"
	"testing"

	"ReaperC2/pkg/dbconnections"

	"go.mongodb.org/mongo-driver/bson"
)

func TestFormatAIChatAuditDetails(t *testing.T) {
	got := formatAIChatAuditDetails(bson.M{
		"provider":        "ollama",
		"model":           "gpt-oss",
		"user_message":    "hello",
		"assistant_reply": "world",
	})
	if !strings.Contains(got, "User:\nhello") || !strings.Contains(got, "Assistant:\nworld") {
		t.Fatalf("formatAIChatAuditDetails() = %q", got)
	}
}

func TestFormatAuditLogDetailsAIChatAction(t *testing.T) {
	got := formatAuditLogDetails(dbconnections.AuditLogEntry{
		Action: dbconnections.AuditActionAIChat,
		Details: bson.M{
			"user_message":    "ping",
			"assistant_reply": "pong",
		},
	})
	if !strings.Contains(got, "ping") || !strings.Contains(got, "pong") {
		t.Fatalf("formatAuditLogDetails() = %q", got)
	}
}

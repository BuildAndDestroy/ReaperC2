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

func TestFormatAuditLogDetailsBeaconOutputPlainText(t *testing.T) {
	got := formatAuditLogDetails(dbconnections.AuditLogEntry{
		Action: dbconnections.AuditActionBeaconOutputReceived,
		Details: bson.M{
			"client_id":      "cli-1",
			"command":        "whoami",
			"output_preview": "line1\nline2",
			"output_bytes":   42,
		},
	})
	if strings.Contains(got, `"client_id"`) {
		t.Fatalf("expected plain text, got %q", got)
	}
	if !strings.Contains(got, "cli-1") || !strings.Contains(got, "whoami") || !strings.Contains(got, "line1") {
		t.Fatalf("formatAuditLogDetails() = %q", got)
	}
}

func TestFormatAuditLogDetailsCommandsDelivered(t *testing.T) {
	got := formatAuditLogDetails(dbconnections.AuditLogEntry{
		Action: dbconnections.AuditActionBeaconCommandsDelivered,
		Details: bson.M{
			"client_id": "c2",
			"commands":  []interface{}{"cmd-a", "cmd-b"},
		},
	})
	if strings.Contains(got, `"commands"`) {
		t.Fatalf("expected plain text, got %q", got)
	}
	if !strings.Contains(got, "1. cmd-a") || !strings.Contains(got, "2. cmd-b") {
		t.Fatalf("formatAuditLogDetails() = %q", got)
	}
}

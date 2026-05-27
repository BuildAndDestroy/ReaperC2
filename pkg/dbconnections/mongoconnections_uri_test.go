package dbconnections

import (
	"net/url"
	"strings"
	"testing"
)

func TestBuildMongoURI_encodesPasswordWithSpecialChars(t *testing.T) {
	t.Setenv("MONGO_HOST", "mongo")
	t.Setenv("MONGO_PORT", "27017")
	t.Setenv("MONGO_USERNAME", "admin")
	t.Setenv("MONGO_PASSWORD", "~AwKG`Nwu6\\sx3t~p`Y`?xKYM-7N5_")
	t.Setenv("MONGO_DATABASE", "api_db")
	t.Setenv("MONGO_AUTH_SOURCE", "admin")

	uri := buildMongoURI("ONPREM")
	parsed, err := url.Parse(uri)
	if err != nil {
		t.Fatalf("parse uri: %v", err)
	}
	if parsed.Path != "/api_db" {
		t.Fatalf("path = %q, want /api_db", parsed.Path)
	}
	if parsed.Query().Get("authSource") != "admin" {
		t.Fatalf("authSource = %q, want admin", parsed.Query().Get("authSource"))
	}
	if strings.Contains(parsed.Path, "authSource") {
		t.Fatalf("authSource leaked into path: %q", parsed.Path)
	}
	pass, _ := parsed.User.Password()
	if pass != "~AwKG`Nwu6\\sx3t~p`Y`?xKYM-7N5_" {
		t.Fatalf("decoded password = %q", pass)
	}
}

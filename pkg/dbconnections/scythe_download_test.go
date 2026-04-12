package dbconnections

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseScytheDownloadPayload(t *testing.T) {
	raw := []byte("hello scythe")
	b64 := base64.StdEncoding.EncodeToString(raw)
	var sb strings.Builder
	sb.WriteString(scytheDownloadHeaderLine)
	sb.WriteString("\npath=/tmp/x.txt\nsize=")
	sb.WriteString("12")
	sb.WriteString("\nencoding=base64\n\n")
	sb.WriteString(b64)
	sb.WriteString("\n")

	path, out, ok := parseScytheDownloadPayload(sb.String())
	if !ok || path != "/tmp/x.txt" || string(out) != "hello scythe" {
		t.Fatalf("got path=%q ok=%v out=%q", path, ok, out)
	}
}

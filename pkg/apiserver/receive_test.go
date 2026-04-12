package apiserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func TestDecodeReceiveUUIDBody_ScytheLowercaseKeys(t *testing.T) {
	body := `{"command":"{\"op\":\"upload\"}","output":"SCYTHE_FILE_UPLOAD v1\npath=/tmp/x\nbytes_written=3\n"}`
	r, _ := http.NewRequest(http.MethodPost, "/receive/x", bytes.NewReader([]byte(body)))
	cmd, out, err := decodeReceiveUUIDBody(r)
	if err != nil {
		t.Fatal(err)
	}
	if cmd == "" || out == "" {
		t.Fatalf("expected non-empty command and output, got cmd=%q out=%q", cmd, out)
	}
}

func TestDecodeReceiveUUIDBody_CapitalKeys(t *testing.T) {
	body := `{"Command":"whoami","Output":"root\n"}`
	r, _ := http.NewRequest(http.MethodPost, "/receive/x", bytes.NewReader([]byte(body)))
	cmd, out, err := decodeReceiveUUIDBody(r)
	if err != nil {
		t.Fatal(err)
	}
	if cmd != "whoami" || out != "root\n" {
		t.Fatalf("got cmd=%q out=%q", cmd, out)
	}
}

func TestJSONStringField(t *testing.T) {
	var m map[string]interface{}
	_ = json.Unmarshal([]byte(`{"command":123}`), &m)
	if s := jsonStringField(m, "command"); s != "123" {
		t.Fatalf("got %q", s)
	}
}

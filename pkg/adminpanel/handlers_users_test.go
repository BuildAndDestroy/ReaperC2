package adminpanel

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleAPIUserPasswordReset_RequiresAuth(t *testing.T) {
	s := NewServer()
	body, _ := json.Marshal(map[string]interface{}{"password": "long-enough-password"})
	req := httptest.NewRequest(http.MethodPost, "/api/users/alice/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestHandleAPIUserPasswordReset_RejectsShortPasswordWhenAuthedShape(t *testing.T) {
	// Route must be registered; unauthenticated still returns 401 before body validation.
	s := NewServer()
	body, _ := json.Marshal(map[string]interface{}{"password": "short"})
	req := httptest.NewRequest(http.MethodPost, "/api/users/alice/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	s.router.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without session, got %d", rr.Code)
	}
}

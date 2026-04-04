package adminpanel

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestArgon2idRoundTrip(t *testing.T) {
	stored, err := HashOperatorPassword("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(stored, argon2idStoredPrefix) {
		t.Fatalf("expected argon2id prefix, got %q", stored[:min(32, len(stored))])
	}
	if !VerifyOperatorPassword(stored, "correct horse battery staple") {
		t.Fatal("verify should succeed")
	}
	if VerifyOperatorPassword(stored, "wrong") {
		t.Fatal("verify should fail for wrong password")
	}
}

func TestVerifyLegacyBcrypt(t *testing.T) {
	h, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if !VerifyOperatorPassword(string(h), "secret") {
		t.Fatal("bcrypt legacy verify should succeed")
	}
	if VerifyOperatorPassword(string(h), "nope") {
		t.Fatal("bcrypt should reject wrong password")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

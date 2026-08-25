package dbconnections

import "testing"

func TestCanonicalOperatorUsername(t *testing.T) {
	got, ok := canonicalOperatorUsername(" Alice.1_b-c ")
	if !ok || got != "Alice.1_b-c" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := canonicalOperatorUsername("alice{$ne:1}"); ok {
		t.Fatal("expected reject")
	}
	if _, ok := canonicalOperatorUsername(""); ok {
		t.Fatal("expected reject empty")
	}
}

func TestCanonicalOperatorRole(t *testing.T) {
	got, ok := canonicalOperatorRole(" ADMIN ")
	if !ok || got != RoleAdmin {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	got, ok = canonicalOperatorRole("operator")
	if !ok || got != RoleOperator {
		t.Fatalf("got %q ok=%v", got, ok)
	}
	if _, ok := canonicalOperatorRole("root"); ok {
		t.Fatal("expected reject")
	}
}

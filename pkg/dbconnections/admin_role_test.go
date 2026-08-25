package dbconnections

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestCanonicalOperatorUsername(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"alice", "alice", true},
		{"  Alice.1_b-c  ", "Alice.1_b-c", true},
		{"", "", false},
		{"   ", "", false},
		{"bad$name", "", false},
		{"alice{$ne:1}", "", false},
		{`{"$ne":""}`, "", false},
	}
	for _, tc := range cases {
		got, ok := canonicalOperatorUsername(tc.in)
		if ok != tc.ok || got != tc.want {
			t.Fatalf("canonicalOperatorUsername(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestCanonicalOperatorRole(t *testing.T) {
	t.Parallel()
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

func TestOperatorUsernameFilterUsesEq(t *testing.T) {
	t.Parallel()
	filter, ok := operatorUsernameFilter("alice")
	if !ok {
		t.Fatal("expected ok")
	}
	inner, ok := filter["username"].(bson.M)
	if !ok {
		t.Fatalf("username filter type %T", filter["username"])
	}
	if inner["$eq"] != "alice" {
		t.Fatalf("expected $eq alice, got %#v", inner)
	}
	if _, ok := operatorUsernameFilter(`{"$gt":""}`); ok {
		t.Fatal("operator-looking username must be rejected")
	}
}

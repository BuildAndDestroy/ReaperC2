package dbconnections

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeClientSearchQuery(t *testing.T) {
	t.Parallel()
	needle, err := NormalizeClientSearchQuery("  ACME  ")
	if err != nil {
		t.Fatal(err)
	}
	if needle != "acme" {
		t.Fatalf("got %q", needle)
	}
	if _, err := NormalizeClientSearchQuery(""); !errors.Is(err, ErrInvalidEngagementClientQuery) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := NormalizeClientSearchQuery("a$b"); !errors.Is(err, ErrInvalidEngagementClientQuery) {
		t.Fatalf("metachar: %v", err)
	}
	if _, err := NormalizeClientSearchQuery("O'Brien & Co."); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("a", maxEngagementClientSearchRunes+1)
	if _, err := NormalizeClientSearchQuery(long); err == nil || !errors.Is(err, ErrInvalidEngagementClientQuery) {
		t.Fatalf("long: %v", err)
	}
}

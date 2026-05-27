package ai

import "testing"

func TestBedrockMaxTokensInt32(t *testing.T) {
	if got := bedrockMaxTokensInt32(2048); got != 2048 {
		t.Fatalf("got %d", got)
	}
	if got := bedrockMaxTokensInt32(0); got != 1 {
		t.Fatalf("zero got %d", got)
	}
	if got := bedrockMaxTokensInt32(1<<62); got != 2147483647 {
		t.Fatalf("overflow got %d", got)
	}
}

package mitreattack

import "testing"

func TestEnterpriseMatrixForVersion_v19Stealth(t *testing.T) {
	c, err := EnterpriseMatrixForVersion(19)
	if err != nil {
		t.Fatal(err)
	}
	if !IsKnownEnterpriseTacticKey("stealth") {
		t.Fatal("v19 should include stealth tactic")
	}
	if _, ok := c.ByTactic["stealth"]; !ok {
		t.Fatal("catalog should list stealth techniques")
	}
}

func TestMatrixTechniquesForTactic_v19(t *testing.T) {
	list, err := MatrixTechniquesForTactic(19, "execution")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 10 {
		t.Fatalf("expected many execution techniques, got %d", len(list))
	}
}

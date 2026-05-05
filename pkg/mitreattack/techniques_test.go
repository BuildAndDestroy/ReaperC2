package mitreattack

import "testing"

func TestNormalizeTechniqueID(t *testing.T) {
	id, err := NormalizeTechniqueID("")
	if err != nil || id != "" {
		t.Fatalf("empty: %q %v", id, err)
	}
	id, err = NormalizeTechniqueID("  t1059  ")
	if err != nil || id != "T1059" {
		t.Fatalf("T1059: %q %v", id, err)
	}
	id, err = NormalizeTechniqueID("t1059.001")
	if err != nil || id != "T1059.001" {
		t.Fatalf("sub: %q %v", id, err)
	}
	_, err = NormalizeTechniqueID("X1059")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNormalizeTechniqueTags_Dedupe(t *testing.T) {
	out, err := NormalizeTechniqueTags([]TechniqueTag{
		{Tactic: "execution", TechniqueID: "T1059", Note: "a"},
		{Tactic: "execution", TechniqueID: "T1059", Note: "b"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Note != "a\n\nb" {
		t.Fatalf("got %#v", out)
	}
}

func TestNavigatorTechniqueLayerObjects(t *testing.T) {
	objs := NavigatorTechniqueLayerObjects([]TechniqueTag{
		{Tactic: "execution", TechniqueID: "T1059", Note: "psh"},
	})
	if len(objs) != 1 {
		t.Fatalf("len %d", len(objs))
	}
	if objs[0]["color"] != NavigatorHighlightColor {
		t.Fatalf("color %v", objs[0]["color"])
	}
	if objs[0]["techniqueID"] != "T1059" || objs[0]["tactic"] != "execution" {
		t.Fatalf("%v", objs[0])
	}
}

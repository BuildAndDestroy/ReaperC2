package mitreattack

import "testing"

func TestParseAttackVersion(t *testing.T) {
	v, err := ParseAttackVersion("")
	if err != nil || v != MaxAttackVersion {
		t.Fatalf("empty: got %d %v", v, err)
	}
	for _, s := range []string{"16", "17", "18", "19"} {
		got, err := ParseAttackVersion(s)
		want := int(s[0]-'0')*10 + int(s[1]-'0')
		if err != nil || got != want {
			t.Fatalf("version %s: got %d %v", s, got, err)
		}
	}
	_, err = ParseAttackVersion("15")
	if err == nil {
		t.Fatal("expected error for 15")
	}
	_, err = ParseAttackVersion("20")
	if err == nil {
		t.Fatal("expected error for 20")
	}
}

func TestNormalizeTacticNotes(t *testing.T) {
	m := NormalizeTacticNotes(map[string]string{
		"execution":      "  ran x  ",
		"bogus":          "drop",
		"initial-access": "",
	})
	if len(m) != 1 || m["execution"] != "ran x" {
		t.Fatalf("got %#v", m)
	}
}

func TestFullTacticNoteMap(t *testing.T) {
	full := FullTacticNoteMap(map[string]string{"impact": "test"})
	if full["impact"] != "test" || full["execution"] != "" {
		t.Fatalf("got %#v", full)
	}
	if len(full) != len(EnterpriseTactics()) {
		t.Fatalf("len %d", len(full))
	}
}

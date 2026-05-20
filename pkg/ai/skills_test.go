package ai

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadSkillsFromDiskMergesFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "SKILLS.md"), []byte("platform"), 0o644); err != nil {
		t.Fatal(err)
	}
	cursorDir := filepath.Join(dir, ".cursor", "skills", "reaper-red-team-operator")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "red_team_operator_skills.md"), []byte("tradecraft"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "mitre_attck_skills.md"), []byte("mitre"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "extra_skills.md"), []byte("extra"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REAPERC2_ROOT", dir)
	t.Setenv("REAPER_AI_SKILLS_FILE", "")

	got := string(readSkillsFromDisk())
	for _, want := range []string{"platform", "tradecraft", "mitre", "extra"} {
		if !strings.Contains(got, want) {
			t.Fatalf("readSkillsFromDisk() missing %q: %q", want, got)
		}
	}
}

func TestSkillPathsForRootDiscoversExtraSkills(t *testing.T) {
	dir := t.TempDir()
	cursorDir := filepath.Join(dir, ".cursor", "skills", "reaper-red-team-operator")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cursorDir, "custom_skills.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	paths := skillPathsForRoot(dir)
	found := false
	for _, p := range paths {
		if strings.HasSuffix(p, "custom_skills.md") {
			found = true
		}
	}
	if !found {
		t.Fatalf("skillPathsForRoot() = %v, want custom_skills.md", paths)
	}
}

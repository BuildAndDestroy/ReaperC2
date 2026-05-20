package ai

import (
	"embed"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

//go:embed SKILLS.md
var skillsFS embed.FS

var (
	skillsOnce sync.Once
	skillsText string
)

// skillFiles lists operator playbook paths relative to a repo root (merged in order; skip missing files).
var skillFiles = []string{
	"SKILLS.md",
	"red_team_operator_skills.md",
	"mitre_attck_skills.md",
	filepath.Join(".cursor", "skills", "reaper-red-team-operator", "red_team_operator_skills.md"),
	filepath.Join(".cursor", "skills", "reaper-red-team-operator", "mitre_attck_skills.md"),
}

// SystemPrompt returns the red team operator skill text for the model system role.
func SystemPrompt() string {
	skillsOnce.Do(func() {
		if b := readSkillsFromDisk(); len(b) > 0 {
			skillsText = string(b)
			return
		}
		skillsText = readSkillsEmbedded()
	})
	return skillsText
}

func readSkillsFromDisk() []byte {
	if p := strings.TrimSpace(os.Getenv("REAPER_AI_SKILLS_FILE")); p != "" {
		if b, err := os.ReadFile(p); err == nil && len(b) > 0 {
			return b
		}
	}
	var roots []string
	if root := strings.TrimSpace(os.Getenv("REAPERC2_ROOT")); root != "" {
		roots = append(roots, root)
	}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	var parts []string
	seen := map[string]bool{}
	for _, root := range roots {
		for _, rel := range skillPathsForRoot(root) {
			p := filepath.Join(root, rel)
			abs, err := filepath.Abs(p)
			if err != nil {
				abs = p
			}
			if seen[abs] {
				continue
			}
			b, err := os.ReadFile(p)
			if err != nil || len(b) == 0 {
				continue
			}
			seen[abs] = true
			parts = append(parts, strings.TrimSpace(string(b)))
		}
	}
	if len(parts) == 0 {
		return nil
	}
	return []byte(strings.Join(parts, "\n\n---\n\n"))
}

// skillPathsForRoot returns skillFiles plus any *_skills.md under the Cursor skill dir (except SKILL.md).
func skillPathsForRoot(root string) []string {
	out := append([]string{}, skillFiles...)
	seen := map[string]bool{}
	for _, rel := range out {
		seen[rel] = true
	}
	dir := filepath.Join(".cursor", "skills", "reaper-red-team-operator")
	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return out
	}
	var extra []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "SKILL.md" || !strings.HasSuffix(strings.ToLower(name), ".md") {
			continue
		}
		if !strings.HasSuffix(name, "_skills.md") {
			continue
		}
		rel := filepath.Join(dir, name)
		if seen[rel] {
			continue
		}
		extra = append(extra, rel)
	}
	sort.Slice(extra, func(i, j int) bool {
		return skillFileRank(filepath.Base(extra[i])) < skillFileRank(filepath.Base(extra[j]))
	})
	return append(out, extra...)
}

func skillFileRank(base string) int {
	switch base {
	case "red_team_operator_skills.md":
		return 0
	case "mitre_attck_skills.md":
		return 1
	default:
		return 50
	}
}

func readSkillsEmbedded() string {
	b, err := skillsFS.ReadFile("SKILLS.md")
	if err != nil || len(b) == 0 {
		return ""
	}
	return string(b)
}

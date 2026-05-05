package mitreattack

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// NavigatorHighlightColor matches ATT&CK Navigator’s default green for scored/annotated techniques.
const NavigatorHighlightColor = "#74c476"

// TechniqueTag ties a technique ID to an enterprise tactic and an optional per-technique note (Navigator comment).
type TechniqueTag struct {
	Tactic      string `json:"tactic" bson:"tactic"`
	TechniqueID string `json:"technique_id" bson:"technique_id"`
	Note        string `json:"note,omitempty" bson:"note,omitempty"`
}

// Limits for stored technique tags (engagement scope).
const (
	MaxTechniqueTags         = 200
	MaxTechniqueTagNoteRunes = 4000
	MaxTechniqueTagsNotesSum = 100000
)

var techniqueIDPattern = regexp.MustCompile(`(?i)^T[0-9]{4}(\.[0-9]{3})?$`)

// NormalizeTechniqueID returns canonical T#### / T####.### or empty string if s is empty.
func NormalizeTechniqueID(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	s = strings.ToUpper(s)
	if !techniqueIDPattern.MatchString(s) {
		return "", fmt.Errorf("invalid technique_id %q (use T#### or T####.###)", s)
	}
	return s, nil
}

// NormalizeTechniqueTags validates, deduplicates (by tactic + technique), merges notes, and sorts for stable export.
func NormalizeTechniqueTags(in []TechniqueTag) ([]TechniqueTag, error) {
	if len(in) > MaxTechniqueTags {
		return nil, fmt.Errorf("attack_techniques: at most %d rows", MaxTechniqueTags)
	}
	type key struct {
		tactic string
		id     string
	}
	type acc struct {
		notes []string
	}
	merged := make(map[key]*acc)
	var sumInputNotes int
	for _, row := range in {
		tac := strings.TrimSpace(row.Tactic)
		tid, err := NormalizeTechniqueID(row.TechniqueID)
		if err != nil {
			return nil, err
		}
		if tid == "" {
			continue
		}
		if !IsKnownEnterpriseTacticKey(tac) {
			return nil, fmt.Errorf("attack_techniques: unknown tactic %q for technique %s", tac, tid)
		}
		note := strings.TrimSpace(row.Note)
		if len([]rune(note)) > MaxTechniqueTagNoteRunes {
			return nil, fmt.Errorf("attack_techniques: note for %s exceeds %d characters", tid, MaxTechniqueTagNoteRunes)
		}
		sumInputNotes += len([]rune(note))
		if sumInputNotes > MaxTechniqueTagsNotesSum {
			return nil, fmt.Errorf("attack_techniques: combined notes too long (max %d characters)", MaxTechniqueTagsNotesSum)
		}
		k := key{tactic: tac, id: tid}
		a, ok := merged[k]
		if !ok {
			a = &acc{}
			merged[k] = a
		}
		if note != "" {
			a.notes = append(a.notes, note)
		}
	}
	if len(merged) == 0 {
		return nil, nil
	}
	tacticOrder := make(map[string]int, len(enterpriseTacticsOrdered))
	for i, t := range enterpriseTacticsOrdered {
		tacticOrder[t.Key] = i
	}
	var out []TechniqueTag
	var sumOut int
	for k, a := range merged {
		comment := strings.TrimSpace(strings.Join(a.notes, "\n\n"))
		if len([]rune(comment)) > MaxTechniqueTagNoteRunes {
			return nil, fmt.Errorf("attack_techniques: merged note for %s in %s exceeds %d characters", k.id, k.tactic, MaxTechniqueTagNoteRunes)
		}
		sumOut += len([]rune(comment))
		if sumOut > MaxTechniqueTagsNotesSum {
			return nil, fmt.Errorf("attack_techniques: combined notes too long (max %d characters)", MaxTechniqueTagsNotesSum)
		}
		out = append(out, TechniqueTag{Tactic: k.tactic, TechniqueID: k.id, Note: comment})
	}
	sort.Slice(out, func(i, j int) bool {
		oi := tacticOrder[out[i].Tactic]
		oj := tacticOrder[out[j].Tactic]
		if oi != oj {
			return oi < oj
		}
		return out[i].TechniqueID < out[j].TechniqueID
	})
	return out, nil
}

// NavigatorTechniqueLayerObjects builds layer "techniques" entries (color + comment + tactic).
func NavigatorTechniqueLayerObjects(tags []TechniqueTag) []map[string]interface{} {
	if len(tags) == 0 {
		return nil
	}
	var objs []map[string]interface{}
	for _, t := range tags {
		objs = append(objs, map[string]interface{}{
			"techniqueID": t.TechniqueID,
			"tactic":      t.Tactic,
			"color":       NavigatorHighlightColor,
			"comment":     t.Note,
			"enabled":     true,
		})
	}
	return objs
}

// NavigatorLegendDemonstrated returns a legend row explaining the highlight color when techniques are present.
func NavigatorLegendDemonstrated() map[string]interface{} {
	return map[string]interface{}{
		"label": "Demonstrated (ReaperC2)",
		"color": NavigatorHighlightColor,
	}
}

// ApplyTechniquesToNavigatorLayer sets layer "techniques" and a matching legend when tags is non-empty.
func ApplyTechniquesToNavigatorLayer(layer map[string]interface{}, tags []TechniqueTag) {
	objs := NavigatorTechniqueLayerObjects(tags)
	if len(objs) == 0 {
		return
	}
	t := make([]interface{}, len(objs))
	for i, o := range objs {
		t[i] = o
	}
	layer["techniques"] = t
	layer["legendItems"] = []interface{}{NavigatorLegendDemonstrated()}
}

// Package mitreattack holds MITRE ATT&CK enterprise matrix metadata used for engagement notes
// and ATT&CK Navigator layer exports. Tactic keys match Navigator's tactic shortnames
// (enterprise-attack) and are stable across ATT&CK releases v16–v19.
package mitreattack

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// MinAttackVersion and MaxAttackVersion bound supported ATT&CK STIX bundle versions for Navigator exports.
const MinAttackVersion = 16
const MaxAttackVersion = 19

// Tactic is one enterprise ATT&CK tactic row (Navigator / layer JSON "tactic" field uses Key).
type Tactic struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

// enterpriseTacticsOrdered is the full enterprise matrix tactic list in standard left-to-right order.
var enterpriseTacticsOrdered = []Tactic{
	{Key: "reconnaissance", Label: "Reconnaissance"},
	{Key: "resource-development", Label: "Resource Development"},
	{Key: "initial-access", Label: "Initial Access"},
	{Key: "execution", Label: "Execution"},
	{Key: "persistence", Label: "Persistence"},
	{Key: "privilege-escalation", Label: "Privilege Escalation"},
	{Key: "defense-evasion", Label: "Defense Evasion"},
	{Key: "credential-access", Label: "Credential Access"},
	{Key: "discovery", Label: "Discovery"},
	{Key: "lateral-movement", Label: "Lateral Movement"},
	{Key: "collection", Label: "Collection"},
	{Key: "command-and-control", Label: "Command and Control"},
	{Key: "exfiltration", Label: "Exfiltration"},
	{Key: "impact", Label: "Impact"},
}

// EnterpriseTactics returns a defensive copy of the ordered enterprise tactic list.
func EnterpriseTactics() []Tactic {
	out := make([]Tactic, len(enterpriseTacticsOrdered))
	copy(out, enterpriseTacticsOrdered)
	return out
}

var validTacticKey = func() map[string]struct{} {
	m := make(map[string]struct{}, len(enterpriseTacticsOrdered))
	for _, t := range enterpriseTacticsOrdered {
		m[t.Key] = struct{}{}
	}
	return m
}()

// IsValidTacticKey reports whether key is a known enterprise tactic shortname.
func IsValidTacticKey(key string) bool {
	_, ok := validTacticKey[strings.TrimSpace(key)]
	return ok
}

// NormalizeTacticNotes keeps only known tactic keys and trims whitespace; omits empty values.
func NormalizeTacticNotes(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string)
	for k, v := range in {
		k = strings.TrimSpace(k)
		if !IsValidTacticKey(k) {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// TacticNotesTotalLen returns the combined rune length of all note values (for size limits).
func TacticNotesTotalLen(m map[string]string) int {
	n := 0
	for _, v := range m {
		n += len([]rune(v))
	}
	return n
}

// FormatTacticNotesDescription builds plain text for Navigator layer description and reports.
func FormatTacticNotesDescription(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		label := k
		for _, t := range enterpriseTacticsOrdered {
			if t.Key == k {
				label = t.Label
				break
			}
		}
		b.WriteString(label)
		b.WriteString("\n")
		b.WriteString(m[k])
		b.WriteString("\n\n")
	}
	return strings.TrimSpace(b.String())
}

// ParseAttackVersion normalizes ?attack_version= for Navigator exports (16–19 inclusive).
func ParseAttackVersion(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return MaxAttackVersion, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("attack_version must be an integer")
	}
	if v < MinAttackVersion || v > MaxAttackVersion {
		return 0, fmt.Errorf("attack_version must be between %d and %d", MinAttackVersion, MaxAttackVersion)
	}
	return v, nil
}

// NavigatorLayer returns a layer object compatible with ATT&CK Navigator when serialized as JSON.
// techniques may be empty; description carries tactic-aligned engagement notes for reporting.
func NavigatorLayer(name, description string, attackVersion int) map[string]interface{} {
	av := strconv.Itoa(attackVersion)
	return map[string]interface{}{
		"name":        name,
		"domain":      "enterprise-attack",
		"description": description,
		"versions": map[string]interface{}{
			"attack":    av,
			"navigator": "4.10.0",
			"layer":     "4.4",
		},
		"filters": map[string]interface{}{},
		"sorting": 0,
		"layout": map[string]interface{}{
			"layout":              "side",
			"aggregateFunction":   "average",
			"showID":              false,
			"showName":            true,
			"showAggregateScores": false,
			"countUnscored":       false,
		},
		"hideDisabled":                  false,
		"techniques":                    []interface{}{},
		"gradient":                      defaultGradient(),
		"legendItems":                   []interface{}{},
		"metadata":                      []interface{}{},
		"links":                         []interface{}{},
		"showTacticRowBackground":       false,
		"selectTechniquesAcrossTactics": true,
		"selectSubtechniquesWithParent": true,
	}
}

func defaultGradient() map[string]interface{} {
	return map[string]interface{}{
		"colors": []string{
			"#ff6666ff",
			"#ffe766ff",
			"#8ec843ff",
		},
		"minValue": 0,
		"maxValue": 100,
	}
}

// MarshalNavigatorLayer JSON-encodes a Navigator layer with stable key ordering where possible.
func MarshalNavigatorLayer(layer map[string]interface{}) ([]byte, error) {
	return json.MarshalIndent(layer, "", "  ")
}

// FullTacticNoteMap returns every enterprise tactic key with stored text or empty string (API / forms).
func FullTacticNoteMap(stored map[string]string) map[string]string {
	out := make(map[string]string, len(enterpriseTacticsOrdered))
	for _, t := range enterpriseTacticsOrdered {
		out[t.Key] = ""
	}
	if stored == nil {
		return out
	}
	for k, v := range stored {
		if !IsValidTacticKey(k) {
			continue
		}
		out[k] = v
	}
	return out
}

// CompactTacticNotesForExport returns only non-empty tactic notes, or nil if none (JSON report / brevity).
func CompactTacticNotesForExport(stored map[string]string) map[string]string {
	return NormalizeTacticNotes(stored)
}

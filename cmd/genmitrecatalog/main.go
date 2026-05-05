// Command genmitrecatalog downloads MITRE ATT&CK enterprise STIX bundles (v16–v19)
// and writes trimmed JSON catalogs for ReaperC2 (tactics + techniques by tactic).
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func main() {
	outDir := "pkg/mitreattack/catalogdata"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		die(err)
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	for _, ver := range []int{16, 17, 18, 19} {
		url := fmt.Sprintf("https://raw.githubusercontent.com/mitre/cti/ATT%%26CK-v%d.0/enterprise-attack/enterprise-attack.json", ver)
		fmt.Fprintf(os.Stderr, "fetching v%d…\n", ver)
		raw, err := fetch(client, url)
		if err != nil {
			die(fmt.Errorf("v%d: %w", ver, err))
		}
		cat, err := buildCatalog(ver, raw)
		if err != nil {
			die(fmt.Errorf("v%d parse: %w", ver, err))
		}
		path := filepath.Join(outDir, fmt.Sprintf("enterprise-%d.json", ver))
		b, err := json.MarshalIndent(cat, "", "  ")
		if err != nil {
			die(err)
		}
		if err := os.WriteFile(path, b, 0o644); err != nil {
			die(err)
		}
		fmt.Fprintf(os.Stderr, "wrote %s (%d tactics, %d technique rows)\n", path, len(cat.Tactics), countTechniques(cat))
	}
}

func die(err error) {
	fmt.Fprintf(os.Stderr, "genmitrecatalog: %v\n", err)
	os.Exit(1)
}

func fetch(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

type bundle struct {
	Objects []map[string]interface{} `json:"objects"`
}

type outCatalog struct {
	Version  int              `json:"version"`
	Tactics  []tacticOut      `json:"tactics"`
	ByTactic map[string][]ref `json:"by_tactic"`
}

type tacticOut struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type ref struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func buildCatalog(version int, raw []byte) (*outCatalog, error) {
	var b bundle
	if err := json.Unmarshal(raw, &b); err != nil {
		return nil, err
	}
	tacticLabel := map[string]string{}
	for _, o := range b.Objects {
		if typ, _ := o["type"].(string); typ != "x-mitre-tactic" {
			continue
		}
		if truthy(o["revoked"]) || truthy(o["x_mitre_deprecated"]) {
			continue
		}
		sn, _ := o["x_mitre_shortname"].(string)
		name, _ := o["name"].(string)
		sn = strings.TrimSpace(sn)
		if sn == "" || name == "" {
			continue
		}
		tacticLabel[sn] = name
	}
	// technique -> set of tactics (dedupe entries per tactic)
	type pair struct{ tactic, id, name string }
	seen := map[pair]struct{}{}
	var pairs []pair
	for _, o := range b.Objects {
		if typ, _ := o["type"].(string); typ != "attack-pattern" {
			continue
		}
		if truthy(o["revoked"]) || truthy(o["x_mitre_deprecated"]) {
			continue
		}
		doms, _ := o["x_mitre_domains"].([]interface{})
		if !hasString(doms, "enterprise-attack") {
			continue
		}
		extID := mitreTechniqueExternalID(o["external_references"])
		if extID == "" || !strings.HasPrefix(extID, "T") {
			continue
		}
		name, _ := o["name"].(string)
		kc, _ := o["kill_chain_phases"].([]interface{})
		for _, rawp := range kc {
			p, ok := rawp.(map[string]interface{})
			if !ok {
				continue
			}
			if kcn, _ := p["kill_chain_name"].(string); kcn != "mitre-attack" {
				continue
			}
			ph, _ := p["phase_name"].(string)
			ph = strings.TrimSpace(ph)
			if ph == "" {
				continue
			}
			pr := pair{tactic: ph, id: extID, name: name}
			if _, ok := seen[pr]; ok {
				continue
			}
			seen[pr] = struct{}{}
			pairs = append(pairs, pr)
		}
	}
	byTactic := map[string][]ref{}
	for _, p := range pairs {
		byTactic[p.tactic] = append(byTactic[p.tactic], ref{ID: p.id, Name: p.name})
	}
	for t, refs := range byTactic {
		sort.Slice(refs, func(i, j int) bool { return techniqueIDLess(refs[i].ID, refs[j].ID) })
		byTactic[t] = dedupeRefs(refs)
	}
	var tactics []tacticOut
	for key := range byTactic {
		label := tacticLabel[key]
		if label == "" {
			label = key
		}
		tactics = append(tactics, tacticOut{Key: key, Label: label})
	}
	sort.Slice(tactics, func(i, j int) bool {
		return tactics[i].Key < tactics[j].Key
	})
	return &outCatalog{Version: version, Tactics: tactics, ByTactic: byTactic}, nil
}

func dedupeRefs(refs []ref) []ref {
	seen := map[string]struct{}{}
	var out []ref
	for _, r := range refs {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}
		out = append(out, r)
	}
	return out
}

func techniqueIDLess(a, b string) bool {
	ap := parseTechID(a)
	bp := parseTechID(b)
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if ap[i] != bp[i] {
			return ap[i] < bp[i]
		}
	}
	return len(ap) < len(bp)
}

func parseTechID(s string) []int {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(strings.ToUpper(s), "T")
	var parts []int
	for _, piece := range strings.Split(s, ".") {
		var n int
		for _, c := range piece {
			if c < '0' || c > '9' {
				return []int{999999}
			}
			n = n*10 + int(c-'0')
		}
		parts = append(parts, n)
	}
	return parts
}

func mitreTechniqueExternalID(raw interface{}) string {
	arr, ok := raw.([]interface{})
	if !ok {
		return ""
	}
	for _, x := range arr {
		m, ok := x.(map[string]interface{})
		if !ok {
			continue
		}
		if sn, _ := m["source_name"].(string); sn != "mitre-attack" {
			continue
		}
		eid, _ := m["external_id"].(string)
		return strings.TrimSpace(eid)
	}
	return ""
}

func hasString(arr []interface{}, want string) bool {
	for _, x := range arr {
		if s, ok := x.(string); ok && s == want {
			return true
		}
	}
	return false
}

func truthy(v interface{}) bool {
	b, ok := v.(bool)
	return ok && b
}

func countTechniques(c *outCatalog) int {
	n := 0
	for _, refs := range c.ByTactic {
		n += len(refs)
	}
	return n
}

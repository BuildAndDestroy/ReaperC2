package mitreattack

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

//go:embed catalogdata
var catalogFS embed.FS

// EnterpriseMatrixData is trimmed MITRE enterprise-attack data for one STIX release (embedded JSON).
type EnterpriseMatrixData struct {
	Version  int                        `json:"version"`
	Tactics  []matrixCatalogTactic      `json:"tactics"`
	ByTactic map[string][]matrixTechRef `json:"by_tactic"`
}

type matrixCatalogTactic struct {
	Key   string `json:"key"`
	Label string `json:"label"`
}

type matrixTechRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var (
	catalogLoadOnce sync.Once
	catalogs        map[int]*EnterpriseMatrixData
	catalogLoadErr  error
)

func loadCatalogs() {
	catalogs = make(map[int]*EnterpriseMatrixData)
	for v := MinAttackVersion; v <= MaxAttackVersion; v++ {
		fp := path.Join("catalogdata", fmt.Sprintf("enterprise-%d.json", v))
		b, err := catalogFS.ReadFile(fp)
		if err != nil {
			catalogLoadErr = fmt.Errorf("read %s: %w", fp, err)
			return
		}
		var c EnterpriseMatrixData
		if err := json.Unmarshal(b, &c); err != nil {
			catalogLoadErr = fmt.Errorf("parse enterprise-%d: %w", v, err)
			return
		}
		catalogs[v] = &c
	}
}

// EnterpriseMatrixForVersion returns embedded catalog data for an ATT&CK STIX major version (16–19).
func EnterpriseMatrixForVersion(version int) (*EnterpriseMatrixData, error) {
	catalogLoadOnce.Do(loadCatalogs)
	if catalogLoadErr != nil {
		return nil, catalogLoadErr
	}
	c, ok := catalogs[version]
	if !ok {
		return nil, fmt.Errorf("unknown ATT&CK catalog version %d", version)
	}
	return c, nil
}

// MatrixTactics returns tactic keys and labels for Navigator/matrix version.
func MatrixTactics(version int) ([]Tactic, error) {
	c, err := EnterpriseMatrixForVersion(version)
	if err != nil {
		return nil, err
	}
	out := make([]Tactic, len(c.Tactics))
	for i, t := range c.Tactics {
		out[i] = Tactic{Key: t.Key, Label: t.Label}
	}
	return out, nil
}

// IsKnownEnterpriseTacticKey reports whether key is a tactic shortname in any embedded matrix catalog (v16–v19).
func IsKnownEnterpriseTacticKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return false
	}
	catalogLoadOnce.Do(loadCatalogs)
	if catalogLoadErr != nil {
		return IsValidTacticKey(key)
	}
	for v := MinAttackVersion; v <= MaxAttackVersion; v++ {
		c := catalogs[v]
		if c == nil {
			continue
		}
		for _, t := range c.Tactics {
			if t.Key == key {
				return true
			}
		}
	}
	return false
}

// MatrixTechniquesForTactic lists techniques under one tactic for a matrix version.
func MatrixTechniquesForTactic(version int, tactic string) ([]CatalogTechnique, error) {
	c, err := EnterpriseMatrixForVersion(version)
	if err != nil {
		return nil, err
	}
	tactic = strings.TrimSpace(tactic)
	refs, ok := c.ByTactic[tactic]
	if !ok || len(refs) == 0 {
		return nil, nil
	}
	out := make([]CatalogTechnique, len(refs))
	for i, r := range refs {
		out[i] = CatalogTechnique{ID: r.ID, Name: r.Name}
	}
	return out, nil
}

// CatalogTechnique is one ATT&CK technique for UI dropdowns and API JSON.
type CatalogTechnique struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// MatrixTechniqueKeysForTactic returns sorted technique IDs (validation helper).
func MatrixTechniqueKeysForTactic(version int, tactic string) ([]string, error) {
	list, err := MatrixTechniquesForTactic(version, tactic)
	if err != nil {
		return nil, err
	}
	keys := make([]string, len(list))
	for i := range list {
		keys[i] = list[i].ID
	}
	sort.Strings(keys)
	return keys, nil
}

package mitreattack

import (
	"sort"
	"strings"
)

// tacticsContainingTechnique returns tactic keys where id appears in the matrix catalog.
func tacticsContainingTechnique(c *EnterpriseMatrixData, id string) []string {
	id = strings.ToUpper(strings.TrimSpace(id))
	var out []string
	for tac, refs := range c.ByTactic {
		for _, r := range refs {
			if strings.ToUpper(strings.TrimSpace(r.ID)) == id {
				out = append(out, tac)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}

func tacticOrderIndex(c *EnterpriseMatrixData) map[string]int {
	m := make(map[string]int, len(c.Tactics))
	for i, t := range c.Tactics {
		m[t.Key] = i
	}
	return m
}

// sortTechniqueTagsForCatalogOrder sorts by embedded matrix tactic order, then technique ID.
func sortTechniqueTagsForCatalogOrder(c *EnterpriseMatrixData, tags []TechniqueTag) {
	order := tacticOrderIndex(c)
	sort.Slice(tags, func(i, j int) bool {
		oi, okI := order[tags[i].Tactic]
		oj, okJ := order[tags[j].Tactic]
		switch {
		case okI && okJ && oi != oj:
			return oi < oj
		case okI != okJ:
			return okI && !okJ
		default:
			return tags[i].TechniqueID < tags[j].TechniqueID
		}
	})
}

type tacticTechniqueKey struct {
	tactic string
	id     string
}

func mergeTechniqueTagsByTacticAndID(in []TechniqueTag) []TechniqueTag {
	mergedNote := make(map[tacticTechniqueKey]string)
	keyOrder := make([]tacticTechniqueKey, 0)
	seen := make(map[tacticTechniqueKey]struct{})
	for _, t := range in {
		k := tacticTechniqueKey{tactic: t.Tactic, id: t.TechniqueID}
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			keyOrder = append(keyOrder, k)
		}
		prev := mergedNote[k]
		note := strings.TrimSpace(t.Note)
		switch {
		case note == "":
			// keep prev
		case prev == "":
			mergedNote[k] = note
		case note == prev:
			// duplicate
		case strings.Contains(prev, note):
			// already covered
		default:
			mergedNote[k] = prev + "\n\n" + note
		}
	}
	out := make([]TechniqueTag, 0, len(keyOrder))
	for _, k := range keyOrder {
		out = append(out, TechniqueTag{
			Tactic:      k.tactic,
			TechniqueID: k.id,
			Note:        mergedNote[k],
		})
	}
	return out
}

// expandTagForNavigatorExport places one stored tag on every matrix tactic column that lists
// the same technique ID (e.g. T1078.003 in Initial Access, Persistence, …). It does not add
// sibling sub-techniques the operator did not tag.
func expandTagForNavigatorExport(c *EnterpriseMatrixData, tag TechniqueTag) []TechniqueTag {
	tactics := tacticsContainingTechnique(c, tag.TechniqueID)
	if len(tactics) == 0 {
		return []TechniqueTag{tag}
	}
	out := make([]TechniqueTag, 0, len(tactics))
	for _, tac := range tactics {
		out = append(out, TechniqueTag{Tactic: tac, TechniqueID: tag.TechniqueID, Note: tag.Note})
	}
	return out
}

// ExpandTechniqueTagsForNavigatorExport duplicates each tag across all tactics where that exact
// technique ID appears in the embedded matrix for attackVersion (16–19: enterprise-{version}.json).
func ExpandTechniqueTagsForNavigatorExport(attackVersion int, tags []TechniqueTag) ([]TechniqueTag, error) {
	if len(tags) == 0 {
		return nil, nil
	}
	c, err := EnterpriseMatrixForVersion(attackVersion)
	if err != nil {
		return nil, err
	}
	var flat []TechniqueTag
	for i := range tags {
		flat = append(flat, expandTagForNavigatorExport(c, tags[i])...)
	}
	out := mergeTechniqueTagsByTacticAndID(flat)
	sortTechniqueTagsForCatalogOrder(c, out)
	return out, nil
}

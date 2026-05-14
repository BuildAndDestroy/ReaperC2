package mitreattack

import (
	"strconv"
	"strings"
	"testing"
)

var matrixCatalogVersions = []int{16, 17, 18, 19}

func TestExpandTechniqueTagsForNavigatorExport_T1078_003_allMatrixVersions(t *testing.T) {
	for _, ver := range matrixCatalogVersions {
		ver := ver
		t.Run("v"+strconv.Itoa(ver), func(t *testing.T) {
			out, err := ExpandTechniqueTagsForNavigatorExport(ver, []TechniqueTag{
				{Tactic: "initial-access", TechniqueID: "T1078.003", Note: "got heem"},
			})
			if err != nil {
				t.Fatal(err)
			}
			c, err := EnterpriseMatrixForVersion(ver)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tacticsContainingTechnique(c, "T1078.003"))
			if want < 1 {
				t.Fatal("catalog should list T1078.003")
			}
			if len(out) != want {
				t.Fatalf("len(out)=%d want %d (catalog tactics for T1078.003): %+v", len(out), want, out)
			}
			seen := make(map[string]bool)
			for _, row := range out {
				if row.TechniqueID != "T1078.003" {
					t.Errorf("unexpected id %s", row.TechniqueID)
				}
				if row.Note != "got heem" {
					t.Errorf("unexpected note %q", row.Note)
				}
				seen[row.Tactic] = true
			}
			if len(seen) != want {
				t.Fatalf("tactic count: got %d want %d", len(seen), want)
			}
		})
	}
}

func TestExpandTechniqueTagsForNavigatorExport_T1078_parent_allMatrixVersions(t *testing.T) {
	for _, ver := range matrixCatalogVersions {
		ver := ver
		t.Run("v"+strconv.Itoa(ver), func(t *testing.T) {
			out, err := ExpandTechniqueTagsForNavigatorExport(ver, []TechniqueTag{
				{Tactic: "persistence", TechniqueID: "T1078", Note: "parent"},
			})
			if err != nil {
				t.Fatal(err)
			}
			c, err := EnterpriseMatrixForVersion(ver)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tacticsContainingTechnique(c, "T1078"))
			if len(out) != want {
				t.Fatalf("len=%d want %d: %+v", len(out), want, out)
			}
			for _, row := range out {
				if row.TechniqueID != "T1078" {
					t.Errorf("want T1078 only, got %s", row.TechniqueID)
				}
			}
		})
	}
}

func TestExpandTechniqueTagsForNavigatorExport_T1059_001_allMatrixVersions(t *testing.T) {
	for _, ver := range matrixCatalogVersions {
		ver := ver
		t.Run("v"+strconv.Itoa(ver), func(t *testing.T) {
			out, err := ExpandTechniqueTagsForNavigatorExport(ver, []TechniqueTag{
				{Tactic: "execution", TechniqueID: "T1059.001", Note: "ps"},
			})
			if err != nil {
				t.Fatal(err)
			}
			c, err := EnterpriseMatrixForVersion(ver)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tacticsContainingTechnique(c, "T1059.001"))
			if len(out) != want {
				t.Fatalf("len=%d want %d: %+v", len(out), want, out)
			}
			for _, row := range out {
				if row.TechniqueID != "T1059.001" || row.Note != "ps" {
					t.Fatalf("%+v", row)
				}
			}
		})
	}
}

func TestExpandTechniqueTagsForNavigatorExport_T1005_allMatrixVersions(t *testing.T) {
	for _, ver := range matrixCatalogVersions {
		ver := ver
		t.Run("v"+strconv.Itoa(ver), func(t *testing.T) {
			out, err := ExpandTechniqueTagsForNavigatorExport(ver, []TechniqueTag{
				{Tactic: "collection", TechniqueID: "T1005", Note: "only here"},
			})
			if err != nil {
				t.Fatal(err)
			}
			c, err := EnterpriseMatrixForVersion(ver)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tacticsContainingTechnique(c, "T1005"))
			if len(out) != want {
				t.Fatalf("len=%d want %d: %+v", len(out), want, out)
			}
			for _, row := range out {
				if row.TechniqueID != "T1005" || row.Note != "only here" {
					t.Fatalf("%+v", row)
				}
			}
		})
	}
}

func TestExpandTechniqueTagsForNavigatorExport_MergeSameID_allMatrixVersions(t *testing.T) {
	for _, ver := range matrixCatalogVersions {
		ver := ver
		t.Run("v"+strconv.Itoa(ver), func(t *testing.T) {
			out, err := ExpandTechniqueTagsForNavigatorExport(ver, []TechniqueTag{
				{Tactic: "initial-access", TechniqueID: "T1078.003", Note: "a"},
				{Tactic: "persistence", TechniqueID: "T1078.003", Note: "b"},
			})
			if err != nil {
				t.Fatal(err)
			}
			c, err := EnterpriseMatrixForVersion(ver)
			if err != nil {
				t.Fatal(err)
			}
			want := len(tacticsContainingTechnique(c, "T1078.003"))
			if len(out) != want {
				t.Fatalf("len=%d want %d", len(out), want)
			}
			for _, row := range out {
				if !strings.Contains(row.Note, "a") || !strings.Contains(row.Note, "b") {
					t.Fatalf("each tactic row should merge notes; got %q for %s", row.Note, row.Tactic)
				}
			}
		})
	}
}

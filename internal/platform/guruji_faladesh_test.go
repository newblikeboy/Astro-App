package platform

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestGurujiSourceCoverage checks every planet/house, remedy, conditional pair and archived source hash.
func TestGurujiSourceCoverage(t *testing.T) {
	seen := map[string]bool{}
	for _, placement := range gurujiReferences.Placements {
		key := placement.Planet + ":" + string(rune('A'+placement.House))
		if seen[key] || placement.House < 1 || placement.House > 12 || len(placement.HindiExcerpt) < 300 || placement.Source.PageStart < 1 || placement.Source.PageEnd < placement.Source.PageStart || placement.Source.PageEnd > 125 || placement.Summary.English == "" {
			t.Fatalf("invalid/duplicate source placement: %s %+v", key, placement.Source)
		}
		seen[key] = true
	}
	if len(seen) != 108 {
		t.Fatal("incomplete planet/house coverage")
	}
	for _, house := range gurujiReferences.HouseReferences {
		if house.House < 1 || house.House > 12 || len(house.Positive.HindiExcerpt) < 150 || len(house.Negative.HindiExcerpt) < 150 {
			t.Fatal("missing conditional house branch")
		}
	}
	for _, source := range gurujiReferences.Sources {
		bytes, err := os.ReadFile(filepath.Join("../..", source.Archive))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(bytes)
		if hex.EncodeToString(hash[:]) != source.SHA256 {
			t.Fatalf("source hash changed: %s", source.ID)
		}
	}
	for _, row := range gurujiRoles {
		seenRoles := map[string]bool{}
		for _, group := range [][]string{row.Karak, row.Marak, row.Neutral} {
			for _, planet := range group {
				if seenRoles[planet] {
					t.Fatalf("duplicate role for lagna %d: %s", row.Lagna, planet)
				}
				seenRoles[planet] = true
			}
		}
	}
}

// TestGurujiNarrativeSourceAnchoring checks complete bilingual copy and retains Hindi polarity conditions verbatim.
func TestGurujiNarrativeSourceAnchoring(t *testing.T) {
	for _, placement := range gurujiReferences.Placements {
		narrative, ok := gurujiNarratives[placement.Planet+":"+fmt.Sprint(placement.House)]
		if !ok || narrative.Version != "m4y-source-reading-v2" || len(narrative.Meaning.Hindi) < 20 || len(narrative.Meaning.English) < 20 {
			t.Fatal("missing personal reading summary", placement.Planet, placement.House)
		}
		normalized := strings.Join(strings.Fields(placement.HindiExcerpt), " ")
		for _, branch := range []string{narrative.Support.Hindi, narrative.Caution.Hindi} {
			if !strings.Contains(normalized, strings.Join(strings.Fields(branch), " ")) {
				t.Fatal("Hindi source condition changed", placement.Planet, placement.House, branch)
			}
		}
	}
}

// TestGurujiRoleAndDignity preserves teacher-specific role differences and contradictory strength evidence.
func TestGurujiRoleAndDignity(t *testing.T) {
	for _, sample := range []struct {
		lagna         int
		planet        string
		sign          int
		role, dignity string
	}{
		{1, "Sun", 1, "karak", "exalted"},
		{1, "Sun", 7, "karak", "debilitated"},
		{1, "Venus", 12, "marak", "exalted"},
		{1, "Venus", 6, "marak", "debilitated"},
		{2, "Sun", 1, "neutral", "exalted"},
		{3, "Jupiter", 4, "karak", "exalted"},
		{12, "Mercury", 6, "karak", "exalted"},
		{1, "Rahu", 2, "unlisted", "exalted"},
		{1, "Rahu", 3, "unlisted", "exalted"},
		{1, "Ketu", 3, "unlisted", "debilitated"},
		{1, "Ketu", 9, "unlisted", "exalted"},
	} {
		role, dignity := gurujiRole(sample.lagna, sample.planet), gurujiDignity(sample.planet, sample.sign)
		if role != sample.role || dignity != sample.dignity {
			t.Fatalf("incorrect supplied convention: %+v; %s %s", sample, role, dignity)
		}
	}
	if suppliedGurujiFaladesh(nil) != nil {
		t.Fatal("missing chart invented readings")
	}
	for _, classical := range []struct {
		planet               string
		exalted, debilitated int
	}{{"Sun", 1, 7}, {"Moon", 2, 8}, {"Mars", 10, 4}, {"Mercury", 6, 12}, {"Jupiter", 4, 10}, {"Venus", 12, 6}, {"Saturn", 7, 1}} {
		if gurujiDignity(classical.planet, classical.exalted) != "exalted" || gurujiDignity(classical.planet, classical.debilitated) != "debilitated" {
			t.Fatal("classical dignity table mismatch", classical)
		}
	}
}

// TestGurujiSnapshotAndApprovalIsolation checks real matching, unknown-time suppression and immutable serialization.
func TestGurujiSnapshotAndApprovalIsolation(t *testing.T) {
	a := testApp(t)
	report, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	if report.GurujiFaladesh == nil || len(report.GurujiFaladesh.Readings) != 9 || len(report.Rules) != 0 {
		t.Fatal("source reading coverage or approval isolation failed")
	}
	for i, planet := range report.Chart.Planets {
		reading := report.GurujiFaladesh.Readings[i]
		if reading.Planet != planet.Name || reading.House != planet.House || reading.Remedy.Planet != planet.Name {
			t.Fatal("reading/placement mismatch")
		}
		if reading.Narrative == nil || reading.Narrative.Version != "m4y-source-reading-v2" {
			t.Fatal("new report omitted snapshotted narrative")
		}
	}
	bytes, _ := json.Marshal(report)
	var restored Report
	if err := json.Unmarshal(bytes, &restored); err != nil || restored.GurujiFaladesh.Readings[0].HindiExcerpt != report.GurujiFaladesh.Readings[0].HindiExcerpt {
		t.Fatal("source snapshot lost", err)
	}
	restored.GurujiFaladesh.Readings[0].HindiExcerpt = "modified copy"
	if report.GurujiFaladesh.Readings[0].HindiExcerpt == "modified copy" {
		t.Fatal("snapshot alias")
	}
	unknown := testBirth()
	unknown.Unknown = true
	u, err := calculate(a.Config, unknown, nil, a.Now())
	if err != nil || u.GurujiFaladesh != nil {
		t.Fatal("unknown-time chart invented source readings", err)
	}
	core, err := calculateCore(a.Config, testBirth(), nil, a.Now())
	if err != nil || core.GurujiFaladesh != nil {
		t.Fatal("source catalogue added to core-only calculation", err)
	}
	legacy := &Report{}
	if err := json.Unmarshal([]byte(`{"rules":[]}`), legacy); err != nil || legacy.GurujiFaladesh != nil {
		t.Fatal("old snapshot backfilled")
	}
	for _, language := range []string{"en", "hi"} {
		l := pdfLocale{language}
		section := l.gurujiSection(report.GurujiFaladesh)
		if len(section.Blocks) != 19 || !strings.Contains(section.Blocks[0].Note, report.GurujiFaladesh.Readings[0].HindiExcerpt) {
			t.Fatal("PDF lost original wording or remedy references")
		}
		if language == "en" {
			for i := range section.Blocks {
				for _, reading := range report.GurujiFaladesh.Readings {
					section.Blocks[i].Note = strings.ReplaceAll(section.Blocks[i].Note, reading.HindiExcerpt, "")
					section.Blocks[i].Note = strings.ReplaceAll(section.Blocks[i].Note, reading.Remedy.HindiExcerpt, "")
				}
				for _, source := range report.GurujiFaladesh.Sources {
					section.Blocks[i].Note = strings.ReplaceAll(section.Blocks[i].Note, source.Filename, "")
				}
			}
			owned, _ := json.Marshal(section)
			if regexp.MustCompile(`[\x{0900}-\x{097f}]`).Match(owned) {
				t.Fatal("untranslated Guruji application labels")
			}
		}
	}
	// Optional synthetic browser fixtures never access application customer storage.
	if directory := os.Getenv("GURUJI_QA_DIR"); directory != "" {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, "report.json"), bytes, 0600); err != nil {
			t.Fatal(err)
		}
		renderer := ChromePDF{FontPath: a.Config.FontPath, Origin: a.Config.Origin}
		for _, language := range []string{"en", "hi"} {
			html, err := renderer.documentLanguage(&Order{ID: "qa-guruji-references", Snapshot: report, Amount: 9900, Demo: true}, language)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(directory, "report-"+language+".html"), html, 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
}

package platform

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// findingRajaScenario isolates the audited Raja Babu D1 positions; astronomy itself remains covered by independent native fixtures.
func findingRajaScenario() *Report {
	r := synthesisScenario()
	r.Chart.Lagna.RashiNumber = 11
	signs := map[string]int{"Sun": 8, "Moon": 11, "Mars": 9, "Mercury": 8, "Jupiter": 8, "Venus": 9, "Saturn": 11, "Rahu": 7, "Ketu": 1}
	for i := range r.Chart.Planets {
		p := &r.Chart.Planets[i]
		p.RashiNumber = signs[p.Name]
		p.House = (p.RashiNumber-11+12)%12 + 1
	}
	yes := true
	for i := range r.Details.Analysis.Conditions {
		if r.Details.Analysis.Conditions[i].Name == "Mercury" {
			r.Details.Analysis.Conditions[i].Combust = &yes
		}
	}
	r.GurujiFaladesh = suppliedGurujiFaladesh(r.Chart)
	return r
}

// TestDetailedFaladeshRajaConnections makes the missing career/gains and wealth/work connections part of the visible, attributed reading.
func TestDetailedFaladeshRajaConnections(t *testing.T) {
	r := findingRajaScenario()
	career := synthesisTopic(t, r, "career")
	if career.Findings[0].ID != "lord-career-10-in-11" || !strings.Contains(career.Findings[0].Basis.English, "Mars") {
		t.Fatal("career did not interpret its actual lord", career.Findings)
	}
	finance := synthesisTopic(t, r, "finance")
	if finance.Findings[0].ID != "lord-finance-2-in-10" || !strings.Contains(finance.Findings[0].Basis.English, "2, 11") {
		t.Fatal("shared wealth/gains owner was not merged")
	}
	for _, f := range finance.Findings {
		if f.Kind == "secondary-lord" {
			t.Fatal("shared owner repeated as a second finding")
		}
	}
	personality := synthesisTopic(t, r, "personality")
	found := false
	for _, f := range personality.Findings {
		if f.Kind == "occupants" {
			found = strings.Contains(f.Text.English, "self-restraint") && len(f.EvidenceIDs) == 2
		}
	}
	if !found {
		t.Fatal("Moon/Saturn personal reading omitted")
	}
	found = false
	for _, f := range career.Findings {
		if f.Kind == "capacity" && strings.Contains(strings.Join(f.EvidenceIDs, ","), "combustion:Mercury") {
			found = true
		}
	}
	if !found {
		t.Fatal("combust career-house occupant was hidden")
	}
	relationships := synthesisTopic(t, r, "relationships")
	merged := relationships.Findings[0]
	if !strings.Contains(merged.Basis.English, "Mercury") || !strings.Contains(strings.Join(merged.EvidenceIDs, ","), "combustion:Mercury") || !strings.Contains(merged.Qualification.English, "combustion") {
		t.Fatal("same-destination owners lost their distinct condition", merged)
	}
	for _, f := range career.Findings {
		if strings.Contains(f.Basis.Hindi, "उनकी स्वराशि/उच्च स्थिति") {
			t.Fatal("combustion falsely asserted own/exalted status")
		}
	}
}

// TestDetailedFaladeshDependsOnHouses proves two equally supportive charts receive different findings when the principal lord changes destination.
func TestDetailedFaladeshDependsOnHouses(t *testing.T) {
	r := synthesisScenario()
	before := synthesisTopic(t, r, "career")
	for i := range r.Chart.Planets {
		if r.Chart.Planets[i].Name == "Saturn" {
			r.Chart.Planets[i].RashiNumber = 11
			r.Chart.Planets[i].House = 10
		}
	}
	r.GurujiFaladesh = suppliedGurujiFaladesh(r.Chart)
	after := synthesisTopic(t, r, "career")
	if before.Assessment != after.Assessment {
		t.Fatal("fixture should retain its supportive assessment")
	}
	if before.Conclusion == after.Conclusion || before.Findings[0].Text == after.Findings[0].Text {
		t.Fatal("house connection still reduced to a generic assessment paragraph")
	}
}

// TestDetailedFaladeshCoverageAndProvenance checks all destination houses, nonempty bilingual detail and references against retained facts rather than invented citations.
func TestDetailedFaladeshCoverageAndProvenance(t *testing.T) {
	for _, spec := range synthesisSpecs {
		if spec.ID == "general" || spec.ID == "dasha" {
			continue
		}
		for destination := 1; destination <= 12; destination++ {
			r := synthesisScenario()
			lord := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
			for i := range r.Chart.Planets {
				if r.Chart.Planets[i].Name == lord {
					r.Chart.Planets[i].House = destination
					r.Chart.Planets[i].RashiNumber = (r.Chart.Lagna.RashiNumber+destination-2)%12 + 1
				}
			}
			topic := synthesisTopic(t, r, spec.ID)
			if len(topic.Findings) < 2 || len(topic.Findings) > 5 || topic.Findings[0].Text != findingLibrary.LordHouses[spec.ID][destination-1].Text {
				t.Fatalf("missing subject/house coverage: %s/%d", spec.ID, destination)
			}
			for _, f := range topic.Findings {
				if len(f.Text.Hindi) < 40 || len(f.Text.English) < 40 || f.Basis.Hindi == "" || f.Basis.English == "" {
					t.Fatal("incomplete readable finding", f.ID)
				}
				for _, id := range f.EvidenceIDs {
					found := false
					for _, e := range topic.Evidence {
						found = found || e.ID == id
					}
					if !found {
						t.Fatal("untraceable finding", f.ID, id)
					}
				}
			}
		}
	}
}

// TestDetailedFaladeshPeriodIsolation requires valid dates for timing while preserving the same natal interpretation as the report's period changes.
func TestDetailedFaladeshPeriodIsolation(t *testing.T) {
	r := findingRajaScenario()
	before := synthesisTopic(t, r, "career")
	dasha := synthesisTopic(t, r, "dasha")
	if len(dasha.Findings) != 3 {
		t.Fatal("both running lords need an individual explanation", len(dasha.Findings))
	}
	r.Dasha.AsOf = r.Dasha.Periods[0].End
	after := synthesisTopic(t, r, "career")
	for _, f := range after.Findings {
		if f.Scope == "period" {
			t.Fatal("expired period received a timed finding")
		}
	}
	if before.Findings[0].Text != after.Findings[0].Text || before.Conclusion != after.Conclusion {
		t.Fatal("a period date rewrote natal content")
	}
	if len(synthesisTopic(t, r, "dasha").Findings) != 0 {
		t.Fatal("invalid current period received detailed timing")
	}
}

// TestDetailedFaladeshSnapshotCompatibility keeps older retained JSON without findings and round-trips new detail without recalculating it.
func TestDetailedFaladeshSnapshotCompatibility(t *testing.T) {
	var old FaladeshTopic
	if err := json.Unmarshal([]byte(`{"category":"career","conclusion":{"hi":"पुरानी पढ़त","en":"Retained reading"}}`), &old); err != nil || old.Findings != nil {
		t.Fatal("legacy finding backfill", err)
	}
	snapshot := buildFaladeshSynthesis(findingRajaScenario())
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	var restored FaladeshSynthesis
	if err := json.Unmarshal(data, &restored); err != nil || restored.Topics[0].Findings[0].Text != snapshot.Topics[0].Findings[0].Text {
		t.Fatal("finding snapshot lost", err)
	}
	for _, language := range []string{"en", "hi"} {
		section := (pdfLocale{language}).combinedFaladeshSection(snapshot, time.UTC)
		if len(section.Blocks) != 10 {
			t.Fatal("unexpected PDF structure")
		}
		career := snapshot.Topics[3]
		if !strings.Contains(section.Blocks[3].Note, (pdfLocale{language}).gurujiWording(career.Findings[0].Text)) || strings.Contains(section.Blocks[3].Note, (pdfLocale{language}).gurujiWording(career.Findings[0].Basis)) {
			t.Fatal("PDF must retain the interpretation without its technical basis paragraph")
		}
	}
}

// TestFaladeshContinuousProse preserves distinct details and practical guidance once under the category, omitting technical bases and intermediate headings.
func TestFaladeshContinuousProse(t *testing.T) {
	topic := FaladeshTopic{Category: "career", Potential: GurujiWording{"अगला कदम अपनाएँ।", "Apply the next step."}, Findings: []FaladeshFinding{
		{Title: GurujiWording{"अलग शीर्षक", "Separate heading"}, Text: GurujiWording{"पहली बात। अगली बात।", "First point. Further detail."}, Qualification: GurujiWording{"सहायक बात।", "Supporting point."}, Basis: GurujiWording{"तकनीकी आधार", "Technical basis"}},
		{Text: GurujiWording{"पहली बात। दूसरी व्याख्या।", "First point. Another explanation."}, Qualification: GurujiWording{"सहायक बात।", "Supporting point."}},
	}}
	for _, language := range []string{"hi", "en"} {
		l := pdfLocale{language}
		paragraphs := l.faladeshParagraphs(topic)
		if len(paragraphs) != 3 || paragraphs[2] != l.gurujiWording(topic.Potential) {
			t.Fatal("guidance must complete the continuous reading", paragraphs)
		}
		text := strings.Join(paragraphs, "\n\n")
		if strings.Count(text, l.gurujiWording(GurujiWording{"पहली बात।", "First point."})) != 1 || strings.Count(text, l.gurujiWording(topic.Findings[0].Qualification)) != 1 {
			t.Fatal("reading repeats overlapping sentences", text)
		}
		if !strings.Contains(text, l.gurujiWording(GurujiWording{"दूसरी व्याख्या।", "Another explanation."})) || strings.Contains(text, l.gurujiWording(topic.Findings[0].Title)) || strings.Contains(text, l.gurujiWording(topic.Findings[0].Basis)) {
			t.Fatal("distinct detail lost or technical section retained", text)
		}
	}
}

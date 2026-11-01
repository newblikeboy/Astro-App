package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// synthesisScenario supplies explicit chart-condition inputs to isolate interpretation decisions from the independently tested astronomical calculations.
func synthesisScenario() *Report {
	at := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	chart := &swisseph.Kundali{}
	chart.Lagna.RashiNumber = 2
	for _, item := range []struct {
		name string
		sign int
	}{{"Sun", 5}, {"Moon", 4}, {"Mars", 1}, {"Mercury", 6}, {"Jupiter", 2}, {"Venus", 7}, {"Saturn", 10}, {"Rahu", 3}, {"Ketu", 9}} {
		p := swisseph.Graha{Name: item.name, House: (item.sign-chart.Lagna.RashiNumber+12)%12 + 1}
		p.RashiNumber = item.sign
		p.Longitude = float64((item.sign-1)*30 + 10)
		chart.Planets = append(chart.Planets, p)
	}
	r := &Report{Chart: chart, Place: Place{Zone: "Asia/Kolkata"}, Created: at, Details: &KundliExpansion{}}
	r.GurujiFaladesh = suppliedGurujiFaladesh(chart)
	for _, p := range chart.Planets {
		c := swisseph.PlanetCondition{Name: p.Name, SignLord: "Sun", Dignity: "Neutral sign"}
		if p.Name == "Saturn" {
			c.Dignity = "Own sign"
			c.SignLord = "Saturn"
		}
		r.Details.Analysis.Conditions = append(r.Details.Analysis.Conditions, c)
	}
	r.Dasha = &swisseph.Vimshottari{AsOf: at, CurrentLord: "Saturn", CurrentSubLord: "Mercury", Periods: []swisseph.DashaPeriod{{Lord: "Saturn", Start: at.AddDate(-1, 0, 0), End: at.AddDate(2, 0, 0), Antardashas: []swisseph.DashaPeriod{{Lord: "Mercury", Start: at.AddDate(0, -1, 0), End: at.AddDate(0, 1, 0)}, {Lord: "Sun", Start: at.AddDate(0, 3, 0), End: at.AddDate(0, 5, 0)}, {Lord: "Venus", Start: at.AddDate(0, 1, 0), End: at.AddDate(0, 3, 0)}}}}}
	return r
}

// TestFaladeshSignificatorQualification ensures a strained relationship significator is considered even when it does not own the seventh house.
func TestFaladeshSignificatorQualification(t *testing.T) {
	r := synthesisScenario()
	r.Chart.Lagna.RashiNumber = 6
	for i := range r.Chart.Planets {
		r.Chart.Planets[i].House = (r.Chart.Planets[i].RashiNumber-6+12)%12 + 1
	}
	r.GurujiFaladesh = suppliedGurujiFaladesh(r.Chart)
	yes := true
	for i := range r.Details.Analysis.Conditions {
		if r.Details.Analysis.Conditions[i].Name == "Venus" {
			r.Details.Analysis.Conditions[i].Combust = &yes
		}
	}
	topic := synthesisTopic(t, r, "relationships")
	found := false
	for _, e := range topic.Evidence {
		if e.Kind == "combustion" && e.Planet == "Venus" && e.Direction == "attention" {
			found = true
		}
	}
	if !found || topic.Assessment == "supportive" {
		t.Fatal("non-lord significator was ignored")
	}
}

// TestFaladeshDivisionalConflict preserves a D9 qualification even when the D1 career lord occupies its own sign.
func TestFaladeshDivisionalConflict(t *testing.T) {
	r := synthesisScenario()
	r.Details.Vargas = []swisseph.Varga{{Factor: 9, Planets: []swisseph.DivisionalPlacement{{Name: "Saturn", RashiNumber: 1, House: 1}}}}
	topic := synthesisTopic(t, r, "career")
	if topic.Assessment != "mixed" {
		t.Fatal("D9 debility was ignored", topic.Assessment)
	}
	found := false
	for _, e := range topic.Evidence {
		if e.Kind == "varga-9" && e.Direction == "attention" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing divisional provenance")
	}
}

// TestFaladeshIncomingAspectsIncludesDisplacedLord retains an aspect to a house lord outside the topic's houses without inventing a node aspect.
func TestFaladeshIncomingAspectsIncludesDisplacedLord(t *testing.T) {
	r := synthesisScenario()
	for i := range r.Chart.Planets {
		if r.Chart.Planets[i].Name == "Saturn" {
			r.Chart.Planets[i].House = 12
			r.Chart.Planets[i].RashiNumber = 12
		}
	}
	r.Details.Analysis.Aspects = []swisseph.GrahaAspect{{Planet: "Jupiter", House: 12, Occupants: []string{"Saturn"}}}
	topic := synthesisTopic(t, r, "career")
	found := false
	for _, e := range topic.Evidence {
		if e.Kind == "aspect" && e.Planet == "Jupiter" && e.House == 12 {
			found = true
		}
	}
	if !found {
		t.Fatal("incoming aspect to displaced lord omitted")
	}
}

// TestFaladeshIncompleteChartFailsClosed prevents partial or malformed native inputs from becoming a personal forecast.
func TestFaladeshIncompleteChartFailsClosed(t *testing.T) {
	r := synthesisScenario()
	r.Chart.Planets = r.Chart.Planets[:8]
	if buildFaladeshSynthesis(r) != nil {
		t.Fatal("incomplete chart received a forecast")
	}
	r = synthesisScenario()
	r.Chart.Lagna.RashiNumber = 0
	if buildFaladeshSynthesis(r) != nil {
		t.Fatal("invalid ascendant received a forecast")
	}
}

// TestFaladeshSubjectConclusionsRejectGenericSourceText prevents a lord's unrelated natal themes from leaking into Education, Relationships or other subject conclusions.
func TestFaladeshSubjectConclusionsRejectGenericSourceText(t *testing.T) {
	r := synthesisScenario()
	before := buildFaladeshSynthesis(r)
	for i := range r.GurujiFaladesh.Readings {
		r.GurujiFaladesh.Readings[i].Narrative.Meaning = GurujiWording{"असंबंधित दैनिक काम और लेखन वाला मूल अनुच्छेद", "UNRELATED daily work and writing source paragraph"}
	}
	after := buildFaladeshSynthesis(r)

	for i, topic := range after.Topics {
		if topic.Conclusion != before.Topics[i].Conclusion || strings.Contains(topic.Conclusion.English, "UNRELATED") {
			t.Fatal("raw planet-house paragraph leaked into", topic.Category)
		}
		if topic.Category == "education" && (strings.Contains(topic.Conclusion.English, "work") || strings.Contains(topic.Conclusion.Hindi, "कार्यशैली")) {
			t.Fatal("career reading shown in Education")
		}
		if topic.Category == "relationships" && (strings.Contains(topic.Conclusion.English, "writing") || strings.Contains(topic.Conclusion.Hindi, "लेखन")) {
			t.Fatal("writing reading shown in Relationships")
		}
	}
}

// TestFaladeshGeneralSummarisesSubjects preserves personal, work, financial and relationship findings with all of their original causes.
func TestFaladeshGeneralSummarisesSubjects(t *testing.T) {
	snapshot := buildFaladeshSynthesis(synthesisScenario())
	general := snapshot.Topics[0]
	for _, category := range []string{"personality", "career", "finance", "relationships"} {
		found := false
		for _, topic := range snapshot.Topics {
			if topic.Category != category {
				continue
			}
			expected := topic.Findings[0]
			if category == "personality" {
				for _, candidate := range topic.Findings {
					if candidate.Kind == "occupants" && len(candidate.EvidenceIDs) > 1 {
						expected = candidate
						break
					}
				}
			}
			for _, finding := range general.Findings {
				if finding.Text != expected.Text {
					continue
				}
				found = true
				for _, id := range finding.EvidenceIDs {
					matched := false
					for _, fact := range general.Evidence {
						matched = matched || fact.ID == id
					}
					if !matched {
						t.Fatal("General lost a finding cause", id)
					}
				}
			}
		}
		if !found {
			t.Fatal("General omitted grounded subject", category)
		}
	}
}

// synthesisTopic locates a retained category and fails clearly if source coverage changes.
func synthesisTopic(t *testing.T, r *Report, category string) FaladeshTopic {
	t.Helper()
	snapshot := buildFaladeshSynthesis(r)
	if snapshot == nil {
		t.Fatal("missing synthesis")
	}
	for _, topic := range snapshot.Topics {
		if topic.Category == category {
			return topic
		}
	}
	t.Fatal("missing topic", category)
	return FaladeshTopic{}
}

// TestFaladeshOpposingConditions verifies that an own-sign lord remains qualified by combustion instead of being called wholly auspicious.
func TestFaladeshOpposingConditions(t *testing.T) {
	r := synthesisScenario()
	before := synthesisTopic(t, r, "career")
	if before.Assessment != "supportive" {
		t.Fatal(before.Assessment)
	}
	yes := true
	for i := range r.Details.Analysis.Conditions {
		if r.Details.Analysis.Conditions[i].Name == "Saturn" {
			r.Details.Analysis.Conditions[i].Combust = &yes
		}
	}
	after := synthesisTopic(t, r, "career")
	if after.Assessment != "mixed" || len(after.MainPoints) != 2 || after.Conclusion.English == before.Conclusion.English {
		t.Fatal("opposing factors were discarded", after.Assessment)
	}
	for _, point := range before.MainPoints {
		if point.Tone == "attention" || strings.Contains(point.Text.English, "challenge;") {
			t.Fatal("supporting chart got an invented obstacle")
		}
	}
}

// TestFaladeshTeacherRoleIsNotAWeaknessScore ensures copied legacy display labels cannot override the authoritative board lookup and sign calculation.
func TestFaladeshTeacherRoleIsNotAWeaknessScore(t *testing.T) {
	r := synthesisScenario()
	for i := range r.GurujiFaladesh.Readings {
		r.GurujiFaladesh.Readings[i].Role = "marak"
		r.GurujiFaladesh.Readings[i].Dignity = "debilitated"
	}
	topic := synthesisTopic(t, r, "career")
	if topic.Assessment != "supportive" {
		t.Fatal("source category overrode native strength", topic.Assessment)
	}
	for _, e := range topic.Evidence {
		if e.Kind == "teacher-role" && e.Direction != "context" {
			t.Fatal("role assigned outcome direction")
		}
	}
}

// TestFaladeshCancellationsAndAbsentYogas never convert a cancelled Mangal placement or missing yoga into a compulsory remedy.
func TestFaladeshCancellationsAndAbsentYogas(t *testing.T) {
	r := synthesisScenario()
	baseline := synthesisTopic(t, r, "relationships")
	r.Details.Mangal.BaseCount = 1
	r.Details.Mangal.ActiveCount = 0
	topic := synthesisTopic(t, r, "relationships")
	found := false
	for _, e := range topic.Evidence {
		if e.Kind == "mangal" {
			found = true
			if e.Direction != "context" || !strings.Contains(e.Text.English, "cancelled") {
				t.Fatal("cancelled Mangal became an active obstacle")
			}
		}
	}
	if !found {
		t.Fatal("cancellation evidence omitted")
	}
	if topic.Assessment != baseline.Assessment || len(topic.MainPoints) != len(baseline.MainPoints) {
		t.Fatal("cancelled Mangal changed the existing assessment")
	}
}

// TestFaladeshTimingBoundaries checks saved-reference timing, stale period-name rejection and the earliest related future interval even in unsorted inputs.
func TestFaladeshTimingBoundaries(t *testing.T) {
	r := synthesisScenario()
	topic := synthesisTopic(t, r, "career")
	if !strings.Contains(topic.Present.English, "Saturn / Mercury") || topic.NextPeriod == nil || topic.NextPeriod.Lord != "Venus" || !topic.NextPeriod.Start.Equal(r.Created.AddDate(0, 1, 0)) {
		t.Fatal("wrong running/future period")
	}
	r.Dasha.AsOf = r.Dasha.Periods[0].End
	end := synthesisTopic(t, r, "career")
	if end.NextPeriod != nil || strings.Contains(end.Present.English, "At the report reference") {
		t.Fatal("end-exclusive boundary violated")
	}
	r.Dasha.AsOf = r.Created
	r.Dasha.CurrentSubLord = "Moon"
	stale := synthesisTopic(t, r, "career")
	if strings.Contains(stale.Present.English, "At the report reference") {
		t.Fatal("stale current_path/name accepted")
	}
}

// TestFaladeshNextPeriodCrossesMahadasha checks that a related interval in the
// next major period is retained and out-of-parent children are never advertised.
func TestFaladeshNextPeriodCrossesMahadasha(t *testing.T) {
	r := synthesisScenario()
	edge := r.Created.AddDate(0, 2, 0)
	r.Dasha.Periods[0].End = edge
	r.Dasha.Periods[0].Antardashas = []swisseph.DashaPeriod{{Lord: "Mercury", Start: r.Created.AddDate(0, -1, 0), End: edge}}
	r.Dasha.Periods = append(r.Dasha.Periods, swisseph.DashaPeriod{
		Lord: "Venus", Start: edge, End: edge.AddDate(20, 0, 0),
		Antardashas: []swisseph.DashaPeriod{
			{Lord: "Saturn", Start: r.Created.AddDate(0, 1, 0), End: edge}, // Invalid: before its parent.
			{Lord: "Venus", Start: edge, End: edge.AddDate(2, 0, 0)},
		},
	})
	topic := synthesisTopic(t, r, "career")
	if topic.NextPeriod == nil || topic.NextPeriod.Lord != "Venus" || !topic.NextPeriod.Start.Equal(edge) {
		t.Fatalf("lost related interval across Mahadasha boundary: %+v", topic.NextPeriod)
	}
}

// TestFaladeshEvidenceAndSnapshotIsolation checks every main-point provenance link, immutable JSON reloads and unknown/core suppression.
func TestFaladeshEvidenceAndSnapshotIsolation(t *testing.T) {
	r := synthesisScenario()
	snapshot := buildFaladeshSynthesis(r)
	if len(snapshot.Topics) != 9 || len(snapshot.Coverage) < 8 || len(snapshot.Limits) < 3 {
		t.Fatal("incomplete declared coverage")
	}
	for _, topic := range snapshot.Topics {
		ids := map[string]bool{}
		for _, e := range topic.Evidence {
			if ids[e.ID] || e.Text.Hindi == "" || e.Text.English == "" {
				t.Fatal("invalid evidence", e.ID)
			}
			ids[e.ID] = true
		}
		for _, point := range topic.MainPoints {
			for _, id := range point.EvidenceIDs {
				if !ids[id] {
					t.Fatal("invented cause", id)
				}
			}
		}
	}
	r.GurujiFaladesh.Analysis = snapshot
	encoded, _ := json.Marshal(r)
	var restored Report
	if err := json.Unmarshal(encoded, &restored); err != nil {
		t.Fatal(err)
	}
	original := restored.GurujiFaladesh.Analysis.Topics[3].Conclusion.English
	yes := true
	for i := range r.Details.Analysis.Conditions {
		if r.Details.Analysis.Conditions[i].Name == "Saturn" {
			r.Details.Analysis.Conditions[i].Combust = &yes
		}
	}
	changed := buildFaladeshSynthesis(r)
	if changed.Topics[3].Conclusion.English == original {
		t.Fatal("changed chart did not change the new interpretation")
	}
	if restored.GurujiFaladesh.Analysis.Topics[3].Conclusion.English != original {
		t.Fatal("snapshot changed on reload")
	}
	restored.GurujiFaladesh.Analysis = nil
	old, _ := json.Marshal(restored)
	var legacy Report
	_ = json.Unmarshal(old, &legacy)
	if legacy.GurujiFaladesh.Analysis != nil {
		t.Fatal("old snapshot gained a synthesis")
	}
	r.Birth.Unknown = true
	if buildFaladeshSynthesis(r) != nil {
		t.Fatal("unknown-time events invented")
	}
	r.Birth.Unknown = false
	r.Details = nil
	if buildFaladeshSynthesis(r) != nil {
		t.Fatal("core report gained unsupported analysis")
	}
}

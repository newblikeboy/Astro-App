package swisseph

import (
	"reflect"
	"testing"
)

// TestRajyogStrengthUsesMatchedParticipantsOnly isolates an unrelated weak
// lord and an independent Yogakaraka, then adds a second, genuinely weak pair.
func TestRajyogStrengthUsesMatchedParticipantsOnly(t *testing.T) {
	chart := expansionChart()
	bala := Shadbala{}
	for _, name := range matchingLords {
		strong := name != "Mercury" && name != "Venus"
		ratio := 1.2
		if !strong {
			ratio = .5
		}
		bala.Planets = append(bala.Planets, ShadbalaPlanet{Planet: name, Ratio: ratio, MeetsRequirement: strong})
	}
	checks := NatalYogaChecks{Rajyog: RajyogAssessment{
		Matches: true,
		Pairs: []RajyogPair{
			{KendraLord: "Sun", TrikonaLord: "Saturn", Links: []string{"conjunction"}},
			{KendraLord: "Mercury", TrikonaLord: "Jupiter", Links: []string{}},
		},
		Yogakarakas: []YogakarakaLord{{Planet: "Venus"}},
	}}
	qualified, err := CalculateYogaQualifications(chart, PlanetAnalysis{}, bala, checks)
	if err != nil {
		t.Fatal(err)
	}
	global := qualified.Yogas[len(qualified.Yogas)-1]
	if global.Status != "supported" || len(global.Limitations) != 0 || !reflect.DeepEqual(global.Planets, []string{"Sun", "Saturn"}) {
		t.Fatalf("unrelated weak lord changed Rajyog: %+v", global)
	}
	if len(qualified.RajyogPairs) != 1 || qualified.RajyogPairs[0].Qualification.Status != "supported" {
		t.Fatal("non-matching pair qualified or actual pair lost")
	}
	checks.Rajyog.Pairs[1].Links = []string{"mutual_aspect"}
	qualified, err = CalculateYogaQualifications(chart, PlanetAnalysis{}, bala, checks)
	if err != nil {
		t.Fatal(err)
	}
	if len(qualified.RajyogPairs) != 2 || qualified.RajyogPairs[0].Qualification.Status != "supported" || qualified.RajyogPairs[1].Qualification.Status != "qualified" {
		t.Fatal("weak pair contaminated strong pair")
	}
	if !reflect.DeepEqual(qualified.RajyogPairs[1].Qualification.Limitations, []string{"below_shadbala:Mercury"}) {
		t.Fatal("lost actual participant obstacle")
	}
	checks.Rajyog.Matches = false
	checks.Rajyog.Pairs = checks.Rajyog.Pairs[:0]
	qualified, err = CalculateYogaQualifications(chart, PlanetAnalysis{}, bala, checks)
	if err != nil {
		t.Fatal(err)
	}
	global = qualified.Yogas[len(qualified.Yogas)-1]
	if global.BasePresent || global.Status != "absent" || len(global.Planets) != 0 || len(qualified.RajyogPairs) != 0 {
		t.Fatal("independent Yogakaraka created an automatic two-planet Rajyog")
	}
}

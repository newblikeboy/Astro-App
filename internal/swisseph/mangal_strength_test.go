package swisseph

import (
	"encoding/json"
	"testing"
)

// TestMangalStrongBeneficRequiresCompleteCapacity isolates the published
// Jupiter exception and holds raw Mars geometry fixed while varying strength,
// combustion and an unresolved war. Own sign alone must not qualify it.
func TestMangalStrongBeneficRequiresCompleteCapacity(t *testing.T) {
	chart := expansionChart()
	chart.Lagna = zodiacPosition(65)
	signs := map[string]int{"Sun": 1, "Moon": 10, "Mars": 6, "Mercury": 2, "Jupiter": 9, "Venus": 7, "Saturn": 2, "Rahu": 11, "Ketu": 5}
	for i := range chart.Planets {
		chart.Planets[i].ZodiacPosition = zodiacPosition(float64(signs[chart.Planets[i].Name]-1)*30 + 5)
		chart.Planets[i].House = relativeSign(chart.Planets[i].RashiNumber, 3)
	}
	base, e := CalculateMangal(chart)
	if e != nil || base.ActiveCount == 0 {
		t.Fatal("fixture needs uncancelled raw Mars", e)
	}
	before, _ := json.Marshal(base)
	bala := Shadbala{Method: ShadbalaMethod}
	for _, n := range matchingLords {
		bala.Planets = append(bala.Planets, ShadbalaPlanet{Planet: n, Total: 500, Required: 300, Ratio: 500.0 / 300, MeetsRequirement: true})
	}
	analysis := CalculatePlanetAnalysis(chart)
	strong, e := QualifyMangalStrength(chart, base, bala, analysis)
	if e != nil || strong.ActiveCount != 0 || strong.BaseCount != base.BaseCount || strong.Strength == nil || !strong.Strength.Conditions[0].Confirmed {
		t.Fatalf("published strong Jupiter exception missing: %+v %v", strong, e)
	}
	for i := range bala.Planets {
		if bala.Planets[i].Planet == "Jupiter" {
			bala.Planets[i].MeetsRequirement = false
			bala.Planets[i].Ratio = .5
		}
	}
	weak, e := QualifyMangalStrength(chart, base, bala, analysis)
	if e != nil || weak.ActiveCount != base.ActiveCount || weak.Strength.Conditions[0].Confirmed {
		t.Fatal("own sign replaced missing strength", e)
	}
	for i := range bala.Planets {
		bala.Planets[i].MeetsRequirement = true
		bala.Planets[i].Ratio = 1.5
	}
	yes := true
	for i := range analysis.Conditions {
		if analysis.Conditions[i].Name == "Jupiter" {
			analysis.Conditions[i].Combust = &yes
		}
	}
	combust, e := QualifyMangalStrength(chart, base, bala, analysis)
	if e != nil || combust.ActiveCount != base.ActiveCount {
		t.Fatal("combust benefic falsely confirmed", e)
	}
	analysis = CalculatePlanetAnalysis(chart)
	bala.Wars = []PlanetWar{{First: "Jupiter", Second: "Mercury", Status: "unresolved"}}
	unresolved, e := QualifyMangalStrength(chart, base, bala, analysis)
	if e != nil || unresolved.ActiveCount != base.ActiveCount || len(unresolved.ReviewConditions) == 0 {
		t.Fatal("unresolved war guessed", e)
	}
	after, _ := json.Marshal(base)
	if string(before) != string(after) {
		t.Fatal("raw assessment mutated")
	}
	pair, e := MangalPairWithStrength(chart, chart, strong, strong)
	if e != nil || pair.Method != MangalStrengthMethod || pair.Status != "no_active_flags" {
		t.Fatal("pair used unqualified raw counts", e)
	}
}

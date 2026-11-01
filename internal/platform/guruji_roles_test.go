package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestGurujiMercurySignAndHouseExamples checks independently stated user examples without calling the native engine or altering its positions.
func TestGurujiMercurySignAndHouseExamples(t *testing.T) {
	for _, sample := range []struct {
		lagna, sign, house int
		role, primary      string
		own                bool
	}{
		{3, 6, 4, "karak", "exalted", true}, {7, 6, 12, "marak", "exalted", true},
		{3, 3, 1, "karak", "own", true}, {3, 12, 10, "karak", "debilitated", false},
	} {
		chart := &swisseph.Kundali{}
		chart.Lagna.RashiNumber = sample.lagna
		p := swisseph.Graha{Name: "Mercury", House: sample.house}
		p.RashiNumber = sample.sign
		chart.Planets = []swisseph.Graha{p}
		before, _ := json.Marshal(chart)
		snapshot := suppliedGurujiFaladesh(chart)
		if snapshot.HouseSystem != "D1 whole-sign" || len(snapshot.Readings) != 1 {
			t.Fatal("missing separate house-system declaration")
		}
		reading := snapshot.Readings[0]
		if reading.Role != sample.role || reading.SignNumber != sample.sign || reading.House != sample.house || reading.SignStatus.Primary != sample.primary || reading.SignStatus.OwnSign != sample.own {
			t.Fatalf("wrong classification: %+v", reading)
		}
		if reading.SignStatus.Exalted != (sample.primary == "exalted") || reading.SignStatus.Debilitated != (sample.primary == "debilitated") {
			t.Fatal("lost independent flags")
		}
		if sample.lagna == 7 && reading.RoleEffect.Direction == "support" {
			t.Fatal("exaltation made board Marak automatically favourable")
		}
		after, _ := json.Marshal(chart)
		if string(before) != string(after) {
			t.Fatal("classification changed native planetary positions")
		}
	}
}

// TestGurujiStatusNeverUsesHouse checks every lagna and every D1 house: Pisces Mercury is always debilitated and Virgo Mercury always exalted and own-sign.
func TestGurujiStatusNeverUsesHouse(t *testing.T) {
	for lagna := 1; lagna <= 12; lagna++ {
		for _, sign := range []int{6, 12} {
			chart := &swisseph.Kundali{}
			chart.Lagna.RashiNumber = lagna
			p := swisseph.Graha{Name: "Mercury", House: (sign-lagna+12)%12 + 1}
			p.RashiNumber = sign
			chart.Planets = []swisseph.Graha{p}
			reading := suppliedGurujiFaladesh(chart).Readings[0]
			if reading.SignStatus.Exalted != (sign == 6) || reading.SignStatus.Debilitated != (sign == 12) || reading.SignStatus.OwnSign != (sign == 6) {
				t.Fatalf("house contaminated sign condition for lagna %d: %+v", lagna, reading)
			}
		}
	}
	if s := gurujiSignStatus("Mercury", 1); s.Primary != "other" || s.OwnSign || s.Exalted || s.Debilitated {
		t.Fatal("ordinary sign misclassified")
	}
}

// TestGurujiRoleEffectsRespectBoard separates functional role from capacity across the complete role/status matrix, including unlisted nodes.
func TestGurujiRoleEffectsRespectBoard(t *testing.T) {
	for _, sample := range []struct {
		lagna           int
		planet          string
		sign            int
		role, direction string
	}{
		{3, "Mercury", 6, "karak", "support"}, {3, "Mercury", 12, "karak", "attention"}, {3, "Mercury", 3, "karak", "support"},
		{7, "Mercury", 6, "marak", "context"}, {7, "Mercury", 12, "marak", "context"}, {7, "Mercury", 3, "marak", "context"},
		{1, "Saturn", 7, "neutral", "context"}, {1, "Saturn", 1, "neutral", "context"},
		{3, "Jupiter", 4, "karak", "support"}, {12, "Mercury", 6, "karak", "support"}, {1, "Rahu", 2, "unlisted", "context"},
	} {
		role := gurujiRole(sample.lagna, sample.planet)
		effect := gurujiRoleEffect(role, gurujiSignStatus(sample.planet, sample.sign))
		if role != sample.role || effect.Direction != sample.direction || effect.Explanation.Hindi == "" || effect.Explanation.English == "" {
			t.Fatalf("board role/condition changed: %+v => %s %+v", sample, role, effect)
		}
	}
	for _, planet := range []string{"Rahu", "Ketu"} {
		for sign := 1; sign <= 12; sign++ {
			if gurujiSignStatus(planet, sign).OwnSign {
				t.Fatal("invented node ownership")
			}
		}
	}
}

// TestGurujiAllClassicalOwnSigns verifies independent ownership for every classical planet in all twelve signs, including overlapping Mercury exaltation.
func TestGurujiAllClassicalOwnSigns(t *testing.T) {
	for planet, signs := range map[string][]int{"Sun": {5}, "Moon": {4}, "Mars": {1, 8}, "Mercury": {3, 6}, "Jupiter": {9, 12}, "Venus": {2, 7}, "Saturn": {10, 11}} {
		for sign := 1; sign <= 12; sign++ {
			want := false
			for _, own := range signs {
				want = want || sign == own
			}
			if gurujiSignStatus(planet, sign).OwnSign != want {
				t.Fatalf("wrong sign ownership: %s/%d", planet, sign)
			}
		}
	}
}

// TestGurujiClassificationSnapshotIsImmutable verifies independent flags survive JSON while historical snapshots gain no live recalculation.
func TestGurujiClassificationSnapshotIsImmutable(t *testing.T) {
	r := synthesisScenario()
	snapshot := suppliedGurujiFaladesh(r.Chart)
	encoded, _ := json.Marshal(snapshot)
	var restored GurujiFaladesh
	if err := json.Unmarshal(encoded, &restored); err != nil || !reflect.DeepEqual(snapshot, &restored) {
		t.Fatal("classification failed round trip", err)
	}
	var old GurujiFaladesh
	if err := json.Unmarshal([]byte(`{"lagna":3,"readings":[{"planet":"Mercury","house":4,"role":"karak","dignity":"exalted"}]}`), &old); err != nil {
		t.Fatal(err)
	}
	if old.Readings[0].SignStatus != nil || old.Readings[0].RoleEffect != nil || old.HouseSystem != "" {
		t.Fatal("historical snapshot gained computed flags")
	}
	for _, reading := range snapshot.Readings {
		if strings.Contains(reading.RoleEffect.Explanation.English, "Neechabhanga") || strings.Contains(reading.RoleEffect.Explanation.English, "Dasha") {
			t.Fatal("role/sign feature claimed uncalculated cancellation or timing")
		}
	}
}

// TestFaladeshBoardRoleQualifiesDignity guards the actual conclusion path against treating every strong planet as helpful, including divisional evidence.
func TestFaladeshBoardRoleQualifiesDignity(t *testing.T) {
	for _, sample := range []struct {
		lagna, sign     int
		role, direction string
	}{
		{3, 6, "karak", "support"}, {7, 6, "marak", "context"}, {3, 12, "karak", "attention"}, {7, 12, "marak", "context"},
	} {
		r := synthesisScenario()
		r.Chart.Lagna.RashiNumber = sample.lagna
		for i := range r.Chart.Planets {
			if r.Chart.Planets[i].Name == "Mercury" {
				r.Chart.Planets[i].RashiNumber = sample.sign
			}
			r.Chart.Planets[i].House = (r.Chart.Planets[i].RashiNumber-sample.lagna+12)%12 + 1
		}
		r.GurujiFaladesh = suppliedGurujiFaladesh(r.Chart)
		// Education always includes Mercury as a subject significator, regardless of seventh/ninth ownership.
		topic := synthesisTopic(t, r, "education")
		found := false
		for _, e := range topic.Evidence {
			if e.Kind == "dignity" && e.Planet == "Mercury" {
				found = true
				if e.Direction != sample.direction {
					t.Fatalf("role ignored: %+v => %+v", sample, e)
				}
			}
		}
		if !found {
			t.Fatal("missing combined role/sign evidence")
		}
	}
	r := synthesisScenario()
	r.Chart.Lagna.RashiNumber = 7
	for i := range r.Chart.Planets {
		if r.Chart.Planets[i].Name == "Mars" {
			r.Chart.Planets[i].RashiNumber = 10
		}
		r.Chart.Planets[i].House = (r.Chart.Planets[i].RashiNumber-7+12)%12 + 1
	}
	r.GurujiFaladesh = suppliedGurujiFaladesh(r.Chart)
	r.Details.Vargas = []swisseph.Varga{{Factor: 9, Planets: []swisseph.DivisionalPlacement{{Name: "Mars", RashiNumber: 10, House: 1}}}}
	topic := synthesisTopic(t, r, "finance")
	for _, e := range topic.Evidence {
		if (e.Kind == "dignity" || e.Kind == "varga-9") && e.Planet == "Mars" && e.Direction != "context" {
			t.Fatal("exalted Marak became automatic support", e)
		}
	}
}

// TestGurujiD1StatusDoesNotUseChalit keeps a displaced Chalit house from changing the stored D1 house or sign condition.
func TestGurujiD1StatusDoesNotUseChalit(t *testing.T) {
	r := synthesisScenario()
	r.Details.Chalit.Planets = []swisseph.BhavaPlanet{{Name: "Mercury", NatalHouse: 5, House: 6}}
	before, _ := json.Marshal(r.Chart)
	_ = buildFaladeshSynthesis(r)
	reading := r.GurujiFaladesh.Readings[3]
	if reading.Planet != "Mercury" || reading.House != 5 || !reading.SignStatus.Exalted || !reading.SignStatus.OwnSign {
		t.Fatal("Chalit contaminated D1 facts", reading)
	}
	after, _ := json.Marshal(r.Chart)
	if string(before) != string(after) {
		t.Fatal("synthesis mutated native chart")
	}
}

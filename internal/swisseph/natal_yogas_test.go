package swisseph

import (
	"math"
	"reflect"
	"testing"
)

// natalYogaChart builds independent longitude fixtures with deliberately absent sign/house metadata.
func natalYogaChart(lagna float64, longitudes [9]float64) Kundali {
	chart := Kundali{Lagna: ZodiacPosition{Longitude: lagna}}
	for i, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"} {
		chart.Planets = append(chart.Planets, Graha{Name: name, ZodiacPosition: ZodiacPosition{Longitude: longitudes[i]}})
	}
	return chart
}

// TestKaalSarpExactArcsAndBoundaries checks both directions, zodiac wrapping,
// same-sign escape, node contacts and a split chart that remains absent despite a node contact.
func TestKaalSarpExactArcsAndBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name, status, arc string
		longitudes        [9]float64
	}{
		{"forward", "present", "rahu_to_ketu", [9]float64{10, 20, 30, 40, 50, 60, 70, 0, 180}},
		{"reverse", "present", "ketu_to_rahu", [9]float64{190, 200, 210, 220, 230, 240, 250, 0, 180}},
		{"wrap", "present", "rahu_to_ketu", [9]float64{355, 0, 20, 40, 80, 110, 160, 350, 170}},
		{"split", "absent", "none", [9]float64{10, 20, 30, 40, 50, 190, 70, 0, 180}},
		{"same-sign-escape", "absent", "none", [9]float64{9, 40, 70, 100, 130, 160, 189, 10, 190}},
		{"rahu-contact", "boundary", "rahu_to_ketu", [9]float64{0, 20, 30, 40, 50, 60, 70, 0, 180}},
		{"ketu-contact", "boundary", "rahu_to_ketu", [9]float64{10, 180, 30, 40, 50, 60, 70, 0, 180}},
		{"reverse-contact", "boundary", "ketu_to_rahu", [9]float64{180, 200, 210, 220, 230, 240, 250, 0, 180}},
		{"wrap-tolerance", "boundary", "rahu_to_ketu", [9]float64{359.99999995, 20, 30, 40, 50, 60, 70, 0, 180}},
		{"outside-tolerance", "absent", "none", [9]float64{359.999999, 20, 30, 40, 50, 60, 70, 0, 180}},
		{"split-with-contact", "absent", "none", [9]float64{0, 20, 30, 40, 50, 190, 70, 0, 180}},
		{"all-on-axis", "boundary", "node_boundary", [9]float64{0, 180, 0, 180, 0, 180, 0, 0, 180}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, rotation := range []float64{0, 23.456, 300.125} {
				longitudes := tc.longitudes
				for i, longitude := range longitudes {
					longitudes[i] = math.Mod(longitude+rotation, 360)
				}
				chart := natalYogaChart(5, longitudes)
				before := append([]Graha{}, chart.Planets...)
				checks, err := CalculateNatalYogaChecks(chart)
				if err != nil || checks.KaalSarp.Status != tc.status || checks.KaalSarp.Arc != tc.arc {
					t.Fatalf("rotation %v: %+v, %v", rotation, checks.KaalSarp, err)
				}
				if len(checks.KaalSarp.Planets) != 7 || !reflect.DeepEqual(chart.Planets, before) {
					t.Fatal("lost classical evidence or mutated natal chart")
				}
				if tc.status != "present" && checks.KaalSarp.Type != "" {
					t.Fatal("assigned a subtype to an absent or undecided enclosure")
				}
			}
		})
	}
}

// TestKaalSarpTwelveTypes verifies house-based naming in both arcs and across the zodiac wrap.
func TestKaalSarpTwelveTypes(t *testing.T) {
	names := []string{"ananta", "kulik", "vasuki", "shankhpal", "padma", "mahapadma", "takshak", "karkotak", "shankhchud", "ghatak", "vishdhar", "sheshnag"}
	for house, name := range names {
		for _, arcOffset := range []float64{0, 180} {
			rahu := float64(house*30) + 10
			longitudes := [9]float64{}
			for i := 0; i < 7; i++ {
				longitudes[i] = math.Mod(rahu+arcOffset+float64(10+i*20), 360)
			}
			longitudes[7], longitudes[8] = rahu, math.Mod(rahu+180, 360)
			checks, err := CalculateNatalYogaChecks(natalYogaChart(5, longitudes))
			if err != nil {
				t.Fatal(err)
			}
			k := checks.KaalSarp
			if k.Status != "present" || k.Type != name || k.RahuHouse != house+1 || k.KetuHouse != (house+6)%12+1 {
				t.Fatalf("house %d, arc %v: %+v", house+1, arcOffset, k)
			}
		}
	}
}

// TestRajyogIndependentLordLinks verifies an Aries 10th/5th Saturn/Sun pair,
// including a one-way aspect that must not be accepted and an exact sign boundary.
func TestRajyogIndependentLordLinks(t *testing.T) {
	for _, tc := range []struct {
		name        string
		sun, saturn float64
		links       []string
	}{
		{"conjunction", 40, 50, []string{"conjunction"}},
		{"opposition", 40, 220, []string{"mutual_aspect"}},
		{"exchange-capricorn-leo", 280, 130, []string{"sign_exchange"}},
		{"exchange-aquarius-leo", 310, 130, []string{"mutual_aspect", "sign_exchange"}},
		{"one-way-saturn-aspect", 75, 15, []string{}},
		{"unrelated", 10, 40, []string{}},
		{"unrounded-sign-boundary", 29.99999999, 30.00000001, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			checks, err := CalculateNatalYogaChecks(natalYogaChart(5, [9]float64{tc.sun, 115, 15, 72, 250, 180, tc.saturn, 300, 120}))
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, pair := range checks.Rajyog.Pairs {
				if pair.KendraHouse == 10 && pair.TrikonaHouse == 5 {
					found = true
					if pair.KendraLord != "Saturn" || pair.TrikonaLord != "Sun" || !reflect.DeepEqual(pair.Links, tc.links) {
						t.Fatalf("incorrect independent house lords or link: %+v", pair)
					}
				}
			}
			if !found {
				t.Fatal("missing checked pair")
			}
		})
	}
	// Leo: Saturn rules house 7; Mars rules house 9. Mars in Aries aspects
	// Saturn in Cancer by its fourth, and Saturn returns its tenth aspect.
	checks, err := CalculateNatalYogaChecks(natalYogaChart(125, [9]float64{40, 115, 10, 72, 250, 180, 100, 300, 120}))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, pair := range checks.Rajyog.Pairs {
		if pair.KendraHouse == 7 && pair.TrikonaHouse == 9 {
			found = reflect.DeepEqual(pair.Links, []string{"mutual_aspect"})
		}
	}
	if !found {
		t.Fatal("missing mutual special aspects")
	}
}

// TestRajyogAllAscendantsAndNoAutomaticLagnaYoga checks independent traditional
// Yogakaraka ownerships and ensures all 12 ascendants omit self-pairs.
func TestRajyogAllAscendantsAndNoAutomaticLagnaYoga(t *testing.T) {
	want := map[int]YogakarakaLord{
		2: {"Saturn", 10, 9}, 4: {"Mars", 10, 5}, 5: {"Mars", 4, 9},
		7: {"Saturn", 4, 5}, 10: {"Venus", 10, 5}, 11: {"Venus", 4, 9},
	}
	for sign := 1; sign <= 12; sign++ {
		checks, err := CalculateNatalYogaChecks(natalYogaChart(float64((sign-1)*30)+5, [9]float64{10, 45, 80, 115, 150, 185, 220, 300, 120}))
		if err != nil {
			t.Fatal(err)
		}
		if expected, ok := want[sign]; ok {
			if !reflect.DeepEqual(checks.Rajyog.Yogakarakas, []YogakarakaLord{expected}) {
				t.Fatalf("ascendant %d: %+v", sign, checks.Rajyog.Yogakarakas)
			}
		} else if len(checks.Rajyog.Yogakarakas) != 0 {
			t.Fatalf("invented yogakaraka for %d", sign)
		}
		matched := false
		for _, pair := range checks.Rajyog.Pairs {
			if pair.KendraLord == pair.TrikonaLord || pair.KendraHouse == pair.TrikonaHouse {
				t.Fatal("self relationship falsely became Rajyog")
			}
			matched = matched || len(pair.Links) > 0
		}
		if matched != checks.Rajyog.Matches {
			t.Fatal("summary disagrees with evidence")
		}
	}
}

// TestNatalYogaRejectsInvalidCharts rejects incomplete, duplicated, non-finite,
// out-of-range and non-opposite-node inputs rather than reporting a false absence.
func TestNatalYogaRejectsInvalidCharts(t *testing.T) {
	for _, kind := range []string{"missing", "duplicate", "nan", "infinity", "negative", "upper-bound", "ascendant", "nodes"} {
		t.Run(kind, func(t *testing.T) {
			chart := natalYogaChart(5, [9]float64{10, 20, 30, 40, 50, 60, 70, 0, 180})
			switch kind {
			case "missing":
				chart.Planets = chart.Planets[1:]
			case "duplicate":
				chart.Planets = append(chart.Planets, chart.Planets[0])
			case "nan":
				chart.Planets[0].Longitude = math.NaN()
			case "infinity":
				chart.Planets[0].Longitude = math.Inf(1)
			case "negative":
				chart.Planets[0].Longitude = -1
			case "upper-bound":
				chart.Planets[0].Longitude = 360
			case "ascendant":
				chart.Lagna.Longitude = math.NaN()
			case "nodes":
				chart.Planets[8].Longitude = 170
			}
			if _, err := CalculateNatalYogaChecks(chart); err == nil {
				t.Fatal("accepted invalid natal input")
			}
		})
	}
}

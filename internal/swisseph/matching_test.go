package swisseph

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
)

// matchingChartForTest builds a Moon-only chart at a chosen longitude for independent Ashtakoota cases.
func matchingChartForTest(lon float64) Kundali {
	return Kundali{Planets: []Graha{{Name: "Moon", ZodiacPosition: ZodiacPosition{Longitude: lon}}}}
}

// matchingNakLongitudeForTest returns the midpoint of a zero-based Nakshatra, avoiding boundary ambiguity.
func matchingNakLongitudeForTest(nak int) float64 { return (float64(nak) + 0.5) * 360 / 27 }

// TestMatchingTaraInclusiveEvidence checks inclusive Tara counting and the supporting score evidence.
func TestMatchingTaraInclusiveEvidence(t *testing.T) {
	for bride := 0; bride < 27; bride++ {
		for groom := 0; groom < 27; groom++ {
			result, err := CalculateMatching(matchingChartForTest(matchingNakLongitudeForTest(bride)), matchingChartForTest(matchingNakLongitudeForTest(groom)))
			if err != nil {
				t.Fatal(err)
			}
			e := result.Factors[2].Evidence
			for _, role := range []string{"bride", "groom"} {
				count, class := e[role+"_count"], e[role+"_class"]
				if count < 1 || count > 27 || class != (count-1)%9+1 {
					t.Fatal("Tara count/class not explained by inclusive nine-cycle")
				}
			}
			if bride == groom {
				if e["bride_count"] != 1 || e["groom_count"] != 1 || result.Factors[2].BrideValue != "Janma (1)" {
					t.Fatal("same-star Janma trace changed")
				}
			} else if e["bride_count"]+e["groom_count"] != 29 {
				t.Fatal("directional counts do not traverse27stars inclusively")
			}
		}
	}
}

// TestMatchingPublishedProkeralaBaseExample compares base matching scores with a published Prokerala example.
func TestMatchingPublishedProkeralaBaseExample(t *testing.T) {
	// Independent publisher's worked example, PDF p.12:
	// https://apiorigin.prokerala.com/reports/sample/guna-milan-en.pdf
	// Bride: Ashwini pada3, Aries7d43m59s; groom: Magha pada3,
	// Leo6d57m39s. The published eight base marks total20/36.
	bride := matchingChartForTest(7 + 43.0/60 + 59.0/3600)
	groom := matchingChartForTest(120 + 6 + 57.0/60 + 39.0/3600)
	got, err := CalculateMatching(bride, groom)
	if err != nil {
		t.Fatal(err)
	}
	want := []float64{1, 0, 3, 2, 5, 1, 0, 8}
	if got.Total != 20 || got.Max != 36 || got.Method != MatchingMethod || len(got.Factors) != len(want) {
		t.Fatalf("published example disagrees: %+v", got)
	}
	for i, score := range want {
		if got.Factors[i].Score != score {
			t.Errorf("published %s got%g want%g", got.Factors[i].Code, got.Factors[i].Score, score)
		}
	}
	if !slices.Contains(got.Cautions, "bhakoot_adverse") || slices.Contains(got.Cautions, "nadi_same") {
		t.Fatalf("incorrect caution classification: %v", got.Cautions)
	}
}

// TestMatchingHandCalculatedFullAndSameStarFixtures compares complete and same-star matching cases with hand-calculated scores.
func TestMatchingHandCalculatedFullAndSameStarFixtures(t *testing.T) {
	// Rohini bride / Taurus Mrigashira groom: all base factors reach their
	// maxima, and their Nadis differ. The reversed Gana row loses one point.
	for _, tc := range []struct {
		bride, groom float64
		total, gana  float64
	}{
		{45, 55, 36, 6},
		{55, 45, 35, 5},
		{45, 50, 28, 6}, // same star, different padas: no Nadi bonus.
	} {
		got, err := CalculateMatching(matchingChartForTest(tc.bride), matchingChartForTest(tc.groom))
		if err != nil {
			t.Fatal(err)
		}
		if got.Total != tc.total || got.Factors[5].Score != tc.gana {
			t.Errorf("hand fixture bride%g groom%g: total%g gana%g", tc.bride, tc.groom, got.Total, got.Factors[5].Score)
		}
		if (got.Factors[7].Score == 0) != slices.Contains(got.Cautions, "nadi_same") {
			t.Fatal("same-star fixture incorrectly received an exception")
		}
	}
}

// TestMatchingAllNakshatraPairsAndBounds checks every Nakshatra pair and the allowed matching score bounds.
func TestMatchingAllNakshatraPairsAndBounds(t *testing.T) {
	codes := []string{"varna", "vashya", "tara", "yoni", "maitri", "gana", "bhakoot", "nadi"}
	for bride := 0; bride < 27; bride++ {
		for groom := 0; groom < 27; groom++ {
			got, err := CalculateMatching(matchingChartForTest(matchingNakLongitudeForTest(bride)), matchingChartForTest(matchingNakLongitudeForTest(groom)))
			if err != nil {
				t.Fatalf("pair%d/%d: %v", bride+1, groom+1, err)
			}
			if len(got.Factors) != 8 || got.Max != 36 || got.Total < 0 || got.Total > 36 || math.Mod(got.Total*2, 1) != 0 {
				t.Fatalf("invalid bounded total: %+v", got)
			}
			total := 0.0
			for i, f := range got.Factors {
				if f.Code != codes[i] || f.Max != float64(i+1) || !finite(f.Score) || f.Score < 0 || f.Score > f.Max || f.BrideValue == "" || f.GroomValue == "" || math.Mod(f.Score*2, 1) != 0 {
					t.Fatalf("invalid factor: %+v", f)
				}
				total += f.Score
			}
			if got.Total != total {
				t.Fatal("total differs from factor sum")
			}
			if got.Factors[2].Score != 1.5 && got.Factors[2].Score != 3 {
				t.Fatal("a valid reciprocal marriage-Tara pair cannot have both sides adverse")
			}
			if bride == groom && (got.Total != 28 || got.Factors[7].Score != 0 || !slices.Contains(got.Cautions, "nadi_same")) {
				t.Fatal("same-star match must retain its uncancelled base Nadi score")
			}
			if _, err := json.Marshal(got); err != nil {
				t.Fatal(err)
			}
		}
	}
}

// TestMatchingAllSignPairBhakootAndVarna checks Bhakoot and Varna for every sign pair.
func TestMatchingAllSignPairBhakootAndVarna(t *testing.T) {
	// AIFAS printed p.62, Aries row, rotated for each bride sign. These
	// are base scores; same-lord cancellation must not turn a zero into7.
	bhakoot := []float64{7, 0, 7, 7, 0, 0, 7, 0, 0, 7, 7, 0}
	// Published Varna order here is Brahmin/Kshatriya/Vaishya/Shudra.
	varna := [4][4]float64{{1, 0, 0, 0}, {1, 1, 0, 0}, {1, 1, 1, 0}, {1, 1, 1, 1}}
	varnaBySign := []int{1, 2, 3, 0, 1, 2, 3, 0, 1, 2, 3, 0}
	for bride := 0; bride < 12; bride++ {
		for groom := 0; groom < 12; groom++ {
			got, err := CalculateMatching(matchingChartForTest(float64(bride*30)+5), matchingChartForTest(float64(groom*30)+5))
			if err != nil {
				t.Fatal(err)
			}
			if got.Factors[6].Score != bhakoot[(groom-bride+12)%12] || got.Factors[0].Score != varna[varnaBySign[bride]][varnaBySign[groom]] {
				t.Errorf("sign pair bride%d groom%d has wrong base Bhakoot/Varna", bride+1, groom+1)
			}
		}
	}
	// Venus owns Taurus/Libra, but the base6/8 Bhakoot still scores zero.
	got, err := CalculateMatching(matchingChartForTest(35), matchingChartForTest(185))
	if err != nil || got.Factors[6].Score != 0 || !slices.Contains(got.Cautions, "bhakoot_adverse") {
		t.Fatal("same-lord Bhakoot cancellation was incorrectly applied")
	}
}

// TestMatchingVashyaHalfSignBoundaries checks Vashya classification at half-sign boundaries.
func TestMatchingVashyaHalfSignBoundaries(t *testing.T) {
	for _, tc := range []struct {
		lon  float64
		want string
	}{
		{0, "Quadruped"}, {60, "Human"}, {90, "Aquatic"}, {120, "Wild animal"}, {210, "Insect"},
		{240, "Human"}, {math.Nextafter(255, 0), "Human"}, {255, "Quadruped"},
		{math.Nextafter(270, 0), "Quadruped"}, {270, "Quadruped"}, {math.Nextafter(285, 0), "Quadruped"}, {285, "Aquatic"},
		{300, "Human"}, {330, "Aquatic"}, {math.Nextafter(360, 0), "Aquatic"},
	} {
		got, err := CalculateMatching(matchingChartForTest(tc.lon), matchingChartForTest(1))
		if err != nil || got.Factors[1].BrideValue != tc.want {
			t.Errorf("Vashya at%.17g got%+v err%v", tc.lon, got, err)
		}
	}
}

// TestMatchingTraditionalTableAssignments checks the declared traditional matching table assignments.
func TestMatchingTraditionalTableAssignments(t *testing.T) {
	// Explicit Nadi groups from the publisher, independent of a modulo shortcut.
	nadiGroups := [][]int{{1, 6, 7, 12, 13, 18, 19, 24, 25}, {2, 5, 8, 11, 14, 17, 20, 23, 26}, {3, 4, 9, 10, 15, 16, 21, 22, 27}}
	for group, stars := range nadiGroups {
		for _, star := range stars {
			got, err := matchingMoon(matchingChartForTest(matchingNakLongitudeForTest(star - 1)))
			if err != nil || got.nadi != group {
				t.Errorf("star%d has wrong Nadi", star)
			}
		}
	}
	// Seven reciprocal great-enemy Yoni pairs, using named representative
	// Nakshatras rather than reproducing the entire matrix as test code.
	for _, pair := range [][2]int{{1, 13}, {2, 23}, {3, 20}, {4, 21}, {6, 17}, {7, 10}, {12, 14}} {
		for _, order := range [][2]int{pair, {pair[1], pair[0]}} {
			got, err := CalculateMatching(matchingChartForTest(matchingNakLongitudeForTest(order[0]-1)), matchingChartForTest(matchingNakLongitudeForTest(order[1]-1)))
			if err != nil || got.Factors[3].Score != 0 || !slices.Contains(got.Cautions, "yoni_enemy") {
				t.Errorf("great-enemy stars%v not retained: %+v %v", order, got, err)
			}
		}
	}
	// Combined natural planetary relationships: friend+neutral4,
	// neutral+neutral3, friend+enemy1, neutral+enemy0.5, enemy+enemy0.
	for _, tc := range []struct {
		bride, groom float64
		want         float64
	}{
		{125, 65, 4}, // Sun/Mercury.
		{5, 35, 3},   // Mars/Venus.
		{95, 65, 1},  // Moon/Mercury.
		{5, 65, 0.5}, // Mars/Mercury.
		{125, 35, 0}, // Sun/Venus.
	} {
		got, err := CalculateMatching(matchingChartForTest(tc.bride), matchingChartForTest(tc.groom))
		if err != nil || got.Factors[4].Score != tc.want {
			t.Errorf("Maitri fixture %+v got%+v err%v", tc, got, err)
		}
	}
}

// TestMatchingMoonValidationAndUnroundedSource checks Moon input validation and use of unrounded source coordinates.
func TestMatchingMoonValidationAndUnroundedSource(t *testing.T) {
	valid := matchingChartForTest(45)
	for _, invalid := range []Kundali{
		{},
		{Planets: []Graha{{Name: "Sun", ZodiacPosition: ZodiacPosition{Longitude: 45}}}},
		matchingChartForTest(math.NaN()), matchingChartForTest(math.Inf(1)), matchingChartForTest(math.Inf(-1)), matchingChartForTest(-1), matchingChartForTest(360),
		{Planets: []Graha{{Name: "Moon"}, {Name: "Moon"}}},
	} {
		for _, pair := range [][2]Kundali{{invalid, valid}, {valid, invalid}} {
			if got, err := CalculateMatching(pair[0], pair[1]); err == nil || got.Method != "" || got.Total != 0 {
				t.Fatalf("invalid Moon accepted: %+v %v", got, err)
			}
		}
	}
	// Inconsistent cached labels cannot override the physical Moon longitude.
	chart := matchingChartForTest(math.Nextafter(255, 0))
	chart.Planets[0].RashiNumber = 1
	chart.Planets[0].NakshatraNumber = 1
	chart.Planets[0].Pada = 1
	got, err := CalculateMatching(chart, valid)
	if err != nil || got.Factors[1].BrideValue != "Human" || got.Factors[6].BrideValue != "Sagittarius" {
		t.Fatal("rounded/cached category used instead of unrounded Moon longitude")
	}
}

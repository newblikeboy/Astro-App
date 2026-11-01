package swisseph

import (
	"context"
	"encoding/json"
	"math"
	"reflect"
	"testing"
	"time"
)

// TestShadbalaPublishedAspectIntervals checks literal chapter-26 verse cases
// and the independent printed base speculum (Santhanam pp.256-260).
func TestShadbalaPublishedAspectIntervals(t *testing.T) {
	for _, tc := range []struct {
		name        string
		angle, want float64
	}{
		{"Sun", 0, 0}, {"Sun", 30, 0}, {"Sun", 45, 7.5}, {"Sun", 60, 15}, {"Sun", 75, 30}, {"Sun", 90, 45}, {"Sun", 105, 37.5}, {"Sun", 120, 30}, {"Sun", 135, 15}, {"Sun", 150, 0}, {"Sun", 165, 30}, {"Sun", 180, 60}, {"Sun", 240, 30}, {"Sun", 270, 15}, {"Sun", 300, 0},
		{"Mars", 75, 37.5}, {"Mars", 90, 60}, {"Mars", 105, 45}, {"Mars", 180, 60}, {"Mars", 210, 60}, {"Mars", 225, 45},
		{"Jupiter", 90, 45}, {"Jupiter", 105, 52.5}, {"Jupiter", 120, 60}, {"Jupiter", 135, 45}, {"Jupiter", 225, 52.5}, {"Jupiter", 240, 60},
		{"Saturn", 45, 30}, {"Saturn", 60, 60}, {"Saturn", 75, 52.5}, {"Saturn", 255, 45}, {"Saturn", 270, 60}, {"Saturn", 285, 30},
	} {
		t.Run(tc.name+"-"+time.Duration(tc.angle).String(), func(t *testing.T) {
			if got := fullAspectVirupas(tc.name, tc.angle); math.Abs(got-tc.want) > 1e-10 {
				t.Fatalf("aspect got %g want %g", got, tc.want)
			}
			if fullAspectVirupas(tc.name, tc.angle+360) != tc.want {
				t.Fatal("circular aspect differs")
			}
		})
	}
}

// TestShadbalaNativeComponentsAndSunrise checks six-group arithmetic, known
// theoretical ranges, no luminary triple-count and the pre-sunrise ritual day.
func TestShadbalaNativeComponentsAndSunrise(t *testing.T) {
	for _, hour := range []int{0, 5, 12, 18, 23} {
		t.Run(time.Duration(hour).String(), func(t *testing.T) {
			at := time.Date(1995, 8, 15, hour, 0, 0, 0, time.FixedZone("IST", 19800))
			chart, err := CalculateKundali("../../ephe", at, 28.6139, 77.209)
			if err != nil {
				t.Fatal(err)
			}
			before, _ := json.Marshal(chart)
			birth, chalit, err := CalculateBirthDetails(context.Background(), "../../ephe", "Asia/Kolkata", at, chart)
			if err != nil {
				t.Fatal(err)
			}
			bala, err := CalculateShadbala(context.Background(), "../../ephe", "Asia/Kolkata", at, chart, birth, chalit)
			if err != nil {
				t.Fatal(err)
			}
			if len(bala.Planets) != 7 || bala.Method != ShadbalaMethod {
				t.Fatal("missing full strength")
			}
			tribhaga := 0.0
			for i, p := range bala.Planets {
				if p.Planet != matchingLords[i] || len(p.SthanaParts) != 5 || len(p.KalaParts) != 9 || len(p.Aspects) != 6 {
					t.Fatal("missing strength components")
				}
				sthana, kala, drik := 0.0, 0.0, 0.0
				for _, part := range p.SthanaParts {
					sthana += part.Virupas
					if part.Code == "uchcha" && (part.Virupas < 0 || part.Virupas > 60) {
						t.Fatal("uchcha range")
					}
				}
				for _, part := range p.KalaParts {
					kala += part.Virupas
					if part.Code == "tribhaga" {
						tribhaga += part.Virupas
					}
					if i == 0 && part.Code == "ayana" && part.Virupas != p.Cheshta {
						t.Fatal("Sun substitution double counted in Kala")
					}
					if i == 1 && part.Code == "paksha" && part.Virupas != p.Cheshta {
						t.Fatal("Moon substitution double counted in Kala")
					}
				}
				for _, a := range p.Aspects {
					drik += a.Contribution
				}
				if math.Abs(sthana-p.Sthana) > 1e-9 || math.Abs(kala-p.Kala) > 1e-9 || math.Abs(drik-p.Drik) > 1e-9 || math.Abs(p.Total-(p.Sthana+p.Dig+p.Kala+p.Cheshta+p.Naisargika+p.Drik)) > 1e-9 || math.Abs(p.Ratio-p.Total/p.Required) > 1e-9 || p.MeetsRequirement != (p.Total >= p.Required) {
					t.Fatal("strength arithmetic mismatch")
				}
				if p.Dig < 0 || p.Dig > 60 || p.Cheshta < 0 || p.Cheshta > 60 || !finite(p.Total) {
					t.Fatal("non-finite or impossible strength")
				}
			}
			if tribhaga != 120 {
				t.Fatalf("one third-lord plus Jupiter required, got %g", tribhaga)
			}
			after, _ := json.Marshal(chart)
			if string(before) != string(after) {
				t.Fatal("natal positions mutated")
			}
			broken := chart
			broken.Planets = append([]Graha{}, chart.Planets...)
			broken.Planets[8].Name = "Unknown"
			if _, e := CalculateShadbala(context.Background(), "../../ephe", "Asia/Kolkata", at, broken, birth, chalit); e == nil {
				t.Fatal("incomplete chart accepted")
			}
			if _, e := CalculateShadbala(context.Background(), "../../ephe", "Asia/Kolkata", at.Add(time.Second), chart, birth, chalit); e == nil {
				t.Fatal("mismatched instant accepted")
			}
		})
	}
	if !moolatrikona(3, zodiacPosition(165)) || moolatrikona(3, zodiacPosition(170)) || moolatrikona(3, zodiacPosition(math.Nextafter(165, 0))) {
		t.Fatal("Mercury degree boundaries")
	}
}

// TestMatchingExceptionsExhaustivePadaPairs checks all 108 by 108 lunar padas,
// preserving the independently tested base scores and direction-sensitive rules.
func TestMatchingExceptionsExhaustivePadaPairs(t *testing.T) {
	counts := map[string]int{}
	for b := 0; b < 108; b++ {
		for g := 0; g < 108; g++ {
			bride, groom := matchingChartForTest((float64(b)+.5)*360/108), matchingChartForTest((float64(g)+.5)*360/108)
			base, e := CalculateMatching(bride, groom)
			if e != nil {
				t.Fatal(e)
			}
			before, _ := json.Marshal(base)
			out, e := CalculateMatchingExceptions(bride, groom)
			if e != nil {
				t.Fatal(e)
			}
			for _, d := range []MatchingDosha{out.Nadi, out.Bhakoot} {
				met := false
				for _, rule := range d.Rules {
					met = met || rule.Met
					if rule.Met {
						counts[rule.Code]++
					}
					if rule.SourcePage < 72 || rule.SourcePage > 80 {
						t.Fatal("missing source locator")
					}
				}
				if d.Active != (d.BasePresent && !met) || d.Status == "cancelled" && (!d.BasePresent || !met) {
					t.Fatal("cancellation erased base or invented defect")
				}
			}
			if out.Nadi.BasePresent != (base.Factors[7].Score == 0) || out.Bhakoot.BasePresent != (base.Factors[6].Score == 0) {
				t.Fatal("raw dosha disagrees with base score")
			}
			afterBase, _ := CalculateMatching(bride, groom)
			after, _ := json.Marshal(afterBase)
			if string(before) != string(after) {
				t.Fatal("exceptions changed base points")
			}
		}
	}
	if len(counts) != 9 {
		t.Fatalf("not all declared exception branches exercised: %v", counts)
	}
	for _, tc := range []struct {
		b, g float64
		code string
		want bool
	}{{5, 35, "two_twelve_groom_even_bride_odd", true}, {35, 5, "two_twelve_groom_even_bride_odd", false}, {5, 155, "six_eight_groom_sixth_even_bride_odd", true}, {155, 5, "six_eight_groom_sixth_even_bride_odd", false}} {
		out, _ := CalculateMatchingExceptions(matchingChartForTest(tc.b), matchingChartForTest(tc.g))
		for _, r := range out.Bhakoot.Rules {
			if r.Code == tc.code && r.Met != tc.want {
				t.Fatal("directional cancellation reversed", tc)
			}
		}
	}
}

// TestPersonalTransitVedhaPublishedTable checks every favourable table entry,
// actual blocking, the two reciprocal exemptions and independent Tara status.
func TestPersonalTransitVedhaPublishedTable(t *testing.T) {
	natal := expansionChart()
	natal.Planets[1].ZodiacPosition = zodiacPosition(0)
	for name, table := range lunarVedha {
		for house, vedha := range table {
			current := expansionChart()
			for i := range current.Planets {
				current.Planets[i].ZodiacPosition = zodiacPosition(float64((house+1)%12)*30 + 5)
				if current.Planets[i].Name == name {
					current.Planets[i].ZodiacPosition = zodiacPosition(float64(house-1)*30 + 5)
				}
			}
			// Keep every other planet away from the table's blocking house.
			for i := range current.Planets {
				if current.Planets[i].Name != name && current.Planets[i].RashiNumber == vedha {
					current.Planets[i].ZodiacPosition = zodiacPosition(float64((vedha+1)%12)*30 + 5)
				}
			}
			got, e := CalculateTransitAssessment(natal, current)
			if e != nil {
				t.Fatal(e)
			}
			for _, row := range got.Planets {
				if row.Planet == name && (row.Status != "support" || row.VedhaHouse != vedha || row.FromMoon != house) {
					t.Fatalf("published favourable row %s %d: %+v", name, house, row)
				}
			}
			for i := range current.Planets {
				if current.Planets[i].Name == "Rahu" {
					current.Planets[i].ZodiacPosition = zodiacPosition(float64(vedha-1)*30 + 5)
				}
			}
			blocked, _ := CalculateTransitAssessment(natal, current)
			for _, row := range blocked.Planets {
				if row.Planet == name && (row.Status != "obstructed" || !synthesisTestContains(row.BlockedBy, "Rahu")) {
					t.Fatal("actual Vedha not applied")
				}
			}
		}
	}
	for _, pair := range [][2]string{{"Sun", "Saturn"}, {"Saturn", "Sun"}, {"Moon", "Mercury"}, {"Mercury", "Moon"}} {
		current := expansionChart()
		h := 3
		vedha := lunarVedha[pair[0]][h]
		if vedha == 0 {
			h = 2
			vedha = lunarVedha[pair[0]][h]
		}
		for i := range current.Planets {
			sign := h
			if current.Planets[i].Name == pair[1] {
				sign = vedha
			}
			current.Planets[i].ZodiacPosition = zodiacPosition(float64(sign-1)*30 + 2)
		}
		out, _ := CalculateTransitAssessment(natal, current)
		for _, p := range out.Planets {
			if p.Planet == pair[0] && p.Status != "support" {
				t.Fatal("reciprocal exception ignored", pair)
			}
		}
	}
}

// synthesisTestContains checks blocker names without importing platform code.
func synthesisTestContains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// TestPlanetWarTransferConservation uses hand-calculated totals to check a
// two-planet war, the exact one-degree boundary, latitude ties and a cluster.
func TestPlanetWarTransferConservation(t *testing.T) {
	var p [7]Graha
	for i, n := range matchingLords {
		p[i] = Graha{Name: n, ZodiacPosition: zodiacPosition(float64(i) * 40)}
	}
	p[2].Longitude = 359.8
	p[3].Longitude = .2
	lat := [7]float64{0, 0, 2, 1, 0, 0, 0}
	totals := [7]float64{400, 400, 500, 350, 450, 400, 300}
	wars, change := strengthWarTransfers(p, lat, totals)
	if len(wars) != 1 || wars[0].Winner != "Mars" || wars[0].Transfer != 150 || change[2] != 150 || change[3] != -150 {
		t.Fatalf("hand war transfer: %+v %v", wars, change)
	}
	p[2].Longitude = 0
	p[3].Longitude = 1
	if wars, _ := strengthWarTransfers(p, lat, totals); len(wars) != 0 {
		t.Fatal("one degree is not within one degree")
	}
	p[3].Longitude = .2
	lat[2] = lat[3]
	wars, change = strengthWarTransfers(p, lat, totals)
	if len(wars) != 1 || wars[0].Status != "unresolved" || wars[0].Winner != "" || change != [7]float64{} {
		t.Fatal("latitude tie guessed")
	}
	p[4].Longitude = .4
	lat[2], lat[3], lat[4] = 3, 2, 1
	wars, change = strengthWarTransfers(p, lat, totals)
	if len(wars) != 3 || change[2] != 200 || change[3] != -50 || change[4] != -150 {
		t.Fatalf("cluster used updated totals: %v", change)
	}
	sum := 0.0
	for _, v := range change {
		sum += v
	}
	if sum != 0 {
		t.Fatal("war created strength")
	}
}

// TestNeechabhangaIndependentConditions isolates each Phaladeepika rule and
// proves cancellation never edits the original sign status or base yoga inputs.
func TestNeechabhangaIndependentConditions(t *testing.T) {
	for _, tc := range []struct {
		venus, mars, lagna, moon int
		want                     [4]bool
	}{{4, 3, 1, 2, [4]bool{true, false, false, false}}, {3, 4, 1, 2, [4]bool{false, true, false, false}}, {3, 6, 1, 2, [4]bool{false, false, true, false}}, {1, 2, 3, 3, [4]bool{false, false, false, true}}} {
		chart := expansionChart()
		chart.Lagna = zodiacPosition(float64(tc.lagna-1)*30 + 5)
		for i := range chart.Planets {
			sign := chart.Planets[i].RashiNumber
			switch chart.Planets[i].Name {
			case "Sun":
				sign = 7
			case "Venus":
				sign = tc.venus
			case "Mars":
				sign = tc.mars
			case "Moon":
				sign = tc.moon
			case "Rahu":
				sign = 1
			case "Ketu":
				sign = 7
			}
			chart.Planets[i].ZodiacPosition = zodiacPosition(float64(sign-1)*30 + 5)
			chart.Planets[i].House = relativeSign(sign, tc.lagna)
		}
		analysis := CalculatePlanetAnalysis(chart)
		checks, e := CalculateNatalYogaChecks(chart)
		if e != nil {
			t.Fatal(e)
		}
		bala := Shadbala{}
		for _, n := range matchingLords {
			bala.Planets = append(bala.Planets, ShadbalaPlanet{Planet: n, Ratio: 1.1, MeetsRequirement: true})
		}
		before, _ := json.Marshal(chart)
		out, e := CalculateYogaQualifications(chart, analysis, bala, checks)
		if e != nil {
			t.Fatal(e)
		}
		sun := out.Neechabhanga[0]
		if !sun.Debilitated || !sun.Cancelled {
			t.Fatal("missing debility/cancellation")
		}
		got := [4]bool{}
		for i, r := range sun.Rules {
			got[i] = r.Met
		}
		if got != tc.want {
			t.Fatalf("isolated cancellation got %v want %v", got, tc.want)
		}
		after, _ := json.Marshal(chart)
		if string(before) != string(after) {
			t.Fatal("cancellation changed natal dignity")
		}
		for i := range bala.Planets {
			bala.Planets[i].MeetsRequirement = false
			bala.Planets[i].Ratio = .4
		}
		weak, _ := CalculateYogaQualifications(chart, analysis, bala, checks)
		if !reflect.DeepEqual(out.Neechabhanga, weak.Neechabhanga) {
			t.Fatal("strength changed geometric cancellation")
		}
		for _, y := range weak.Yogas {
			if y.BasePresent && y.Status != "qualified" {
				t.Fatal("weak participants ignored")
			}
		}
	}
}

package swisseph

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// mangalFixture creates a deterministic chart for checking the separate Mangal reference frames.
func mangalFixture() Kundali {
	chart := Kundali{Lagna: zodiacPosition(160)}
	for _, p := range []struct {
		name string
		lon  float64
	}{{"Sun", 310}, {"Moon", 340}, {"Mars", 160}, {"Mercury", 315}, {"Jupiter", 222}, {"Venus", 100}, {"Saturn", 305}, {"Rahu", 220}, {"Ketu", 40}} {
		chart.Planets = append(chart.Planets, Graha{Name: p.name, ZodiacPosition: zodiacPosition(p.lon), Speed: 0.2})
	}
	return chart
}

// setMangalFixture changes one fixture graha using the normal zodiac classifier.
func setMangalFixture(chart *Kundali, name string, lon float64) {
	for i := range chart.Planets {
		if chart.Planets[i].Name == name {
			chart.Planets[i].ZodiacPosition = zodiacPosition(lon)
		}
	}
}

// TestMangalTriadRawAndCancellationEvidence checks raw Mangal results and cancellation evidence for the three reference frames.
func TestMangalTriadRawAndCancellationEvidence(t *testing.T) {
	base := mangalFixture()
	got, err := CalculateMangal(base)
	if err != nil || got.BaseCount != 2 || got.ActiveCount != 2 || got.BaseLevel != "high" || got.Level != "high" {
		t.Fatalf("base %+v %v", got, err)
	}
	for i, h := range []int{1, 7, 3} {
		if got.References[i].House != h {
			t.Fatal("reference houses mixed")
		}
	}
	for _, tc := range []struct {
		code, planet string
		lon          float64
	}{
		{"mars_own_or_exalted", "Mars", 10}, {"mars_own_or_exalted", "Mars", 225}, {"mars_own_or_exalted", "Mars", 280},
		{"mars_debilitated", "Mars", 100}, {"mars_combust", "Sun", 177},
		{"mars_conjunct_jupiter", "Jupiter", 170}, {"mars_conjunct_moon", "Moon", 170}, {"mars_conjunct_rahu", "Rahu", 170},
	} {
		chart := mangalFixture()
		setMangalFixture(&chart, tc.planet, tc.lon)
		v, e := CalculateMangal(chart)
		if e != nil {
			t.Fatal(e)
		}
		met := false
		for _, r := range v.Rules {
			if r.Code == tc.code {
				met = r.Met
			}
		}
		if !met || v.ActiveCount != 0 {
			t.Fatalf("%s not applied: %+v", tc.code, v)
		}
		for _, ref := range v.References {
			if ref.Flagged && (!ref.Cancelled || len(ref.Exceptions) == 0) {
				t.Fatal("cancellation hides raw evidence")
			}
		}
	}
	chart := mangalFixture()
	chart.Planets[2].Speed = -0.001
	if v, _ := CalculateMangal(chart); v.ActiveCount != 0 || !v.Rules[2].Met {
		t.Fatal("retrograde cancellation not derived from speed")
	}
	chart = mangalFixture()
	setMangalFixture(&chart, "Sun", math.Nextafter(177, 360))
	if v, _ := CalculateMangal(chart); v.Rules[3].Met || v.ActiveCount != 2 {
		t.Fatal("combustion boundary was rounded")
	}
	chart = mangalFixture()
	chart.Lagna = zodiacPosition(340)
	setMangalFixture(&chart, "Jupiter", 40)
	if v, _ := CalculateMangal(chart); !v.Rules[7].Met || v.ActiveCount != 0 {
		t.Fatal("Jupiter fifth aspect on seventh-house Mars not applied")
	}
	chart = mangalFixture()
	setMangalFixture(&chart, "Jupiter", 340)
	v, _ := CalculateMangal(chart)
	if len(v.ReviewConditions) == 0 || v.ActiveCount == 0 {
		t.Fatal("uncomputed benefic strength was treated as established")
	}
}

// TestMangalHouseExceptionsAndCoverage checks house-specific Mangal exceptions and coverage.
func TestMangalHouseExceptionsAndCoverage(t *testing.T) {
	expected := map[int][]int{1: {1, 10}, 4: {1, 4, 8}, 7: {1, 2, 4, 10}, 8: {11, 12}, 12: {1, 2, 3, 6, 7, 8, 9}}
	for house := 1; house <= 12; house++ {
		for sign := 1; sign <= 12; sign++ {
			want := false
			for _, v := range expected[house] {
				want = want || sign == v
			}
			if mangalHouseException(sign, house) != want {
				t.Fatalf("sign/house %d/%d", sign, house)
			}
		}
	}
	// Every possible three-frame house combination, including the regional2nd
	// house, has a bounded and fully accounted raw/active count.
	for lagna := 1; lagna <= 12; lagna++ {
		for moon := 1; moon <= 12; moon++ {
			for venus := 1; venus <= 12; venus++ {
				c := mangalFixture()
				c.Lagna = zodiacPosition(float64((lagna-1)*30 + 10))
				setMangalFixture(&c, "Moon", float64((moon-1)*30+10))
				setMangalFixture(&c, "Venus", float64((venus-1)*30+10))
				v, err := CalculateMangal(c)
				if err != nil {
					t.Fatal(err)
				}
				raw, active := 0, 0
				for _, ref := range v.References {
					if ref.Flagged {
						raw++
					}
					if ref.Flagged && !ref.Cancelled {
						active++
					}
					if ref.Cancelled && !ref.Flagged {
						t.Fatal("unflagged house canceled")
					}
				}
				if raw != v.BaseCount || active != v.ActiveCount || v.BaseLevel != mangalLevel(raw) || v.Level != mangalLevel(active) {
					t.Fatal("coverage not conserved")
				}
			}
		}
	}
}

// TestMangalPairAndInvalidCharts checks paired Mangal assessment and rejection of invalid charts.
func TestMangalPairAndInvalidCharts(t *testing.T) {
	bride, groom := mangalFixture(), mangalFixture()
	if p, e := CalculateMangalPair(bride, groom); e != nil || p.Status != "similar_active_levels" || p.Method != MangalMethod {
		t.Fatalf("pair %+v %v", p, e)
	}
	setMangalFixture(&groom, "Mars", 280)
	p, e := CalculateMangalPair(bride, groom)
	if e != nil || p.Groom.Level != "none" || p.Bride.Level != "high" {
		t.Fatal("raw cancellation and pair balance mixed")
	}
	for _, mutate := range []func(*Kundali){
		func(c *Kundali) { c.Planets = c.Planets[1:] },
		func(c *Kundali) { c.Planets = append(c.Planets, c.Planets[2]) },
		func(c *Kundali) { c.Planets[2].Longitude = math.NaN() },
		func(c *Kundali) { c.Planets[2].Speed = math.Inf(1) },
		func(c *Kundali) { c.Lagna.Longitude = 360 },
	} {
		c := mangalFixture()
		mutate(&c)
		if _, e := CalculateMangal(c); e == nil {
			t.Fatal("incomplete assessment returned")
		}
	}
}

// TestMatchingClockAgainstOfficialSwissUTC compares matching birth-time conversion with official Swiss UTC reference calculations.
func TestMatchingClockAgainstOfficialSwissUTC(t *testing.T) {
	var fixture struct {
		Tolerance float64 `json:"tolerance_degrees"`
		Cases     []struct {
			At      string  `json:"at"`
			Lat     float64 `json:"latitude"`
			Lon     float64 `json:"longitude"`
			Lagna   float64 `json:"lagna"`
			Planets []struct {
				Name      string  `json:"name"`
				Longitude float64 `json:"longitude"`
				Speed     float64 `json:"speed"`
			} `json:"planets"`
		} `json:"cases"`
	}
	data, err := os.ReadFile("testdata/matching-clock-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	for _, tc := range fixture.Cases {
		at, _ := time.Parse(time.RFC3339, tc.At)
		chart, e := CalculateKundali("../../ephe", at, tc.Lat, tc.Lon)
		if e != nil {
			t.Fatal(e)
		}
		if math.Abs(chart.Lagna.Longitude-tc.Lagna) > fixture.Tolerance {
			t.Fatal("Lagna differs from official UTC reference")
		}
		for i, want := range tc.Planets {
			got := chart.Planets[i]
			if got.Name != want.Name || math.Abs(got.Longitude-want.Longitude) > fixture.Tolerance || math.Abs(got.Speed-want.Speed) > fixture.Tolerance {
				t.Fatalf("%s %s mismatch", tc.At, want.Name)
			}
		}
	}
}

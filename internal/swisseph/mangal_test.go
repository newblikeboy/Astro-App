package swisseph

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGurujiMangalAudioProvenance pins the runtime configuration to both unmodified user-supplied recordings.
func TestGurujiMangalAudioProvenance(t *testing.T) {
	var source struct {
		Sources []struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(gurujiMangalJSON, &source); err != nil || len(source.Sources) != 2 {
		t.Fatal("two audio sources required", err)
	}
	for _, item := range source.Sources {
		data, err := os.ReadFile(filepath.Join("../../references/guruji/2026-11-24/mangal-audio", item.File))
		if err != nil || fmt.Sprintf("%x", sha256.Sum256(data)) != item.SHA256 {
			t.Fatal("changed source recording", item.File, err)
		}
	}
}

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

// TestGurujiMangalAllSignsAndReferences keeps every recorded base placement intact after the later text's separate parihar stage.
func TestGurujiMangalAllSignsAndReferences(t *testing.T) {
	flagged := map[int]bool{1: true, 4: true, 7: true, 8: true, 12: true}
	for sign := 1; sign <= 12; sign++ {
		for lagnaHouse := 1; lagnaHouse <= 12; lagnaHouse++ {
			for moonHouse := 1; moonHouse <= 12; moonHouse++ {
				c := mangalFixture()
				c.Lagna = zodiacPosition(float64((sign-lagnaHouse+12)%12)*30 + 10)
				setMangalFixture(&c, "Moon", float64((sign-moonHouse+12)%12)*30+10)
				setMangalFixture(&c, "Mars", float64(sign-1)*30+10)
				setMangalFixture(&c, "Sun", float64(sign-1)*30+11)
				setMangalFixture(&c, "Jupiter", float64(sign-1)*30+12)
				setMangalFixture(&c, "Rahu", float64(sign-1)*30+13)
				c.Planets[2].Speed = -.2
				got, err := CalculateMangal(c)
				if err != nil {
					t.Fatal(err)
				}
				class, level := "none", "none"
				if flagged[lagnaHouse] {
					class, level = "mangal", "high"
				} else if flagged[moonHouse] {
					class, level = "partial", "low"
				}
				count := 0
				for _, h := range []int{lagnaHouse, moonHouse} {
					if flagged[h] {
						count++
					}
				}
				if got.BaseClassification != class || got.BaseLevel != level || got.BaseCount != count || got.ActiveCount > count || len(got.References) != 2 || got.Strength != nil {
					t.Fatalf("%d/%d/%d: %+v", sign, lagnaHouse, moonHouse, got)
				}
				for i, h := range []int{lagnaHouse, moonHouse} {
					ref := got.References[i]
					if ref.House != h || ref.Flagged != flagged[h] || ref.Cancelled && !ref.Flagged {
						t.Fatalf("incorrect reference: %+v", ref)
					}
				}
				// Venus cannot create a base indication; rule 24 may cancel an existing one.
				setMangalFixture(&c, "Venus", float64(sign-1)*30+14)
				changed, _ := CalculateMangal(c)
				if changed.BaseClassification != got.BaseClassification || changed.BaseCount != got.BaseCount {
					t.Fatal("Venus changed Guruji base assessment")
				}
			}
		}
	}
}

// TestGurujiMangalBoundariesAndPair keeps sign boundaries and recorded partial/full precedence distinct from text partner balance.
func TestGurujiMangalBoundariesAndPair(t *testing.T) {
	c := mangalFixture()
	c.Lagna = zodiacPosition(10)
	setMangalFixture(&c, "Moon", 10)
	setMangalFixture(&c, "Mars", math.Nextafter(30, 0))
	if got, e := CalculateMangal(c); e != nil || got.BaseClassification != "mangal" {
		t.Fatal("end of first house", got, e)
	}
	setMangalFixture(&c, "Mars", 30)
	if got, e := CalculateMangal(c); e != nil || got.Classification != "none" {
		t.Fatal("second house must not be flagged", got, e)
	}
	// Mars in Virgo: Lagna Cancer gives house 3; Moon Pisces gives house 7.
	partial := mangalFixture()
	partial.Lagna = zodiacPosition(100)
	full := mangalFixture()
	pair, e := CalculateMangalPair(full, partial)
	if e != nil || pair.Method != MangalMethod || pair.Bride.Classification != "mangal" || pair.Groom.Classification != "partial" || pair.Status != "different_active_levels" || len(pair.Rules) != 5 {
		t.Fatal("full and partial must differ", pair, e)
	}
	same, e := CalculateMangalPair(partial, partial)
	if e != nil || same.Status != "pair_exception_found" || !same.Rules[0].Met || same.Bride.Classification != "partial" || same.Groom.Classification != "partial" {
		t.Fatal(same, e)
	}
	none := partial
	none.Planets = append([]Graha{}, partial.Planets...)
	setMangalFixture(&none, "Moon", 100)
	pair, e = CalculateMangalPair(none, none)
	if e != nil || pair.Status != "no_active_flags" {
		t.Fatal(pair, e)
	}
	for _, mutate := range []func(*Kundali){
		func(c *Kundali) { c.Planets = append(c.Planets[:1], c.Planets[2:]...) },
		func(c *Kundali) { c.Planets = append(c.Planets, c.Planets[2]) },
		func(c *Kundali) { c.Planets[2].Longitude = math.NaN() },
		func(c *Kundali) { c.Planets[2].Speed = math.Inf(1) },
		func(c *Kundali) { c.Lagna.Longitude = 360 },
	} {
		invalid := mangalFixture()
		mutate(&invalid)
		if _, e := CalculateMangal(invalid); e == nil {
			t.Fatal("invalid chart accepted")
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

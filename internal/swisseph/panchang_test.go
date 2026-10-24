package swisseph

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"sync"
	"testing"
	"time"
)

// TestPanchangCLIReferences compares Panchang astronomy with independent Swiss command-line fixtures.
func TestPanchangCLIReferences(t *testing.T) {
	var refs struct {
		Cases []struct {
			Date, Convention    string
			Latitude, Longitude float64
			Sunrise, Sunset     time.Time
			NextSunrise         time.Time `json:"next_sunrise"`
			Limbs               [][]struct {
				Number int
				End    time.Time
			}
		}
	}
	b, err := os.ReadFile("testdata/panchang-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &refs); err != nil {
		t.Fatal(err)
	}
	for _, c := range refs.Cases {
		t.Run(c.Date+"/"+c.Convention, func(t *testing.T) {
			p, err := CalculatePanchang(context.Background(), "../../ephe", c.Date, "Asia/Kolkata", c.Convention, c.Latitude, c.Longitude)
			if err != nil {
				t.Fatal(err)
			}
			// closeTime compares a native event against its official CLI reference within the stated time tolerance.
			closeTime := func(name string, got, want time.Time) {
				t.Helper()
				if math.Abs(got.Sub(want).Seconds()) > 1 {
					t.Errorf("%s got %s want %s (CLI UT1 tolerance 1 second)", name, got, want)
				}
			}
			closeTime("sunrise", p.Sunrise, c.Sunrise)
			closeTime("sunset", p.Sunset, c.Sunset)
			closeTime("next sunrise", p.NextSunrise, c.NextSunrise)
			for kind, list := range [][]PanchangPeriod{p.Tithi, p.Nakshatra, p.Yoga, p.Karana} {
				if len(list) != len(c.Limbs[kind]) {
					t.Fatalf("limb %d count got %d want %d", kind, len(list), len(c.Limbs[kind]))
				}
				for i, v := range list {
					if v.Number != c.Limbs[kind][i].Number {
						t.Errorf("kind %d classification got %d want %d", kind, v.Number, c.Limbs[kind][i].Number)
					}
					closeTime("limb end", v.End, c.Limbs[kind][i].End)
					if i == 0 && !v.Start.Equal(p.Sunrise) || i > 0 && !v.Start.Equal(list[i-1].End) || !v.End.After(v.Start) {
						t.Fatal("gap or inverted limb")
					}
					// Independently classify +/- one second around every solved boundary.
					for _, delta := range []time.Duration{-time.Second, time.Second} {
						chart, err := CalculateKundali("../../ephe", v.End.Add(delta), c.Latitude, c.Longitude)
						if err != nil {
							t.Fatal(err)
						}
						sun, moon := chart.Planets[0].Longitude, chart.Planets[1].Longitude
						angle, width, count := normalizeDegrees(moon-sun), 12.0, 30
						switch kind {
						case 1:
							angle, width, count = moon, 360.0/27, 27
						case 2:
							angle, width, count = normalizeDegrees(moon+sun), 360.0/27, 27
						case 3:
							width, count = 6, 60
						}
						want := v.Number
						if delta > 0 {
							want = want%count + 1
						}
						if got := int(angle/width) + 1; got != want {
							t.Errorf("boundary classification kind %d delta %s: got %d want %d", kind, delta, got, want)
						}
					}
				}
			}
		})
	}
}

// TestDailyWeekdayRules checks weekday-dependent daily timing rules.
func TestDailyWeekdayRules(t *testing.T) {
	// Independently tabulated weekday results. Synthetic 06:00-18:00 daylight
	// makes each eighth exactly 90 minutes, each Hora exactly 60 minutes.
	rahu := [7]int{8, 2, 7, 5, 6, 4, 3}
	yama := [7]int{5, 4, 3, 2, 1, 7, 6}
	gulika := [7]int{7, 6, 5, 4, 3, 2, 1}
	firstDay := [7]string{"Udveg", "Amrit", "Rog", "Labh", "Shubh", "Char", "Kaal"}
	firstNight := [7]string{"Shubh", "Char", "Kaal", "Udveg", "Amrit", "Rog", "Labh"}
	lords := [7]string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"}
	kaalSlots := [7]int{5, 2, 6, 3, 7, 4, 1}
	for w := 0; w < 7; w++ {
		rise := time.Date(2026, 10, 18+w, 6, 0, 0, 0, time.UTC)
		p := Panchang{Weekday: w, Sunrise: rise, Sunset: rise.Add(12 * time.Hour), NextSunrise: rise.Add(24 * time.Hour)}
		populateDailyPeriods(&p)
		for i, n := range []int{rahu[w], yama[w], gulika[w]} {
			if !p.Timings[i].Start.Equal(rise.Add(time.Duration(n-1)*90*time.Minute)) || p.Timings[i].End.Sub(p.Timings[i].Start) != 90*time.Minute {
				t.Fatalf("weekday %d eighth %d wrong", w, i)
			}
		}
		if p.ChoghadiyaDay[0].Name != firstDay[w] || p.ChoghadiyaNight[0].Name != firstNight[w] || p.HoraDay[0].Name != lords[w] {
			t.Fatalf("weekday %d cycle incorrect", w)
		}
		if p.ChoghadiyaDay[kaalSlots[w]-1].Name != "Kaal" {
			t.Fatal("Kaal Vela slot incorrect")
		}
		if (p.Timings[3].Quality == "Caution") != (w == 3) {
			t.Fatal("Wednesday exception missing")
		}
		if p.Timings[3].Start.Sub(rise) != 336*time.Minute || p.Timings[3].End.Sub(rise) != 384*time.Minute {
			t.Fatal("Abhijit not middle fifteenth")
		}
		for _, list := range [][]PanchangPeriod{p.ChoghadiyaDay, p.ChoghadiyaNight, p.HoraDay, p.HoraNight} {
			for i, v := range list {
				if i > 0 && !v.Start.Equal(list[i-1].End) {
					t.Fatal("period gap")
				}
			}
		}
		if !p.HoraNight[11].End.Equal(p.NextSunrise) || !p.ChoghadiyaNight[7].End.Equal(p.NextSunrise) {
			t.Fatal("night not bounded by next sunrise")
		}
		if w == 6 {
			for _, i := range []int{0, 7} {
				found := false
				for _, s := range p.ChoghadiyaDay[i].Cautions {
					found = found || s == "Kaal Vela"
				}
				if !found {
					t.Fatal("repeated Saturday Kaal Vela missing")
				}
			}
		}
	}
}

// TestPanchangValidationAndConcurrency checks invalid Panchang inputs and concurrent access to shared native state.
func TestPanchangValidationAndConcurrency(t *testing.T) {
	for _, date := range []string{"", "2026-2-01", "2026-02-30", "1939-12-31", "2099-12-31"} {
		if _, err := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", "upper-limb", 28, 77); err == nil {
			t.Fatalf("accepted invalid date %q", date)
		}
	}
	if _, err := CalculatePanchang(context.Background(), t.TempDir(), "2026-10-18", "Asia/Kolkata", "upper-limb", 28, 77); err == nil {
		t.Fatal("missing data allowed fallback")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CalculatePanchang(ctx, "../../ephe", "2026-10-18", "Asia/Kolkata", "upper-limb", 28, 77); err == nil {
		t.Fatal("ignored cancellation")
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				p, err := CalculatePanchang(context.Background(), "../../ephe", "2026-10-18", "Asia/Kolkata", "hindu", 28.6139, 77.209)
				if err != nil || p.Tithi[0].Name != "Shukla Saptami" {
					t.Errorf("Panchang state leak: %v", err)
				}
			} else {
				p, err := CalculateKundali("../../ephe", time.Date(1995, 8, 15, 5, 0, 0, 0, time.UTC), 28.6139, 77.209)
				if err != nil || p.Planets[1].Nakshatra != "Revati" {
					t.Errorf("natal state leak: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()
}

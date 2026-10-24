package swisseph

import (
	"context"
	"math"
	"testing"
	"time"
)

// TestPanchangExtendedAstronomy checks supplementary Panchang astronomy and its interval invariants.
func TestPanchangExtendedAstronomy(t *testing.T) {
	for _, tc := range []struct {
		date, amanta, purnimanta string
		adhika                   bool
		vikram                   int
	}{
		{"2026-10-18", "Ashwina", "Ashwina", false, 2083},
		{"2026-03-19", "Phalguna", "Chaitra", false, 2082},
		{"2023-07-20", "Shravana", "Shravana", true, 2080},
		{"2023-08-07", "Shravana", "Shravana", true, 2080},
		{"2023-08-20", "Shravana", "Shravana", false, 2080},
		{"2026-01-01", "Pausha", "Pausha", false, 2082},
	} {
		t.Run(tc.date, func(t *testing.T) {
			p, err := CalculatePanchang(context.Background(), "../../ephe", tc.date, "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
			if err != nil {
				t.Fatal(err)
			}
			d := p.Details
			if d == nil {
				t.Fatal("missing details")
			}
			c := d.Calendar
			t.Logf("%s/%s adhika=%v Vikram=%d", c.Amanta, c.Purnimanta, c.Adhika, c.Vikram)
			if c.Amanta != tc.amanta || c.Purnimanta != tc.purnimanta || c.Adhika != tc.adhika || c.Vikram != tc.vikram {
				t.Errorf("calendar %+v", c)
			}
			if !c.LunationStart.Before(p.Sunrise) || !c.LunationEnd.After(p.Sunrise) || c.LunationEnd.Sub(c.LunationStart) < 28*24*time.Hour || c.LunationEnd.Sub(c.LunationStart) > 31*24*time.Hour {
				t.Fatal("invalid lunation")
			}
			for _, instant := range []time.Time{c.LunationStart, c.LunationEnd} {
				chart, e := CalculateKundali("../../ephe", instant, 28.6139, 77.209)
				if e != nil {
					t.Fatal(e)
				}
				distance := normalizeDegrees(chart.Planets[1].Longitude - chart.Planets[0].Longitude)
				if math.Min(distance, 360-distance) > 0.00003 {
					t.Errorf("new moon error %.8f", distance)
				}
			}
			for _, rows := range [][]PanchangPeriod{d.Lagna, d.Panchaka} {
				if len(rows) == 0 || !rows[0].Start.Equal(p.Sunrise) || !rows[len(rows)-1].End.Equal(p.NextSunrise) {
					t.Fatal("incomplete interval coverage")
				}
				for i, r := range rows {
					if !r.End.After(r.Start) || i > 0 && !r.Start.Equal(rows[i-1].End) {
						t.Fatal("gap/overlap")
					}
				}
			}
			for _, r := range d.Lagna {
				chart, e := CalculateKundali("../../ephe", r.Start.Add(time.Second), 28.6139, 77.209)
				if e != nil {
					t.Fatal(e)
				}
				if chart.Lagna.RashiNumber != r.Number {
					t.Errorf("lagna %d want %d", chart.Lagna.RashiNumber, r.Number)
				}
			}
			if math.Abs(float64(d.DaySeconds+d.NightSeconds)-p.NextSunrise.Sub(p.Sunrise).Seconds()) > 1 {
				t.Fatal("duration mismatch")
			}
		})
	}
}

// TestPanchangExtendedDelhiWindows checks supplementary daily timing windows for Delhi fixtures.
func TestPanchangExtendedDelhiWindows(t *testing.T) {
	p, err := CalculatePanchang(context.Background(), "../../ephe", "2026-10-18", "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Moonrise %s Moonset %s", p.Details.Moonrise, p.Details.Moonset)
	// Official swetest -p1 -rise, same coordinates/atmosphere; UTC/UT1 <1 sec.
	for i, value := range []*time.Time{p.Details.Moonrise, p.Details.Moonset} {
		want, _ := time.Parse(time.RFC3339, []string{"2026-10-18T13:05:59+05:30", "2026-10-18T23:31:41+05:30"}[i])
		if value == nil || math.Abs(value.Sub(want).Seconds()) > 1 {
			t.Fatalf("Moon event differs from CLI: %v", value)
		}
	}
	// Public same-date/location reference observations, minute-level tolerance.
	windows := map[string][2]string{"Brahma Muhurta": {"04:43", "05:33"}, "Vijaya Muhurta": {"14:00", "14:46"}, "Sayahna Sandhya": {"17:49", "19:04"}, "Nishita Muhurta": {"23:41", "00:32"}, "Dur Muhurtam": {"16:17", "17:03"}, "Varjyam": {"21:45", "23:32"}, "Amrit Kalam": {"07:24", "09:12"}}
	for name, want := range windows {
		found := false
		for _, v := range p.Details.Muhurats {
			if v.Name != name {
				continue
			}
			found = true
			for i, at := range []time.Time{v.Start, v.End} {
				target, _ := time.Parse("15:04", want[i])
				minutes := at.Hour()*60 + at.Minute()
				expected := target.Hour()*60 + target.Minute()
				if math.Abs(float64(minutes-expected)) > 1 {
					t.Errorf("%s got %s want around %s", name, at.Format("15:04:05"), want[i])
				}
			}
		}
		if !found {
			t.Errorf("missing %s", name)
		}
	}
	for _, r := range p.Details.Balam {
		if r.MoonSign == 9 {
			want := []int{3, 4, 7, 9, 11, 12}
			for i, v := range want {
				if r.Chandra[i] != v {
					t.Fatal("Chandrabalam mismatch")
				}
			}
		}
	}
}

// TestMonotoneCrossConjunctionWrap checks angular boundary search across a conjunction at zero degrees.
func TestMonotoneCrossConjunctionWrap(t *testing.T) {
	// Starting immediately before conjunction must still find the preceding one,
	// even though a coarse bracket can cross both the target and 360-degree wrap.
	fn := func(jd float64) (float64, error) { return normalizeDegrees(359.8 + 12*jd), nil }
	got, err := monotoneCross(context.Background(), 0, 0, -1, 35, fn)
	if err != nil || math.Abs(got+359.8/12) > 0.1/86400 {
		t.Fatalf("got %.9f %v", got, err)
	}
}

// TestSarvarthaOctoberReferences checks Sarvartha windows against the documented October reference cases.
func TestSarvarthaOctoberReferences(t *testing.T) {
	// Public Drik New Delhi October 2026 list, observed 19 October 2026.
	want := []int{4, 5, 6, 14, 18, 19, 25, 27, 28}
	got := []int{}
	for day := 1; day <= 31; day++ {
		date := time.Date(2026, 10, day, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		p, e := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
		if e != nil {
			t.Fatal(e)
		}
		for _, r := range p.Details.Special {
			if r.Name == "Sarvartha Siddhi Yoga" {
				got = append(got, day)
				break
			}
		}
	}
	if len(got) != len(want) {
		t.Fatalf("Sarvartha days got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("Sarvartha days got %v want %v", got, want)
		}
	}
}

// TestKshayaCombination checks the declared combination rule for a Kshaya month.
func TestKshayaCombination(t *testing.T) {
	p, e := CalculatePanchang(context.Background(), "../../ephe", "1983-01-20", "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
	if e != nil {
		t.Fatal(e)
	}
	if p.Details.Calendar.Amanta != "Pausha" || p.Details.Calendar.SkippedMonth != "Magha" || p.Details.Calendar.Adhika {
		t.Fatalf("Kshaya combination %+v", p.Details.Calendar)
	}
}

package swisseph

import (
	"context"
	"math"
	"testing"
	"time"
)

// TestHistoricalNativeInstantBoundaries checks native instants at the supported historical date boundaries.
func TestHistoricalNativeInstantBoundaries(t *testing.T) {
	for _, tc := range []struct {
		utc string
		ok  bool
	}{
		{"1939-12-30T23:59:59.999999999Z", false},
		{"1939-12-31T00:00:00Z", true},
		{"1940-01-01T00:00:00Z", true},
		{"2099-12-31T23:59:59Z", true},
		{"2100-01-01T00:00:00Z", false},
	} {
		at, err := time.Parse(time.RFC3339Nano, tc.utc)
		if err != nil {
			t.Fatal(err)
		}
		if supportedCalculationInstant(at) != tc.ok {
			t.Errorf("native instant boundary %s", tc.utc)
		}
		if _, err := SunPosition("../../ephe", at); (err == nil) != tc.ok {
			t.Errorf("Sun boundary %s: %v", tc.utc, err)
		}
		if _, err := CalculatePlanetPositions(context.Background(), "../../ephe", at); (err == nil) != tc.ok {
			t.Errorf("sky position boundary %s: %v", tc.utc, err)
		}
	}
}

// TestHistoricalSunKundaliAndSkyAgree checks that historical Sun positions agree between Kundali and sky calculations.
func TestHistoricalSunKundaliAndSkyAgree(t *testing.T) {
	for _, tc := range []struct {
		date, zone string
		lat, lon   float64
	}{
		{"1940-01-01", "Asia/Kolkata", 28.6139, 77.209},
		{"1940-01-01", "Australia/Sydney", -33.8688, 151.2093},
		{"1950-01-01", "Asia/Kolkata", 28.6139, 77.209},
		{"1960-02-29", "Asia/Kolkata", 13.0827, 80.2707},
	} {
		t.Run(tc.date+"/"+tc.zone, func(t *testing.T) {
			loc, err := time.LoadLocation(tc.zone)
			if err != nil {
				t.Fatal(err)
			}
			at, err := time.ParseInLocation("2006-01-02", tc.date, loc)
			if err != nil {
				t.Fatal(err)
			}
			if tc.date == MinDate && at.UTC().Format("2006-01-02") != "1939-12-31" {
				t.Fatal("historical local-midnight case does not exercise the UTC buffer")
			}
			chart, err := CalculateKundali("../../ephe", at, tc.lat, tc.lon)
			if err != nil {
				t.Fatal(err)
			}
			sun, err := SunPosition("../../ephe", at)
			if err != nil {
				t.Fatal(err)
			}
			sky, err := CalculatePlanetPositions(context.Background(), "../../ephe", at)
			if err != nil {
				t.Fatal(err)
			}
			if len(chart.Planets) != 9 || len(chart.Houses) != 12 || len(sky.Planets) != 9 {
				t.Fatal("incomplete historical natal/sky result")
			}
			if math.Abs(sun.Longitude-chart.Planets[0].Longitude) > 1e-9 || math.Abs(sun.Speed-chart.Planets[0].Speed) > 1e-9 {
				t.Fatal("historical Sun and Kundali disagree")
			}
			for i, p := range chart.Planets {
				if !finite(p.Longitude) || p.RashiNumber < 1 || p.RashiNumber > 12 || p.House < 1 || p.House > 12 || math.Abs(p.Longitude-sky.Planets[i].Longitude) > 1e-9 || math.Abs(p.Speed-sky.Planets[i].Speed) > 1e-9 {
					t.Errorf("historical native entrypoints disagree for %s", p.Name)
				}
			}
			if tc.date == MinDate {
				dasha := CalculateVimshottari(chart.Planets[1].Longitude, at, time.Date(2099, time.December, 30, 0, 0, 0, 0, time.UTC))
				if dasha.CurrentLord == "" || dasha.CurrentSubLord == "" || len(dasha.Periods) < 18 {
					t.Fatal("1940 birth does not retain complete dasha coverage to 2099")
				}
			}
		})
	}
}

// TestHistoricalPanchangStartsAndLeapDays checks historical Panchang starts and leap-day handling.
func TestHistoricalPanchangStartsAndLeapDays(t *testing.T) {
	for _, date := range []string{MinDate, "1940-02-28", "1940-02-29", "1940-03-01", "1950-01-01", "1950-06-21", "1960-01-01", "1960-02-29", "1972-01-01"} {
		for _, method := range []string{"upper-limb", "hindu"} {
			t.Run(date+"/"+method, func(t *testing.T) {
				p, err := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", method, 28.6139, 77.209)
				if err != nil {
					t.Fatal(err)
				}
				if p.Date != date || p.Sunrise.Format("2006-01-02") != date || !p.Sunset.After(p.Sunrise) || !p.NextSunrise.After(p.Sunset) || p.Details == nil {
					t.Fatal("invalid historical sunrise day")
				}
				for _, limb := range [][]PanchangPeriod{p.Tithi, p.Nakshatra, p.Yoga, p.Karana} {
					if len(limb) == 0 || !limb[0].Start.Equal(p.Sunrise) || limb[len(limb)-1].End.Before(p.NextSunrise) {
						t.Fatal("incomplete historical limb coverage")
					}
				}
				for _, event := range []*time.Time{p.Details.Moonrise, p.Details.Moonset} {
					if event != nil && (event.Before(p.Sunrise) || !event.Before(p.NextSunrise)) {
						t.Fatal("historical Moon event belongs to a different sunrise day")
					}
				}
				if date == "1940-02-29" && p.NextSunrise.Format("2006-01-02") != "1940-03-01" {
					t.Fatal("leap day did not advance to 1 March")
				}
			})
		}
	}
	for _, date := range []string{"1939-12-31", "1940-02-30", "1950-02-29"} {
		if _, err := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", "upper-limb", 28.6139, 77.209); err == nil {
			t.Errorf("invalid historical day accepted: %s", date)
		}
	}
}

// TestHistorical1940AnnualSkyTools checks annual sky calculations for the earliest supported year, 1940.
func TestHistorical1940AnnualSkyTools(t *testing.T) {
	transits, err := CalculatePlanetTransitions(context.Background(), "../../ephe", MinYear)
	if err != nil {
		t.Fatal(err)
	}
	solar := 0
	for i, event := range transits.Events {
		if event.At.Year() != MinYear || i > 0 && event.At.Before(transits.Events[i-1].At) {
			t.Fatal("historical transit outside civil year or out of order")
		}
		if event.Planet == "Sun" && event.Kind == "Sign ingress" {
			solar++
		}
	}
	if solar != 12 {
		t.Fatalf("1940 must contain 12 solar sign ingresses, got %d", solar)
	}
	eclipses, err := CalculateEclipses(context.Background(), "../../ephe", MinYear, 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	if len(eclipses.Events) < 2 {
		t.Fatal("incomplete 1940 eclipse year")
	}
	for i, event := range eclipses.Events {
		if event.Maximum.Year() != MinYear || event.Type == "Unknown" || i > 0 && event.Maximum.Before(eclipses.Events[i-1].Maximum) {
			t.Fatal("historical eclipse outside civil year, unknown or out of order")
		}
		if event.Local != nil && !event.Local.VisibleEnd.After(event.Local.VisibleStart) {
			t.Fatal("invalid historical local eclipse window")
		}
	}
}

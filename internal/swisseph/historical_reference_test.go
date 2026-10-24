package swisseph

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// TestHistoricalOfficialCLIReferences compares historical calculations with official Swiss command-line fixtures.
func TestHistoricalOfficialCLIReferences(t *testing.T) {
	var reference struct {
		Tolerance float64 `json:"tolerance_degrees"`
		Cases     []struct {
			At                         string
			Latitude, Longitude, Lagna float64
			Planets                    []struct {
				Name             string
				Longitude, Speed float64
			}
		}
	}
	raw, err := os.ReadFile("testdata/historical-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &reference); err != nil {
		t.Fatal(err)
	}
	if len(reference.Cases) != 4 || reference.Tolerance <= 0 || reference.Tolerance > 1e-6 {
		t.Fatal("missing historical reference fixtures")
	}
	for _, c := range reference.Cases {
		t.Run(c.At, func(t *testing.T) {
			at, err := time.Parse(time.RFC3339, c.At)
			if err != nil {
				t.Fatal(err)
			}
			chart, err := CalculateKundali("../../ephe", at, c.Latitude, c.Longitude)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(chart.Lagna.Longitude-c.Lagna) > reference.Tolerance {
				t.Fatalf("historical Lagna %v want %v", chart.Lagna.Longitude, c.Lagna)
			}
			for i, want := range c.Planets {
				got := chart.Planets[i]
				if got.Name != want.Name || math.Abs(got.Longitude-want.Longitude) > reference.Tolerance || math.Abs(got.Speed-want.Speed) > reference.Tolerance {
					t.Fatalf("historical %s longitude/speed %+v want %+v", want.Name, got, want)
				}
			}
			sun, err := SunPosition("../../ephe", at)
			if err != nil || math.Abs(sun.Longitude-chart.Planets[0].Longitude) > reference.Tolerance {
				t.Fatalf("Sun wrapper disagreement: %v", err)
			}
			// Swiss explicitly treats pre-1972 input clocks as UT1, so these
			// fields must equal the direct Gregorian clock JD without a UTC
			// leap-second or manual Delta-T correction.
			wantJD := float64(at.Unix())/86400 + 2440587.5
			if math.Abs(chart.JulianDayUT1-wantJD) > 1e-9 {
				t.Fatal("historical input clock was not interpreted as documented UT1")
			}
		})
	}
}

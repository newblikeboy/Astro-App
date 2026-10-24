package swisseph

import (
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// TestOfficialSwetestReference compares native chart results with official Swiss command-line reference values.
func TestOfficialSwetestReference(t *testing.T) {
	b, e := os.ReadFile("testdata/swetest-reference.json")
	if e != nil {
		t.Fatal(e)
	}
	var fixtures []struct {
		UTC     string             `json:"utc"`
		Lat     float64            `json:"latitude"`
		Lon     float64            `json:"longitude"`
		Lagna   float64            `json:"lagna"`
		Planets map[string]float64 `json:"planets"`
	}
	if e = json.Unmarshal(b, &fixtures); e != nil {
		t.Fatal(e)
	}
	for _, f := range fixtures {
		t.Run(f.UTC, func(t *testing.T) {
			instant, _ := time.Parse(time.RFC3339, f.UTC)
			chart, e := CalculateKundali("../../ephe", instant, f.Lat, f.Lon)
			if e != nil {
				t.Fatal(e)
			}
			if math.Abs(chart.Lagna.Longitude-f.Lagna) > 1e-6 {
				t.Fatalf("lagna got %.9f want %.9f", chart.Lagna.Longitude, f.Lagna)
			}
			for _, p := range chart.Planets {
				if want, ok := f.Planets[p.Name]; ok && math.Abs(p.Longitude-want) > 1e-6 {
					t.Errorf("%s got %.9f want %.9f", p.Name, p.Longitude, want)
				}
			}
		})
	}
}

package swisseph

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"sort"
	"testing"
	"time"
)

// NASA supplies the planet's apparent tropical longitude. The fixture shares
// the declared Lahiri coordinate origin only; planetary coordinates and annual
// event searches are independently supplied/derived from Horizons DE441.
// This is not an independent certification of the ayanamsa or mean lunar nodes.
type jplSkyFixture struct {
	Source            string  `json:"source"`
	PositionTolerance float64 `json:"position_tolerance_arcsec"`
	IngressTolerance  float64 `json:"ingress_tolerance_seconds"`
	StationTolerance  float64 `json:"station_tolerance_seconds"`
	Year              int     `json:"year"`
	Positions         []struct {
		Planet   string    `json:"planet"`
		At       time.Time `json:"at"`
		Tropical float64   `json:"jpl_apparent_tropical_longitude"`
		Origin   float64   `json:"lahiri_origin"`
		Sidereal float64   `json:"sidereal_longitude"`
	} `json:"positions"`
	Events []struct {
		Planet     string    `json:"planet"`
		Kind       string    `json:"kind"`
		At         time.Time `json:"at"`
		From       int       `json:"from_sign_number"`
		To         int       `json:"to_sign_number"`
		Retrograde bool      `json:"retrograde"`
	} `json:"events"`
	DenseEvents []struct {
		Planet string    `json:"planet"`
		At     time.Time `json:"at"`
	} `json:"dense_events"`
}

// loadJPLSkyFixture loads the offline NASA fixture and rejects incomplete cases or widened acceptance limits.
func loadJPLSkyFixture(t *testing.T) jplSkyFixture {
	t.Helper()
	raw, err := os.ReadFile("testdata/jpl-sky-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var f jplSkyFixture
	if err := json.Unmarshal(raw, &f); err != nil {
		t.Fatal(err)
	}
	if f.Year != 2026 || len(f.Positions) != 42 || len(f.Events) < 200 || len(f.DenseEvents) != 3 ||
		f.PositionTolerance <= 0 || f.PositionTolerance > 0.5 ||
		f.IngressTolerance <= 0 || f.IngressTolerance > 60 ||
		f.StationTolerance <= 0 || f.StationTolerance > 120 {
		t.Fatal("incomplete independent JPL fixture or invalid tolerance")
	}
	return f
}

// jplAngleDifference returns the signed shortest angular separation for independent NASA comparisons.
func jplAngleDifference(a, b float64) float64 {
	return math.Mod(a-b+540, 360) - 180
}

// TestSkyIndependentJPLPositions compares physical planet positions with independent NASA/JPL coordinates.
func TestSkyIndependentJPLPositions(t *testing.T) {
	f := loadJPLSkyFixture(t)
	cache := map[time.Time]PlanetPositions{}
	maximum := map[string]float64{}
	for _, want := range f.Positions {
		if !finite(want.Tropical) || !finite(want.Origin) || !finite(want.Sidereal) ||
			want.Origin < 20 || want.Origin > 30 ||
			math.Abs(jplAngleDifference(want.Tropical-want.Origin, want.Sidereal)) > 1e-10 {
			t.Fatal("invalid NASA longitude or declared coordinate origin")
		}
		positions, exists := cache[want.At]
		if !exists {
			var err error
			positions, err = CalculatePlanetPositions(context.Background(), "../../ephe", want.At)
			if err != nil {
				t.Fatal(err)
			}
			cache[want.At] = positions
		}
		found := false
		for _, got := range positions.Planets {
			if got.Name != want.Planet {
				continue
			}
			found = true
			residual := jplAngleDifference(got.Longitude, want.Sidereal) * 3600
			maximum[got.Name] = math.Max(maximum[got.Name], math.Abs(residual))
			t.Logf("JPL_POSITION planet=%s at=%s residual_arcsec=%.6f", got.Name, want.At.Format(time.RFC3339), residual)
			if math.Abs(residual) > f.PositionTolerance {
				t.Errorf("%s %s differs from independent NASA coordinate by %.6f arcsec (limit %.3f)",
					want.At.Format(time.RFC3339), got.Name, residual, f.PositionTolerance)
			}
		}
		if !found {
			t.Fatalf("NASA reference planet missing: %s", want.Planet)
		}
	}
	for _, planet := range skyNames[:7] {
		t.Logf("JPL_POSITION_MAX planet=%s residual_arcsec=%.6f", planet, maximum[planet])
	}
}

// TestSkyIndependentJPLAnnualEvents compares annual physical planet ingresses and stations with NASA/JPL-derived events.
func TestSkyIndependentJPLAnnualEvents(t *testing.T) {
	f := loadJPLSkyFixture(t)
	actual, err := CalculatePlanetTransitions(context.Background(), "../../ephe", f.Year)
	if err != nil {
		t.Fatal(err)
	}
	groups := map[string][]PlanetTransition{}
	for _, e := range actual.Events {
		// Mean nodes are mathematical orbit points, not Horizons bodies.
		if e.Planet == "Rahu" || e.Planet == "Ketu" {
			continue
		}
		key := e.Planet + "/" + e.Kind
		groups[key] = append(groups[key], e)
	}
	for key := range groups {
		sort.Slice(groups[key], func(i, j int) bool { return groups[key][i].At.Before(groups[key][j].At) })
	}
	indices := map[string]int{}
	maximumIngress, maximumStation := 0.0, 0.0
	for _, want := range f.Events {
		key := want.Planet + "/" + want.Kind
		i := indices[key]
		if i >= len(groups[key]) {
			t.Errorf("NASA event missing from annual output: %s %s", key, want.At)
			continue
		}
		got := groups[key][i]
		indices[key]++
		residual := got.At.Sub(want.At).Seconds()
		tolerance := f.StationTolerance
		if want.Kind == "Sign ingress" {
			tolerance = f.IngressTolerance
			maximumIngress = math.Max(maximumIngress, math.Abs(residual))
			if want.From < 1 || want.From > 12 || want.To < 1 || want.To > 12 ||
				got.FromSign != rashiNames[want.From-1] || got.ToSign != rashiNames[want.To-1] {
				t.Errorf("%s has different sign crossing from NASA reference: %+v", key, got)
			}
		} else {
			maximumStation = math.Max(maximumStation, math.Abs(residual))
		}
		t.Logf("JPL_EVENT planet=%s kind=%q ours=%s reference=%s residual_seconds=%.6f",
			got.Planet, got.Kind, got.At.Format(time.RFC3339Nano), want.At.Format(time.RFC3339Nano), residual)
		if math.Abs(residual) > tolerance || got.Retrograde != want.Retrograde {
			t.Errorf("%s %s differs from NASA: residual %.3fs, limit %.0fs; direction got=%t want=%t",
				key, want.At, residual, tolerance, got.Retrograde, want.Retrograde)
		}
	}
	for key, rows := range groups {
		if indices[key] != len(rows) {
			t.Errorf("extra local events absent from independent NASA year search: %s local=%d NASA=%d", key, len(rows), indices[key])
		}
	}
	for _, want := range f.DenseEvents {
		rows := groups[want.Planet+"/Sign ingress"]
		closest := math.Inf(1)
		for _, got := range rows {
			closest = math.Min(closest, math.Abs(got.At.Sub(want.At).Seconds()))
		}
		if closest > f.IngressTolerance {
			t.Errorf("dense one-minute NASA %s ingress differs by %.3fs", want.Planet, closest)
		}
		t.Logf("JPL_DENSE_EVENT planet=%s reference=%s residual_seconds=%.6f", want.Planet, want.At.Format(time.RFC3339Nano), closest)
	}
	t.Logf("JPL_EVENT_MAX ingress_seconds=%.6f station_seconds=%.6f count=%d", maximumIngress, maximumStation, len(f.Events))
}

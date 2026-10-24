package swisseph

import (
	"context"
	"errors"
	"math"
	"testing"
)

// TestCombustionLimits preserves inclusive unrounded boundaries and the Kundli rule for every eligible graha.
func TestCombustionLimits(t *testing.T) {
	for _, row := range []struct {
		name          string
		direct, retro float64
	}{
		{"Moon", 12, 12}, {"Mars", 17, 17}, {"Mercury", 14, 12}, {"Jupiter", 11, 11}, {"Venus", 10, 8}, {"Saturn", 15, 15},
	} {
		for _, retro := range []bool{false, true} {
			limit, ok := combustionLimit(row.name, retro)
			want := row.direct
			if retro {
				want = row.retro
			}
			if !ok || limit != want {
				t.Fatalf("%s retro=%v: %g", row.name, retro, limit)
			}
			at := combustionSample{separation: limit, limit: limit}
			if !at.active() {
				t.Fatal("boundary must be inclusive")
			}
			at.separation = math.Nextafter(limit, math.Inf(1))
			if at.active() {
				t.Fatal("classification used rounded separation")
			}
		}
	}
	for _, name := range []string{"Sun", "Rahu", "Ketu", "unknown"} {
		if _, ok := combustionLimit(name, false); ok {
			t.Fatalf("ineligible combustion body %s", name)
		}
	}
}

// TestCombustionReversals preserves close entry/exit pairs at a relative-longitude extremum and threshold jumps at a station.
func TestCombustionReversals(t *testing.T) {
	// turn has two threshold crossings inside a six-hour bracket whose endpoints are both non-combust.
	turn := func(jd float64) (combustionSample, error) {
		x := jd - .1
		return combustionSample{planet: SkyPlanet{Speed: 1 + 2*x}, sunSpeed: 1, separation: 14 + x*x - .0001, limit: 14}, nil
	}
	l, _ := turn(0)
	r, _ := turn(.25)
	points, err := combustionTurningPoints(turn, 0, .25, l, r)
	if err != nil || len(points) != 3 {
		t.Fatalf("missing relative turning point %v %v", points, err)
	}
	for i, want := range []float64{.09, .11} {
		at, _, reason, err := refineCombustionChange(turn, points[i], points[i+1])
		if err != nil || math.Abs(at-want) > .05/86400 || reason != "Sun separation" {
			t.Fatalf("bad close root %g %v %s", at, err, reason)
		}
	}
	// jump isolates Mercury's inclusive 14-to-12 degree rule change at a speed zero.
	jump := func(jd float64) (combustionSample, error) {
		limit, _ := combustionLimit("Mercury", jd >= .1)
		return combustionSample{planet: SkyPlanet{Speed: .1 - jd}, separation: 13, limit: limit}, nil
	}
	at, after, reason, err := refineCombustionChange(jump, 0, .25)
	if err != nil || math.Abs(at-.1) > .05/86400 || after.active() || reason != "Motion-dependent limit" {
		t.Fatalf("bad threshold jump %g %+v %s %v", at, after, reason, err)
	}
}

// TestSkyAnnualCombustion checks root states, angular evidence, completeness, civil-year edges and native failure handling.
func TestSkyAnnualCombustion(t *testing.T) {
	for _, year := range []int{1940, 2026, 2099} {
		out, err := CalculatePlanetCombustions(context.Background(), "../../ephe", year)
		if err != nil {
			t.Fatal(err)
		}
		if len(out.Events) < 30 || out.Settings["method"] == "" {
			t.Fatalf("incomplete combustion year %d", year)
		}
		err = withSky(context.Background(), "../../ephe", "Asia/Kolkata", func(s *skySession) error {
			start, end, err := s.yearBounds(year)
			if err != nil {
				return err
			}
			for i, e := range out.Events {
				if e.At.Year() != year || (i > 0 && e.At.Before(out.Events[i-1].At)) {
					t.Fatalf("invalid ordering/year %+v", e)
				}
				index := -1
				for k, name := range skyNames {
					if name == e.Planet {
						index = k
					}
				}
				if index < 1 || index > 6 {
					t.Fatalf("ineligible planet %+v", e)
				}
				jd, err := s.jd(e.At)
				if err != nil {
					return err
				}
				// active independently classifies raw native coordinates rather than calling the event solver.
				active := func(at float64) (bool, error) {
					p, err := s.position(at, index)
					if err != nil {
						return false, err
					}
					sun, err := s.position(at, 0)
					if err != nil {
						return false, err
					}
					limit, _ := combustionLimit(p.Name, p.Retrograde)
					return math.Abs(math.Remainder(p.Longitude-sun.Longitude, 360)) <= limit, nil
				}
				before, err := active(jd - 2.0/86400)
				if err != nil {
					return err
				}
				after, err := active(jd + 2.0/86400)
				if err != nil {
					return err
				}
				if before == after || after != (e.Kind == "Combustion begins") {
					t.Fatalf("incorrect boundary %+v", e)
				}
				if e.Boundary == "Sun separation" && math.Abs(e.SunSeparation-e.CombustLimit) > .00002 {
					t.Fatalf("bad angular evidence %+v", e)
				}
			}
			// Independently sample hourly states; every observed toggle must have exactly one returned boundary.
			for index := 1; index <= 6; index++ {
				// state calculates an hourly physical state for a completeness sweep, without event refinement.
				state := func(jd float64) (bool, error) {
					p, err := s.position(jd, index)
					if err != nil {
						return false, err
					}
					sun, err := s.position(jd, 0)
					if err != nil {
						return false, err
					}
					limit, _ := combustionLimit(p.Name, p.Retrograde)
					return math.Abs(math.Remainder(p.Longitude-sun.Longitude, 360)) <= limit, nil
				}
				current, err := state(start)
				if err != nil {
					return err
				}
				changes := 0
				for at := start + 1.0/24; at < end; at += 1.0 / 24 {
					next, err := state(at)
					if err != nil {
						return err
					}
					if next != current {
						changes++
						current = next
					}
				}
				count := 0
				for _, e := range out.Events {
					if e.Planet == skyNames[index] {
						count++
					}
				}
				if count != changes {
					t.Fatalf("%d %s: events %d, independent hourly changes %d", year, skyNames[index], count, changes)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, year := range []int{1939, 2100} {
		if _, err := CalculatePlanetCombustions(context.Background(), "../../ephe", year); err == nil {
			t.Fatal("unsupported year accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CalculatePlanetCombustions(ctx, "../../ephe", 2026); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
	if _, err := CalculatePlanetCombustions(context.Background(), t.TempDir(), 2026); err == nil {
		t.Fatal("missing ephemeris accepted")
	}
}

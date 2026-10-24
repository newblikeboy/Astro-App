package swisseph

import (
	"context"
	"math"
	"sort"
	"time"
)

type PlanetCombustionEvent struct {
	Planet        string    `json:"planet"`
	Kind          string    `json:"kind"`
	At            time.Time `json:"at"`
	Sign          string    `json:"to_sign"`
	Retrograde    bool      `json:"retrograde"`
	SunSeparation float64   `json:"sun_separation"`
	CombustLimit  float64   `json:"combust_limit"`
	Boundary      string    `json:"boundary"`
}

type PlanetCombustions struct {
	Year     int                     `json:"year"`
	Events   []PlanetCombustionEvent `json:"events"`
	Settings map[string]string       `json:"settings"`
}

// combustionLimit supplies the shared inclusive longitude-separation rule in degrees; the Sun and lunar nodes are excluded.
func combustionLimit(name string, retrograde bool) (float64, bool) {
	switch name {
	case "Moon":
		return 12, true
	case "Mars":
		return 17, true
	case "Mercury":
		if retrograde {
			return 12, true
		}
		return 14, true
	case "Jupiter":
		return 11, true
	case "Venus":
		if retrograde {
			return 8, true
		}
		return 10, true
	case "Saturn":
		return 15, true
	}
	return 0, false
}

type combustionSample struct {
	planet                      SkyPlanet
	sunSpeed, separation, limit float64
}

// active classifies unrounded shortest longitude separation using the declared inclusive threshold.
func (c combustionSample) active() bool { return c.separation <= c.limit }

// refineCombustionChange brackets a state change to 0.05 seconds, including motion-dependent threshold jumps.
func refineCombustionChange(get func(float64) (combustionSample, error), a, b float64) (float64, combustionSample, string, error) {
	l, err := get(a)
	if err != nil {
		return 0, l, "", err
	}
	r, err := get(b)
	if err != nil {
		return 0, r, "", err
	}
	for b-a > .05/86400 {
		mid := (a + b) / 2
		p, err := get(mid)
		if err != nil {
			return 0, p, "", err
		}
		if p.active() == l.active() {
			a, l = mid, p
		} else {
			b, r = mid, p
		}
	}
	boundary := "Sun separation"
	if l.limit != r.limit {
		boundary = "Motion-dependent limit"
	}
	return (a + b) / 2, r, boundary, nil
}

// combustionTurningPoints splits six-hour samples at planet and relative-Sun speed zeros so a reversal cannot hide two crossings.
func combustionTurningPoints(get func(float64) (combustionSample, error), a, b float64, l, r combustionSample) ([]float64, error) {
	points := []float64{a, b}
	for _, relative := range []bool{false, true} {
		// speed selects physical longitude motion or the derivative of Sun-relative longitude.
		speed := func(p combustionSample) float64 {
			if relative {
				return p.planet.Speed - p.sunSpeed
			}
			return p.planet.Speed
		}
		if (speed(l) < 0) == (speed(r) < 0) {
			continue
		}
		lo, hi := a, b
		for hi-lo > .01/86400 {
			mid := (lo + hi) / 2
			p, err := get(mid)
			if err != nil {
				return nil, err
			}
			if (speed(p) < 0) == (speed(l) < 0) {
				lo = mid
			} else {
				hi = mid
			}
		}
		points = append(points, (lo+hi)/2)
	}
	sort.Float64s(points)
	return points, nil
}

// CalculatePlanetCombustions finds native annual combustion entry/exit instants under the same angular rules as Kundli analysis.
// The India civil year is half-open; no sunrise, atmospheric visibility or heliacal rise/set is inferred.
func CalculatePlanetCombustions(ctx context.Context, epheDir string, year int) (PlanetCombustions, error) {
	out := PlanetCombustions{Year: year, Events: []PlanetCombustionEvent{}}
	err := withSky(ctx, epheDir, "Asia/Kolkata", func(s *skySession) error {
		start, end, err := s.yearBounds(year)
		if err != nil {
			return err
		}
		for index := 1; index <= 6; index++ {
			// get evaluates the graha and Sun in one locked native session without display rounding.
			get := func(jd float64) (combustionSample, error) {
				p, err := s.position(jd, index)
				if err != nil {
					return combustionSample{}, err
				}
				sun, err := s.position(jd, 0)
				if err != nil {
					return combustionSample{}, err
				}
				limit, _ := combustionLimit(p.Name, p.Retrograde)
				return combustionSample{p, sun.Speed, math.Abs(math.Remainder(p.Longitude-sun.Longitude, 360)), limit}, nil
			}
			a := start - 1.0/86400
			left, err := get(a)
			if err != nil {
				return err
			}
			for a < end {
				b := math.Min(a+.25, end)
				right, err := get(b)
				if err != nil {
					return err
				}
				points, err := combustionTurningPoints(get, a, b, left, right)
				if err != nil {
					return err
				}
				for k := 0; k < len(points)-1; k++ {
					l, err := get(points[k])
					if err != nil {
						return err
					}
					r, err := get(points[k+1])
					if err != nil {
						return err
					}
					if l.active() == r.active() {
						continue
					}
					at, after, boundary, err := refineCombustionChange(get, points[k], points[k+1])
					if err != nil {
						return err
					}
					if at < start || at >= end {
						continue
					}
					kind := "Combustion ends"
					if after.active() {
						kind = "Combustion begins"
					}
					out.Events = append(out.Events, PlanetCombustionEvent{skyNames[index], kind, s.instant(at), after.planet.Rashi, after.planet.Retrograde, after.separation, after.limit, boundary})
				}
				a, left = b, right
			}
		}
		sort.Slice(out.Events, func(i, j int) bool {
			if out.Events[i].At.Equal(out.Events[j].At) {
				return out.Events[i].Planet < out.Events[j].Planet
			}
			return out.Events[i].At.Before(out.Events[j].At)
		})
		out.Settings = s.settings()
		out.Settings["method"] = "m4y-combustion-v1; inclusive shortest Sun longitude separation; Moon 12; Mars 17; Mercury 14/12 retrograde; Jupiter 11; Venus 10/8 retrograde; Saturn 15 degrees"
		out.Settings["search"] = "India civil year [start,end); six-hour samples split at motion and relative-longitude extrema; state boundaries to 0.05 second"
		return nil
	})
	return out, err
}

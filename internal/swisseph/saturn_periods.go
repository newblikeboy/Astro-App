package swisseph

import (
	"context"
	"fmt"
	"math"
	"time"
)

type SaturnPeriod struct {
	Kind         string    `json:"kind"`
	Phase        string    `json:"phase"`
	Sign         int       `json:"sign"`
	FromMoon     int       `json:"from_moon"`
	Start        time.Time `json:"start"`
	End          time.Time `json:"end"`
	StartClipped bool      `json:"start_clipped"`
	EndClipped   bool      `json:"end_clipped"`
}
type SaturnPeriods struct {
	Method   string         `json:"method"`
	MoonSign int            `json:"moon_sign"`
	Start    time.Time      `json:"start"`
	End      time.Time      `json:"end"`
	Periods  []SaturnPeriod `json:"periods"`
}

// saturnKind classifies Saturn's whole-sign distance from natal Moon as Sade Sati, Dhaiya or neither.
func saturnKind(sign, moon int) (string, string, int) {
	distance := (sign-moon+12)%12 + 1
	switch distance {
	case 12:
		return "Sade Sati", "Rising", distance
	case 1:
		return "Sade Sati", "Middle", distance
	case 2:
		return "Sade Sati", "Setting", distance
	case 4:
		return "Dhaiya", "Fourth from Moon", distance
	case 8:
		return "Dhaiya", "Eighth from Moon", distance
	}
	return "", "", distance
}

// Each six-hour bracket is split at a motion station before ingress searches.
// This preserves retrograde exit/re-entry, including a close pair at a station.
func saturnCrossings(ctx context.Context, start, end float64, get func(float64) (SkyPlanet, error)) ([]*signIngressRoot, error) {
	out := []*signIngressRoot{}
	left, err := get(start)
	if err != nil {
		return nil, err
	}
	for a := start; a < end; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		b := math.Min(a+0.25, end)
		right, e := get(b)
		if e != nil {
			return nil, e
		}
		points := []float64{a, b}
		if (left.Speed < 0) != (right.Speed < 0) {
			lo, hi := a, b
			for hi-lo > 0.01/86400 {
				mid := (lo + hi) / 2
				p, e := get(mid)
				if e != nil {
					return nil, e
				}
				if (p.Speed < 0) == (left.Speed < 0) {
					lo = mid
				} else {
					hi = mid
				}
			}
			points = []float64{a, (lo + hi) / 2, b}
		}
		for i := 0; i < len(points)-1; i++ {
			crossing, e := refineSignIngress(get, points[i], points[i+1])
			if e != nil {
				return nil, e
			}
			if crossing != nil && crossing.at >= start && crossing.at < end {
				out = append(out, crossing)
			}
		}
		a, left = b, right
	}
	return out, nil
}

// CalculateSaturnPeriods finds bounded Saturn transit periods, retaining retrograde exits, re-entries and clipped edges.
func CalculateSaturnPeriods(ctx context.Context, epheDir, zone string, moonSign, startYear, endYear int) (SaturnPeriods, error) {
	out := SaturnPeriods{Method: "Lahiri geocentric Saturn; 12/1/2 from natal Moon = Sade Sati; 4/8 = Dhaiya; every direct/retrograde ingress; [start,end)", MoonSign: moonSign, Periods: []SaturnPeriod{}}
	if moonSign < 1 || moonSign > 12 || startYear < 1940 || endYear > 2100 || endYear <= startYear || endYear-startYear > 30 {
		return out, fmt.Errorf("choose a 1-30 year window within 1940-2100")
	}
	err := withSky(ctx, epheDir, zone, func(s *skySession) error {
		out.Start = time.Date(startYear, 1, 1, 0, 0, 0, 0, s.loc)
		out.End = time.Date(endYear, 1, 1, 0, 0, 0, 0, s.loc)
		start, e := s.jd(out.Start)
		if e != nil {
			return e
		}
		end, e := s.jd(out.End)
		if e != nil {
			return e
		}
		// get evaluates Saturn's native sidereal position during the requested timeline search.
		get := func(jd float64) (SkyPlanet, error) { return s.position(jd, 6) }
		initial, e := get(start)
		if e != nil {
			return e
		}
		roots, e := saturnCrossings(ctx, start, end, get)
		if e != nil {
			return e
		}
		cursor, sign := out.Start, initial.RashiNumber
		// appendPeriod records a positive-duration Saturn period with its classification and clipped-edge evidence.
		appendPeriod := func(edge time.Time, last bool) {
			kind, phase, distance := saturnKind(sign, moonSign)
			if kind != "" && edge.After(cursor) {
				out.Periods = append(out.Periods, SaturnPeriod{kind, phase, sign, distance, cursor, edge, cursor.Equal(out.Start), last})
			}
		}
		for _, root := range roots {
			instant := s.instant(root.at)
			appendPeriod(instant, false)
			cursor, sign = instant, root.to.RashiNumber
		}
		appendPeriod(out.End, true)
		return nil
	})
	return out, err
}

package swisseph

/*
#include <stdlib.h>
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
	"unsafe"
)

type SkyPlanet struct {
	Name string `json:"name"`
	ZodiacPosition
	Speed      float64 `json:"longitude_speed_deg_per_day"`
	Retrograde bool    `json:"retrograde"`
}
type PlanetPositions struct {
	At       time.Time         `json:"at"`
	Planets  []SkyPlanet       `json:"planets"`
	Settings map[string]string `json:"settings"`
}
type PlanetTransition struct {
	Planet     string    `json:"planet"`
	Kind       string    `json:"kind"`
	At         time.Time `json:"at"`
	FromSign   string    `json:"from_sign"`
	ToSign     string    `json:"to_sign"`
	Retrograde bool      `json:"retrograde"`
}
type PlanetTransitions struct {
	Year     int                `json:"year"`
	Events   []PlanetTransition `json:"events"`
	Settings map[string]string  `json:"settings"`
}
type EclipseContact struct {
	Name         string    `json:"name"`
	At           time.Time `json:"at"`
	AboveHorizon *bool     `json:"above_horizon,omitempty"`
}
type LocalEclipse struct {
	Type                string           `json:"type"`
	Maximum             time.Time        `json:"maximum"`
	MaximumAboveHorizon bool             `json:"maximum_above_horizon"`
	Magnitude           float64          `json:"magnitude"`
	Obscuration         *float64         `json:"obscuration,omitempty"`
	VisibleStart        time.Time        `json:"visible_start"`
	VisibleEnd          time.Time        `json:"visible_end"`
	Contacts            []EclipseContact `json:"contacts"`
}
type Eclipse struct {
	Kind     string           `json:"kind"`
	Type     string           `json:"type"`
	Maximum  time.Time        `json:"maximum"`
	Contacts []EclipseContact `json:"contacts"`
	Visible  bool             `json:"visible"`
	Local    *LocalEclipse    `json:"local,omitempty"`
}
type EclipseYear struct {
	Year     int               `json:"year"`
	Events   []Eclipse         `json:"events"`
	Settings map[string]string `json:"settings"`
}

type skySession struct {
	ctx context.Context
	loc *time.Location
}

var skyNames = [...]string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"}

// withSky locks shared Swiss state, verifies required data files and initializes the fixed Lahiri calculation session.
func withSky(ctx context.Context, epheDir, zone string, work func(*skySession) error) error {
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return fmt.Errorf("invalid timezone")
	}
	abs, err := filepath.Abs(epheDir)
	if err != nil {
		return err
	}
	for _, name := range []string{"sepl_18.se1", "semo_18.se1"} {
		info, e := os.Stat(filepath.Join(abs, name))
		if e != nil || info.IsDir() {
			return fmt.Errorf("required ephemeris data unavailable")
		}
	}
	engineMu.Lock()
	defer engineMu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	cPath := C.CString(abs)
	defer C.free(unsafe.Pointer(cPath))
	C.swe_set_ephe_path(cPath)
	defer C.swe_close()
	C.swe_set_sid_mode(C.SE_SIDM_LAHIRI, 0, 0)
	return work(&skySession{ctx, loc})
}

// jd converts a precise UTC instant to UT1 Julian day through Swiss's time-scale conversion.
func (s *skySession) jd(t time.Time) (float64, error) {
	u := t.UTC()
	var jd [2]C.double
	var msg [256]C.char
	ret := C.swe_utc_to_jd(C.int(u.Year()), C.int(u.Month()), C.int(u.Day()), C.int(u.Hour()), C.int(u.Minute()), C.double(float64(u.Second())+float64(u.Nanosecond())/1e9), C.SE_GREG_CAL, &jd[0], &msg[0])
	if ret < 0 {
		return 0, fmt.Errorf("UTC conversion failed")
	}
	return float64(jd[1]), nil
}

// instant converts a native UT1 Julian day to the session's local timezone without truncating precision.
func (s *skySession) instant(jd float64) time.Time {
	var y, m, d, h, min C.int
	var sec C.double
	C.swe_jdut1_to_utc(C.double(jd), C.SE_GREG_CAL, &y, &m, &d, &h, &min, &sec)
	return time.Date(int(y), time.Month(m), int(d), int(h), int(min), 0, 0, time.UTC).Add(time.Duration(float64(sec) * float64(time.Second))).In(s.loc)
}

// position calculates an apparent sidereal graha, using mean Rahu and exactly opposite Ketu.
func (s *skySession) position(jd float64, index int) (SkyPlanet, error) {
	var out SkyPlanet
	if err := s.ctx.Err(); err != nil {
		return out, err
	}
	ids := [...]C.int32{C.SE_SUN, C.SE_MOON, C.SE_MARS, C.SE_MERCURY, C.SE_JUPITER, C.SE_VENUS, C.SE_SATURN, C.SE_MEAN_NODE, C.SE_MEAN_NODE}
	var xx [6]C.double
	var msg [256]C.char
	flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SIDEREAL | C.SEFLG_SPEED)
	ret := C.swe_calc_ut(C.double(jd), ids[index], flags, &xx[0], &msg[0])
	if !swissFlagsMatch(int32(ret), int32(flags)) || !finite(float64(xx[0])) || !finite(float64(xx[3])) {
		return out, fmt.Errorf("Swiss ephemeris position unavailable (flags=%d)", int32(ret))
	}
	lon := float64(xx[0])
	if index == 8 {
		lon += 180
	}
	return SkyPlanet{skyNames[index], zodiacPosition(lon), float64(xx[3]), xx[3] < 0}, nil
}

// settings returns the engine, coordinate, ayanamsa, node and timezone conventions used by the result.
func (s *skySession) settings() map[string]string {
	var version [256]C.char
	C.swe_version(&version[0])
	return map[string]string{"engine": "m4y-sky-v2", "swiss_version": C.GoString(&version[0]), "ephemeris": "Swiss data files", "ayanamsa": "Lahiri", "coordinates": "Geocentric apparent", "nodes": "Mean Rahu; Ketu opposite", "timezone": s.loc.String()}
}

// yearBounds resolves the supported civil year's start and exclusive end to UT1 Julian days.
func (s *skySession) yearBounds(year int) (float64, float64, error) {
	if year < MinYear || year > MaxYear {
		return 0, 0, fmt.Errorf("year must be between %d and %d", MinYear, MaxYear)
	}
	a, err := s.jd(time.Date(year, 1, 1, 0, 0, 0, 0, s.loc))
	if err != nil {
		return 0, 0, err
	}
	b, err := s.jd(time.Date(year+1, 1, 1, 0, 0, 0, 0, s.loc))
	return a, b, err
}

// CalculatePlanetPositions returns all nine grahas at the requested supported instant under the declared native settings.
func CalculatePlanetPositions(ctx context.Context, epheDir string, at time.Time) (PlanetPositions, error) {
	out := PlanetPositions{At: at, Planets: []SkyPlanet{}}
	if !supportedCalculationInstant(at) {
		return out, fmt.Errorf("unsupported date")
	}
	err := withSky(ctx, epheDir, "Asia/Kolkata", func(s *skySession) error {
		jd, err := s.jd(at)
		if err != nil {
			return err
		}
		out.At = at.In(s.loc)
		for i := range skyNames {
			p, e := s.position(jd, i)
			if e != nil {
				return e
			}
			out.Planets = append(out.Planets, p)
		}
		out.Settings = s.settings()
		return nil
	})
	return out, err
}

type signIngressRoot struct {
	at       float64
	from, to SkyPlanet
}

// refineSignIngress searches one monotonic interval between samples or a
// station. The six-hour outer sampling limits the fastest body (the Moon) to
// less than one sign per interval. Keep the signs of this bracket: probing a
// fixed duration before/after the root can cross the neighbouring station and
// erase both members of a close direct/retrograde ingress pair.
func refineSignIngress(get func(float64) (SkyPlanet, error), a, b float64) (*signIngressRoot, error) {
	l, err := get(a)
	if err != nil {
		return nil, err
	}
	r, err := get(b)
	if err != nil {
		return nil, err
	}
	if l.RashiNumber == r.RashiNumber {
		return nil, nil
	}
	delta := (r.RashiNumber - l.RashiNumber + 12) % 12
	if delta != 1 && delta != 11 {
		return nil, fmt.Errorf("invalid sign ingress bracket")
	}
	lo, hi := a, b
	for hi-lo > 0.05/86400 {
		mid := (lo + hi) / 2
		p, err := get(mid)
		if err != nil {
			return nil, err
		}
		if p.RashiNumber == l.RashiNumber {
			lo = mid
		} else {
			hi = mid
		}
	}
	return &signIngressRoot{(lo + hi) / 2, l, r}, nil
}

// Annual sidereal sign crossings, including retrograde re-entry. Six-hour
// samples bracket stations; splitting at each speed zero prevents losing a
// pair of crossings around a reversal. Planetary stations are separated by
// much more than six hours; the Moon and mean nodes have no stations.
func CalculatePlanetTransitions(ctx context.Context, epheDir string, year int) (PlanetTransitions, error) {
	out := PlanetTransitions{Year: year, Events: []PlanetTransition{}}
	err := withSky(ctx, epheDir, "Asia/Kolkata", func(s *skySession) error {
		start, end, err := s.yearBounds(year)
		if err != nil {
			return err
		}
		for index := range skyNames {
			// get evaluates the current graha for annual ingress and motion-station bracketing.
			get := func(jd float64) (SkyPlanet, error) { return s.position(jd, index) }
			left, err := get(start - 1.0/86400)
			if err != nil {
				return err
			}
			for a := start - 1.0/86400; a < end; {
				b := math.Min(a+0.25, end)
				right, err := get(b)
				if err != nil {
					return err
				}
				points := []float64{a, b}
				if (left.Speed < 0) != (right.Speed < 0) {
					lo, hi := a, b
					for hi-lo > 0.05/86400 {
						mid := (lo + hi) / 2
						p, e := get(mid)
						if e != nil {
							return e
						}
						if (p.Speed < 0) == (left.Speed < 0) {
							lo = mid
						} else {
							hi = mid
						}
					}
					station := (lo + hi) / 2
					points = []float64{a, station, b}
					if station >= start && station < end {
						p, e := get(station)
						if e != nil {
							return e
						}
						kind := "Direct motion begins"
						if right.Speed < 0 {
							kind = "Retrograde begins"
						}
						out.Events = append(out.Events, PlanetTransition{skyNames[index], kind, s.instant(station), p.Rashi, p.Rashi, right.Speed < 0})
					}
				}
				for k := 0; k < len(points)-1; k++ {
					root, e := refineSignIngress(get, points[k], points[k+1])
					if e != nil {
						return e
					}
					if root == nil {
						continue
					}
					if root.at >= start && root.at < end {
						retrograde := (root.to.RashiNumber-root.from.RashiNumber+12)%12 == 11
						out.Events = append(out.Events, PlanetTransition{skyNames[index], "Sign ingress", s.instant(root.at), root.from.Rashi, root.to.Rashi, retrograde})
					}
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
		out.Settings["search"] = "Civil year; sign ingresses and motion stations; roots to 0.05 second"
		return nil
	})
	return out, err
}

// eclipseType translates native Swiss eclipse flags into the corresponding global/local eclipse type.
func eclipseType(flags C.int32) string {
	switch {
	case flags&C.SE_ECL_ANNULAR_TOTAL != 0:
		return "Hybrid"
	case flags&C.SE_ECL_TOTAL != 0:
		return "Total"
	case flags&C.SE_ECL_ANNULAR != 0:
		return "Annular"
	case flags&C.SE_ECL_PARTIAL != 0:
		return "Partial"
	case flags&C.SE_ECL_PENUMBRAL != 0:
		return "Penumbral"
	}
	return "Unknown"
}

// CalculateEclipses finds annual global eclipses and matches independently calculated selected-place circumstances.
func CalculateEclipses(ctx context.Context, epheDir string, year int, latitude, longitude float64) (EclipseYear, error) {
	out := EclipseYear{Year: year, Events: []Eclipse{}}
	if !finite(latitude) || math.Abs(latitude) > 66 || !finite(longitude) || math.Abs(longitude) > 180 {
		return out, fmt.Errorf("unsupported coordinates")
	}
	err := withSky(ctx, epheDir, "Asia/Kolkata", func(s *skySession) error {
		start, end, err := s.yearBounds(year)
		if err != nil {
			return err
		}
		// Verify requested Swiss files/flags before eclipse routines, whose
		// return values are eclipse flags rather than ephemeris flags.
		for _, id := range []int{0, 1} {
			if _, e := s.position(start, id); e != nil {
				return e
			}
		}
		geo := [3]C.double{C.double(longitude), C.double(latitude), 0}
		for _, kind := range []string{"Solar", "Lunar"} {
			// Collect all local visible events once. A next local eclipse may
			// be outside this year; never attach it to an unrelated global event.
			type localEvent struct {
				jd   float64
				data LocalEclipse
			}
			locals := []localEvent{}
			for cursor, n := start-1, 0; n < 12; n++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				var tret [10]C.double
				var attr [20]C.double
				var msg [256]C.char
				var flags C.int32
				if kind == "Solar" {
					flags = C.swe_sol_eclipse_when_loc(C.double(cursor), C.SEFLG_SWIEPH, &geo[0], &tret[0], &attr[0], 0, &msg[0])
				} else {
					flags = C.swe_lun_eclipse_when_loc(C.double(cursor), C.SEFLG_SWIEPH, &geo[0], &tret[0], &attr[0], 0, &msg[0])
				}
				if flags < 0 || tret[0] <= C.double(cursor) || !finite(float64(tret[0])) {
					return fmt.Errorf("local eclipse search unavailable")
				}
				jd := float64(tret[0])
				if jd > end+1 {
					break
				}
				// Both local search routines can move tret[0] to a horizon
				// crossing when the geometric maximum is hidden. attr[] is
				// recomputed there, but MAX_VISIBLE still describes the original
				// geometric maximum. Report the altitude of the returned instant.
				local := LocalEclipse{Type: eclipseType(flags), Maximum: s.instant(jd), MaximumAboveHorizon: attr[6] > 0, Magnitude: float64(attr[0]), Contacts: []EclipseContact{}}
				// add records a valid local eclipse contact with its native time and horizon state.
				add := func(name string, index int, bit C.int32) {
					if tret[index] > 0 {
						above := flags&bit != 0
						local.Contacts = append(local.Contacts, EclipseContact{name, s.instant(float64(tret[index])), &above})
					}
				}
				if kind == "Solar" {
					v := float64(attr[2])
					// Swiss returns the lunar/solar disc AREA RATIO during
					// totality, which may exceed one (swecl.c, eclipse_how).
					// Our API field is the fraction of the Sun actually hidden.
					if flags&C.SE_ECL_TOTAL != 0 {
						v = 1
					}
					if !finite(v) || v < 0 || v > 1 {
						return fmt.Errorf("invalid solar obscuration")
					}
					local.Obscuration = &v
					add("First contact", 1, C.SE_ECL_1ST_VISIBLE)
					add("Central phase begins", 2, C.SE_ECL_2ND_VISIBLE)
					add("Central phase ends", 3, C.SE_ECL_3RD_VISIBLE)
					add("Last contact", 4, C.SE_ECL_4TH_VISIBLE)
					lo, hi := float64(tret[1]), float64(tret[4])
					if tret[5] > 0 {
						lo = math.Max(lo, float64(tret[5]))
					}
					if tret[6] > 0 {
						hi = math.Min(hi, float64(tret[6]))
					}
					if lo <= 0 || hi <= lo {
						return fmt.Errorf("invalid solar visibility window")
					}
					local.VisibleStart, local.VisibleEnd = s.instant(lo), s.instant(hi)
					if tret[5] > 0 {
						local.Contacts = append(local.Contacts, EclipseContact{Name: "Sunrise during eclipse", At: s.instant(float64(tret[5]))})
					}
					if tret[6] > 0 {
						local.Contacts = append(local.Contacts, EclipseContact{Name: "Sunset during eclipse", At: s.instant(float64(tret[6]))})
					}
				} else {
					add("Penumbral phase begins", 6, C.SE_ECL_PENUMBBEG_VISIBLE)
					add("Partial phase begins", 2, C.SE_ECL_PARTBEG_VISIBLE)
					add("Totality begins", 4, C.SE_ECL_TOTBEG_VISIBLE)
					add("Totality ends", 5, C.SE_ECL_TOTEND_VISIBLE)
					add("Partial phase ends", 3, C.SE_ECL_PARTEND_VISIBLE)
					add("Penumbral phase ends", 7, C.SE_ECL_PENUMBEND_VISIBLE)
					lo, hi := float64(tret[6]), float64(tret[7])
					if tret[8] > 0 {
						lo = float64(tret[8])
						local.Contacts = append(local.Contacts, EclipseContact{Name: "Moonrise during eclipse", At: s.instant(lo)})
					}
					if tret[9] > 0 {
						hi = float64(tret[9])
						local.Contacts = append(local.Contacts, EclipseContact{Name: "Moonset during eclipse", At: s.instant(hi)})
					}
					if lo <= 0 || hi <= lo {
						return fmt.Errorf("invalid lunar visibility window")
					}
					local.VisibleStart, local.VisibleEnd = s.instant(lo), s.instant(hi)
				}
				if flags&C.SE_ECL_VISIBLE != 0 {
					sort.Slice(local.Contacts, func(i, j int) bool { return local.Contacts[i].At.Before(local.Contacts[j].At) })
					locals = append(locals, localEvent{jd, local})
				}
				cursor = jd + 1
			}
			for cursor, n := start-1, 0; n < 12; n++ {
				if err := ctx.Err(); err != nil {
					return err
				}
				var tret [10]C.double
				var msg [256]C.char
				var flags C.int32
				if kind == "Solar" {
					flags = C.swe_sol_eclipse_when_glob(C.double(cursor), C.SEFLG_SWIEPH, 0, &tret[0], 0, &msg[0])
				} else {
					flags = C.swe_lun_eclipse_when(C.double(cursor), C.SEFLG_SWIEPH, 0, &tret[0], 0, &msg[0])
				}
				if flags <= 0 || tret[0] <= C.double(cursor) || !finite(float64(tret[0])) {
					return fmt.Errorf("global eclipse search unavailable")
				}
				jd := float64(tret[0])
				if jd >= end {
					break
				}
				cursor = jd + 1
				if jd < start {
					continue
				}
				e := Eclipse{Kind: kind, Type: eclipseType(flags), Maximum: s.instant(jd), Contacts: []EclipseContact{}}
				// add records a valid global eclipse contact without inventing a local visibility interval.
				add := func(name string, index int) {
					if tret[index] > 0 {
						e.Contacts = append(e.Contacts, EclipseContact{Name: name, At: s.instant(float64(tret[index]))})
					}
				}
				if kind == "Solar" {
					add("Eclipse begins worldwide", 2)
					add("Central phase begins worldwide", 4)
					add("Central phase ends worldwide", 5)
					add("Eclipse ends worldwide", 3)
				} else {
					add("Penumbral phase begins", 6)
					add("Partial phase begins", 2)
					add("Totality begins", 4)
					add("Totality ends", 5)
					add("Partial phase ends", 3)
					add("Penumbral phase ends", 7)
				}
				for _, v := range locals {
					if math.Abs(v.jd-jd) < 0.75 {
						copy := v.data
						e.Local = &copy
						e.Visible = true
						break
					}
				}
				out.Events = append(out.Events, e)
			}
		}
		sort.Slice(out.Events, func(i, j int) bool { return out.Events[i].Maximum.Before(out.Events[j].Maximum) })
		out.Settings = s.settings()
		out.Settings["eclipse_coordinates"] = "Swiss global and local eclipse searches; elevation 0 m; horizon visibility only"
		out.Settings["year_boundary"] = "Year of global maximum in Asia/Kolkata"
		out.Settings["local_maximum"] = "Maximum visible locally; may be at sunrise, sunset, moonrise or moonset"
		out.Settings["magnitude"] = "Solar: IMCCE diameter magnitude; lunar: umbral magnitude (zero for penumbral eclipses)"
		out.Settings["obscuration"] = "Fraction of solar disc hidden, 0 to 1; exactly 1 during totality"
		return nil
	})
	return out, err
}

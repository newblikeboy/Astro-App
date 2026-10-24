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
	"time"
	"unsafe"
)

// Periods use [Start, End). The first limb starts at sunrise (not its
// astronomical beginning); its End is the actual transition, even when that
// lies after the next sunrise. Subsequent limbs are never omitted/skipped.
type PanchangPeriod struct {
	Number   int       `json:"number"`
	Name     string    `json:"name"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Quality  string    `json:"quality,omitempty"`
	Cautions []string  `json:"cautions,omitempty"`
}

type Panchang struct {
	Details         *PanchangDetails  `json:"details,omitempty"`
	Date            string            `json:"date"`
	Zone            string            `json:"zone"`
	Weekday         int               `json:"weekday"`
	Sunrise         time.Time         `json:"sunrise"`
	Sunset          time.Time         `json:"sunset"`
	NextSunrise     time.Time         `json:"next_sunrise"`
	Tithi           []PanchangPeriod  `json:"tithi"`
	Nakshatra       []PanchangPeriod  `json:"nakshatra"`
	Yoga            []PanchangPeriod  `json:"yoga"`
	Karana          []PanchangPeriod  `json:"karana"`
	Timings         []PanchangPeriod  `json:"timings"`
	ChoghadiyaDay   []PanchangPeriod  `json:"choghadiya_day"`
	ChoghadiyaNight []PanchangPeriod  `json:"choghadiya_night"`
	HoraDay         []PanchangPeriod  `json:"hora_day"`
	HoraNight       []PanchangPeriod  `json:"hora_night"`
	Settings        map[string]string `json:"settings"`
}

var tithiNames = [...]string{"Pratipada", "Dwitiya", "Tritiya", "Chaturthi", "Panchami", "Shashthi", "Saptami", "Ashtami", "Navami", "Dashami", "Ekadashi", "Dwadashi", "Trayodashi", "Chaturdashi"}
var yogaNames = [...]string{"Vishkambha", "Priti", "Ayushman", "Saubhagya", "Shobhana", "Atiganda", "Sukarma", "Dhriti", "Shula", "Ganda", "Vriddhi", "Dhruva", "Vyaghata", "Harshana", "Vajra", "Siddhi", "Vyatipata", "Variyana", "Parigha", "Shiva", "Siddha", "Sadhya", "Shubha", "Shukla", "Brahma", "Indra", "Vaidhriti"}

// limbName returns the traditional label for the selected Panchang limb index.
func limbName(kind, n int) string {
	switch kind {
	case 0:
		if n == 14 {
			return "Purnima"
		}
		if n == 29 {
			return "Amavasya"
		}
		paksha := "Shukla"
		if n >= 15 {
			paksha = "Krishna"
		}
		return paksha + " " + tithiNames[n%15]
	case 1:
		return nakshatraNames[n]
	case 2:
		return yogaNames[n]
	default:
		if n == 0 {
			return "Kimstughna"
		}
		if n >= 57 {
			return [...]string{"Shakuni", "Chatushpada", "Naga"}[n-57]
		}
		return [...]string{"Bava", "Balava", "Kaulava", "Taitila", "Gara", "Vanija", "Vishti"}[(n-1)%7]
	}
}

// CalculatePanchang owns the same process-wide mutex as natal calculations.
// UTC/UT1 conversions go through Swiss, including its leap-second/delta-T model.
func CalculatePanchang(ctx context.Context, epheDir, date, zone, convention string, latitude, longitude float64) (Panchang, error) {
	var out Panchang
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return out, fmt.Errorf("invalid timezone")
	}
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil || day.Format("2006-01-02") != date || date < MinDate || date > MaxDate {
		return out, fmt.Errorf("date must be YYYY-MM-DD between %s and %s", MinDate, MaxDate)
	}
	if !finite(latitude) || math.Abs(latitude) > 66 || !finite(longitude) || math.Abs(longitude) > 180 {
		return out, fmt.Errorf("unsupported coordinates")
	}
	if convention != "upper-limb" && convention != "hindu" {
		return out, fmt.Errorf("unsupported sunrise convention")
	}
	absDir, err := filepath.Abs(epheDir)
	if err != nil {
		return out, err
	}
	for _, name := range []string{"sepl_18.se1", "semo_18.se1"} {
		info, err := os.Stat(filepath.Join(absDir, name))
		if err != nil || info.IsDir() {
			return out, fmt.Errorf("required ephemeris data unavailable")
		}
	}
	engineMu.Lock()
	defer engineMu.Unlock()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	cPath := C.CString(absDir)
	defer C.free(unsafe.Pointer(cPath))
	C.swe_set_ephe_path(cPath)
	defer C.swe_close()
	C.swe_set_sid_mode(C.SE_SIDM_LAHIRI, 0, 0)
	var message [256]C.char
	// toJD converts a civil instant to the UT1 Julian day required by Swiss calculation calls.
	toJD := func(t time.Time) (float64, error) {
		u := t.UTC()
		var jd [2]C.double
		status := C.swe_utc_to_jd(C.int(u.Year()), C.int(u.Month()), C.int(u.Day()), C.int(u.Hour()), C.int(u.Minute()), C.double(u.Second()), C.SE_GREG_CAL, &jd[0], &message[0])
		if status < 0 {
			return 0, fmt.Errorf("UTC conversion failed")
		}
		// jd[0] is TT; swe_calc_ut and swe_rise_trans require jd[1] (UT1).
		// swe_calc_ut applies delta-T internally; do not add it here.
		return float64(jd[1]), nil
	}
	// fromJD converts a native UT1 Julian day back to the requested local civil timezone.
	fromJD := func(jd float64) time.Time {
		var y, m, d, h, min C.int
		var sec C.double
		C.swe_jdut1_to_utc(C.double(jd), C.SE_GREG_CAL, &y, &m, &d, &h, &min, &sec)
		return time.Date(int(y), time.Month(m), int(d), int(h), int(min), 0, 0, time.UTC).Add(time.Duration(float64(sec) * float64(time.Second))).In(loc)
	}
	// Cache shared Sun/Moon evaluations within this calculation, never globally.
	positions := map[float64][2]float64{}
	// angles calculates the native Sun/Moon longitudes used by Panchang limb searches.
	angles := func(jd float64) ([2]float64, error) {
		if v, ok := positions[jd]; ok {
			return v, nil
		}
		var v [2]float64
		if err := ctx.Err(); err != nil {
			return v, err
		}
		for i := range v {
			var xx [6]C.double
			// Same apparent, geocentric Lahiri flags for Sun and Moon. No
			// TRUEPOS/TOPOCTR flag, manual ayanamsa subtraction or TT input.
			flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SIDEREAL)
			got := C.swe_calc_ut(C.double(jd), C.int32(i), flags, &xx[0], &message[0])
			if !swissFlagsMatch(int32(got), int32(flags)) || !finite(float64(xx[0])) {
				return v, fmt.Errorf("Swiss sidereal ephemeris calculation failed (flags=%d)", int32(got))
			}
			v[i] = float64(xx[0])
		}
		positions[jd] = v
		return v, nil
	}
	base, err := toJD(day)
	if err != nil {
		return out, err
	}
	if _, err := angles(base); err != nil {
		return out, err
	} // Reject fallback before solar searches.
	geo := [3]C.double{C.double(longitude), C.double(latitude), 0}
	// solar finds a local solar rise/set event under the selected sunrise convention.
	solar := func(after float64, event C.int32) (float64, error) {
		var jd C.double
		bits := event
		if convention == "hindu" {
			bits |= C.SE_BIT_HINDU_RISING
		}
		status := C.swe_rise_trans(C.double(after), C.SE_SUN, nil, C.SEFLG_SWIEPH, bits, &geo[0], 1013.25, 15, &jd, &message[0])
		if status < 0 || !finite(float64(jd)) {
			return 0, fmt.Errorf("sunrise or sunset unavailable for this date and location")
		}
		return float64(jd), nil
	}
	rise, err := solar(base, C.SE_CALC_RISE)
	if err != nil {
		return out, err
	}
	set, err := solar(rise, C.SE_CALC_SET)
	if err != nil {
		return out, err
	}
	next, err := solar(set, C.SE_CALC_RISE)
	if err != nil {
		return out, err
	}
	if fromJD(rise).Format("2006-01-02") != date || !(rise < set && set < next) || next-rise > 1.5 {
		return out, fmt.Errorf("invalid solar day")
	}
	var version [256]C.char
	C.swe_version(&version[0])
	out = Panchang{Date: date, Zone: zone, Weekday: int(day.Weekday()), Sunrise: fromJD(rise), Sunset: fromJD(set), NextSunrise: fromJD(next), Settings: map[string]string{
		"engine": "m4y-panchang-v2", "swiss_version": C.GoString(&version[0]), "ayanamsa": "Lahiri", "coordinates": "Geocentric apparent Sun and Moon", "ephemeris": "Swiss data files", "sunrise_convention": convention, "elevation": "0 m", "atmosphere": "1013.25 hPa, 15 C (upper-limb only)", "day_boundary": "Sunrise to next sunrise", "transition_precision": "Bisection to 0.1 second; display rounded to nearest minute",
	}}
	// value evaluates the current limb's angular quantity while retaining wraparound continuity.
	value := func(kind int, jd float64) (float64, error) {
		v, err := angles(jd)
		if err != nil {
			return 0, err
		}
		switch kind {
		case 0, 3:
			return normalizeDegrees(v[1] - v[0]), nil
		case 1:
			return v[1], nil
		default:
			return normalizeDegrees(v[1] + v[0]), nil
		}
	}
	widths := [...]float64{12, 360.0 / 27, 360.0 / 27, 6}
	counts := [...]int{30, 27, 27, 60}
	lists := []*[]PanchangPeriod{&out.Tithi, &out.Nakshatra, &out.Yoga, &out.Karana}
	for kind, width := range widths {
		start := rise
		angle, err := value(kind, start)
		if err != nil {
			return Panchang{}, err
		}
		n := int(math.Floor(angle/width)) % counts[kind]
		for steps := 0; steps < 6; steps++ {
			// Use the same guarded solver as conjunction/sign searches. In
			// particular, a target just ahead must not be skipped by an epsilon
			// shortcut, and cancellation must also interrupt bisection.
			end, err := monotoneCross(ctx, start, float64(n+1)*width, 1, 3,
				func(jd float64) (float64, error) { return value(kind, jd) })
			if err != nil {
				return Panchang{}, err
			}
			*lists[kind] = append(*lists[kind], PanchangPeriod{Number: n + 1, Name: limbName(kind, n), Start: fromJD(start), End: fromJD(end)})
			if end >= next {
				break
			}
			start, n = end, (n+1)%counts[kind]
		}
		if len(*lists[kind]) == 0 || (*lists[kind])[len(*lists[kind])-1].End.Before(out.NextSunrise) {
			return Panchang{}, fmt.Errorf("incomplete Panchang day")
		}
	}
	populateDailyPeriods(&out)
	if err := populatePanchangDetails(ctx, &out, latitude, longitude, base, rise, set, next, angles, fromJD); err != nil {
		return Panchang{}, err
	}
	return out, nil
}

// fractionalTime interpolates an exact fraction of a time interval without civil-clock rounding.
func fractionalTime(start, end time.Time, numerator, denominator int) time.Time {
	return start.Add(time.Duration(int64(end.Sub(start)) * int64(numerator) / int64(denominator)))
}

// populateDailyPeriods derives weekday-based daily timings, Choghadiya and Hora from sunrise and sunset.
func populateDailyPeriods(p *Panchang) {
	w := p.Weekday
	for _, rule := range []struct {
		name  string
		parts [7]int
	}{
		{"Rahu Kaal", [7]int{8, 2, 7, 5, 6, 4, 3}},
		{"Yamaganda", [7]int{5, 4, 3, 2, 1, 7, 6}},
		{"Gulika", [7]int{7, 6, 5, 4, 3, 2, 1}},
	} {
		n := rule.parts[w]
		p.Timings = append(p.Timings, PanchangPeriod{Name: rule.name, Number: n, Start: fractionalTime(p.Sunrise, p.Sunset, n-1, 8), End: fractionalTime(p.Sunrise, p.Sunset, n, 8), Quality: "Caution"})
	}
	abhijit := PanchangPeriod{Name: "Abhijit", Number: 8, Start: fractionalTime(p.Sunrise, p.Sunset, 7, 15), End: fractionalTime(p.Sunrise, p.Sunset, 8, 15), Quality: "Favourable"}
	if w == 3 {
		abhijit.Quality = "Caution"
		abhijit.Cautions = []string{"Wednesday exception"}
	}
	for _, v := range p.Timings {
		if v.Start.Before(abhijit.End) && v.End.After(abhijit.Start) {
			abhijit.Cautions = append(abhijit.Cautions, v.Name)
		}
	}
	p.Timings = append(p.Timings, abhijit)
	dayNames := [...]string{"Udveg", "Char", "Labh", "Amrit", "Kaal", "Shubh", "Rog"}
	nightNames := [...]string{"Shubh", "Amrit", "Char", "Rog", "Kaal", "Labh", "Udveg"}
	dayStart := [...]int{0, 3, 6, 2, 5, 1, 4}
	nightStart := [...]int{0, 2, 4, 6, 1, 3, 5}
	// Traditional weekday cautions, one-based slots; no automatic event Muhurat.
	vaar := [...]int{4, 7, 2, 5, 8, 3, 6}
	ratri := [...]int{6, 4, 2, 7, 5, 3, 1}
	for half := 0; half < 2; half++ {
		start, end := p.Sunrise, p.Sunset
		if half == 1 {
			start, end = p.Sunset, p.NextSunrise
		}
		for i := 0; i < 8; i++ {
			name := dayNames[(dayStart[w]+i)%7]
			if half == 1 {
				name = nightNames[(nightStart[w]+i)%7]
			}
			quality := "Favourable"
			if name == "Char" {
				quality = "Neutral"
			}
			if name == "Udveg" || name == "Kaal" || name == "Rog" {
				quality = "Caution"
			}
			v := PanchangPeriod{Name: name, Number: i + 1, Start: fractionalTime(start, end, i, 8), End: fractionalTime(start, end, i+1, 8), Quality: quality}
			if half == 0 {
				if i+1 == vaar[w] {
					v.Cautions = append(v.Cautions, "Vaar Vela")
				}
				if name == "Kaal" {
					v.Cautions = append(v.Cautions, "Kaal Vela")
				}
				for _, t := range p.Timings[:3] {
					if v.Start.Before(t.End) && v.End.After(t.Start) {
						v.Cautions = append(v.Cautions, t.Name)
					}
				}
				p.ChoghadiyaDay = append(p.ChoghadiyaDay, v)
			} else {
				if (i+1-ratri[w]+7)%7 == 0 {
					v.Cautions = append(v.Cautions, "Kaal Ratri")
				}
				p.ChoghadiyaNight = append(p.ChoghadiyaNight, v)
			}
		}
		// Chaldean order; the first daytime Hora is the weekday lord.
		planets := [...]string{"Sun", "Venus", "Mercury", "Moon", "Saturn", "Jupiter", "Mars"}
		for i := 0; i < 12; i++ {
			v := PanchangPeriod{Name: planets[(dayStart[w]+half*12+i)%7], Number: half*12 + i + 1, Start: fractionalTime(start, end, i, 12), End: fractionalTime(start, end, i+1, 12)}
			if half == 0 {
				p.HoraDay = append(p.HoraDay, v)
			} else {
				p.HoraNight = append(p.HoraNight, v)
			}
		}
	}
}

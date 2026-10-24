package swisseph

/*
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"fmt"
	"math"
	"time"
)

type BirthChakra struct {
	LagnaLord     string `json:"lagna_lord"`
	RashiLord     string `json:"rashi_lord"`
	NakshatraLord string `json:"nakshatra_lord"`
	Varna         string `json:"varna"`
	Vashya        string `json:"vashya"`
	Yoni          string `json:"yoni"`
	Gana          string `json:"gana"`
	Nadi          string `json:"nadi"`
}
type BirthDetails struct {
	At           time.Time         `json:"at"`
	CivilWeekday int               `json:"civil_weekday"`
	HinduWeekday int               `json:"hindu_weekday"`
	Paksha       string            `json:"paksha"`
	Limbs        []PanchangPeriod  `json:"limbs"`
	Sunrise      time.Time         `json:"sunrise"`
	Sunset       time.Time         `json:"sunset"`
	DayStart     time.Time         `json:"day_start"`
	DayEnd       time.Time         `json:"day_end"`
	Ayanamsa     float64           `json:"ayanamsa"`
	JulianDayUT1 float64           `json:"julian_day_ut1"`
	Chakra       BirthChakra       `json:"chakra"`
	Settings     map[string]string `json:"settings"`
}
type BhavaHouse struct {
	House  int     `json:"house"`
	Start  float64 `json:"start_longitude"`
	Middle float64 `json:"middle_longitude"`
	End    float64 `json:"end_longitude"`
}
type BhavaPlanet struct {
	Name        string `json:"name"`
	RashiNumber int    `json:"rashi_number"`
	NatalHouse  int    `json:"natal_house"`
	House       int    `json:"house"`
}
type Chalit struct {
	Method    string        `json:"method"`
	Houses    []BhavaHouse  `json:"houses"`
	Planets   []BhavaPlanet `json:"planets"`
	LagnaSign int           `json:"lagna_sign"`
}

// arcContains tests membership in a half-open zodiac arc, including arcs that wrap through zero degrees.
func arcContains(start, end, longitude float64) bool {
	return normalizeDegrees(longitude-start) < normalizeDegrees(end-start)
}

// CalculateBirthDetails uses the actual birth instant for all four limbs.
// Sunrise is only the Hindu weekday boundary, never a substitute birth time.
func CalculateBirthDetails(ctx context.Context, epheDir, zone string, at time.Time, chart Kundali) (BirthDetails, Chalit, error) {
	var out BirthDetails
	var chalit Chalit
	if !supportedCalculationInstant(at) || !finite(chart.Latitude) || math.Abs(chart.Latitude) > 66 || !finite(chart.Longitude) || math.Abs(chart.Longitude) > 180 {
		return out, chalit, fmt.Errorf("unsupported birth details")
	}
	err := withSky(ctx, epheDir, zone, func(s *skySession) error {
		jd, err := s.jd(at)
		if err != nil {
			return err
		}
		sun, err := s.position(jd, 0)
		if err != nil {
			return err
		}
		moon, err := s.position(jd, 1)
		if err != nil {
			return err
		}
		cat, err := matchingMoon(chart)
		if err != nil {
			return err
		}
		out = BirthDetails{At: at.In(s.loc), CivilWeekday: int(at.In(s.loc).Weekday()), Paksha: "Shukla", Limbs: []PanchangPeriod{}, JulianDayUT1: jd, Ayanamsa: float64(C.swe_get_ayanamsa_ut(C.double(jd))),
			Chakra:   BirthChakra{matchingLords[matchingSignLords[chart.Lagna.RashiNumber-1]], matchingLords[cat.lord], dashaLords[(moon.NakshatraNumber-1)%9], matchingVarnas[cat.varna], matchingVashyas[cat.vashya], matchingYonis[cat.yoni], matchingGanas[cat.gana], matchingNadis[cat.nadi]},
			Settings: map[string]string{"method": "m4y-birth-panchang-v1", "ayanamsa": "Lahiri", "limbs": "At actual birth instant; full astronomical [start,end) boundaries", "sunrise": "Hindu solar centre; no refraction; zero elevation", "weekday": "Hindu weekday starts at preceding sunrise", "chakra": "Same versioned Moon categories as Ashtakoota matching"}}
		elongation := normalizeDegrees(moon.Longitude - sun.Longitude)
		if elongation >= 180 {
			out.Paksha = "Krishna"
		}
		widths := []float64{12, 360.0 / 27, 360.0 / 27, 6}
		values := []float64{elongation, moon.Longitude, normalizeDegrees(moon.Longitude + sun.Longitude), elongation}
		for kind, width := range widths {
			index := int(math.Floor(values[kind] / width))
			// value evaluates the selected birth Panchang angle for native interval-boundary searches.
			value := func(point float64) (float64, error) {
				m, e := s.position(point, 1)
				if e != nil {
					return 0, e
				}
				if kind == 1 {
					return m.Longitude, nil
				}
				a, e := s.position(point, 0)
				if e != nil {
					return 0, e
				}
				if kind == 2 {
					return normalizeDegrees(m.Longitude + a.Longitude), nil
				}
				return normalizeDegrees(m.Longitude - a.Longitude), nil
			}
			start := jd
			if values[kind] != float64(index)*width {
				start, err = monotoneCross(ctx, jd, float64(index)*width, -1, 4, value)
				if err != nil {
					return err
				}
			}
			end, e := monotoneCross(ctx, jd, float64(index+1)*width, 1, 4, value)
			if e != nil {
				return e
			}
			out.Limbs = append(out.Limbs, PanchangPeriod{Number: index + 1, Name: limbName(kind, index), Start: s.instant(start), End: s.instant(end)})
		}
		geo := [3]C.double{C.double(chart.Longitude), C.double(chart.Latitude), 0}
		// solar finds a geometric Hindu sunrise/sunset event at the recorded birthplace.
		solar := func(after float64, bits C.int32) (float64, error) {
			var result C.double
			var message [256]C.char
			status := C.swe_rise_trans(C.double(after), C.SE_SUN, nil, C.SEFLG_SWIEPH, bits|C.SE_BIT_HINDU_RISING, &geo[0], 0, 0, &result, &message[0])
			if status < 0 || !finite(float64(result)) {
				return 0, fmt.Errorf("birth sunrise unavailable")
			}
			return float64(result), nil
		}
		local := at.In(s.loc)
		midnight, e := s.jd(time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc))
		if e != nil {
			return e
		}
		rise, e := solar(midnight, C.SE_CALC_RISE)
		if e != nil {
			return e
		}
		set, e := solar(rise, C.SE_CALC_SET)
		if e != nil {
			return e
		}
		dayStart := rise
		if jd < rise {
			dayStart, e = solar(midnight-1, C.SE_CALC_RISE)
			if e != nil {
				return e
			}
		}
		dayEnd, e := solar(dayStart+0.2, C.SE_CALC_RISE)
		if e != nil {
			return e
		}
		if dayStart > jd || dayEnd <= jd || s.instant(rise).Format("2006-01-02") != local.Format("2006-01-02") {
			return fmt.Errorf("invalid birth solar day")
		}
		out.Sunrise, out.Sunset, out.DayStart, out.DayEnd = s.instant(rise), s.instant(set), s.instant(dayStart), s.instant(dayEnd)
		out.HinduWeekday = int(out.DayStart.Weekday())
		// Swiss S returns Sripati house STARTS. Porphyry O returns the
		// corresponding bhava middles. Do not midpoint S a second time.
		var starts, middles [13]C.double
		var axes [10]C.double
		if C.swe_houses_ex(C.double(jd), C.SEFLG_SIDEREAL, C.double(chart.Latitude), C.double(chart.Longitude), C.int('S'), &starts[0], &axes[0]) < 0 {
			return fmt.Errorf("Sripati houses unavailable")
		}
		if C.swe_houses_ex(C.double(jd), C.SEFLG_SIDEREAL, C.double(chart.Latitude), C.double(chart.Longitude), C.int('O'), &middles[0], &axes[0]) < 0 {
			return fmt.Errorf("bhava middles unavailable")
		}
		chalit = Chalit{Method: "Swiss Sripati (S) boundaries; Porphyry (O) bhava middles; Lahiri; ecliptic-longitude assignment", Houses: []BhavaHouse{}, Planets: []BhavaPlanet{}, LagnaSign: chart.Lagna.RashiNumber}
		for i := 1; i <= 12; i++ {
			start, end := float64(starts[i]), float64(starts[i%12+1])
			middle := float64(middles[i])
			if !finite(start) || !finite(end) || !finite(middle) || !arcContains(start, end, middle) {
				return fmt.Errorf("invalid Sripati interval")
			}
			chalit.Houses = append(chalit.Houses, BhavaHouse{i, start, middle, end})
		}
		for _, p := range chart.Planets {
			house := 0
			for _, h := range chalit.Houses {
				if arcContains(h.Start, h.End, p.Longitude) {
					house = h.House
					break
				}
			}
			if house == 0 {
				return fmt.Errorf("planet outside Chalit houses")
			}
			chalit.Planets = append(chalit.Planets, BhavaPlanet{p.Name, p.RashiNumber, p.House, house})
		}
		return nil
	})
	return out, chalit, err
}

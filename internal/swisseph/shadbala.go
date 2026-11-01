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

// ShadbalaMethod declares the BPHS/Santhanam six-component convention, using
// modern Swiss orbital means for Cheshta rather than a rounded historical table.
const ShadbalaMethod = "m4y-shadbala-bphs-swiss-means-v1"

type BalaPart struct {
	Code    string  `json:"code"`
	Virupas float64 `json:"virupas"`
}
type ShadbalaPlanet struct {
	Planet           string       `json:"planet"`
	Sthana           float64      `json:"sthana"`
	Dig              float64      `json:"dig"`
	Kala             float64      `json:"kala"`
	Cheshta          float64      `json:"cheshta"`
	Naisargika       float64      `json:"naisargika"`
	Drik             float64      `json:"drik"`
	Total            float64      `json:"total_virupas"`
	Rupas            float64      `json:"rupas"`
	Required         float64      `json:"required_virupas"`
	Ratio            float64      `json:"ratio"`
	MeetsRequirement bool         `json:"meets_requirement"`
	SthanaParts      []BalaPart   `json:"sthana_parts"`
	KalaParts        []BalaPart   `json:"kala_parts"`
	Aspects          []BalaAspect `json:"aspects"`
}
type BalaAspect struct {
	From         string  `json:"from"`
	Virupas      float64 `json:"virupas"`
	Contribution float64 `json:"contribution"`
}
type PlanetWar struct {
	First      string  `json:"first"`
	Second     string  `json:"second"`
	Separation float64 `json:"separation"`
	Winner     string  `json:"winner,omitempty"`
	Transfer   float64 `json:"transfer"`
	Status     string  `json:"status"`
}
type Shadbala struct {
	Method   string            `json:"method"`
	At       time.Time         `json:"at"`
	Planets  []ShadbalaPlanet  `json:"planets"`
	Wars     []PlanetWar       `json:"wars"`
	Settings map[string]string `json:"settings"`
}

type balaAstronomy struct {
	Latitude     [7]float64
	Declination  [7]float64
	Mean         [7]float64
	ApparentHour float64
}

// shortestAngle measures the unsigned circular distance without display rounding.
func shortestAngle(a, b float64) float64 { return math.Abs(math.Remainder(a-b, 360)) }

// moolatrikona tests the classical degree intervals in D1; it is independent of
// Guruji's functional role and does not overwrite exalted or own-sign flags.
func moolatrikona(index int, p ZodiacPosition) bool {
	signs := [7]int{5, 2, 1, 6, 9, 7, 11}
	starts := [7]float64{0, 3, 0, 15, 0, 0, 0}
	ends := [7]float64{20, 30, 12, 20, 10, 15, 20}
	return p.RashiNumber == signs[index] && p.Degree >= starts[index] && p.Degree < ends[index]
}

// fullAspectVirupas implements BPHS chapter 26's degree-based directed aspect
// intervals, including the exact seventh and the special Mars/Jupiter/Saturn arcs.
func fullAspectVirupas(planet string, angle float64) float64 {
	a := normalizeDegrees(angle)
	v := 0.0
	switch {
	case a < 30:
		v = 0
	case a < 60:
		v = (a - 30) / 2
	case a < 90:
		v = a - 45
	case a < 120:
		v = (120-a)/2 + 30
	case a < 150:
		v = 30 - (a - 120)
	case a < 180:
		v = (a - 150) * 2
	case a < 300:
		v = (300 - a) / 2
	}
	// Chapter 26's special rules override, rather than add to, the base arc.
	switch planet {
	case "Mars":
		switch {
		case a >= 60 && a < 90:
			v = 15 + (a-60)*1.5
		case a >= 90 && a < 120:
			v = 60 - (a - 90)
		case a >= 180 && a < 210:
			v = 60
		case a >= 210 && a < 240:
			v = 60 - (a - 210)
		}
	case "Jupiter":
		switch {
		case a >= 90 && a < 120:
			v = 45 + (a-90)/2
		case a >= 120 && a < 150:
			v = 60 - (a - 120)
		case a >= 210 && a < 240:
			v = 45 + (a-210)/2
		case a >= 240 && a < 270:
			v = 60 - (a - 240)
		}
	case "Saturn":
		switch {
		case a >= 30 && a < 60:
			v = 2 * (a - 30)
		case a >= 60 && a < 90:
			v = 60 - (a-60)/2
		case a >= 240 && a < 270:
			v = 30 + (a - 240)
		case a >= 270 && a < 300:
			v = 2 * (300 - a)
		}
	}
	return math.Max(0, math.Min(60, v))
}

// naturalBenefics classifies benefic/malefic for strength arithmetic only; it
// does not replace the authoritative ascendant-role table or give node strengths.
func naturalBenefics(planets [7]Graha, nodes []Graha) [7]bool {
	benefic := [7]bool{false, normalizeDegrees(planets[1].Longitude-planets[0].Longitude) < 180, false, true, true, true, false}
	for j, p := range planets {
		if j != 3 && !benefic[j] && p.RashiNumber == planets[3].RashiNumber {
			benefic[3] = false
		}
	}
	for _, p := range nodes {
		if (p.Name == "Rahu" || p.Name == "Ketu") && p.RashiNumber == planets[3].RashiNumber {
			benefic[3] = false
		}
	}
	return benefic
}

// shadbalaAstronomy obtains extra astronomical inputs under the shared native
// lock. Orbital means never replace the apparent physical natal positions.
func shadbalaAstronomy(s *skySession, jd float64, longitude float64) (balaAstronomy, error) {
	var out balaAstronomy
	ids := [7]C.int{C.SE_SUN, C.SE_MOON, C.SE_MARS, C.SE_MERCURY, C.SE_JUPITER, C.SE_VENUS, C.SE_SATURN}
	var msg [256]C.char
	for i, id := range ids {
		var xx [6]C.double
		flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SPEED)
		ret := C.swe_calc_ut(C.double(jd), id, flags, &xx[0], &msg[0])
		if !swissFlagsMatch(int32(ret), int32(flags)) || !finite(float64(xx[1])) {
			return out, fmt.Errorf("Swiss strength latitude unavailable")
		}
		out.Latitude[i] = float64(xx[1])
		flags |= C.SEFLG_EQUATORIAL
		ret = C.swe_calc_ut(C.double(jd), id, flags, &xx[0], &msg[0])
		if !swissFlagsMatch(int32(ret), int32(flags)) || !finite(float64(xx[1])) {
			return out, fmt.Errorf("Swiss strength declination unavailable")
		}
		out.Declination[i] = float64(xx[1])
	}
	// Swiss elements are referred to the mean ecliptic J2000. The fixed
	// Lahiri origin at J2000 keeps their circular means in a single frame.
	ayanJ2000 := float64(C.swe_get_ayanamsa(C.double(2451545.0)))
	var elements [50]C.double
	tt := jd + float64(C.swe_deltat_ex(C.double(jd), C.SEFLG_SWIEPH, &msg[0]))
	for i, id := range ids {
		if i == 1 {
			continue
		}
		if i == 0 {
			id = C.SE_EARTH
		}
		var check [6]C.double
		flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_HELCTR | C.SEFLG_J2000 | C.SEFLG_SPEED)
		returned := C.swe_calc(C.double(tt), id, flags, &check[0], &msg[0])
		if !swissFlagsMatch(int32(returned), int32(flags)) {
			return out, fmt.Errorf("Swiss orbital input unavailable")
		}
		ret := C.swe_get_orbital_elements(C.double(tt), id, C.SEFLG_SWIEPH|C.SEFLG_HELCTR|C.SEFLG_J2000, &elements[0], &msg[0])
		if ret < 0 || !finite(float64(elements[9])) {
			return out, fmt.Errorf("Swiss mean orbital longitude unavailable")
		}
		out.Mean[i] = normalizeDegrees(float64(elements[9]) - ayanJ2000)
		if i == 0 {
			out.Mean[i] = normalizeDegrees(out.Mean[i] + 180)
		}
	}
	var equation C.double
	if C.swe_time_equ(C.double(jd), &equation, &msg[0]) < 0 || !finite(float64(equation)) {
		return out, fmt.Errorf("apparent solar time unavailable")
	}
	out.ApparentHour = normalizeDegrees((jd+.5+longitude/360+float64(equation))*360) / 15
	return out, nil
}

// CalculateShadbala calculates all six strength groups and their subcomponents.
// The solar Ayana and lunar Paksha substitutions are counted once in Kala and
// once in Cheshta, never doubled in Kala and then counted a third time.
func CalculateShadbala(ctx context.Context, epheDir, zone string, at time.Time, chart Kundali, birth BirthDetails, chalit Chalit) (Shadbala, error) {
	out := Shadbala{Method: ShadbalaMethod, At: at.UTC(), Planets: []ShadbalaPlanet{}, Wars: []PlanetWar{}, Settings: map[string]string{
		"units":        "Virupas; 60 virupas = 1 rupa; unrounded totals",
		"source":       "BPHS chapters 3,26,27; Santhanam volume 1 pp.263-286",
		"saptavargaja": "D1/2/3/7/9/12/30; natal compound friendship; 45/30/20/15/10/4/2; D1 degree-based Moolatrikona",
		"cheshta":      "BPHS circular mean/true midpoint; Swiss J2000 orbital means minus fixed Lahiri J2000 origin; inner-planet mean/Sun interchange",
		"luminaries":   "Sun raw Ayana and Moon phase Paksha each counted once in Kala and once in Cheshta",
		"war":          "Classical five planets within 1 degree; greater ecliptic latitude wins; equal latitude remains unresolved; simultaneous transfers use pre-war totals",
		"calendar":     "Sunrise day; abbreviated Ahargana 65142 on 1983-12-31; 360-day year, 30-day month; equal 24 sunrise-day Horas",
	}}
	if !supportedCalculationInstant(at) || !birth.At.Equal(at) || len(chart.Planets) != 9 || !finite(chart.Lagna.Longitude) || chart.Lagna.Longitude < 0 || chart.Lagna.Longitude >= 360 || !finite(chart.Latitude) || !finite(chart.Longitude) || len(chalit.Houses) != 12 || !birth.DayEnd.After(birth.DayStart) || at.Before(birth.DayStart) || !at.Before(birth.DayEnd) {
		return out, fmt.Errorf("complete matching birth details required for Shadbala")
	}
	for i, h := range chalit.Houses {
		if h.House != i+1 || !finite(h.Middle) || h.Middle < 0 || h.Middle >= 360 {
			return out, fmt.Errorf("invalid strength axes")
		}
	}
	positions := map[string]Graha{}
	for _, p := range chart.Planets {
		if _, ok := positions[p.Name]; ok || !finite(p.Longitude) || p.Longitude < 0 || p.Longitude >= 360 {
			return out, fmt.Errorf("invalid strength planet")
		}
		p.ZodiacPosition = zodiacPosition(p.Longitude)
		p.House = relativeSign(p.RashiNumber, zodiacPosition(chart.Lagna.Longitude).RashiNumber)
		positions[p.Name] = p
	}
	for _, name := range skyNames {
		if _, ok := positions[name]; !ok {
			return out, fmt.Errorf("missing strength planet %s", name)
		}
	}
	if chart.Latitude < -90 || chart.Latitude > 90 || chart.Longitude < -180 || chart.Longitude > 180 {
		return out, fmt.Errorf("invalid strength coordinates")
	}
	chartInstant, parseErr := time.Parse(time.RFC3339, chart.UTC)
	if parseErr != nil || !chartInstant.Equal(at) {
		return out, fmt.Errorf("strength chart instant differs from birth")
	}
	var planets [7]Graha
	for i, name := range matchingLords {
		p, ok := positions[name]
		if !ok {
			return out, fmt.Errorf("missing strength %s", name)
		}
		planets[i] = p
	}
	var sky balaAstronomy
	daySet := birth.Sunset
	err := withSky(ctx, epheDir, zone, func(s *skySession) error {
		jd, e := s.jd(at)
		if e != nil {
			return e
		}
		sky, e = shadbalaAstronomy(s, jd, chart.Longitude)
		if e != nil {
			return e
		}
		if at.Before(birth.Sunrise) {
			start, e := s.jd(birth.DayStart)
			if e != nil {
				return e
			}
			geo := [3]C.double{C.double(chart.Longitude), C.double(chart.Latitude), 0}
			var sunset C.double
			var msg [256]C.char
			if C.swe_rise_trans(C.double(start), C.SE_SUN, nil, C.SEFLG_SWIEPH, C.SE_CALC_SET|C.SE_BIT_HINDU_RISING, &geo[0], 0, 0, &sunset, &msg[0]) < 0 {
				return fmt.Errorf("preceding birth sunset unavailable")
			}
			daySet = s.instant(float64(sunset))
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	if !daySet.After(birth.DayStart) || !birth.DayEnd.After(daySet) {
		return out, fmt.Errorf("invalid strength day/night boundaries")
	}
	benefic := naturalBenefics(planets, chart.Planets)
	deep := [7]float64{10, 33, 298, 165, 95, 357, 200}
	required := [7]float64{390, 360, 300, 420, 390, 330, 300}
	natural := [7]float64{60, 360.0 / 7, 120.0 / 7, 180.0 / 7, 240.0 / 7, 300.0 / 7, 60.0 / 7}
	phase := shortestAngle(planets[1].Longitude, planets[0].Longitude) / 3
	// Calendar arithmetic uses civil dates of the preceding local sunrise.
	d := birth.DayStart
	day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
	ahargana := int64(day.Sub(time.Date(1983, 12, 31, 0, 0, 0, 0, time.UTC))/(24*time.Hour)) + 65142
	weekLords := [7]int{0, 1, 2, 3, 4, 5, 6}
	yearLord := weekLords[((ahargana/360*3+1)+6)%7]
	monthLord := weekLords[((ahargana/30*2+1)+6)%7]
	dayLord := weekLords[(ahargana+6)%7]
	// Weekday order above is Sun/Moon/Mars/Mercury/Jupiter/Venus/Saturn.
	horaSequence := [7]int{6, 4, 2, 0, 5, 3, 1}
	firstHora := 0
	for i, v := range horaSequence {
		if v == dayLord {
			firstHora = i
		}
	}
	hora := min(23, int(float64(at.Sub(birth.DayStart))/float64(birth.DayEnd.Sub(birth.DayStart))*24))
	horaLord := horaSequence[(firstHora+hora)%7]
	for i, p := range planets {
		r := ShadbalaPlanet{Planet: p.Name, Required: required[i], Naisargika: natural[i], SthanaParts: []BalaPart{}, KalaParts: []BalaPart{}, Aspects: []BalaAspect{}}
		uchcha := (180 - shortestAngle(p.Longitude, deep[i])) / 3
		saptavarga := 0.0
		for _, factor := range []int{1, 2, 3, 7, 9, 12, 30} {
			sign, e := vargaSign(p.Longitude, factor)
			if e != nil {
				return out, e
			}
			owner := matchingSignLords[sign-1]
			points := 30.0
			if factor == 1 && moolatrikona(i, p.ZodiacPosition) {
				points = 45
			} else if owner != i {
				temporary := -1
				distance := relativeSign(planets[owner].RashiNumber, p.RashiNumber)
				if distance == 2 || distance == 3 || distance == 4 || distance == 10 || distance == 11 || distance == 12 {
					temporary = 1
				}
				points = []float64{2, 4, 10, 15, 20}[naturalRelations[i][owner]+temporary+2]
			}
			saptavarga += points
		}
		d9, _ := vargaSign(p.Longitude, 9)
		oja := 0.0
		for _, sign := range []int{p.RashiNumber, d9} {
			if ((i == 1 || i == 5) && sign%2 == 0) || ((i != 1 && i != 5) && sign%2 == 1) {
				oja += 15
			}
		}
		kendradi := []float64{60, 30, 15}[(p.House-1)%3]
		gender := 0
		if i == 1 || i == 5 {
			gender = 1
		} else if i == 3 || i == 6 {
			gender = 2
		}
		drekkana := 0.0
		if int(p.Degree/10) == gender {
			drekkana = 15
		}
		r.SthanaParts = []BalaPart{{"uchcha", uchcha}, {"saptavargaja", saptavarga}, {"ojayugma", oja}, {"kendradi", kendradi}, {"drekkana", drekkana}}
		for _, part := range r.SthanaParts {
			r.Sthana += part.Virupas
		}
		powerless := []int{4, 10, 4, 7, 7, 10, 1}[i]
		r.Dig = shortestAngle(p.Longitude, chalit.Houses[powerless-1].Middle) / 3
		dayStrength := (12 - math.Abs(sky.ApparentHour-12)) * 5
		natha := dayStrength
		if i == 1 || i == 2 || i == 6 {
			natha = 60 - dayStrength
		} else if i == 3 {
			natha = 60
		}
		paksha := phase
		if i != 1 && !benefic[i] {
			paksha = 60 - phase
		}
		tribhaga := 0.0
		segmentLord := -1
		if at.Before(daySet) {
			part := min(2, int(float64(at.Sub(birth.DayStart))/float64(daySet.Sub(birth.DayStart))*3))
			segmentLord = []int{3, 0, 6}[part]
		} else {
			start, end := daySet, birth.DayEnd
			if end.After(start) {
				part := min(2, int(float64(at.Sub(start))/float64(end.Sub(start))*3))
				segmentLord = []int{1, 5, 2}[part]
			}
		}
		if i == 4 || i == segmentLord {
			tribhaga = 60
		}
		year, month, weekday, hour := 0.0, 0.0, 0.0, 0.0
		if i == yearLord {
			year = 15
		}
		if i == monthLord {
			month = 30
		}
		if i == dayLord {
			weekday = 45
		}
		if i == horaLord {
			hour = 60
		}
		decl := sky.Declination[i]
		if i == 1 || i == 6 {
			decl = -decl
		} else if i == 3 {
			decl = math.Abs(decl)
		}
		ayana := math.Max(0, math.Min(60, (23.45+decl)*60/46.9))
		r.KalaParts = []BalaPart{{"nathonnatha", natha}, {"paksha", paksha}, {"tribhaga", tribhaga}, {"abda", year}, {"masa", month}, {"vara", weekday}, {"hora", hour}, {"ayana", ayana}, {"yuddha", 0}}
		for _, part := range r.KalaParts {
			r.Kala += part.Virupas
		}
		if i == 0 {
			r.Cheshta = ayana
		} else if i == 1 {
			r.Cheshta = phase
		} else {
			mean, apogee := sky.Mean[i], sky.Mean[0]
			if i == 3 || i == 5 {
				mean, apogee = sky.Mean[0], sky.Mean[i]
			}
			midpoint := normalizeDegrees(mean + math.Remainder(p.Longitude-mean, 360)/2)
			r.Cheshta = shortestAngle(apogee, midpoint) / 3
		}
		for j, other := range planets {
			if i == j {
				continue
			}
			aspect := fullAspectVirupas(other.Name, p.Longitude-other.Longitude)
			contribution := -aspect / 4
			if benefic[j] {
				contribution = aspect / 4
			}
			if j == 3 || j == 4 {
				contribution += aspect
			}
			r.Drik += contribution
			r.Aspects = append(r.Aspects, BalaAspect{other.Name, aspect, contribution})
		}
		r.Total = r.Sthana + r.Dig + r.Kala + r.Cheshta + r.Naisargika + r.Drik
		out.Planets = append(out.Planets, r)
	}
	// War transfers conserve the sum of all seven totals; they are computed
	// simultaneously from pre-war totals so input order cannot change the result.
	var totals [7]float64
	for i, p := range out.Planets {
		totals[i] = p.Total
	}
	wars, adjustment := strengthWarTransfers(planets, sky.Latitude, totals)
	out.Wars = wars
	for i := range out.Planets {
		r := &out.Planets[i]
		r.KalaParts[len(r.KalaParts)-1].Virupas = adjustment[i]
		r.Kala += adjustment[i]
		r.Total += adjustment[i]
		r.Rupas = r.Total / 60
		r.Ratio = r.Total / r.Required
		r.MeetsRequirement = r.Total >= r.Required
		if !finite(r.Total) || !finite(r.Ratio) {
			return Shadbala{}, fmt.Errorf("invalid Shadbala result")
		}
	}
	return out, nil
}

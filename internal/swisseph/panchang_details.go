package swisseph

/*
#include "swephexp.h"
*/
import "C"

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// All added astronomy runs inside CalculatePanchang's existing engine mutex.
// Ritual rules are versioned separately from the ephemeris, never scraped.
type PanchangCalendar struct {
	Amanta        string    `json:"amanta"`
	Purnimanta    string    `json:"purnimanta"`
	MonthNumber   int       `json:"month_number"`
	Adhika        bool      `json:"adhika"`
	SkippedMonth  string    `json:"skipped_month,omitempty"`
	Paksha        string    `json:"paksha"`
	Vikram        int       `json:"vikram"`
	Shaka         int       `json:"shaka"`
	LunationStart time.Time `json:"lunation_start"`
	LunationEnd   time.Time `json:"lunation_end"`
}
type PanchangBalam struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	MoonSign  int       `json:"moon_sign"`
	Nakshatra int       `json:"nakshatra"`
	Chandra   []int     `json:"chandra"`
	Tara      []int     `json:"tara"`
}
type PanchangDirection struct {
	Name  string    `json:"name"`
	Value string    `json:"value"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}
type PanchangDetails struct {
	Moonrise      *time.Time          `json:"moonrise"`
	Moonset       *time.Time          `json:"moonset"`
	DaySeconds    int                 `json:"day_seconds"`
	NightSeconds  int                 `json:"night_seconds"`
	Midday        time.Time           `json:"midday"`
	Sun           ZodiacPosition      `json:"sun"`
	Moon          ZodiacPosition      `json:"moon"`
	MoonSigns     []PanchangPeriod    `json:"moon_signs"`
	SunSigns      []PanchangPeriod    `json:"sun_signs"`
	SunNakshatras []PanchangPeriod    `json:"sun_nakshatras"`
	Padas         []PanchangPeriod    `json:"padas"`
	Lagna         []PanchangPeriod    `json:"lagna"`
	Panchaka      []PanchangPeriod    `json:"panchaka"`
	Muhurats      []PanchangPeriod    `json:"muhurats"`
	Special       []PanchangPeriod    `json:"special"`
	Balam         []PanchangBalam     `json:"balam"`
	Directions    []PanchangDirection `json:"directions"`
	Calendar      PanchangCalendar    `json:"calendar"`
	TropicalRitu  string              `json:"tropical_ritu"`
	LunarRitu     string              `json:"lunar_ritu"`
	TropicalAyana string              `json:"tropical_ayana"`
	SiderealAyana string              `json:"sidereal_ayana"`
	Ayanamsa      float64             `json:"ayanamsa"`
	JulianDayUT1  float64             `json:"julian_day_ut1"`
}

var lunarMonths = [...]string{"Chaitra", "Vaishakha", "Jyeshtha", "Ashadha", "Shravana", "Bhadrapada", "Ashwina", "Kartika", "Margashirsha", "Pausha", "Magha", "Phalguna"}

// monotoneCross finds the first target crossing strictly after/before at.
// An exactly equal initial angle selects the next full cycle; a merely nearby
// target is not skipped. The longitude must increase with JD, moving less than
// 180 degrees per six-hour sample. direction chooses the direction in time,
// not the body's direction of motion. This is not a retrograde transit solver.
// Sampled reverse motion is rejected; unsampled reversals cannot be ruled out.
func monotoneCross(ctx context.Context, at, target float64, direction int, maxDays float64, value func(float64) (float64, error)) (float64, error) {
	if direction != 1 && direction != -1 {
		return 0, fmt.Errorf("invalid crossing direction")
	}
	if !finite(at) || !finite(target) || !finite(maxDays) || maxDays <= 0 || value == nil {
		return 0, fmt.Errorf("invalid longitude search inputs")
	}
	// evaluate validates a search sample and normalizes the callback's longitude with cancellation checks.
	evaluate := func(jd float64) (float64, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if !finite(jd) {
			return 0, fmt.Errorf("invalid longitude evaluation time")
		}
		v, err := value(jd)
		if err != nil {
			return 0, err
		}
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		if !finite(v) {
			return 0, fmt.Errorf("non-finite longitude")
		}
		return normalizeDegrees(v), nil
	}
	a, err := evaluate(at)
	if err != nil {
		return 0, err
	}
	distance := normalizeDegrees(float64(direction) * (normalizeDegrees(target) - a))
	if distance == 0 {
		distance = 360
	}
	const angularTolerance = 1e-9
	// progress measures directed angular travel without discarding precision at an exact boundary.
	progress := func(from, to float64) float64 {
		// Both angles are already normalized. Avoid adding/subtracting 180
		// here: that can erase a low bit at a target on the search limit.
		delta := to - from
		if delta >= 180 {
			delta -= 360
		} else if delta < -180 {
			delta += 360
		}
		return float64(direction) * delta
	}
	lo := 0.0
	travel, previous := 0.0, a
	for lo < maxDays {
		hi := math.Min(lo+0.25, maxDays)
		v, e := evaluate(at + float64(direction)*hi)
		if e != nil {
			return 0, e
		}
		increment := progress(previous, v)
		if increment < -angularTolerance {
			return 0, fmt.Errorf("non-monotonic longitude at JD %.9f", at+float64(direction)*hi)
		}
		increment = math.Max(0, increment)
		if travel+increment >= distance {
			left, right := lo, hi
			leftProgress, rightProgress := 0.0, increment
			for right-left > 0.1/86400 {
				mid := (left + right) / 2
				v, e = evaluate(at + float64(direction)*mid)
				if e != nil {
					return 0, e
				}
				moved := progress(previous, v)
				if moved < leftProgress-angularTolerance || moved > rightProgress+angularTolerance {
					return 0, fmt.Errorf("non-monotonic longitude inside crossing bracket")
				}
				moved = math.Max(leftProgress, math.Min(rightProgress, moved))
				if travel+moved < distance {
					left, leftProgress = mid, moved
				} else {
					right, rightProgress = mid, moved
				}
			}
			return at + float64(direction)*right, nil
		}
		travel += increment
		previous = v
		lo = hi
	}
	return 0, fmt.Errorf("longitude boundary not bracketed")
}

// populatePanchangDetails adds lunar calendar facts, rise/set events, ascendants and extended timing windows.
func populatePanchangDetails(ctx context.Context, p *Panchang, lat, lon, base, rise, set, next float64, angles func(float64) ([2]float64, error), instant func(float64) time.Time) error {
	d := &PanchangDetails{Muhurats: []PanchangPeriod{}, Special: []PanchangPeriod{}, Directions: []PanchangDirection{}}
	p.Details = d
	// value creates a Sun or Moon longitude callback for the extended Panchang searches.
	value := func(body int) func(float64) (float64, error) {
		return func(jd float64) (float64, error) { v, e := angles(jd); return v[body], e }
	}
	// phase returns the normalized Moon-minus-Sun angle used for lunar-phase boundaries.
	phase := func(jd float64) (float64, error) { v, e := angles(jd); return normalizeDegrees(v[1] - v[0]), e }
	atRise, err := angles(rise)
	if err != nil {
		return err
	}
	d.Sun, d.Moon = zodiacPosition(atRise[0]), zodiacPosition(atRise[1])
	d.DaySeconds = int(math.Round(p.Sunset.Sub(p.Sunrise).Seconds()))
	d.NightSeconds = int(math.Round(p.NextSunrise.Sub(p.Sunset).Seconds()))
	d.Midday = p.Sunrise.Add(p.Sunset.Sub(p.Sunrise) / 2)
	d.Ayanamsa = float64(C.swe_get_ayanamsa_ut(C.double(rise)))
	d.JulianDayUT1 = rise
	geo := [3]C.double{C.double(lon), C.double(lat), 0}
	// event finds a local Sun/Moon rise or set event and preserves an unavailable-event result.
	event := func(after float64, body, event int) (*time.Time, error) {
		var jd C.double
		var msg [256]C.char
		bits := C.int32(event)
		if body == int(C.SE_SUN) && p.Settings["sunrise_convention"] == "hindu" {
			bits |= C.SE_BIT_HINDU_RISING
		}
		result := C.swe_rise_trans(C.double(after), C.int32(body), nil, C.SEFLG_SWIEPH, bits, &geo[0], 1013.25, 15, &jd, &msg[0])
		if result == -2 {
			return nil, nil
		}
		if result < 0 || !finite(float64(jd)) {
			return nil, fmt.Errorf("rise/set search failed")
		}
		t := instant(float64(jd))
		return &t, nil
	}
	// Dainik Panchang is a sunrise day, not a civil midnight day. A search
	// from midnight can repeat yesterday's post-midnight moonset, or attach a
	// pre-sunrise moonrise to the wrong Panchang day. Retain only events inside
	// the same half-open interval as the other daily calculations.
	moonEvent := func(kind int) (*time.Time, error) {
		t, e := event(rise, int(C.SE_MOON), kind)
		if e != nil || t == nil {
			return t, e
		}
		if t.Before(p.Sunrise) || !t.Before(p.NextSunrise) {
			return nil, nil
		}
		return t, nil
	}
	d.Moonrise, err = moonEvent(int(C.SE_CALC_RISE))
	if err != nil {
		return err
	}
	d.Moonset, err = moonEvent(int(C.SE_CALC_SET))
	if err != nil {
		return err
	}
	// Previous sunset is needed for pre-sunrise Brahma/Pratah windows.
	previousSet, err := event(base-1, int(C.SE_SUN), int(C.SE_CALC_SET))
	if err != nil || previousSet == nil {
		return fmt.Errorf("previous sunset unavailable")
	}
	if !previousSet.Before(p.Sunrise) {
		return fmt.Errorf("invalid previous night")
	}
	// add adds a positive-duration Muhurat window to the extended daily details.
	add := func(name string, start, end time.Time, quality string) {
		if end.After(start) {
			d.Muhurats = append(d.Muhurats, PanchangPeriod{Name: name, Start: start, End: end, Quality: quality})
		}
	}
	night := p.NextSunrise.Sub(p.Sunset)
	previousNight := p.Sunrise.Sub(*previousSet)
	add("Brahma Muhurta", p.Sunrise.Add(-previousNight*2/15), p.Sunrise.Add(-previousNight/15), "Favourable")
	add("Pratah Sandhya", p.Sunrise.Add(-previousNight/10), p.Sunrise, "Traditional")
	add("Vijaya Muhurta", fractionalTime(p.Sunrise, p.Sunset, 10, 15), fractionalTime(p.Sunrise, p.Sunset, 11, 15), "Favourable")
	add("Sayahna Sandhya", p.Sunset, p.Sunset.Add(night/10), "Traditional")
	add("Nishita Muhurta", fractionalTime(p.Sunset, p.NextSunrise, 7, 15), fractionalTime(p.Sunset, p.NextSunrise, 8, 15), "Traditional")
	// Durmuhurta: one-based daylight fifteenths; Tuesday's second is at night.
	dur := [7][]int{{14}, {9, 12}, {4}, {8}, {6, 12}, {4, 9}, {1, 2}}
	for _, n := range dur[p.Weekday] {
		add("Dur Muhurtam", fractionalTime(p.Sunrise, p.Sunset, n-1, 15), fractionalTime(p.Sunrise, p.Sunset, n, 15), "Caution")
	}
	if p.Weekday == 2 {
		add("Dur Muhurtam", fractionalTime(p.Sunset, p.NextSunrise, 6, 15), fractionalTime(p.Sunset, p.NextSunrise, 7, 15), "Caution")
	}
	// Segments retain the exact end crossing, even if it falls after next sunrise.
	segments := func(width float64, fn func(float64) (float64, error), name func(int) string, clip bool) ([]PanchangPeriod, error) {
		rows := []PanchangPeriod{}
		start := rise
		for n := 0; n < 32; n++ {
			v, e := fn(start)
			if e != nil {
				return nil, e
			}
			index := int(math.Floor(v / width))
			end, e := monotoneCross(ctx, start, float64(index+1)*width, 1, 35, fn)
			if e != nil {
				return nil, e
			}
			end = math.Max(end, start+0.05/86400)
			shownEnd := end
			if clip {
				shownEnd = math.Min(end, next)
			}
			rows = append(rows, PanchangPeriod{Number: index + 1, Name: name(index), Start: instant(start), End: instant(shownEnd)})
			if end >= next {
				return rows, nil
			}
			start = end
		}
		return nil, fmt.Errorf("too many daily segments")
	}
	d.MoonSigns, err = segments(30, value(1), func(i int) string { return rashiNames[i%12] }, false)
	if err != nil {
		return err
	}
	d.SunSigns, err = segments(30, value(0), func(i int) string { return rashiNames[i%12] }, false)
	if err != nil {
		return err
	}
	d.Padas, err = segments(360.0/108, value(1), func(i int) string { return nakshatraNames[(i/4)%27] }, false)
	if err != nil {
		return err
	}
	d.SunNakshatras, err = segments(360.0/27, value(0), func(i int) string { return nakshatraNames[i%27] }, false)
	if err != nil {
		return err
	}
	// asc calculates the local sidereal ascendant for the daily Lagna interval search.
	asc := func(jd float64) (float64, error) {
		var cusps [13]C.double
		var axes [10]C.double
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		ret := C.swe_houses_ex(C.double(jd), C.SEFLG_SIDEREAL, C.double(lat), C.double(lon), C.int('W'), &cusps[0], &axes[0])
		if ret < 0 || !finite(float64(axes[0])) {
			return 0, fmt.Errorf("ascendant unavailable")
		}
		return float64(axes[0]), nil
	}
	// Ascendant needs much smaller brackets than solar/lunar longitudes.
	d.Lagna, err = dailyAscendants(ctx, rise, next, asc, instant)
	if err != nil {
		return err
	}
	// Astronomical lunar month: label by solar sign at the preceding conjunction.
	// Lunations include their opening conjunction. monotoneCross is purposely
	// exclusive, so at an exactly equal conjunction the current month begins
	// at sunrise itself rather than at the preceding lunation.
	prevNM := rise
	if normalizeDegrees(atRise[1]-atRise[0]) != 0 {
		prevNM, err = monotoneCross(ctx, rise, 0, -1, 35, phase)
		if err != nil {
			return err
		}
	}
	nextNM, err := monotoneCross(ctx, rise, 0, 1, 35, phase)
	if err != nil {
		return err
	}
	a, err := angles(prevNM)
	if err != nil {
		return err
	}
	b, err := angles(nextNM)
	if err != nil {
		return err
	}
	firstSign, lastSign := int(a[0]/30), int(b[0]/30)
	month := (firstSign + 1) % 12
	ingressCount := (lastSign - firstSign + 12) % 12
	lunarYear := p.Sunrise.Year()
	// Before Chaitra's March/April new year, the winter months still belong
	// to the preceding year. Margashirsha can continue into January (e.g.
	// 1 January 2024), so it must be included, not just Pausha onward.
	if p.Sunrise.Month() <= 4 && month >= 8 {
		lunarYear--
	}
	pm := month
	paksha := "Shukla"
	if p.Tithi[0].Number > 15 {
		// Intercalary/contracted lunations retain their combined Amanta label.
		// Only an ordinary one-ingress lunation advances during Krishna.
		if ingressCount == 1 {
			pm = (month + 1) % 12
		}
		paksha = "Krishna"
	}
	d.Calendar = PanchangCalendar{Amanta: lunarMonths[month], Purnimanta: lunarMonths[pm], MonthNumber: month + 1, Adhika: ingressCount == 0, Paksha: paksha, Vikram: lunarYear + 57, Shaka: lunarYear - 78, LunationStart: instant(prevNM), LunationEnd: instant(nextNM)}
	if ingressCount > 1 {
		d.Calendar.SkippedMonth = lunarMonths[(month+1)%12]
	}
	var xx [6]C.double
	var msg [256]C.char
	returned := C.swe_calc_ut(C.double(rise), C.SE_SUN, C.SEFLG_SWIEPH, &xx[0], &msg[0])
	if !swissFlagsMatch(int32(returned), int32(C.SEFLG_SWIEPH)) || !finite(float64(xx[0])) {
		return fmt.Errorf("tropical Swiss Sun unavailable (flags=%d)", int32(returned))
	}
	tropical := normalizeDegrees(float64(xx[0]))
	ritus := [6]string{"Vasanta", "Grishma", "Varsha", "Sharad", "Hemanta", "Shishira"}
	d.TropicalRitu = tropicalRitu(tropical)
	d.LunarRitu = ritus[month/2]
	d.TropicalAyana = "Dakshinayana"
	if tropical >= 270 || tropical < 90 {
		d.TropicalAyana = "Uttarayana"
	}
	d.SiderealAyana = "Dakshinayana"
	if first := d.Sun.RashiNumber; first >= 10 || first <= 3 {
		d.SiderealAyana = "Uttarayana"
	}
	// Nakshatra windows use the actual full nakshatra duration, not a fixed day.
	varjyam := [27][]int{{50}, {24}, {30}, {40}, {14}, {21}, {30}, {20}, {32}, {30}, {20}, {18}, {21}, {20}, {14}, {14}, {10}, {14}, {20, 56}, {24}, {20}, {10}, {10}, {18}, {16}, {24}, {30}}
	amrit := [27]int{42, 48, 54, 52, 38, 35, 54, 44, 56, 54, 44, 42, 45, 44, 38, 38, 34, 38, 44, 48, 44, 34, 34, 42, 40, 48, 54}
	// Include the previous nakshatra: its late window can overlap sunrise.
	for _, n := range p.Nakshatra {
		endJD := rise + n.End.Sub(p.Sunrise).Seconds()/86400
		startJD, e := monotoneCross(ctx, endJD-1.0/86400, float64(n.Number-1)*360/27, -1, 3, value(1))
		if e != nil {
			return e
		}
		start, end := instant(startJD), n.End
		for _, offset := range varjyam[n.Number-1] {
			a := fractionalTime(start, end, offset, 60)
			b := fractionalTime(start, end, offset+4, 60)
			if a.Before(p.NextSunrise) && b.After(p.Sunrise) {
				add("Varjyam", a, b, "Caution")
			}
		}
		offset := amrit[n.Number-1]
		a := fractionalTime(start, end, offset, 60)
		b := fractionalTime(start, end, offset+4, 60)
		if a.Before(p.NextSunrise) && b.After(p.Sunrise) {
			add("Amrit Kalam", a, b, "Favourable")
		}
	}
	populatePanchangRules(p)
	sort.SliceStable(d.Muhurats, func(i, j int) bool { return d.Muhurats[i].Start.Before(d.Muhurats[j].Start) })
	p.Settings["details_rules"] = "m4y-panchang-rules-v2"
	p.Settings["moon_event_window"] = "Sunrise inclusive to next sunrise exclusive; no event in the interval is null"
	p.Settings["moon_horizon"] = "Topocentric upper limb; refraction at 1013.25 hPa/15 C; sea-level horizon"
	p.Settings["ritu"] = "Sunrise snapshot; tropical Vasanta begins at 330 degrees, each season spans 60 degrees; lunar seasons use Amanta month pairs"
	p.Settings["calendar"] = "Astronomical new-moon lunations; Amanta/Purnimanta; Chaitra-start Shaka/Vikram years"
	p.Settings["ritual_rules"] = "Sunrise-based weekdays; variable day/night fifteenths; 60-part actual nakshatra duration. Tradition-dependent, not an event-specific Muhurat."
	return nil
}

// Tropical seasons are centred on the equinoxes, not begun at them. Vasanta
// spans tropical Pisces and Aries (330 to 30 degrees); Grishma Taurus/Gemini,
// Varsha Cancer/Leo, Sharad Virgo/Libra, Hemanta Scorpio/Sagittarius and
// Shishira Capricorn/Aquarius. This differs deliberately from lunar Ritu.
func tropicalRitu(longitude float64) string {
	ritus := [6]string{"Vasanta", "Grishma", "Varsha", "Sharad", "Hemanta", "Shishira"}
	return ritus[int(normalizeDegrees(longitude+30)/60)]
}

// dailyAscendants finds consecutive Lagna sign intervals across the complete sunrise-to-sunrise day.
func dailyAscendants(ctx context.Context, rise, next float64, value func(float64) (float64, error), instant func(float64) time.Time) ([]PanchangPeriod, error) {
	rows := []PanchangPeriod{}
	start := rise
	for len(rows) < 16 {
		v, e := value(start)
		if e != nil {
			return nil, e
		}
		index := int(v / 30)
		lo, hi := start, start
		for hi < next+0.2 {
			hi += 1.0 / 144
			val, e := value(hi)
			if e != nil {
				return nil, e
			}
			if int(val/30) != index {
				break
			}
			lo = hi
		}
		for hi-lo > 0.1/86400 {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
			mid := (lo + hi) / 2
			val, e := value(mid)
			if e != nil {
				return nil, e
			}
			if int(val/30) == index {
				lo = mid
			} else {
				hi = mid
			}
		}
		rows = append(rows, PanchangPeriod{Number: index + 1, Name: rashiNames[index], Start: instant(start), End: instant(math.Min(hi, next))})
		if hi >= next {
			return rows, nil
		}
		start = hi
	}
	return nil, fmt.Errorf("incomplete ascendant day")
}

// populatePanchangRules splits the day at every relevant change before evaluating traditional timing/direction rules.
func populatePanchangRules(p *Panchang) {
	d := p.Details
	// Every rule is split on every relevant transition; no noon-only snapshots.
	bounds := []time.Time{p.Sunrise, p.NextSunrise}
	for _, list := range [][]PanchangPeriod{p.Tithi, p.Nakshatra, p.Karana, d.MoonSigns, d.SunNakshatras, d.Lagna} {
		for _, v := range list {
			if v.End.After(p.Sunrise) && v.End.Before(p.NextSunrise) {
				bounds = append(bounds, v.End)
			}
		}
	}
	sort.Slice(bounds, func(i, j int) bool { return bounds[i].Before(bounds[j]) })
	unique := bounds[:0]
	for _, b := range bounds {
		if len(unique) == 0 || b.Sub(unique[len(unique)-1]) > time.Millisecond {
			unique = append(unique, b)
		}
	}
	bounds = unique
	// get selects the interval containing an instant using start-inclusive/end-exclusive boundaries.
	get := func(rows []PanchangPeriod, t time.Time) PanchangPeriod {
		for _, v := range rows {
			if !t.Before(v.Start) && t.Before(v.End) {
				return v
			}
		}
		return rows[len(rows)-1]
	}
	// add appends a traditional timing result, merging directly adjacent windows with the same name.
	add := func(name string, a, b time.Time, quality string) {
		for i := range d.Special {
			v := &d.Special[i]
			if v.Name == name && v.End.Equal(a) {
				v.End = b
				return
			}
		}
		d.Special = append(d.Special, PanchangPeriod{Name: name, Start: a, End: b, Quality: quality})
	}
	// direction appends a directional rule, joining adjacent intervals with the same name and value.
	direction := func(name, v string, a, b time.Time) {
		for i := range d.Directions {
			r := &d.Directions[i]
			if r.Name == name && r.Value == v && r.End.Equal(a) {
				r.End = b
				return
			}
		}
		d.Directions = append(d.Directions, PanchangDirection{name, v, a, b})
	}
	sarva := [7][]int{{1, 8, 12, 13, 19, 21, 26}, {4, 5, 8, 17, 22}, {1, 3, 9, 26}, {3, 4, 5, 13, 17}, {1, 7, 8, 17, 27}, {1, 7, 17, 22, 27}, {4, 15, 22}}
	amrita := [7]int{13, 5, 1, 17, 8, 27, 4}
	// has checks membership in a traditional-rule index list.
	has := func(xs []int, n int) bool {
		for _, x := range xs {
			if x == n {
				return true
			}
		}
		return false
	}
	for i := 0; i < len(bounds)-1; i++ {
		a, b := bounds[i], bounds[i+1]
		mid := a.Add(b.Sub(a) / 2)
		ti, na, ka, mo, su, la := get(p.Tithi, mid), get(p.Nakshatra, mid), get(p.Karana, mid), get(d.MoonSigns, mid), get(d.SunNakshatras, mid), get(d.Lagna, mid)
		if has(sarva[p.Weekday], na.Number) {
			add("Sarvartha Siddhi Yoga", a, b, "Favourable")
		}
		if amrita[p.Weekday] == na.Number {
			add("Amrita Siddhi Yoga", a, b, "Favourable")
		}
		if has([]int{4, 6, 9, 10, 13, 20}, (na.Number-su.Number+27)%27+1) {
			add("Ravi Yoga", a, b, "Favourable")
		}
		if has([]int{0, 2, 6}, p.Weekday) && has([]int{2, 7, 12}, (ti.Number-1)%15+1) {
			if has([]int{5, 14, 23}, na.Number) {
				add("Dwipushkar Yoga", a, b, "Traditional")
			}
			if has([]int{3, 7, 12, 16, 21, 25}, na.Number) {
				add("Tripushkar Yoga", a, b, "Traditional")
			}
		}
		if p.Weekday == 0 && na.Number == 8 {
			add("Ravi Pushya Yoga", a, b, "Favourable")
		}
		if p.Weekday == 4 && na.Number == 8 {
			add("Guru Pushya Yoga", a, b, "Favourable")
		}
		if ka.Name == "Vishti" {
			add("Bhadra", a, b, "Caution")
			v := "Patala"
			if has([]int{1, 2, 3, 8}, mo.Number) {
				v = "Swarga"
			}
			if has([]int{4, 5, 11, 12}, mo.Number) {
				v = "Prithvi"
			}
			direction("Bhadra Vasa", v, a, b)
		}
		if has([]int{1, 9, 10, 18, 19, 27}, na.Number) {
			add("Ganda Moola", a, b, "Caution")
		}
		// Panchaka starts in the latter half of Dhanishtha (Aquarius), not Capricorn.
		if mo.Number == 11 || mo.Number == 12 {
			add("Panchaka", a, b, "Caution")
		}
		if mo.Number == 8 {
			add("Vinchudo", a, b, "Caution")
		}
		n := (ti.Number + na.Number + p.Weekday + 1 + la.Number) % 9
		names := map[int]string{1: "Mrityu Panchaka", 2: "Agni Panchaka", 4: "Raja Panchaka", 6: "Chora Panchaka", 8: "Roga Panchaka"}
		name, ok := names[n]
		quality := "Caution"
		if !ok {
			name = "Panchaka Rahita"
			quality = "Favourable"
		}
		d.Panchaka = append(d.Panchaka, PanchangPeriod{Name: name, Start: a, End: b, Quality: quality})
		bal := PanchangBalam{Start: a, End: b, MoonSign: mo.Number, Nakshatra: na.Number, Chandra: []int{}, Tara: []int{}}
		for birth := 1; birth <= 12; birth++ {
			if has([]int{1, 3, 6, 7, 10, 11}, (mo.Number-birth+12)%12+1) {
				bal.Chandra = append(bal.Chandra, birth)
			}
		}
		for birth := 1; birth <= 27; birth++ {
			tara := ((na.Number-birth+27)%27)%9 + 1
			if has([]int{2, 4, 6, 8, 9}, tara) {
				bal.Tara = append(bal.Tara, birth)
			}
		}
		if len(d.Balam) > 0 && d.Balam[len(d.Balam)-1].MoonSign == bal.MoonSign && d.Balam[len(d.Balam)-1].Nakshatra == bal.Nakshatra {
			d.Balam[len(d.Balam)-1].End = b
		} else {
			d.Balam = append(d.Balam, bal)
		}
		direction("Disha Shool", [7]string{"West", "East", "North", "North", "South", "West", "East"}[p.Weekday], a, b)
		direction("Rahu Vasa", [7]string{"North", "North-West", "West", "South-West", "South", "South-East", "East"}[p.Weekday], a, b)
		direction("Chandra Vasa", [4]string{"East", "South", "West", "North"}[(mo.Number-1)%4], a, b)
		direction("Agni Vasa", [4]string{"Prithvi", "Akasha", "Patala", "Prithvi"}[(ti.Number+p.Weekday+2)%4], a, b)
		direction("Shiva Vasa", [7]string{"Shmashana", "Kailash", "Gauri", "Nandi", "Sabha", "Bhojana", "Krida"}[(ti.Number*2+5)%7], a, b)
	}
	// Coalesce adjacent identical Panchaka results after unrelated transitions.
	rows := d.Panchaka
	d.Panchaka = nil
	for _, r := range rows {
		n := len(d.Panchaka)
		if n > 0 && d.Panchaka[n-1].Name == r.Name && d.Panchaka[n-1].End.Equal(r.Start) {
			d.Panchaka[n-1].End = r.End
		} else {
			d.Panchaka = append(d.Panchaka, r)
		}
	}
}

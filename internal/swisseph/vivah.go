package swisseph

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// VivahWindow is a preliminary marriage interval with constant limbs and Lagna
// throughout [Start, End). Birth-chart compatibility and full Lagna Shuddhi are separate.
type VivahWindow struct {
	Start     time.Time `json:"start"`
	End       time.Time `json:"end"`
	Tithi     string    `json:"tithi"`
	Nakshatra string    `json:"nakshatra"`
	Yoga      string    `json:"yoga"`
	Karana    string    `json:"karana"`
	Lagna     string    `json:"lagna"`
}

type VivahDay struct {
	Date    string        `json:"date"`
	Windows []VivahWindow `json:"windows"`
	Reasons []string      `json:"reasons"`
}

type VivahMonth struct {
	Month    string            `json:"month"`
	Zone     string            `json:"zone"`
	Days     []VivahDay        `json:"days"`
	Settings map[string]string `json:"settings"`
}

// vivahLunation resolves the Amanta month at an instant just after a new moon.
// It uses the same conjunction/solar-ingress convention as daily Panchang.
func vivahLunation(s *skySession, at time.Time) (PanchangCalendar, error) {
	jd, err := s.jd(at)
	if err != nil {
		return PanchangCalendar{}, err
	}
	// phase supplies the unrounded geocentric Sun-Moon elongation for conjunction searches.
	phase := func(jd float64) (float64, error) {
		moon, err := s.position(jd, 1)
		if err != nil {
			return 0, err
		}
		sun, err := s.position(jd, 0)
		return normalizeDegrees(moon.Longitude - sun.Longitude), err
	}
	prev, err := monotoneCross(s.ctx, jd, 0, -1, 35, phase)
	if err != nil {
		return PanchangCalendar{}, err
	}
	next, err := monotoneCross(s.ctx, jd, 0, 1, 35, phase)
	if err != nil {
		return PanchangCalendar{}, err
	}
	a, err := s.position(prev, 0)
	if err != nil {
		return PanchangCalendar{}, err
	}
	b, err := s.position(next, 0)
	if err != nil {
		return PanchangCalendar{}, err
	}
	first, last := int(a.Longitude/30), int(b.Longitude/30)
	count := (last - first + 12) % 12
	month := (first + 1) % 12
	c := PanchangCalendar{MonthNumber: month + 1, Amanta: lunarMonths[month], Adhika: count == 0, LunationStart: s.instant(prev), LunationEnd: s.instant(next)}
	if count > 1 {
		c.SkippedMonth = lunarMonths[(month+1)%12]
	}
	return c, nil
}

// vivahCombustions finds Jupiter/Venus angular-combustion intervals over the
// requested solar days, including an interval already active at the first sunrise.
// These are the app's shared longitude limits, not heliacal visibility or Asta buffers.
func vivahCombustions(s *skySession, start, end time.Time) ([]PanchangPeriod, error) {
	a, err := s.jd(start)
	if err != nil {
		return nil, err
	}
	b, err := s.jd(end)
	if err != nil {
		return nil, err
	}
	out := []PanchangPeriod{}
	for _, index := range []int{4, 5} {
		// get samples the shared angular and motion-dependent combustion rule.
		get := func(jd float64) (combustionSample, error) {
			planet, err := s.position(jd, index)
			if err != nil {
				return combustionSample{}, err
			}
			sun, err := s.position(jd, 0)
			limit, _ := combustionLimit(planet.Name, planet.Retrograde)
			return combustionSample{planet, sun.Speed, math.Abs(math.Remainder(planet.Longitude-sun.Longitude, 360)), limit}, err
		}
		left, err := get(a)
		if err != nil {
			return nil, err
		}
		active, opened := left.active(), start
		name := "Guru combustion"
		if index == 5 {
			name = "Shukra combustion"
		}
		for x := a; x < b; {
			y := math.Min(x+.25, b)
			right, err := get(y)
			if err != nil {
				return nil, err
			}
			points, err := combustionTurningPoints(get, x, y, left, right)
			if err != nil {
				return nil, err
			}
			for i := 0; i < len(points)-1; i++ {
				l, err := get(points[i])
				if err != nil {
					return nil, err
				}
				r, err := get(points[i+1])
				if err != nil {
					return nil, err
				}
				if l.active() == r.active() {
					continue
				}
				root, after, _, err := refineCombustionChange(get, points[i], points[i+1])
				if err != nil {
					return nil, err
				}
				boundary := s.instant(root)
				if active {
					out = append(out, PanchangPeriod{Name: name, Start: opened, End: boundary})
				}
				active = after.active()
				opened = boundary
			}
			x, left = y, right
		}
		if active {
			out = append(out, PanchangPeriod{Name: name, Start: opened, End: end})
		}
	}
	return out, nil
}

// vivahPeriodAt returns the limb active at an instant using half-open intervals.
func vivahPeriodAt(periods []PanchangPeriod, at time.Time) PanchangPeriod {
	for _, p := range periods {
		if !at.Before(p.Start) && at.Before(p.End) {
			return p
		}
	}
	return PanchangPeriod{}
}

// vivahReasons evaluates a declared conservative marriage shortlist rule set.
// Chaturmas spans Ashadha Shukla Ekadashi through Kartika Shukla Ekadashi;
// Full Lagna Shuddhi, Tara/Chandra bala, regional exceptions and Asta buffers are separate.
func vivahReasons(tithi, nakshatra, yoga, karana, sun, pada PanchangPeriod, calendar PanchangCalendar, blocked []PanchangPeriod, at time.Time) []string {
	reasons := []string{}
	switch nakshatra.Number {
	case 1, 4, 5, 10, 12, 13, 14, 15, 17, 19, 21, 22, 23, 26:
	default:
		reasons = append(reasons, "Marriage Nakshatra unavailable")
	}
	// Retain the existing Magha/Mula first-quarter safeguards pending the Vedha review.
	if pada.Number == 37 || pada.Number == 73 {
		reasons = append(reasons, "Excluded Nakshatra quarter")
	}
	if tithi.Number == 0 || tithi.Number == 30 || tithi.Number%15 == 4 || tithi.Number%15 == 9 || tithi.Number%15 == 14 {
		reasons = append(reasons, "Excluded Tithi")
	}
	switch yoga.Number {
	case 0, 1, 6, 9, 10, 13, 15, 17, 19, 27:
		reasons = append(reasons, "Excluded Yoga")
	}
	if karana.Number == 0 || karana.Name == "Vishti" || karana.Name == "Shakuni" || karana.Name == "Chatushpada" || karana.Name == "Naga" {
		reasons = append(reasons, "Excluded Karana")
	}
	switch sun.Number {
	case 1, 2, 3, 8, 10, 11:
	default:
		reasons = append(reasons, "Excluded solar month")
	}
	if calendar.Adhika {
		reasons = append(reasons, "Adhika Masa")
	}
	if calendar.SkippedMonth != "" {
		reasons = append(reasons, "Kshaya Masa")
	}
	m, n := calendar.MonthNumber, tithi.Number
	if (m == 4 && n >= 11) || (m >= 5 && m <= 7) || (m == 8 && n <= 11) {
		reasons = append(reasons, "Chaturmas")
	}
	if m == 12 && n >= 8 && n <= 15 {
		reasons = append(reasons, "Holashtak")
	}
	if m == 0 || pada.Number == 0 {
		reasons = append(reasons, "Incomplete calendar data")
	}
	for _, period := range blocked {
		if !at.Before(period.Start) && at.Before(period.End) {
			reasons = append(reasons, period.Name)
		}
	}
	return reasons
}

// vivahElectionReasons applies the unambiguous Saturday exclusion and six marriage
// Lagnas in Guruji's supplied notes. Tuesday awaits clarification; missing Lagna fails closed.
func vivahElectionReasons(lagna PanchangPeriod, weekday int) []string {
	reasons := []string{}
	if weekday == int(time.Saturday) {
		reasons = append(reasons, "Saturday excluded for marriage")
	}
	if lagna.Number < 1 || lagna.Number > 12 {
		reasons = append(reasons, "Incomplete Lagna data")
		return reasons
	}
	switch lagna.Number {
	case 2, 3, 6, 7, 9, 12:
	default:
		reasons = append(reasons, "Excluded marriage Lagna")
	}
	return reasons
}

// vivahDay splits a solar day at every applied rule boundary, retains constant-
// evidence intervals of at least five minutes, and records exclusion reasons.
func vivahDay(p Panchang, nextCalendar PanchangCalendar, combustion []PanchangPeriod) VivahDay {
	out := VivahDay{Date: p.Date, Windows: []VivahWindow{}, Reasons: []string{}}
	blocked := append([]PanchangPeriod{}, combustion...)
	for _, period := range p.Timings {
		if period.Name == "Rahu Kaal" {
			blocked = append(blocked, period)
		}
	}
	for _, period := range p.Details.Muhurats {
		if period.Name == "Dur Muhurtam" || period.Name == "Varjyam" {
			blocked = append(blocked, period)
		}
	}
	boundaries := []time.Time{p.Sunrise, p.NextSunrise}
	for _, list := range [][]PanchangPeriod{p.Tithi, p.Nakshatra, p.Yoga, p.Karana, p.Details.SunSigns, p.Details.Padas, p.Details.Lagna, blocked} {
		for _, period := range list {
			for _, at := range []time.Time{period.Start, period.End} {
				if at.After(p.Sunrise) && at.Before(p.NextSunrise) {
					boundaries = append(boundaries, at)
				}
			}
		}
	}
	if at := p.Details.Calendar.LunationEnd; at.After(p.Sunrise) && at.Before(p.NextSunrise) {
		boundaries = append(boundaries, at)
	}
	sort.Slice(boundaries, func(i, j int) bool { return boundaries[i].Before(boundaries[j]) })
	seen := map[string]bool{}
	for i := 0; i < len(boundaries)-1; i++ {
		start, end := boundaries[i], boundaries[i+1]
		if !end.After(start) {
			continue
		}
		at := start.Add(end.Sub(start) / 2)
		tithi, star := vivahPeriodAt(p.Tithi, at), vivahPeriodAt(p.Nakshatra, at)
		yoga, karana := vivahPeriodAt(p.Yoga, at), vivahPeriodAt(p.Karana, at)
		calendar := p.Details.Calendar
		if !at.Before(calendar.LunationEnd) {
			calendar = nextCalendar
		}
		reasons := vivahReasons(tithi, star, yoga, karana, vivahPeriodAt(p.Details.SunSigns, at), vivahPeriodAt(p.Details.Padas, at), calendar, blocked, at)
		lagna := vivahPeriodAt(p.Details.Lagna, at)
		reasons = append(reasons, vivahElectionReasons(lagna, p.Weekday)...)
		for _, reason := range reasons {
			seen[reason] = true
		}
		if len(reasons) != 0 {
			continue
		}
		// Reuse canonical English zodiac labels for the existing bilingual interface.
		window := VivahWindow{Start: start, End: end, Tithi: tithi.Name, Nakshatra: star.Name, Yoga: yoga.Name, Karana: karana.Name, Lagna: matchingSigns[lagna.Number-1]}
		// Adjacent acceptable quarters retain one interval when displayed evidence is identical.
		if n := len(out.Windows); n > 0 {
			last := &out.Windows[n-1]
			if last.End.Equal(start) && last.Tithi == window.Tithi && last.Nakshatra == window.Nakshatra && last.Yoga == window.Yoga && last.Karana == window.Karana && last.Lagna == window.Lagna {
				last.End = end
				continue
			}
		}
		out.Windows = append(out.Windows, window)
	}
	kept := []VivahWindow{}
	for _, window := range out.Windows {
		if window.End.Sub(window.Start) >= 5*time.Minute {
			kept = append(kept, window)
		} else {
			seen["Window shorter than five minutes"] = true
		}
	}
	out.Windows = kept
	if len(kept) == 0 {
		for reason := range seen {
			out.Reasons = append(out.Reasons, reason)
		}
		sort.Strings(out.Reasons)
	}
	return out
}

// CalculateVivahMonth calculates a bounded India civil month of preliminary
// marriage windows from native Panchang boundaries, never hardcoded dates.
func CalculateVivahMonth(ctx context.Context, epheDir, month, zone, convention string, latitude, longitude float64) (VivahMonth, error) {
	out := VivahMonth{Month: month, Zone: zone, Days: []VivahDay{}}
	start, err := time.Parse("2006-01", month)
	if err != nil || start.Format("2006-01") != month || month < MinDate[:7] || month > MaxDate[:7] || zone != "Asia/Kolkata" {
		return out, fmt.Errorf("unsupported marriage month or timezone")
	}
	days := []Panchang{}
	for day := start; day.Before(start.AddDate(0, 1, 0)) && day.Format("2006-01-02") <= MaxDate; day = day.AddDate(0, 0, 1) {
		p, err := CalculatePanchang(ctx, epheDir, day.Format("2006-01-02"), zone, convention, latitude, longitude)
		if err != nil {
			return out, err
		}
		days = append(days, p)
	}
	err = withSky(ctx, epheDir, zone, func(s *skySession) error {
		blocked, err := vivahCombustions(s, days[0].Sunrise, days[len(days)-1].NextSunrise)
		if err != nil {
			return err
		}
		for _, p := range days {
			if err := ctx.Err(); err != nil {
				return err
			}
			next := PanchangCalendar{}
			if p.Details.Calendar.LunationEnd.Before(p.NextSunrise) {
				// One second lies safely beyond the subsecond conjunction search tolerance.
				next, err = vivahLunation(s, p.Details.Calendar.LunationEnd.Add(time.Second))
				if err != nil {
					return err
				}
			}
			out.Days = append(out.Days, vivahDay(p, next, blocked))
		}
		out.Settings = s.settings()
		out.Settings["rules"] = "m4y-vivah-shortlist-v2"
		out.Settings["guruji_applied"] = "Supplied fourteen-Nakshatra list; Saturday excluded; Taurus, Gemini, Virgo, Libra, Sagittarius and Pisces Lagna signs"
		out.Settings["guruji_pending"] = "Tuesday, Tithi list and Paksha, Masa convention, Panchashalaka Vedha, planetary house rules and exceptions, and Trijyeshtha await confirmation; existing additional safeguards retained"
		out.Settings["sunrise_convention"] = convention
		out.Settings["day_boundary"] = "Sunrise to next sunrise; half-open intervals; minimum window five minutes"
		out.Settings["combustion"] = "Angular only: Jupiter 11 degrees; Venus 10 direct / 8 retrograde; inclusive; no heliacal Asta or bala/vriddha buffer"
		out.Settings["scope"] = "Preliminary shortlist with marriage Lagna sign selection; no full Lagna/Navamsha Shuddhi, planetary house exceptions, Trijyeshtha, personal Tara/Chandra bala, Panchashalaka Vedha, Lattadi, eclipse/Sankranti buffers or regional exceptions"
		return nil
	})
	return out, err
}

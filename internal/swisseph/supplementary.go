package swisseph

import (
	"fmt"
	"math"
	"time"
)

// DivisionalPlacement deliberately excludes physical longitude/nakshatra:
// D9 is a sign mapping, not another observed astronomical position.
type DivisionalPlacement struct {
	Name        string `json:"name"`
	RashiNumber int    `json:"rashi_number"`
	House       int    `json:"house"`
	Vargottama  bool   `json:"vargottama"`
}
type Navamsa struct {
	LagnaSign int                   `json:"lagna_sign"`
	Planets   []DivisionalPlacement `json:"planets"`
}

// navamsaSign maps an unrounded zodiac position to its traditional Parashari D9 sign.
func navamsaSign(longitude float64) int {
	if !finite(longitude) {
		return 0
	}
	// Traditional Parashari D9: successive 3°20′ divisions progress Aries
	// through Pisces, repeating nine times around the sidereal zodiac.
	lon := normalizeDegrees(longitude)
	sign := int(lon / 30)
	return ([]int{0, 9, 6, 3}[sign%4]+vargaPart(lon, 9))%12 + 1
}

// CalculateNavamsa derives the Navamsa Lagna and planetary placements from a calculated birth chart.
func CalculateNavamsa(chart Kundali) Navamsa {
	out := Navamsa{LagnaSign: navamsaSign(chart.Lagna.Longitude), Planets: []DivisionalPlacement{}}
	if out.LagnaSign == 0 {
		return out
	}
	for _, p := range chart.Planets {
		sign := navamsaSign(p.Longitude)
		if sign == 0 {
			continue
		}
		out.Planets = append(out.Planets, DivisionalPlacement{p.Name, sign, (sign-out.LagnaSign+12)%12 + 1, sign == p.RashiNumber})
	}
	return out
}

const DashaYearDays = 365.25
const VimshottariMethod = "m4y-vimshottari-julian36525-five-level-v1"

var dashaLords = [...]string{"Ketu", "Venus", "Sun", "Moon", "Mars", "Rahu", "Jupiter", "Saturn", "Mercury"}
var dashaYears = [...]float64{7, 20, 6, 10, 7, 18, 16, 19, 17}

type DashaPeriod struct {
	Lord        string        `json:"lord"`
	Start       time.Time     `json:"start"`
	End         time.Time     `json:"end"`
	Antardashas []DashaPeriod `json:"antardashas,omitempty"`
}
type Vimshottari struct {
	Method         string        `json:"method"`
	MaxDepth       int           `json:"max_depth"`
	CurrentPath    []DashaStep   `json:"current_path"`
	AsOf           time.Time     `json:"as_of"`
	YearDays       float64       `json:"year_days"`
	BirthLord      string        `json:"birth_lord"`
	BalanceYears   float64       `json:"balance_years"`
	CurrentLord    string        `json:"current_lord"`
	CurrentSubLord string        `json:"current_sub_lord"`
	Periods        []DashaPeriod `json:"periods"`
}

// DashaStep is one active period at a named depth. Path contains zero-based
// indices from the first retained Mahadasha, including subsequent cycles.
type DashaStep struct {
	Level int       `json:"level"`
	Path  []int     `json:"path"`
	Lord  string    `json:"lord"`
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// DashaChildren divides the FULL parent period, starting with its own lord.
// Integer cumulative proportions prevent gaps, overlaps and repeated rounding
// drift. Quotient/remainder arithmetic avoids multiplying a multi-year
// nanosecond duration by 120, which would overflow int64.
func DashaChildren(parent DashaPeriod) ([]DashaPeriod, error) {
	lord := -1
	for i, name := range dashaLords {
		if name == parent.Lord {
			lord = i
			break
		}
	}
	span := parent.End.Sub(parent.Start)
	if lord < 0 || span <= 0 || span > 21*365*24*time.Hour {
		return nil, fmt.Errorf("invalid Vimshottari parent")
	}
	children := make([]DashaPeriod, 0, 9)
	elapsed := int64(0)
	start := parent.Start
	for i := 0; i < 9; i++ {
		index := (lord + i) % 9
		elapsed += int64(dashaYears[index])
		ns := int64(span)
		offset := (ns/120)*elapsed + ((ns%120)*elapsed+60)/120
		end := parent.Start.Add(time.Duration(offset))
		if i == 8 {
			end = parent.End
		}
		children = append(children, DashaPeriod{Lord: dashaLords[index], Start: start, End: end})
		start = end
	}
	return children, nil
}

// DashaBranch calculates only the nine periods requested, through Prana (5).
// Existing Antardasha boundaries are retained for compatibility; descendants
// partition those exact boundaries rather than a rounded display date.
func DashaBranch(d Vimshottari, path []int) (*DashaPeriod, []DashaPeriod, error) {
	if len(path) > 4 {
		return nil, nil, fmt.Errorf("Vimshottari supports five levels")
	}
	periods := d.Periods
	var parent *DashaPeriod
	for _, index := range path {
		if index < 0 || index >= len(periods) {
			return nil, nil, fmt.Errorf("invalid Vimshottari path")
		}
		p := periods[index]
		parent = &p
		if len(p.Antardashas) == 9 {
			periods = p.Antardashas
		} else {
			var err error
			periods, err = DashaChildren(p)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	// Keep payloads bounded; children are fetched one level at a time.
	out := make([]DashaPeriod, len(periods))
	for i, p := range periods {
		out[i] = DashaPeriod{Lord: p.Lord, Start: p.Start, End: p.End}
	}
	if parent != nil {
		parent.Antardashas = nil
	}
	return parent, out, nil
}

// addDashaYears advances an instant by the declared 365.25-day Julian-year duration.
func addDashaYears(t time.Time, years float64) time.Time {
	return t.Add(time.Duration(math.Round(years * DashaYearDays * 86400 * 1e9)))
}

// inPeriod tests an instant against a start-inclusive/end-exclusive Dasha interval.
func inPeriod(at time.Time, p DashaPeriod) bool { return !at.Before(p.Start) && at.Before(p.End) }

// CalculateVimshottari builds Moon-Nakshatra-based Vimshottari periods and identifies the currently running sequence.
func CalculateVimshottari(moonLongitude float64, birth, asOf time.Time) Vimshottari {
	if !finite(moonLongitude) {
		return Vimshottari{AsOf: asOf.UTC(), YearDays: DashaYearDays, Periods: []DashaPeriod{}}
	}
	starPosition := normalizeDegrees(moonLongitude) * 27 / 360
	star := int(math.Floor(starPosition))
	fraction := starPosition - float64(star)
	first := star % 9
	out := Vimshottari{Method: VimshottariMethod, MaxDepth: 5, CurrentPath: []DashaStep{}, AsOf: asOf.UTC(), YearDays: DashaYearDays, BirthLord: dashaLords[first], BalanceYears: (1 - fraction) * dashaYears[first], Periods: []DashaPeriod{}}
	// Compute subperiods from the FULL mahadasha beginning before birth.
	// Scaling the remaining balance would produce incorrect antardashas.
	start := addDashaYears(birth.UTC(), -fraction*dashaYears[first])
	// Keep the birth cycle and, for an older native, continue complete cycles
	// until asOf is covered. The supported 1940-2099 instants can be over
	// 120 years apart; returning no running lord after the first cycle is wrong.
	for i := 0; i < 9 || (!asOf.Before(birth) && (i%9 != 0 || !asOf.Before(start))); i++ {
		lord := (first + i) % 9
		p := DashaPeriod{Lord: dashaLords[lord], Start: start, End: addDashaYears(start, dashaYears[lord]), Antardashas: []DashaPeriod{}}
		elapsed := 0.0
		for j := 0; j < 9; j++ {
			sub := (lord + j) % 9
			part := DashaPeriod{Lord: dashaLords[sub], Start: addDashaYears(start, elapsed)}
			elapsed += dashaYears[lord] * dashaYears[sub] / 120
			part.End = addDashaYears(start, elapsed)
			if j == 8 {
				part.End = p.End
			}
			p.Antardashas = append(p.Antardashas, part)
			if !asOf.Before(birth) && inPeriod(asOf, p) && inPeriod(asOf, part) {
				out.CurrentLord = p.Lord
				out.CurrentSubLord = part.Lord
			}
		}
		out.Periods = append(out.Periods, p)
		start = p.End
	}
	if !asOf.Before(birth) {
		path := []int{}
		for level := 1; level <= 5; level++ {
			_, periods, err := DashaBranch(out, path)
			if err != nil {
				break
			}
			found := false
			for i, p := range periods {
				if inPeriod(asOf, p) {
					path = append(path, i)
					out.CurrentPath = append(out.CurrentPath, DashaStep{Level: level, Path: append([]int{}, path...), Lord: p.Lord, Start: p.Start, End: p.End})
					found = true
					break
				}
			}
			if !found {
				break
			}
		}
	}
	return out
}

package swisseph

import (
	"fmt"
	"math"
	"time"
)

type YoginiLord struct {
	Name  string `json:"name"`
	Lord  string `json:"lord"`
	Years int    `json:"years"`
}
type Yogini struct {
	Method         string        `json:"method"`
	AsOf           time.Time     `json:"as_of"`
	YearDays       float64       `json:"year_days"`
	BirthLord      string        `json:"birth_lord"`
	BalanceYears   float64       `json:"balance_years"`
	CurrentLord    string        `json:"current_lord"`
	CurrentSubLord string        `json:"current_sub_lord"`
	Lords          []YoginiLord  `json:"lords"`
	Periods        []DashaPeriod `json:"periods"`
}

var yoginiLords = []YoginiLord{{"Mangala", "Moon", 1}, {"Pingala", "Sun", 2}, {"Dhanya", "Jupiter", 3}, {"Bhramari", "Mars", 4}, {"Bhadrika", "Mercury", 5}, {"Ulka", "Saturn", 6}, {"Siddha", "Venus", 7}, {"Sankata", "Rahu", 8}}

// CalculateYogini builds Yogini and subperiod timelines from the natal Moon Nakshatra and birth balance.
func CalculateYogini(moonLongitude float64, birth, asOf time.Time) (Yogini, error) {
	out := Yogini{Method: "Moon nakshatra: (one-based star + 3) modulo 8; 36-year cycle; full-parent proportional Antardashas; 365.25-day Julian year", AsOf: asOf.UTC(), YearDays: DashaYearDays, Lords: append([]YoginiLord{}, yoginiLords...), Periods: []DashaPeriod{}}
	if !finite(moonLongitude) || moonLongitude < 0 || moonLongitude >= 360 || asOf.Before(birth) {
		return out, fmt.Errorf("invalid Yogini inputs")
	}
	star := int(moonLongitude * 27 / 360)
	first := (star + 3) % 8 // zero-based lord; Ashwini -> Bhramari
	fraction := moonLongitude*27/360 - float64(star)
	out.BirthLord = yoginiLords[first].Lord
	out.BalanceYears = (1 - fraction) * float64(yoginiLords[first].Years)
	start := addDashaYears(birth, -fraction*float64(yoginiLords[first].Years))
	for i := 0; i < 48; i++ {
		index := (first + i) % 8
		lord := yoginiLords[index]
		end := addDashaYears(start, float64(lord.Years))
		p := DashaPeriod{Lord: lord.Lord, Start: start, End: end, Antardashas: []DashaPeriod{}}
		span := int64(end.Sub(start))
		elapsed := int64(0)
		previous := start
		for j := 0; j < 8; j++ {
			sub := yoginiLords[(index+j)%8]
			elapsed += int64(sub.Years)
			// Cumulative quotient/remainder, avoiding multiplying a full duration.
			edge := start.Add(time.Duration(span/36*elapsed + span%36*elapsed/36))
			child := DashaPeriod{Lord: sub.Lord, Start: previous, End: edge}
			p.Antardashas = append(p.Antardashas, child)
			if inPeriod(asOf, child) {
				out.CurrentSubLord = sub.Lord
			}
			previous = edge
		}
		if inPeriod(asOf, p) {
			out.CurrentLord = lord.Lord
		}
		out.Periods = append(out.Periods, p)
		start = end
	}
	if out.CurrentLord == "" || math.IsNaN(out.BalanceYears) {
		return out, fmt.Errorf("Yogini reference outside cycle coverage")
	}
	return out, nil
}

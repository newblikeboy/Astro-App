// Package numerology calculates date-based Indian numerology independently of astronomical charts.
package numerology

import (
	"fmt"
	"time"
)

const Method = "m4y-numerology-civil-date-single-digit-v1"

type Number struct {
	Value  int    `json:"value"`
	Planet string `json:"planet"`
	Digits []int  `json:"digits"`
	Steps  []int  `json:"steps"`
}

type Reading struct {
	Method    string           `json:"method"`
	BirthDate string           `json:"birth_date"`
	Mulank    Number           `json:"mulank"`
	Bhagyank  Number           `json:"bhagyank"`
	Extended  *ExtendedReading `json:"extended,omitempty"`
}

// Calculate uses the recorded Gregorian civil date, not its UTC date or sunrise.
// Mulank reduces the day; Bhagyank reduces every DDMMYYYY digit. Totals 11/22/33
// also reduce to 1–9. Invalid dates fail instead of producing a plausible reading.
func Calculate(date string) (Reading, error) {
	at, err := time.Parse("2006-01-02", date)
	if err != nil || at.Year() < 1 || at.Format("2006-01-02") != date {
		return Reading{}, fmt.Errorf("numerology requires a valid YYYY-MM-DD birth date")
	}
	return Reading{Method: Method, BirthDate: date, Mulank: reduce(fmt.Sprint(at.Day())), Bhagyank: reduce(at.Format("02012006"))}, nil
}

// reduce records each addition and its single-digit result; planet names denote traditional number associations only.
func reduce(source string) Number {
	out := Number{Digits: []int{}, Steps: []int{}}
	sum := 0
	for _, digit := range source {
		value := int(digit - '0')
		out.Digits = append(out.Digits, value)
		sum += value
	}
	out.Steps = append(out.Steps, sum)
	for sum > 9 {
		next := 0
		for remaining := sum; remaining > 0; remaining /= 10 {
			next += remaining % 10
		}
		sum = next
		out.Steps = append(out.Steps, sum)
	}
	out.Value = sum
	out.Planet = []string{"Sun", "Moon", "Jupiter", "Rahu", "Mercury", "Venus", "Ketu", "Saturn", "Mars"}[sum-1]
	return out
}

package numerology

import (
	"reflect"
	"testing"
)

// TestDateNumbers covers hand-worked dates, leap days, leading zeroes and intermediate 11/22/33 reductions.
func TestDateNumbers(t *testing.T) {
	for _, tc := range []struct {
		date                       string
		mulank, bhagyank           int
		mulankSteps, bhagyankSteps []int
	}{
		{"1995-11-29", 2, 1, []int{11, 2}, []int{37, 10, 1}},
		{"2024-02-29", 2, 3, []int{11, 2}, []int{21, 3}},
		{"2000-01-01", 1, 4, []int{1}, []int{4}},
		{"1990-10-22", 4, 6, []int{4}, []int{24, 6}},
		{"2000-09-09", 9, 2, []int{9}, []int{20, 2}},
		{"2000-01-08", 8, 2, []int{8}, []int{11, 2}},
		{"2000-08-12", 3, 4, []int{3}, []int{13, 4}},
		{"1999-01-04", 4, 6, []int{4}, []int{33, 6}},
		{"2000-09-29", 2, 4, []int{11, 2}, []int{22, 4}},
	} {
		r, err := Calculate(tc.date)
		if err != nil || r.Method != Method || r.BirthDate != tc.date || r.Mulank.Value != tc.mulank || r.Bhagyank.Value != tc.bhagyank || !reflect.DeepEqual(r.Mulank.Steps, tc.mulankSteps) || !reflect.DeepEqual(r.Bhagyank.Steps, tc.bhagyankSteps) {
			t.Fatalf("%s: %+v, %v", tc.date, r, err)
		}
		if len(r.Bhagyank.Digits) != 8 || r.Mulank.Planet == "" || r.Bhagyank.Planet == "" {
			t.Fatal("missing calculation evidence")
		}
	}
}

// TestNumberRulers verifies the declared nine-number association without borrowing chart placements.
func TestNumberRulers(t *testing.T) {
	for i, planet := range []string{"Sun", "Moon", "Jupiter", "Rahu", "Mercury", "Venus", "Ketu", "Saturn", "Mars"} {
		r, err := Calculate("2000-01-0" + string(rune('1'+i)))
		if err != nil || r.Mulank.Value != i+1 || r.Mulank.Planet != planet {
			t.Fatalf("number %d: %+v, %v", i+1, r, err)
		}
	}
}

// TestInvalidDates rejects impossible, noncanonical and non-Gregorian input strings.
func TestInvalidDates(t *testing.T) {
	for _, date := range []string{"", "1995-2-09", "29-11-1995", "1900-02-29", "2024-02-30", "2000-00-01", "2000-01-00", "0000-01-01", "2000-01-01T00:00:00Z", " 2000-01-01"} {
		if _, err := Calculate(date); err == nil {
			t.Fatalf("accepted invalid date %q", date)
		}
	}
}

package swisseph

import (
	"context"
	"fmt"
	"math"
	"slices"
	"testing"
	"time"
)

// TestAuditTropicalRituReferenceSectors checks tropical-longitude sectors used to classify Ritu.
func TestAuditTropicalRituReferenceSectors(t *testing.T) {
	// Publisher's definition, not a snapshot of this implementation:
	// https://www.drikpanchang.com/seasons/season-tropical-timings.html
	// Vasanta Pisces/Aries, Grishma Taurus/Gemini, Varsha Cancer/Leo,
	// Sharad Virgo/Libra, Hemanta Scorpio/Sagittarius, Shishira Capricorn/Aquarius.
	want := []string{"Vasanta", "Grishma", "Grishma", "Varsha", "Varsha", "Sharad", "Sharad", "Hemanta", "Hemanta", "Shishira", "Shishira", "Vasanta"}
	for sign, name := range want {
		for _, longitude := range []float64{float64(sign) * 30, float64(sign)*30 + 15, float64(sign)*30 + 30 - 1e-8} {
			if got := tropicalRitu(longitude); got != name {
				t.Errorf("tropical longitude %.9f: %s want %s", longitude, got, name)
			}
		}
	}
	if tropicalRitu(360) != "Vasanta" || tropicalRitu(-30) != "Vasanta" {
		t.Fatal("season angle wrap")
	}
}

// TestAuditTropicalRituReferenceDates checks Ritu classification against dated tropical reference cases.
func TestAuditTropicalRituReferenceDates(t *testing.T) {
	// Dates after the six published 2026 transitions. Exact times are not
	// inferred from a rounded page; every fixture is safely inside its season.
	for _, tc := range []struct{ date, ritu string }{
		{"2026-02-19", "Vasanta"}, {"2026-04-21", "Grishma"},
		{"2026-06-22", "Varsha"}, {"2026-08-24", "Sharad"},
		{"2026-10-24", "Hemanta"}, {"2026-12-23", "Shishira"},
	} {
		t.Run(tc.date, func(t *testing.T) {
			p := auditPanchang(t, tc.date)
			if p.Details.TropicalRitu != tc.ritu {
				t.Fatalf("%s want %s", p.Details.TropicalRitu, tc.ritu)
			}
		})
	}
}

// TestAuditMoonEventsUseSunriseDay checks that Moon events belong to the sunrise-to-next-sunrise Panchang day.
func TestAuditMoonEventsUseSunriseDay(t *testing.T) {
	// Independently invoked official swetest, upper limb/refraction/sea level:
	// swetest -b20.10.2026 -ut0:55 -p1 -rise -edirephe -eswe
	//         -geopos77.209,28.6139,0 -head
	// UT rise20Oct08:49:05.3, set20Oct19:55:42.4 (21Oct01:25:42.4 IST).
	// Starting at midnight incorrectly returned 20Oct00:29 from the prior night.
	p := auditPanchang(t, "2026-10-20")
	for _, tc := range []struct {
		event *time.Time
		want  string
	}{{p.Details.Moonrise, "2026-10-20T14:19:05+05:30"}, {p.Details.Moonset, "2026-10-21T01:25:42+05:30"}} {
		want, err := time.Parse(time.RFC3339, tc.want)
		if err != nil {
			t.Fatal(err)
		}
		if tc.event == nil || math.Abs(tc.event.Sub(want).Seconds()) > 1 {
			t.Errorf("Moon event got %v want %s (1s CLI tolerance)", tc.event, want)
		}
	}
	previous := auditPanchang(t, "2026-10-19")
	if p.Details.Moonset != nil && previous.Details.Moonset != nil && p.Details.Moonset.Equal(*previous.Details.Moonset) {
		t.Fatal("moonset repeated across different sunrise days")
	}
	missingRise, missingSet := false, false
	for day := 1; day <= 31; day++ {
		q := auditPanchang(t, fmt.Sprintf("2026-10-%02d", day))
		for _, event := range []*time.Time{q.Details.Moonrise, q.Details.Moonset} {
			if event != nil && (event.Before(q.Sunrise) || !event.Before(q.NextSunrise)) {
				t.Errorf("%s moon event %s outside [%s,%s)", q.Date, event, q.Sunrise, q.NextSunrise)
			}
		}
		missingRise = missingRise || q.Details.Moonrise == nil
		missingSet = missingSet || q.Details.Moonset == nil
	}
	if !missingRise || !missingSet {
		t.Fatal("lunation must retain sunrise days without a Moon rise/set instead of borrowing a later day")
	}
}

// TestAuditKshayaKrishnaKeepsCombinedMonth checks that Krishna Paksha retains a combined month during Kshaya handling.
func TestAuditKshayaKrishnaKeepsCombinedMonth(t *testing.T) {
	// Kamakoti identifies 15Jan-12Feb1983 as Pausha(-Magha) Kshaya:
	// https://www.kamakoti.org/kamakoti/details/calendar.html
	// Panchang Karta Mahesh Shastri explicitly retains the Amanta label for
	// an Adhika/Kshaya month in the Purnimanta convention:
	// https://drishyaganitpanchang.com/newsite/kartikamasa.php
	for _, date := range []string{"1983-01-20", "1983-02-03", "1983-02-10"} {
		p := auditPanchang(t, date)
		c := p.Details.Calendar
		if c.Amanta != "Pausha" || c.Purnimanta != "Pausha" || c.SkippedMonth != "Magha" || c.Adhika {
			t.Errorf("%s Kshaya labels %+v", date, c)
		}
	}
}

// TestAuditChaitraYearIncludesJanuaryMargashirsha checks Chaitra-year assignment for a January Margashirsha case.
func TestAuditChaitraYearIncludesJanuaryMargashirsha(t *testing.T) {
	// 1Jan2024 is Margashirsha Krishna in Amanta, Purnimanta Pausha;
	// Chaitra new year has not occurred, so Shaka1945 / Vikram2080 remain.
	// https://www.drikpanchang.com/panchang/day-panchang.html?date=01/01/2024
	p := auditPanchang(t, "2024-01-01")
	c := p.Details.Calendar
	if c.Amanta != "Margashirsha" || c.Purnimanta != "Pausha" || c.Vikram != 2080 || c.Shaka != 1945 {
		t.Fatalf("January calendar/year %+v", c)
	}
	for _, tc := range []struct {
		date          string
		vikram, shaka int
	}{
		{"2024-04-08", 2080, 1945}, {"2024-04-09", 2081, 1946},
	} {
		q := auditPanchang(t, tc.date)
		if q.Details.Calendar.Vikram != tc.vikram || q.Details.Calendar.Shaka != tc.shaka {
			t.Errorf("%s year %+v", tc.date, q.Details.Calendar)
		}
	}
}

// TestAuditTraditionalIntersectionsAndNivas checks traditional timing intersections and Nivas rules.
func TestAuditTraditionalIntersectionsAndNivas(t *testing.T) {
	// Hand-built astronomical sectors test the rule layer independently of
	// boundary searches. Krishna Dwitiya=17, Tuesday=3, Mrigashirsha=5,
	// Kumbha=11: sum36 leaves0, Panchaka Rahita; Dwipushkar requires all
	// three factors together and ends exactly when Dwitiya ends.
	a := time.Date(2026, 10, 20, 6, 0, 0, 0, time.UTC)
	b, end := a.Add(4*time.Hour), a.Add(24*time.Hour)
	// row builds one full-day limb interval for a hand-constructed traditional-rule fixture.
	row := func(number int, name string) []PanchangPeriod {
		return []PanchangPeriod{{Number: number, Name: name, Start: a, End: end}}
	}
	p := &Panchang{
		Sunrise: a, NextSunrise: end, Weekday: 2,
		Tithi:     []PanchangPeriod{{Number: 17, Start: a, End: b}, {Number: 18, Start: b, End: end}},
		Nakshatra: row(5, "Mrigashirsha"), Karana: row(7, "Vishti"),
		Details: &PanchangDetails{MoonSigns: row(11, "Aquarius"), SunNakshatras: row(2, "Bharani"), Lagna: row(11, "Aquarius")},
	}
	populatePanchangRules(p)
	// find extracts every timing interval matching the requested traditional-rule name.
	find := func(name string) []PanchangPeriod {
		var result []PanchangPeriod
		for _, r := range p.Details.Special {
			if r.Name == name {
				result = append(result, r)
			}
		}
		return result
	}
	if r := find("Dwipushkar Yoga"); len(r) != 1 || !r[0].Start.Equal(a) || !r[0].End.Equal(b) {
		t.Fatalf("Dwipushkar intersection %+v", r)
	}
	if len(find("Tripushkar Yoga")) != 0 || len(find("Ganda Moola")) != 0 || len(find("Vinchudo")) != 0 {
		t.Fatal("unrelated traditions falsely emitted")
	}
	if r := p.Details.Panchaka; len(r) != 2 || r[0].Name != "Panchaka Rahita" || r[1].Name != "Mrityu Panchaka" || !r[0].End.Equal(b) || !r[1].Start.Equal(b) {
		t.Fatalf("Panchaka arithmetic/intersection %+v", r)
	}
	if r := find("Panchaka"); len(r) != 1 || !r[0].End.Equal(end) {
		t.Fatal("Aquarius Dhanishtha-half Panchaka must cover the day")
	}
	if len(p.Details.Balam) != 1 || len(p.Details.Balam[0].Chandra) != 6 || len(p.Details.Balam[0].Tara) != 15 {
		t.Fatal("unrelated Tithi change must not split Balam")
	}
	for _, tc := range []struct{ name, value string }{{"Disha Shool", "North"}, {"Rahu Vasa", "West"}, {"Chandra Vasa", "West"}, {"Bhadra Vasa", "Prithvi"}} {
		found := false
		for _, r := range p.Details.Directions {
			if r.Name == tc.name && r.Value == tc.value && r.Start.Equal(a) && r.End.Equal(end) {
				found = true
			}
		}
		if !found {
			t.Errorf("missing complete %s=%s", tc.name, tc.value)
		}
	}
	// Direct same-day publisher observations: 18Oct2026 Saptami ->Ashtami,
	// Agni Akasha->Patala; Shiva Bhojana->Shmashana. These values differ
	// from geometric astronomy and must be checked as traditional rules.
	q := auditPanchang(t, "2026-10-18")
	for _, tc := range []struct {
		name   string
		values []string
	}{
		{"Agni Vasa", []string{"Akasha", "Patala"}},
		{"Shiva Vasa", []string{"Bhojana", "Shmashana"}},
	} {
		var got []string
		for _, r := range q.Details.Directions {
			if r.Name == tc.name {
				got = append(got, r.Value)
			}
		}
		if !slices.Equal(got, tc.values) {
			t.Errorf("%s got %v want %v", tc.name, got, tc.values)
		}
	}
}

// auditPanchang calculates the fixed New Delhi audit day and fails immediately if native calculation fails.
func auditPanchang(t *testing.T, date string) *Panchang {
	t.Helper()
	p, err := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	return &p
}

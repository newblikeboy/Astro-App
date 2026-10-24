package swisseph

import (
	"context"
	"slices"
	"testing"
	"time"
)

// vivahFixture supplies constant favourable limbs; individual tests inject
// independent clock boundaries rather than regenerating expected results.
func vivahFixture() Panchang {
	loc, _ := time.LoadLocation("Asia/Kolkata")
	start := time.Date(2026, 4, 20, 6, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)
	// period builds one constant limb across the synthetic solar day.
	period := func(number int, name string) []PanchangPeriod {
		return []PanchangPeriod{{Number: number, Name: name, Start: start, End: end}}
	}
	return Panchang{Date: "2026-04-20", Weekday: int(time.Monday), Sunrise: start, Sunset: start.Add(12 * time.Hour), NextSunrise: end,
		Tithi: period(3, "Shukla Tritiya"), Nakshatra: period(4, "Rohini"), Yoga: period(2, "Priti"), Karana: period(2, "Bava"),
		Details: &PanchangDetails{SunSigns: period(1, "Aries"), Padas: period(13, "Rohini"), Lagna: period(2, "Vrishabh"), Calendar: PanchangCalendar{MonthNumber: 2, LunationEnd: end.Add(10 * 24 * time.Hour)}}}
}

// TestVivahExactRuleIntersections checks sunrise clipping, blockers already in
// progress, a Tithi transition, and overnight date retention against hand-set clocks.
func TestVivahExactRuleIntersections(t *testing.T) {
	p := vivahFixture()
	start := p.Sunrise
	p.Timings = []PanchangPeriod{{Name: "Rahu Kaal", Start: start.Add(4 * time.Hour), End: start.Add(5 * time.Hour)}}
	p.Details.Muhurats = []PanchangPeriod{{Name: "Dur Muhurtam", Start: start.Add(8 * time.Hour), End: start.Add(8*time.Hour + 30*time.Minute)}}
	p.Tithi[0].End = start.Add(12 * time.Hour)
	p.Tithi = append(p.Tithi, PanchangPeriod{Number: 4, Name: "Shukla Chaturthi", Start: start.Add(12 * time.Hour), End: p.NextSunrise})
	blocked := []PanchangPeriod{{Name: "Guru combustion", Start: start.Add(-time.Hour), End: start.Add(2 * time.Hour)}}
	got := vivahDay(p, PanchangCalendar{}, blocked)
	want := [][2]time.Duration{{2 * time.Hour, 4 * time.Hour}, {5 * time.Hour, 8 * time.Hour}, {8*time.Hour + 30*time.Minute, 12 * time.Hour}}
	if len(got.Windows) != len(want) {
		t.Fatalf("unexpected windows: %+v", got)
	}
	for i, pair := range want {
		if !got.Windows[i].Start.Equal(start.Add(pair[0])) || !got.Windows[i].End.Equal(start.Add(pair[1])) {
			t.Fatalf("window %d: %+v", i, got.Windows[i])
		}
	}
	p = vivahFixture()
	p.Nakshatra[0].Start = start.Add(18 * time.Hour)
	p.Nakshatra = append([]PanchangPeriod{{Number: 3, Name: "Krittika", Start: start, End: start.Add(18 * time.Hour)}}, p.Nakshatra...)
	got = vivahDay(p, PanchangCalendar{}, nil)
	if len(got.Windows) != 1 || got.Windows[0].Start.Format("2006-01-02 15:04") != "2026-04-21 00:00" || !got.Windows[0].End.Equal(p.NextSunrise) {
		t.Fatalf("overnight interval lost: %+v", got)
	}
}

// TestVivahCalendarChangeAndShortWindows checks exact lunation changes, quarter
// exclusions, adjacent acceptable quarters and the five-minute minimum.
func TestVivahCalendarChangeAndShortWindows(t *testing.T) {
	p := vivahFixture()
	p.Details.Calendar.LunationEnd = p.Sunrise.Add(6 * time.Hour)
	got := vivahDay(p, PanchangCalendar{MonthNumber: 3, Adhika: true}, nil)
	if len(got.Windows) != 1 || !got.Windows[0].End.Equal(p.Details.Calendar.LunationEnd) {
		t.Fatalf("lunation boundary ignored: %+v", got)
	}
	p = vivahFixture()
	p.Nakshatra[0].Number, p.Nakshatra[0].Name = 10, "Magha"
	p.Details.Padas[0].Number = 37
	p.Details.Padas[0].End = p.Sunrise.Add(2 * time.Hour)
	p.Details.Padas = append(p.Details.Padas, PanchangPeriod{Number: 38, Start: p.Sunrise.Add(2 * time.Hour), End: p.NextSunrise})
	got = vivahDay(p, PanchangCalendar{}, nil)
	if len(got.Windows) != 1 || !got.Windows[0].Start.Equal(p.Sunrise.Add(2*time.Hour)) {
		t.Fatalf("first Magha quarter retained: %+v", got)
	}
	p = vivahFixture()
	p.Details.Padas[0].End = p.Sunrise.Add(3 * time.Minute)
	p.Details.Padas = append(p.Details.Padas, PanchangPeriod{Number: 14, Start: p.Sunrise.Add(3 * time.Minute), End: p.NextSunrise})
	got = vivahDay(p, PanchangCalendar{}, nil)
	if len(got.Windows) != 1 || !got.Windows[0].Start.Equal(p.Sunrise) {
		t.Fatalf("acceptable quarters were not merged: %+v", got)
	}
	p.Timings = []PanchangPeriod{{Name: "Rahu Kaal", Start: p.Sunrise.Add(4 * time.Minute), End: p.NextSunrise}}
	got = vivahDay(p, PanchangCalendar{}, nil)
	if len(got.Windows) != 0 || !slices.Contains(got.Reasons, "Window shorter than five minutes") {
		t.Fatalf("short window retained: %+v", got)
	}
}

// TestVivahSeasonAndLimbRules checks explicit selection tables and Chaturmas
// edges independently of native ephemeris fixtures.
func TestVivahSeasonAndLimbRules(t *testing.T) {
	for _, tc := range []struct {
		month, tithi int
		reason       string
	}{
		{4, 10, ""}, {4, 11, "Chaturmas"}, {4, 29, "Chaturmas"}, {5, 3, "Chaturmas"}, {7, 3, "Chaturmas"},
		{8, 11, "Chaturmas"}, {8, 12, ""}, {8, 18, ""}, {12, 7, ""}, {12, 8, "Holashtak"}, {12, 15, "Holashtak"}, {12, 17, ""},
	} {
		p := vivahFixture()
		p.Details.Calendar.MonthNumber = tc.month
		p.Tithi[0].Number = tc.tithi
		got := vivahDay(p, PanchangCalendar{}, nil)
		if tc.reason == "" && len(got.Windows) != 1 || tc.reason != "" && (len(got.Windows) != 0 || !slices.Contains(got.Reasons, tc.reason)) {
			t.Fatalf("month %d tithi %d: %+v", tc.month, tc.tithi, got)
		}
	}
	for _, tc := range []struct {
		kind         string
		number       int
		name, reason string
	}{
		{"tithi", 4, "Shukla Chaturthi", "Excluded Tithi"}, {"tithi", 24, "Krishna Navami", "Excluded Tithi"}, {"tithi", 30, "Amavasya", "Excluded Tithi"},
		{"nakshatra", 3, "Krittika", "Marriage Nakshatra unavailable"}, {"yoga", 17, "Vyatipata", "Excluded Yoga"},
		{"karana", 8, "Vishti", "Excluded Karana"}, {"sun", 9, "Sagittarius", "Excluded solar month"},
	} {
		p := vivahFixture()
		rows := map[string]*[]PanchangPeriod{"tithi": &p.Tithi, "nakshatra": &p.Nakshatra, "yoga": &p.Yoga, "karana": &p.Karana, "sun": &p.Details.SunSigns}
		(*rows[tc.kind])[0].Number, (*rows[tc.kind])[0].Name = tc.number, tc.name
		got := vivahDay(p, PanchangCalendar{}, nil)
		if len(got.Windows) != 0 || !slices.Contains(got.Reasons, tc.reason) {
			t.Fatalf("%s: %+v", tc.kind, got)
		}
	}
}

// TestVivahGurujiNakshatras compares the supplied fourteen-star list against every
// star, independently of the retained first-quarter safeguards.
func TestVivahGurujiNakshatras(t *testing.T) {
	allowed := map[int]bool{1: true, 4: true, 5: true, 10: true, 12: true, 13: true, 14: true, 15: true, 17: true, 19: true, 21: true, 22: true, 23: true, 26: true}
	for number := 1; number <= 27; number++ {
		p := vivahFixture()
		p.Nakshatra[0].Number, p.Nakshatra[0].Name = number, nakshatraNames[number-1]
		p.Details.Padas[0].Number = (number-1)*4 + 2
		got := vivahDay(p, PanchangCalendar{}, nil)
		if (len(got.Windows) > 0) != allowed[number] {
			t.Fatalf("Guruji Nakshatra %d %s: %+v", number, nakshatraNames[number-1], got)
		}
	}
}

// TestVivahGurujiLagnaAndWeekday checks the six supplied signs, disallowed/missing
// Lagna data, Saturday's sunrise-day exclusion and the still-unresolved Tuesday.
func TestVivahGurujiLagnaAndWeekday(t *testing.T) {
	allowed := map[int]bool{2: true, 3: true, 6: true, 7: true, 9: true, 12: true}
	for number := 0; number <= 13; number++ {
		p := vivahFixture()
		p.Details.Lagna[0].Number = number
		got := vivahDay(p, PanchangCalendar{}, nil)
		if (len(got.Windows) > 0) != allowed[number] {
			t.Fatalf("Guruji Lagna %d: %+v", number, got)
		}
	}
	p := vivahFixture()
	p.Details.Lagna = nil
	if got := vivahDay(p, PanchangCalendar{}, nil); len(got.Windows) != 0 || !slices.Contains(got.Reasons, "Incomplete Lagna data") {
		t.Fatalf("missing Lagna accepted: %+v", got)
	}
	p = vivahFixture()
	p.Weekday = int(time.Saturday)
	if got := vivahDay(p, PanchangCalendar{}, nil); len(got.Windows) != 0 || !slices.Contains(got.Reasons, "Saturday excluded for marriage") {
		t.Fatalf("Saturday retained: %+v", got)
	}
	p.Weekday = int(time.Tuesday)
	if got := vivahDay(p, PanchangCalendar{}, nil); len(got.Windows) != 1 {
		t.Fatalf("Tuesday changed without clarification: %+v", got)
	}

	p = vivahFixture()
	start := p.Sunrise
	p.Details.Lagna = []PanchangPeriod{
		{Number: 2, Start: start, End: start.Add(2 * time.Hour)},
		{Number: 5, Start: start.Add(2 * time.Hour), End: start.Add(4 * time.Hour)},
		{Number: 6, Start: start.Add(4 * time.Hour), End: start.Add(6 * time.Hour)},
		{Number: 7, Start: start.Add(6 * time.Hour), End: start.Add(8 * time.Hour)},
		{Number: 8, Start: start.Add(8 * time.Hour), End: p.NextSunrise},
	}
	got := vivahDay(p, PanchangCalendar{}, nil)
	want := []struct {
		start, end time.Duration
		name       string
	}{{0, 2 * time.Hour, "Taurus"}, {4 * time.Hour, 6 * time.Hour, "Virgo"}, {6 * time.Hour, 8 * time.Hour, "Libra"}}
	if len(got.Windows) != len(want) {
		t.Fatalf("Lagna intersections: %+v", got)
	}
	for i, row := range want {
		window := got.Windows[i]
		if !window.Start.Equal(start.Add(row.start)) || !window.End.Equal(start.Add(row.end)) || window.Lagna != row.name {
			t.Fatalf("Lagna window %d crossed or merged: %+v", i, window)
		}
	}
}

// TestVivahNativeMonth checks actual monthly coverage, combustion spanning the
// first day, bounded edge months, input failures and cancellation.
func TestVivahNativeMonth(t *testing.T) {
	for _, tc := range []struct {
		month string
		count int
	}{{"2026-01", 31}, {"2026-11", 30}, {"1940-01", 31}, {"2099-12", 30}} {
		got, err := CalculateVivahMonth(context.Background(), "../../ephe", tc.month, "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
		if err != nil {
			t.Fatal(err)
		}
		if len(got.Days) != tc.count || got.Settings["rules"] != "m4y-vivah-shortlist-v2" {
			t.Fatalf("incomplete month %+v", got)
		}
		windows := 0
		for _, day := range got.Days {
			date, _ := time.Parse("2006-01-02", day.Date)
			if date.Weekday() == time.Saturday && len(day.Windows) > 0 {
				t.Fatalf("native Saturday retained: %+v", day)
			}
			windows += len(day.Windows)
			if day.Windows == nil || day.Reasons == nil {
				t.Fatal("null lists are not supported")
			}
			for i, window := range day.Windows {
				if !slices.Contains([]string{"Taurus", "Gemini", "Virgo", "Libra", "Sagittarius", "Pisces"}, window.Lagna) {
					t.Fatalf("native Lagna unavailable: %+v", window)
				}
				if window.End.Sub(window.Start) < 5*time.Minute || i > 0 && window.Start.Before(day.Windows[i-1].End) {
					t.Fatalf("invalid interval %+v", window)
				}
			}
			if tc.month == "2026-11" && len(day.Windows) > 0 {
				p, err := CalculatePanchang(context.Background(), "../../ephe", day.Date, "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
				if err != nil {
					t.Fatal(err)
				}
				for _, window := range day.Windows {
					first := vivahPeriodAt(p.Details.Lagna, window.Start)
					last := vivahPeriodAt(p.Details.Lagna, window.End.Add(-time.Nanosecond))
					if first.Number == 0 || first.Number != last.Number || matchingSigns[first.Number-1] != window.Lagna {
						t.Fatalf("native Lagna boundary crossed: %+v", window)
					}
				}
			}
		}
		if tc.month == "2026-01" && windows != 0 || tc.month == "2026-11" && windows == 0 {
			t.Fatalf("unexpected native availability %s: %d", tc.month, windows)
		}
	}
	for _, month := range []string{"2026-1", "2026-13", "1939-12", "2100-01"} {
		if _, err := CalculateVivahMonth(context.Background(), "../../ephe", month, "Asia/Kolkata", "hindu", 28, 77); err == nil {
			t.Fatalf("accepted %s", month)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CalculateVivahMonth(ctx, "../../ephe", "2026-11", "Asia/Kolkata", "hindu", 28, 77); err == nil {
		t.Fatal("cancellation ignored")
	}
}

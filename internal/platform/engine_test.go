package platform

import (
	"encoding/json"
	"math"
	"testing"
	"time"
)

// TestCalculationUsesBirthplaceCivilTodayAndActualInstant checks birth-date validation in the birthplace timezone and calculation at the actual instant.
func TestCalculationUsesBirthplaceCivilTodayAndActualInstant(t *testing.T) {
	// No App/Store or configuration loader is used; these pure calculations
	// neither read .env nor connect to Atlas/providers.
	c := Config{EpheDir: "../../ephe"}
	for _, tc := range []struct {
		name, now, place, date, clock string
		unknown, wantOK               bool
	}{
		{"Delhi today before UTC midnight", "2026-10-18T20:00:00Z", "delhi", "2026-10-19", "01:00", false, true},
		{"Delhi future clock today", "2026-10-18T20:00:00Z", "delhi", "2026-10-19", "02:00", false, false},
		{"Delhi same UTC date future clock", "2026-10-19T06:00:00Z", "delhi", "2026-10-19", "12:00", false, false},
		{"Delhi unknown time today", "2026-10-18T20:00:00Z", "delhi", "2026-10-19", "", true, true},
		{"New York tomorrow before local midnight", "2026-10-19T01:00:00Z", "new-york", "2026-10-19", "00:00", true, false},
		{"New York actual current local date", "2026-10-19T01:00:00Z", "new-york", "2026-10-18", "20:00", false, true},
		{"Birth exactly at now", "2026-10-19T00:00:00Z", "delhi", "2026-10-19", "05:30", false, true},
		{"Delhi 1940 midnight uses UTC buffer", "2026-10-19T00:00:00Z", "delhi", "1940-01-01", "00:00", false, true},
		{"Sydney 1940 midnight uses UTC buffer", "2026-10-19T00:00:00Z", "sydney", "1940-01-01", "00:00", false, true},
		{"Unknown time 1940 lower date", "2026-10-19T00:00:00Z", "delhi", "1940-01-01", "", true, true},
		{"Known time before public lower date", "2026-10-19T00:00:00Z", "delhi", "1939-12-31", "23:59", false, false},
		{"Unknown time before public lower date", "2026-10-19T00:00:00Z", "delhi", "1939-12-31", "", true, false},
		{"1940 leap day", "2026-10-19T00:00:00Z", "delhi", "1940-02-29", "10:30", false, true},
		{"Non-leap 1941 February29", "2026-10-19T00:00:00Z", "delhi", "1941-02-29", "10:30", false, false},
		{"1968 leap day", "2026-10-19T00:00:00Z", "london", "1968-02-29", "10:30", false, true},
		{"1972 former UTC edge is accepted", "2026-10-19T00:00:00Z", "delhi", "1972-01-01", "01:00", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tc.now)
			if err != nil {
				t.Fatal(err)
			}
			b := Birth{Name: "Audit Fixture", PlaceID: tc.place, Date: tc.date, Time: tc.clock, Unknown: tc.unknown}
			r, err := calculate(c, b, nil, now)
			if (err == nil) != tc.wantOK {
				t.Fatalf("acceptance = %v want %v: %v", err == nil, tc.wantOK, err)
			}
			if r != nil && tc.unknown && (r.Chart != nil || r.Dasha != nil || r.Navamsa != nil || r.Transits != nil) {
				t.Fatal("unknown-time birth generated astronomical details")
			}
		})
	}
}

// TestHistoricalIndiaWartimeZoneUsesIANAOffsets checks historical Indian wartime offsets against the IANA timezone database.
func TestHistoricalIndiaWartimeZoneUsesIANAOffsets(t *testing.T) {
	// IANA's primary Asia/Kolkata table includes wartime UTC+06:30 as well
	// as UTC+05:30. https://data.iana.org/time-zones/tzdb/asia
	// Its comments warn that the historical1941-1945 source is uncertain;
	// these fixtures establish packagedtzdata behavior, not documentary proof.
	for _, tc := range []struct {
		date, clock, utc string
		offset           *int
	}{
		{"1940-01-01", "00:00", "1939-12-31T18:30:00Z", nil},
		{"1940-02-29", "10:30", "1940-02-29T05:00:00Z", nil},
		{"1941-10-01", "00:30", "", nil},
		{"1942-01-01", "10:30", "1942-01-01T04:00:00Z", nil},
		{"1942-05-14", "23:30", "", nil},
		{"1942-05-14", "23:30", "1942-05-14T17:00:00Z", ptr(390)},
		{"1942-05-14", "23:30", "1942-05-14T18:00:00Z", ptr(330)},
		{"1942-09-01", "00:30", "", nil},
		{"1945-10-14", "23:30", "1945-10-14T17:00:00Z", ptr(390)},
		{"1945-10-14", "23:30", "1945-10-14T18:00:00Z", ptr(330)},
	} {
		got, err := ResolveWall(tc.date, tc.clock, "Asia/Kolkata", tc.offset)
		if tc.utc == "" {
			if err == nil {
				t.Fatalf("wartime gap/fold accepted without resolution: %+v -> %s", tc, got)
			}
		} else if err != nil || got.UTC().Format(time.RFC3339) != tc.utc {
			t.Fatalf("wartime IANA reference %+v -> %s, %v", tc, got, err)
		}
	}
	// Check the real chart path records the resolved wartime instant, not a
	// modern fixed IST substitution. No Store/configloader is involved.
	r, err := calculate(Config{EpheDir: "../../ephe"}, Birth{Name: "Historical audit", Date: "1942-01-01", Time: "10:30", PlaceID: "delhi"}, nil, time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))
	if err != nil || r.Chart == nil || r.Chart.UTC != "1942-01-01T04:00:00Z" {
		t.Fatalf("chart lost historical IANA instant: %+v, %v", r, err)
	}
}

// TestGuestKundali1940LowerBirthDate checks the earliest supported guest birth date in 1940.
func TestGuestKundali1940LowerBirthDate(t *testing.T) {
	a := testApp(t)
	b := Birth{Name: "Historical guest", Date: "1940-01-01", Time: "00:00", PlaceID: "delhi"}
	w := request(t, a, "POST", "/api/kundli", b, nil, "")
	if w.Code != 200 {
		t.Fatalf("1940 guest chart unavailable: %d %s", w.Code, w.Body)
	}
	var got Report
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Chart == nil || got.Chart.UTC != "1939-12-31T18:30:00Z" || got.Navamsa == nil || got.Dasha == nil {
		t.Fatalf("1940 guest chart lost prior-day UTC or supplementary calculations: %+v", got)
	}
	b.Date = "1939-12-31"
	if w = request(t, a, "POST", "/api/kundli", b, nil, ""); w.Code != 422 {
		t.Fatalf("pre-1940 public birth accepted: %d %s", w.Code, w.Body)
	}
}

// TestResolveWallNonHourTransitionsAndOffsetOverflow checks non-hour timezone transitions and rejects overflowing offsets.
func TestResolveWallNonHourTransitionsAndOffsetOverflow(t *testing.T) {
	for _, tc := range []struct {
		date, clock, zone, utc string
		offset                 *int
	}{
		{"2024-04-07", "01:45", "Australia/Lord_Howe", "2024-04-06T14:45:00Z", ptr(660)},
		{"2024-04-07", "01:45", "Australia/Lord_Howe", "2024-04-06T15:15:00Z", ptr(630)},
		{"2024-04-07", "01:45", "Australia/Lord_Howe", "", nil},
		{"2024-10-06", "02:15", "Australia/Lord_Howe", "", nil},
		{"2011-12-30", "12:00", "Pacific/Apia", "", nil},
		{"1986-01-01", "00:15", "Asia/Kathmandu", "1985-12-31T18:30:00Z", ptr(345)},
		{"2026-10-19", "12:00", "Asia/Kolkata", "", ptr(math.MaxInt)},
		{"2026-10-19", "12:00", "Asia/Kolkata", "", ptr(math.MinInt)},
		// (2^(wordSize-2)+330)*60 wraps to 19800, exactly IST's seconds.
		{"2026-10-19", "12:00", "Asia/Kolkata", "", ptr(math.MaxInt/2 + 331)},
	} {
		got, err := ResolveWall(tc.date, tc.clock, tc.zone, tc.offset)
		if tc.utc == "" {
			if err == nil {
				t.Fatalf("invalid/ambiguous wall time accepted: %+v -> %s", tc, got)
			}
		} else if err != nil || got.UTC().Format(time.RFC3339) != tc.utc {
			t.Fatalf("wall-time reference %+v -> %s, %v", tc, got, err)
		}
	}
}

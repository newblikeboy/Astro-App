package platform

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestPanchangCalendar checks weekly and monthly Panchang calendar results.
func TestPanchangCalendar(t *testing.T) {
	a := testApp(t)
	h := a.Handler()
	// get dispatches a read-only calendar request through the test HTTP handler.
	get := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/panchang/calendar?"+query, nil))
		return w
	}
	for _, q := range []string{"", "date=2026-10-18&place_id=delhi&view=year", "date=2026-02-30&place_id=delhi&view=month", "date=2026-10-18&place_id=london&view=month", "date=2026-10-18&place_id=delhi&view=month&sunrise=fake", "date=2026-10-18&place_id=delhi&view=month&view=week", "date=1939-12-31&place_id=delhi&view=month", "date=1939-12-31&place_id=delhi&view=week", "date=1940-02-30&place_id=delhi&view=month", "date=2099-12-31&place_id=delhi&view=month"} {
		if w := get(q); w.Code != 400 {
			t.Fatalf("%q status %d", q, w.Code)
		}
	}
	for _, tc := range []struct {
		q           string
		count       int
		first, last string
	}{
		{"date=2026-10-18&place_id=delhi&view=week", 7, "2026-10-18", "2026-10-24"},
		{"date=2024-02-19&place_id=delhi&view=month", 29, "2024-02-01", "2024-02-29"},
		{"date=1972-01-02&place_id=delhi&view=month", 31, "1972-01-01", "1972-01-31"},
		{"date=1940-01-01&place_id=delhi&view=month", 31, "1940-01-01", "1940-01-31"},
		{"date=1940-01-01&place_id=delhi&view=week", 6, "1940-01-01", "1940-01-06"},
		{"date=1940-02-19&place_id=delhi&view=month", 29, "1940-02-01", "1940-02-29"},
		{"date=1940-02-29&place_id=delhi&view=week", 7, "1940-02-25", "1940-03-02"},
		{"date=2099-12-30&place_id=delhi&view=week", 4, "2099-12-27", "2099-12-30"},
	} {
		w := get(tc.q)
		if w.Code != 200 {
			t.Fatalf("%s %d %s", tc.q, w.Code, w.Body.String())
		}
		var result struct {
			Days []calendarDay `json:"days"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Days) != tc.count || result.Days[0].Date != tc.first || result.Days[len(result.Days)-1].Date != tc.last {
			t.Fatalf("incorrect calendar range %s", tc.q)
		}
		for _, d := range result.Days {
			if len(d.Tithi) == 0 || len(d.Nakshatra) == 0 || d.Calendar.Amanta == "" || !d.Sunset.After(d.Sunrise) {
				t.Fatal("incomplete calendar day")
			}
		}
		if cached := get(tc.q); cached.Body.String() != w.Body.String() {
			t.Fatal("cached calendar differs")
		}
	}
}

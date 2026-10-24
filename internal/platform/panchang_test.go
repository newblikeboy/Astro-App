package platform

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// TestPanchangHTTP checks Panchang endpoint validation and response contents.
func TestPanchangHTTP(t *testing.T) {
	a := testApp(t)
	h := a.Handler()
	// get dispatches a read-only Panchang request and records its HTTP response.
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	for _, q := range []string{
		"", "date=2026-10-18", "date=2026-10-18&place_id=unknown", "date=2026-10-18&place_id=london", "date=2026-02-30&place_id=delhi", "date=2026-2-01&place_id=delhi", "date=1939-12-31&place_id=delhi", "date=2099-12-31&place_id=delhi", "date=2026-10-18&place_id=delhi&sunrise=fake", "date=2026-10-18&date=2026-10-19&place_id=delhi", "date=2026-10-18&place_id=delhi&lat=30",
	} {
		if w := get("/api/panchang?" + q); w.Code != 400 {
			t.Fatalf("query %q: got %d", q, w.Code)
		}
	}
	valid := "/api/panchang?date=2026-10-18&place_id=delhi"
	w := get(valid)
	if w.Code != 200 {
		t.Fatalf("calculation failed %s", w.Body.String())
	}
	var p dailyResult
	if err := json.Unmarshal(w.Body.Bytes(), &p); err != nil {
		t.Fatal(err)
	}
	if p.Place.ID != "delhi" || p.Zone != "Asia/Kolkata" || p.Tithi[0].Name != "Shukla Saptami" || p.Settings["sunrise_convention"] != "upper-limb" {
		t.Fatalf("unexpected result %+v", p)
	}
	if w.Header().Get("Cache-Control") == "" {
		t.Fatal("cache header missing")
	}
	if second := get(valid); second.Body.String() != w.Body.String() {
		t.Fatal("cache changed result")
	}
	other := get(valid + "&sunrise=hindu")
	var hindu dailyResult
	_ = json.Unmarshal(other.Body.Bytes(), &hindu)
	if !hindu.Sunrise.After(p.Sunrise) || !hindu.Sunset.Before(p.Sunset) {
		t.Fatal("sunrise conventions not independent")
	}
	if len(a.daily.entries) != 2 {
		t.Fatal("cache key omitted convention")
	}
	// Error paths must not expose filesystem paths or library internals.
	a.Config.EpheDir = t.TempDir()
	failure := get("/api/panchang?date=2026-10-19&place_id=delhi")
	if failure.Code != 503 || failure.Body.String() != "{\"error\":\"Panchang calculation unavailable; please retry\"}\n" {
		t.Fatalf("unexpected failure %d %s", failure.Code, failure.Body.String())
	}
}

// TestPanchangHTTPHistoricalDateRange checks the supported historical date range at the Panchang endpoint.
func TestPanchangHTTPHistoricalDateRange(t *testing.T) {
	a := testApp(t)
	h := a.Handler()
	for _, date := range []string{"1940-01-01", "1940-02-29", "1972-01-01"} {
		t.Run(date, func(t *testing.T) {
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest("GET", "/api/panchang?date="+date+"&place_id=delhi", nil))
			if w.Code != 200 {
				t.Fatalf("historical date rejected: %d %s", w.Code, w.Body.String())
			}
			var result dailyResult
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Date != date || result.Sunrise.Format("2006-01-02") != date || result.Place.ID != "delhi" || len(result.Tithi) == 0 || result.Details == nil || !result.NextSunrise.After(result.Sunset) {
				t.Fatalf("incomplete historical day %+v", result)
			}
		})
	}
}

// TestPanchangCacheBoundExpiryAndRateLimit checks bounded Panchang caching, expiry and request throttling.
func TestPanchangCacheBoundExpiryAndRateLimit(t *testing.T) {
	a := testApp(t)
	// Handler called directly here to test the cache cap independently of the
	// network rate limiter; no persistent store or customer records involved.
	for i := 0; i < 130; i++ {
		date := time.Date(2026, 1, 1+i, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
		w := httptest.NewRecorder()
		a.dailyPanchang(w, httptest.NewRequest("GET", "/api/panchang?date="+date+"&place_id=delhi", nil))
		if w.Code != 200 {
			t.Fatalf("calculation %d failed: %s", i, w.Body.String())
		}
	}
	if len(a.daily.entries) != 128 {
		t.Fatalf("cache not bounded: %d", len(a.daily.entries))
	}
	if _, ok := a.daily.entries["delhi/2026-01-01/upper-limb"]; ok {
		t.Fatal("old cache entry retained")
	}
	a.Config.EpheDir = t.TempDir()
	oldNow := a.Now()
	a.Now = func() time.Time { return oldNow.Add(7 * time.Hour) }
	w := httptest.NewRecorder()
	a.dailyPanchang(w, httptest.NewRequest("GET", "/api/panchang?date=2026-05-10&place_id=delhi", nil))
	if w.Code != 503 {
		t.Fatal("expired cache returned without recalculation")
	}
	a = testApp(t)
	h := a.Handler()
	for i := 0; i < 61; i++ {
		r := httptest.NewRequest("GET", "/api/panchang?date=2026-10-18&place_id=delhi", nil)
		r.RemoteAddr = "192.0.2.1:1234"
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 200
		if i == 60 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("request %d status %d want %d", i, w.Code, want)
		}
	}
	if !a.limits.allow("calc:192.0.2.1", 20, time.Minute) {
		t.Fatal("daily browsing consumed birth-chart allowance")
	}
}

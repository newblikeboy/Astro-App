package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"net/http/httptest"
	"testing"
)

// TestVivahHTTP checks strict month/place/query validation, independent convention
// caching, public result metadata and sanitised native failures.
func TestVivahHTTP(t *testing.T) {
	a := testApp(t)
	// get records a read-only marriage endpoint response in isolated test storage.
	get := func(query string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		a.vivahMuhurat(w, httptest.NewRequest("GET", "/api/panchang/vivah?"+query, nil))
		return w
	}
	for _, query := range []string{"", "month=2026-11", "month=2026-11&place_id=unknown", "month=2026-11&place_id=london", "month=2026-13&place_id=delhi", "month=2026-1&place_id=delhi", "month=1939-12&place_id=delhi", "month=2100-01&place_id=delhi", "month=2026-11&place_id=delhi&sunrise=fake", "month=2026-11&place_id=delhi&month=2026-12", "month=2026-11&place_id=delhi&lat=30"} {
		if w := get(query); w.Code != 400 {
			t.Fatalf("accepted %s: %d", query, w.Code)
		}
	}
	query := "month=2026-11&place_id=delhi"
	w := get(query)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("failed: %d %s", w.Code, w.Body.String())
	}
	var result struct {
		swisseph.VivahMonth
		Place Place `json:"place"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil || result.Place.ID != "delhi" || result.Month != "2026-11" || len(result.Days) != 30 {
		t.Fatalf("invalid result: %v %+v", err, result)
	}
	if result.Settings["rules"] != "m4y-vivah-shortlist-v2" || result.Settings["guruji_pending"] == "" {
		t.Fatal("Guruji review scope is missing")
	}
	for _, day := range result.Days {
		if day.Date == "2026-11-21" && len(day.Windows) > 0 {
			t.Fatal("previous Saturday/Revati audit example remains available")
		}
		for _, window := range day.Windows {
			if window.Lagna == "" {
				t.Fatal("marriage Lagna evidence is missing")
			}
		}
	}
	if again := get(query); again.Body.String() != w.Body.String() {
		t.Fatal("cache response changed")
	}
	if hindu := get(query + "&sunrise=hindu"); hindu.Code != 200 || hindu.Body.String() == w.Body.String() || len(a.sky.entries) != 2 {
		t.Fatal("conventions share a cache entry")
	}
	a.Config.EpheDir = t.TempDir()
	if failure := get("month=2026-12&place_id=delhi"); failure.Code != 503 || failure.Body.String() != "{\"error\":\"Marriage calculations unavailable. Please try again.\"}\n" {
		t.Fatalf("internal error escaped: %d %s", failure.Code, failure.Body.String())
	}
	h := a.Handler()
	for i := 0; i < 13; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/panchang/vivah?"+query, nil))
		want := 200
		if i == 12 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("request %d: expected %d, got %d", i+1, want, w.Code)
		}
	}
}

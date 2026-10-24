package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestSkyHTTPValidationAndResponses checks input validation and response shapes for the public sky tools.
func TestSkyHTTPValidationAndResponses(t *testing.T) {
	a := testApp(t)
	h := a.Handler()
	// get dispatches a read-only sky request and records its HTTP response.
	get := func(path string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		return w
	}
	for _, path := range []string{
		"/positions?place_id=delhi&date=2026-02-30&time=12:00", "/positions?place_id=delhi&date=2026-10-18&time=24:00", "/positions?place_id=delhi&date=1939-12-31&time=12:00", "/positions?place_id=delhi&date=2026-10-18&time=12:00&year=2026", "/positions?place_id=london&date=2026-10-18&time=12:00", "/positions?place_id=delhi&date=2026-10-18&time=12:00&time=10:00",
		"/transits?year=1939&place_id=delhi", "/transits?year=2100&place_id=delhi", "/transits?year=02026&place_id=delhi", "/transits?year=2026&place_id=unknown", "/eclipses?year=2026&place_id=delhi&latitude=30", "/eclipses?year=2026&place_id=delhi&year=2025", "/eclipses?place_id=delhi", "/transits?year=2026&place_id=delhi&date=2026-10-18",
		"/combustion?year=1939&place_id=delhi", "/combustion?year=2100&place_id=delhi", "/combustion?year=2026&year=2025&place_id=delhi", "/combustion?year=2026&place_id=london", "/combustion?year=2026&place_id=delhi&time=12:00",
	} {
		if w := get("/api/sky" + path); w.Code != 400 {
			t.Fatalf("%s got %d %s", path, w.Code, w.Body.String())
		}
	}
	if w := get("/api/sky/fake?year=2026&place_id=delhi"); w.Code != 404 {
		t.Fatal("unknown tool accepted")
	}
	for _, path := range []string{"positions?place_id=delhi&date=2026-10-18&time=12:00", "transits?year=2026&place_id=delhi", "combustion?year=2026&place_id=delhi", "eclipses?year=2026&place_id=delhi"} {
		w := get("/api/sky/" + path)
		if w.Code != 200 {
			t.Fatalf("%s %d %s", path, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") == "" {
			t.Fatal("missing public cache header")
		}
		var value map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &value); err != nil {
			t.Fatal(err)
		}
		if _, ok := value["place_id"]; ok {
			t.Fatal("unexpected private location data")
		}
		if strings.Contains(w.Body.String(), "\"house\"") {
			t.Fatal("public sky positions must not pretend to be natal houses")
		}
		if again := get("/api/sky/" + path); again.Body.String() != w.Body.String() {
			t.Fatal("unstable cache")
		}
	}
	// Geocentric positions/transits share one cache entry across India. Eclipse
	// visibility must be keyed by place and must never share this cache key.
	for _, path := range []string{"positions?place_id=mumbai&date=2026-10-18&time=12:00", "transits?year=2026&place_id=mumbai", "combustion?year=2026&place_id=mumbai", "eclipses?year=2026&place_id=mumbai"} {
		if w := get("/api/sky/" + path); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	if len(a.sky.entries) != 5 {
		t.Fatalf("incorrect sky cache keys: %d", len(a.sky.entries))
	}
	a.Config.EpheDir = t.TempDir()
	w := get("/api/sky/eclipses?year=2027&place_id=delhi")
	if w.Code != 503 || strings.Contains(w.Body.String(), a.Config.EpheDir) {
		t.Fatalf("unsafe failure: %d %s", w.Code, w.Body.String())
	}
}

// TestSkyHTTPHistoricalDateRange checks historical date support in the public sky endpoints.
func TestSkyHTTPHistoricalDateRange(t *testing.T) {
	a := testApp(t)
	h := a.Handler()
	// get requests a historical sky tool through the shared test handler.
	get := func(path string) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", "/api/sky/"+path, nil))
		return w
	}
	// The first civil midnight is in UTC1939. Accept the civil date instead
	// of accidentally rejecting its first five and a half hours after parsing.
	for _, at := range []string{"1940-01-01 00:00", "1940-01-01 12:00", "1940-02-29 12:00", "1972-01-01 12:00"} {
		t.Run(at, func(t *testing.T) {
			w := get("positions?place_id=delhi&date=" + at[:10] + "&time=" + at[11:])
			if w.Code != 200 {
				t.Fatalf("historical positions rejected: %d %s", w.Code, w.Body.String())
			}
			var result swisseph.PlanetPositions
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			want, err := time.ParseInLocation("2006-01-02 15:04", at, mustIndiaZone())
			if err != nil {
				t.Fatal(err)
			}
			if !result.At.Equal(want) || len(result.Planets) != 9 {
				t.Fatalf("incorrect historical position instant %s / %s, planets=%d", result.At, want, len(result.Planets))
			}
			for _, p := range result.Planets {
				if !(p.Longitude >= 0 && p.Longitude < 360) || p.RashiNumber < 1 || p.RashiNumber > 12 {
					t.Fatalf("invalid historical position %+v", p)
				}
			}
		})
	}
	for _, tool := range []string{"transits", "combustion", "eclipses"} {
		t.Run(tool, func(t *testing.T) {
			w := get(tool + "?place_id=delhi&year=1940")
			if w.Code != 200 {
				t.Fatalf("historical year rejected: %d %s", w.Code, w.Body.String())
			}
			var result struct {
				Year   int `json:"year"`
				Events []struct {
					At      time.Time `json:"at"`
					Maximum time.Time `json:"maximum"`
				} `json:"events"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if result.Year != 1940 || len(result.Events) == 0 {
				t.Fatalf("incomplete historical annual result: %s", w.Body.String())
			}
			for _, event := range result.Events {
				at := event.At
				if tool == "eclipses" {
					at = event.Maximum
				}
				if at.IsZero() || at.In(mustIndiaZone()).Year() != 1940 {
					t.Fatalf("event escaped historical civil year: %s", at)
				}
			}
		})
	}
	for _, path := range []string{
		"positions?place_id=delhi&date=1940-02-30&time=12:00",
		"positions?place_id=delhi&date=1939-12-31&time=23:59",
		"eclipses?place_id=delhi&year=1939",
	} {
		if w := get(path); w.Code != 400 {
			t.Fatalf("invalid historical query accepted: %s status%d", path, w.Code)
		}
	}
}

// TestSkyCacheBoundsExpiryAndCancellation checks bounded sky caches, expiry and cancelled requests.
func TestSkyCacheBoundsExpiryAndCancellation(t *testing.T) {
	a := testApp(t)
	calls := 0
	// calculate counts cache misses and returns a distinct value for cache expiry/eviction assertions.
	calculate := func() (any, error) { calls++; return map[string]int{"value": calls}, nil }
	for i := 0; i < 130; i++ {
		if _, err := a.cachedSky(context.Background(), fmt.Sprint(i), calculate); err != nil {
			t.Fatal(err)
		}
	}
	if len(a.sky.entries) != 128 {
		t.Fatal("unbounded sky cache")
	}
	if _, ok := a.sky.entries["0"]; ok {
		t.Fatal("cache eviction failed")
	}
	_, _ = a.cachedSky(context.Background(), "129", calculate)
	if calls != 130 {
		t.Fatal("cache miss on valid entry")
	}
	now := a.Now()
	a.Now = func() time.Time { return now.Add(7 * time.Hour) }
	_, _ = a.cachedSky(context.Background(), "129", calculate)
	if calls != 131 {
		t.Fatal("expired cache used")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := a.cachedSky(ctx, "cancelled", calculate); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, err := a.cachedSky(context.Background(), "large", func() (any, error) { return strings.Repeat("x", 513*1024), nil }); err == nil {
		t.Fatal("unbounded response cache size")
	}
}

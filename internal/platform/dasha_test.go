package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestDashaBranchAPIPrivacyAndFiveLevels checks five-level branch responses without persisting guest birth details.
func TestDashaBranchAPIPrivacyAndFiveLevels(t *testing.T) {
	a := testApp(t)
	b := testBirth()
	b.Time = "10:30:47"
	var before, after []byte
	_ = a.Store.View(context.Background(), func(s *State) error { before, _ = json.Marshal(s); return nil })
	report, err := calculate(a.Config, b, nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Level     int                    `json:"level"`
		Path      []int                  `json:"path"`
		Periods   []swisseph.DashaPeriod `json:"periods"`
		Parent    *swisseph.DashaPeriod  `json:"parent"`
		Current   []swisseph.DashaStep   `json:"current_path"`
		Ancestors []swisseph.DashaStep   `json:"ancestors"`
		Zone      string                 `json:"timezone"`
		Method    string                 `json:"method"`
	}
	path := []int{}
	for level := 1; level <= 5; level++ {
		w := request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": b, "path": path, "as_of": a.Now().Format(time.RFC3339Nano)}, nil, "")
		if w.Code != 200 {
			t.Fatalf("level %d: %d %s", level, w.Code, w.Body)
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out.Level != level || len(out.Ancestors) != level-1 || len(out.Periods) != 9 || len(out.Current) != 5 || out.Zone != "Asia/Kolkata" || out.Method != swisseph.VimshottariMethod {
			t.Fatalf("bad Dasha metadata %+v", out)
		}
		if w.Header().Get("Cache-Control") != "no-store" || len(w.Result().Cookies()) > 0 || strings.Contains(w.Body.String(), "antardashas") {
			t.Fatal("branch privacy or bounded payload failed")
		}
		_, want, e := swisseph.DashaBranch(*report.Dasha, path)
		if e != nil {
			t.Fatal(e)
		}
		for i, p := range out.Periods {
			if p.Lord != want[i].Lord || !p.Start.Equal(want[i].Start) || !p.End.Equal(want[i].End) {
				t.Fatalf("native branch mismatch level%d/%d", level, i)
			}
		}
		if level > 1 && (out.Parent == nil || !out.Periods[0].Start.Equal(out.Parent.Start) || !out.Periods[8].End.Equal(out.Parent.End)) {
			t.Fatal("parent not partitioned")
		}
		path = append(path, 0)
	}
	_ = a.Store.View(context.Background(), func(s *State) error { after, _ = json.Marshal(s); return nil })
	if string(before) != string(after) {
		t.Fatal("Dasha navigation mutated application state")
	}
}

// TestDashaBranchAPIValidationAndLimits rejects invalid Dasha branch inputs and unsupported depths.
func TestDashaBranchAPIValidationAndLimits(t *testing.T) {
	a := testApp(t)
	b := testBirth()
	for _, path := range [][]int{{-1}, {18}, {0, 9}, {0, 0, 0, 0, 0}} {
		if w := request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": b, "path": path}, nil, ""); w.Code != 422 {
			t.Fatal("bad path accepted", path, w.Code)
		}
	}
	for _, asOf := range []string{"bad", "1940-01-01T00:00:00Z", a.Now().Add(time.Hour).Format(time.RFC3339)} {
		if w := request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": b, "path": []int{}, "as_of": asOf}, nil, ""); w.Code != 422 {
			t.Fatal("bad as-of accepted", asOf, w.Code)
		}
	}
	for _, change := range []func(*Birth){func(b *Birth) { b.Unknown = true }, func(b *Birth) { b.Time = "" }, func(b *Birth) { b.Date = "1939-12-31" }, func(b *Birth) { b.PlaceID = "invented" }, func(b *Birth) { b.Time = "10:30:60" }, func(b *Birth) { b.Date = "2024-11-03"; b.Time = "01:30:31"; b.PlaceID = "new-york" }} {
		bad := b
		change(&bad)
		if w := request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": bad, "path": []int{}}, nil, ""); w.Code != 422 {
			t.Fatal("invalid birth accepted", bad, w.Code)
		}
	}
	w := request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": b, "path": []int{}, "longitude": 20, "start": "2000-01-01"}, nil, "")
	if w.Code != 400 {
		t.Fatal("untrusted calculation accepted")
	}
	a.Config.Production = true
	if w = request(t, a, "POST", "/api/kundli/dasha", map[string]any{"birth": b, "path": []int{}}, nil, ""); w.Code != 503 {
		t.Fatal("licence gate bypassed")
	}
	a.Config.Production = false
	a.limits = newLimiter()
	for i := 0; i < 60; i++ {
		if !a.limits.allow("dasha:192.0.2.1", 60, time.Minute) {
			t.Fatal("rate setup")
		}
	}
	r := httptest.NewRequest("POST", "/api/kundli/dasha", strings.NewReader(`{}`))
	r.RemoteAddr = "192.0.2.1:1234"
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatalf("Dasha rate limit missing: %d", w.Code)
	}
	a.limits = newLimiter()
	r = httptest.NewRequest("POST", "/api/kundli/dasha", strings.NewReader(`{}`))
	ctx, cancel := context.WithCancel(r.Context())
	cancel()
	r = r.WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "periods") {
		t.Fatal("canceled request returned periods")
	}
}

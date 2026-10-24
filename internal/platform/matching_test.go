package platform

import (
	"astroapp/backend/internal/swisseph"
	"bytes"
	"context"
	"encoding/json"
	"math"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestMatchingParticipantMangalGeometry checks Mangal reference-house geometry for matching participants.
func TestMatchingParticipantMangalGeometry(t *testing.T) {
	// position creates a position safely inside the requested sign for Mangal geometry tests.
	position := func(sign int) swisseph.ZodiacPosition {
		return swisseph.ZodiacPosition{Longitude: float64(sign-1)*30 + 10, RashiNumber: sign}
	}
	flagged := map[int]bool{1: true, 2: true, 4: true, 7: true, 8: true, 12: true}
	for mars := 1; mars <= 12; mars++ {
		report := &Report{Chart: &swisseph.Kundali{Lagna: position(12), Planets: []swisseph.Graha{
			{Name: "Moon", ZodiacPosition: position(1)}, {Name: "Mars", ZodiacPosition: position(mars)},
		}}}
		got, err := matchingParticipant(report)
		lagnaHouse := mars%12 + 1
		if err != nil || got.MarsChecks.FromLagna != lagnaHouse || got.MarsChecks.FromMoon != mars ||
			got.MarsChecks.LagnaFlagged != flagged[lagnaHouse] || got.MarsChecks.MoonFlagged != flagged[mars] {
			t.Fatalf("Mars sign %d: %+v, %v", mars, got.MarsChecks, err)
		}
	}
	for _, invalid := range []swisseph.ZodiacPosition{
		{Longitude: math.NaN(), RashiNumber: 1}, {Longitude: math.Inf(1), RashiNumber: 1},
		{Longitude: -1, RashiNumber: 1}, {Longitude: 360, RashiNumber: 12},
		{Longitude: 10, RashiNumber: 0}, {Longitude: 10, RashiNumber: 2},
	} {
		for _, index := range []int{0, 1, 2} {
			chart := swisseph.Kundali{Lagna: position(12), Planets: []swisseph.Graha{
				{Name: "Moon", ZodiacPosition: position(1)}, {Name: "Mars", ZodiacPosition: position(7)},
			}}
			if index == 2 {
				chart.Lagna = invalid
			} else {
				chart.Planets[index].ZodiacPosition = invalid
			}
			if _, err := matchingParticipant(&Report{Chart: &chart}); err == nil {
				t.Fatalf("malformed position accepted: index %d, %+v", index, invalid)
			}
		}
	}
}

// TestMatchingHTTPStatelessNativeResult checks that guest matching returns the native result without saving participant data.
func TestMatchingHTTPStatelessNativeResult(t *testing.T) {
	a := testApp(t)
	bride := Birth{Name: "Bride Fixture", Date: "1995-08-15", Time: "10:30", PlaceID: "delhi"}
	groom := Birth{Name: "Groom Fixture", Date: "1992-02-29", Time: "18:15", PlaceID: "mumbai"}
	var before, after []byte
	_ = a.Store.View(context.Background(), func(s *State) error { before, _ = json.Marshal(s); return nil })
	w := request(t, a, "POST", "/api/kundli/match", MatchingRequest{Bride: bride, Groom: groom}, nil, "")
	if w.Code != 200 {
		t.Fatalf("matching status %d: %s", w.Code, w.Body)
	}
	var got MatchingResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Max != 36 || len(got.Factors) != 8 || got.Total < 0 || got.Total > 36 || !got.CalculatedAt.Equal(a.Now()) {
		t.Fatalf("incomplete matching result: %+v", got)
	}
	if got.Bride.Birth.Name != bride.Name || got.Groom.Place.ID != groom.PlaceID || got.Bride.Moon.Longitude == got.Groom.Moon.Longitude {
		t.Fatal("partner inputs were mixed up")
	}
	br, err := calculate(a.Config, bride, nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	gr, err := calculate(a.Config, groom, nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	want, err := swisseph.CalculateMatching(*br.Chart, *gr.Chart)
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != want.Total || got.Method != want.Method {
		t.Fatal("HTTP result differs from native matching")
	}
	sum := 0.0
	for i, factor := range got.Factors {
		if factor.Score != want.Factors[i].Score || factor.Max != float64(i+1) {
			t.Fatal("factor arithmetic changed during serialization")
		}
		sum += factor.Score
	}
	if sum != got.Total || math.Mod(got.Total*2, 1) != 0 {
		t.Fatal("score lost its half-point arithmetic")
	}
	for _, person := range []MatchingParticipant{got.Bride, got.Groom} {
		if person.MarsChecks.FromLagna < 1 || person.MarsChecks.FromLagna > 12 || person.MarsChecks.FromMoon < 1 || person.MarsChecks.FromMoon > 12 {
			t.Fatal("missing separate Mangal geometry")
		}
	}
	_ = a.Store.View(context.Background(), func(s *State) error { after, _ = json.Marshal(s); return nil })
	if string(before) != string(after) {
		t.Fatal("matching wrote to application state")
	}
	if w.Header().Get("Cache-Control") != "no-store" || len(w.Result().Cookies()) != 0 {
		t.Fatal("personal results were cached or created a session")
	}
	var root map[string]json.RawMessage
	_ = json.Unmarshal(w.Body.Bytes(), &root)
	if strings.Contains(w.Body.String(), "user_id") || root["rules"] != nil || strings.Contains(w.Body.String(), "payment_key") {
		t.Fatal("unnecessary account/content/provider data exposed")
	}
	if got.Bride.Chart == nil || got.Groom.Navamsa == nil || got.Bride.Dasha == nil || len(got.Bride.Chart.Planets) != 9 || got.Mangal.Method != swisseph.MangalMethod || len(got.Mangal.Bride.References) != 3 || got.Summary.Band == "" {
		t.Fatal("expanded native comparison incomplete")
	}
}

// TestMatchingHTTPInputValidation rejects invalid participant birth inputs at the matching endpoint.
func TestMatchingHTTPInputValidation(t *testing.T) {
	valid := testBirth()
	for _, tc := range []struct {
		name, role string
		change     func(*Birth)
	}{
		{"unknown bride time", "bride", func(b *Birth) { b.Unknown = true }},
		{"missing groom time", "groom", func(b *Birth) { b.Time = "" }},
		{"invalid bride clock", "bride", func(b *Birth) { b.Time = "24:00" }},
		{"untrusted groom place", "groom", func(b *Birth) { b.PlaceID = "not-a-trusted-city" }},
		{"future groom", "groom", func(b *Birth) { b.Date = "2027-01-01" }},
		{"pre-1940 bride", "bride", func(b *Birth) { b.Date = "1939-12-31" }},
		{"invalid groom leap day", "groom", func(b *Birth) { b.Date = "1993-02-29" }},
		{"missing bride name", "bride", func(b *Birth) { b.Name = "" }},
		{"ambiguous groom wall time", "groom", func(b *Birth) { b.Date = "2024-11-03"; b.Time = "01:30"; b.PlaceID = "new-york" }},
		{"overflowing bride offset", "bride", func(b *Birth) { b.Offset = ptr(math.MaxInt) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := testApp(t)
			in := MatchingRequest{Bride: valid, Groom: valid}
			if tc.role == "bride" {
				tc.change(&in.Bride)
			} else {
				tc.change(&in.Groom)
			}
			w := request(t, a, "POST", "/api/kundli/match", in, nil, "")
			var body map[string]string
			_ = json.Unmarshal(w.Body.Bytes(), &body)
			if w.Code != 422 || body["person"] != tc.role || body["error"] == "" {
				t.Fatalf("validation error %d: %s", w.Code, w.Body)
			}
			if strings.Contains(w.Body.String(), "\"total\"") {
				t.Fatal("invalid input returned a score")
			}
		})
	}
	// Historical opening midnight and an explicit repeated-time offset remain valid.
	for _, birth := range []Birth{
		{Name: "Historical Fixture", Date: "1940-01-01", Time: "00:00", PlaceID: "delhi"},
		{Name: "Repeated Time Fixture", Date: "2024-11-03", Time: "01:30", PlaceID: "new-york", Offset: ptr(-240)},
	} {
		a := testApp(t)
		if w := request(t, a, "POST", "/api/kundli/match", MatchingRequest{Bride: valid, Groom: birth}, nil, ""); w.Code != 200 {
			t.Fatalf("valid edge case %d: %s", w.Code, w.Body)
		}
	}
}

// TestMatchingHTTPStrictBodyAndLicence checks strict request decoding and ephemeris licence enforcement for matching.
func TestMatchingHTTPStrictBodyAndLicence(t *testing.T) {
	a := testApp(t)
	for _, body := range []string{
		`{"bride":{"latitude":10},"groom":{}}`,
		`{"bride":{},"groom":{},"total":36}`,
		`{"bride":{},"groom":{}} {}`,
	} {
		r := httptest.NewRequest("POST", "/api/kundli/match", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("untrusted body accepted: %d %s", w.Code, w.Body)
		}
	}
	a.Config.Production = true
	w := request(t, a, "POST", "/api/kundli/match", MatchingRequest{Bride: testBirth(), Groom: testBirth()}, nil, "")
	if w.Code != 503 || strings.Contains(w.Body.String(), "\"total\"") {
		t.Fatal("production licence gate bypassed")
	}
}

// TestMatchingHTTPRateLimitAndCancellation checks matching request throttling and cancellation.
func TestMatchingHTTPRateLimitAndCancellation(t *testing.T) {
	a := testApp(t)
	in := MatchingRequest{Bride: testBirth(), Groom: testBirth()}
	for i := 0; i < 10; i++ {
		w := request(t, a, "POST", "/api/kundli/match", in, nil, "")
		if w.Code != 200 {
			t.Fatalf("matching request %d: %d %s", i+1, w.Code, w.Body)
		}
	}
	if w := request(t, a, "POST", "/api/kundli/match", in, nil, ""); w.Code != 429 {
		t.Fatal("two-chart endpoint is not rate limited")
	}
	// A canceled request must never calculate or serialize a personal score.
	canceled := testApp(t)
	body, _ := json.Marshal(in)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("POST", "/api/kundli/match", bytes.NewReader(body)).WithContext(ctx)
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	canceled.Handler().ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), "\"total\"") || strings.Contains(w.Body.String(), "\"factors\"") {
		t.Fatal("canceled matching request returned a score")
	}
}

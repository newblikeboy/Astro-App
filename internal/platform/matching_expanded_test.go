package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type matchingPDFStub struct {
	calls    int
	got      *MatchingResponse
	language string
	fail     bool
}

// TestMatchingSharedTranslations checks shared matching translations for consistent keys and labels.
func TestMatchingSharedTranslations(t *testing.T) {
	var catalog map[string]string
	if err := json.Unmarshal(matchingLocaleJSON, &catalog); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for hi, en := range catalog {
		if strings.ContainsAny(hi, "?\ufffd") || strings.Contains(en, "\ufffd") || seen[en] {
			t.Fatalf("damaged or ambiguous matching translation: %q", en)
		}
		seen[en] = true
	}
	for _, values := range []map[string]string{matchingFactorWhy, matchingMangalRuleText, matchingMangalStatusText, matchingBandText} {
		for _, en := range values {
			if matchingText("hi", en) == en {
				t.Fatalf("missing Hindi matching explanation: %s", en)
			}
		}
	}
	if got := matchingText("hi", "High Mangal level"); got != "मंगल का उच्च स्तर" {
		t.Fatalf("damaged Mangal label: %q", got)
	}
}

// RenderMatching captures the recalculated PDF input/language and can simulate rendering failure.
func (p *matchingPDFStub) RenderMatching(ctx context.Context, r *MatchingResponse, language string) ([]byte, error) {
	p.calls++
	p.got = r
	p.language = language
	if p.fail {
		return nil, fmt.Errorf("private internal failure")
	}
	return []byte("%PDF-1.4\nmatching-fixture"), nil
}

// TestMatchingPDFRecalculationPrivacyAndFailures checks server-side PDF recalculation, guest privacy and renderer failures.
func TestMatchingPDFRecalculationPrivacyAndFailures(t *testing.T) {
	a := testApp(t)
	renderer := &matchingPDFStub{}
	a.MatchingPDF = renderer
	b := testBirth()
	b.Time = "10:30:47"
	in := map[string]any{"bride": b, "groom": testBirth(), "language": "hi"}
	var before, after []byte
	_ = a.Store.View(context.Background(), func(s *State) error { before, _ = json.Marshal(s); return nil })
	w := request(t, a, "POST", "/api/kundli/match/pdf", in, nil, "")
	if w.Code != 200 || !strings.HasPrefix(w.Body.String(), "%PDF-") || renderer.calls != 1 || renderer.language != "hi" || renderer.got.Bride.Chart.UTC != "1995-08-15T05:00:47Z" {
		t.Fatalf("PDF %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Content-Type") != "application/pdf" || !strings.Contains(w.Header().Get("Content-Disposition"), "matching-hi.pdf") || w.Header().Get("Cache-Control") != "no-store" || len(w.Result().Cookies()) > 0 {
		t.Fatal("private file headers invalid")
	}
	_ = a.Store.View(context.Background(), func(s *State) error { after, _ = json.Marshal(s); return nil })
	if !bytes.Equal(before, after) {
		t.Fatal("PDF export saved records")
	}
	// The downloaded score is recomputed; client reports/charts are rejected.
	in["total"] = 36
	if got := request(t, a, "POST", "/api/kundli/match/pdf", in, nil, ""); got.Code != 400 || renderer.calls != 1 {
		t.Fatal("client score reached renderer")
	}
	delete(in, "total")
	in["language"] = "fr"
	if got := request(t, a, "POST", "/api/kundli/match/pdf", in, nil, ""); got.Code != 400 {
		t.Fatal("unsupported report language accepted")
	}
	fresh := testApp(t)
	broken := &matchingPDFStub{fail: true}
	fresh.MatchingPDF = broken
	in["language"] = "en"
	got := request(t, fresh, "POST", "/api/kundli/match/pdf", in, nil, "")
	if got.Code != 503 || strings.Contains(got.Body.String(), "private internal") {
		t.Fatal("render failure leaked internals")
	}
	busy := testApp(t)
	busy.MatchingPDF = &matchingPDFStub{}
	busy.matchingPDFSlots <- struct{}{}
	if w := request(t, busy, "POST", "/api/kundli/match/pdf", in, nil, ""); w.Code != 429 {
		t.Fatal("PDF concurrency unbounded")
	}
}

// TestMatchingProfilesOwnershipAndOptIn checks ownership and explicit opt-in when selecting saved matching profiles.
func TestMatchingProfilesOwnershipAndOptIn(t *testing.T) {
	a := testApp(t)
	if w := request(t, a, "GET", "/api/kundli/match/profiles", nil, nil, ""); w.Code != 401 {
		t.Fatal("saved profiles public")
	}
	putUser(t, a, "owner")
	putUser(t, a, "other")
	owned := saveTestReport(t, a, "owner")
	other := saveTestReport(t, a, "other")
	_ = a.Store.Update(context.Background(), func(s *State) error {
		s.Reports["unknown"] = &Report{ID: "unknown", UserID: "owner", Birth: Birth{Name: "Unknown time", Date: "1995-01-01", Unknown: true}}
		return nil
	})
	cookie, _ := authorize(t, a, "owner", "customer")
	w := request(t, a, "GET", "/api/kundli/match/profiles", nil, cookie, "")
	var out struct {
		Profiles []MatchingProfile `json:"profiles"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != 200 || len(out.Profiles) != 1 || out.Profiles[0].ID != owned.ID || strings.Contains(w.Body.String(), other.ID) || strings.Contains(w.Body.String(), "phone") || strings.Contains(w.Body.String(), "rules") {
		t.Fatalf("profile ownership %d %s", w.Code, w.Body)
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("private profiles cached")
	}
	admin, _ := authorize(t, a, "owner", "admin")
	if w := request(t, a, "GET", "/api/kundli/match/profiles", nil, admin, ""); w.Code != 401 {
		t.Fatal("admin cookie exposed customer profiles")
	}
}

// TestBirthSecondsAndHistoricalOffsetPrecision checks that birth instants retain seconds and historical timezone precision internally.
func TestBirthSecondsAndHistoricalOffsetPrecision(t *testing.T) {
	for _, tc := range []struct {
		date, clock, zone, want string
		offset                  *int
	}{
		{"1995-08-15", "10:30:47", "Asia/Kolkata", "1995-08-15T05:00:47Z", nil},
		{"1940-01-01", "00:00:31", "Asia/Kolkata", "1939-12-31T18:30:31Z", nil},
		{"2024-11-03", "01:30:59", "America/New_York", "2024-11-03T05:30:59Z", ptr(-240)},
		{"2024-11-03", "01:30:59", "America/New_York", "2024-11-03T06:30:59Z", ptr(-300)},
	} {
		got, e := ResolveWall(tc.date, tc.clock, tc.zone, tc.offset)
		if e != nil || got.UTC().Format(time.RFC3339) != tc.want {
			t.Fatalf("seconds %+v: %v %v", tc, got, e)
		}
	}
	for _, clock := range []string{"10:30:60", "10:30:59.1", "10:30:00junk", "1:30", "24:00:00"} {
		if _, e := ResolveWall("1995-08-15", clock, "Asia/Kolkata", nil); e == nil {
			t.Fatalf("noncanonical time accepted: %s", clock)
		}
	}
	if _, e := ResolveWall("2024-03-10", "02:30:31", "America/New_York", nil); e == nil {
		t.Fatal("second-level DST gap accepted")
	}
	if _, e := ResolveWall("2024-11-03", "01:30:31", "America/New_York", nil); e == nil {
		t.Fatal("second-level DST ambiguity hidden")
	}
}

// TestMatchingRealPDFRenderer checks the real matching PDF renderer when its integration-test flag is enabled.
func TestMatchingRealPDFRenderer(t *testing.T) {
	if os.Getenv("RUN_MATCHING_PDF_TEST") != "1" {
		t.Skip("set RUN_MATCHING_PDF_TEST=1 for real English/Hindi Chrome rendering")
	}
	a := testApp(t)
	b := testBirth()
	b.Time = "10:30:47"
	out, failure := a.calculateMatching(context.Background(), MatchingRequest{Bride: b, Groom: Birth{Name: "Groom PDF Fixture", Date: "1992-02-29", Time: "18:15:23", PlaceID: "mumbai"}})
	if failure != nil {
		t.Fatal(failure)
	}
	renderer := ChromePDF{FontPath: a.Config.FontPath, ChromePath: os.Getenv("CHROME_PATH")}
	for _, language := range []string{"en", "hi"} {
		data, e := renderer.RenderMatching(context.Background(), out, language)
		if e != nil || !bytes.HasPrefix(data, []byte("%PDF-")) {
			t.Fatalf("real PDF %s: %v", language, e)
		}
		dir := os.Getenv("MATCHING_PDF_OUTPUT_DIR")
		if dir == "" {
			dir = t.TempDir()
		}
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		if e := os.WriteFile(filepath.Join(dir, "matching-expanded-"+language+".pdf"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
}

// TestMatchingStrictPDFBodyAndLicense checks strict PDF request decoding and ephemeris licence enforcement.
func TestMatchingStrictPDFBodyAndLicense(t *testing.T) {
	a := testApp(t)
	stub := &matchingPDFStub{}
	a.MatchingPDF = stub
	for _, body := range []string{`{"bride":{},"groom":{},"language":"en","chart":{}}`, `{"bride":{},"groom":{},"language":"en"} {}`} {
		r := httptest.NewRequest("POST", "/api/kundli/match/pdf", strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatal("PDF trusted extra data")
		}
	}
	a.Config.Production = true
	if w := request(t, a, "POST", "/api/kundli/match/pdf", map[string]any{"bride": testBirth(), "groom": testBirth(), "language": "en"}, nil, ""); w.Code != 503 || stub.calls != 0 {
		t.Fatal("PDF licence gate bypassed")
	}
}

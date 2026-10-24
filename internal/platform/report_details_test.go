package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestExpandedReportAndUnknownTime checks expanded Kundli details and restrictions for an unknown birth time.
func TestExpandedReportAndUnknownTime(t *testing.T) {
	a := testApp(t)
	r, e := calculate(a.Config, testBirth(), seed().Rules, a.Now())
	if e != nil {
		t.Fatal(e)
	}
	if r.Navamsa == nil || len(r.Navamsa.Planets) != 9 || r.Dasha == nil || len(r.Dasha.Periods) != 9 || r.Transits == nil || len(r.Transits.Planets) != 9 || len(r.Checks) != 4 {
		t.Fatalf("expanded report incomplete: %+v", r)
	}
	// Serialization is also the saved-report snapshot contract. An exported
	// fixture is used only by the isolated, intercepted browser QA below.
	b, e := json.Marshal(r)
	if e != nil {
		t.Fatal(e)
	}
	var restored Report
	if e = json.Unmarshal(b, &restored); e != nil || restored.Dasha.CurrentLord != r.Dasha.CurrentLord || !restored.Transits.AsOf.Equal(a.Now()) {
		t.Fatal("snapshot fields lost")
	}
	if path := os.Getenv("REPORT_QA_FIXTURE"); path != "" {
		if e = os.MkdirAll(filepath.Dir(path), 0700); e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	unknown := testBirth()
	unknown.Unknown = true
	u, e := calculate(a.Config, unknown, seed().Rules, a.Now())
	if e != nil || u.Chart != nil || u.Navamsa != nil || u.Dasha != nil || u.Transits != nil || len(u.Checks) != 0 || len(u.Rules) != 0 {
		t.Fatalf("unknown time leaked calculated details: %+v %v", u, e)
	}
}

// TestPlacementChecksAndTransitFrames checks placement predicates against their declared natal and transit frames.
func TestPlacementChecksAndTransitFrames(t *testing.T) {
	natal := swisseph.Kundali{Lagna: swisseph.ZodiacPosition{RashiNumber: 1}}
	for i, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"} {
		sign := []int{5, 4, 7, 5, 7, 2, 9, 6, 12}[i]
		natal.Planets = append(natal.Planets, swisseph.Graha{Name: name, ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: sign}, House: sign})
	}
	checks := placementChecks(natal)
	for _, check := range checks {
		if !check.Matches {
			t.Fatalf("expected base placement: %+v", check)
		}
	}
	for _, f := range []struct {
		sign  int
		phase string
	}{{3, "rising"}, {4, "peak"}, {5, "setting"}, {6, "none"}} {
		current := swisseph.Kundali{Planets: []swisseph.Graha{{Name: "Saturn", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: f.sign}}}}
		snapshot := transitSnapshot(natal, current, time.Now())
		if snapshot.SadeSati != f.phase || snapshot.Planets[0].House != f.sign {
			t.Fatalf("wrong natal reference frame: %+v", snapshot)
		}
	}
	natal.Planets[2].House = 6
	natal.Planets[2].RashiNumber = 6
	natal.Planets[4].RashiNumber = 5
	natal.Planets[3].RashiNumber = 6
	for _, check := range placementChecks(natal) {
		if check.Matches {
			t.Fatalf("expected no placement match: %+v", check)
		}
	}
}

// TestPlacementAndTransitDoNotInventMissingReferenceFrames checks that missing chart reference frames cannot generate invented placements or transits.
func TestPlacementAndTransitDoNotInventMissingReferenceFrames(t *testing.T) {
	for _, chart := range []swisseph.Kundali{
		{},
		{Planets: []swisseph.Graha{{Name: "Sun", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 0}}, {Name: "Mercury", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 0}}}},
	} {
		for _, check := range placementChecks(chart) {
			if check.Matches || check.House != 0 {
				t.Fatalf("missing/invalid chart generated a placement claim: %+v", check)
			}
		}
	}
	at := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	current := swisseph.Kundali{Planets: []swisseph.Graha{
		{Name: "Saturn", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 1}, House: 9},
		{Name: "Invalid", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 13}},
	}}
	got := transitSnapshot(swisseph.Kundali{}, current, at)
	if len(got.Planets) != 1 || got.Planets[0].House != 0 || got.Planets[0].FromMoon != 0 || got.SadeSati != "none" || !got.AsOf.Equal(at) {
		t.Fatalf("missing natal frame invented transit houses/phase: %+v", got)
	}
}

// TestFaladeshConditionsAndApproval checks Faladesh condition matching and approved-content selection.
func TestFaladeshConditionsAndApproval(t *testing.T) {
	now := time.Now()
	r := &Report{Chart: &swisseph.Kundali{Lagna: swisseph.ZodiacPosition{RashiNumber: 6}, Planets: []swisseph.Graha{{Name: "Moon", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 12, NakshatraNumber: 27}, House: 7}}}, Dasha: &swisseph.Vimshottari{CurrentLord: "Mercury"}}
	base := Rule{ID: "career", Version: 1, Planet: "Moon", Sign: 12, House: 7, LagnaSign: 6, Nakshatra: 27, DashaLord: "Mercury", Category: "career", Status: "approved", Approver: "Test approval fixture", ApprovedAt: &now}
	// check asserts how many approved interpretation rules match the fixture chart.
	check := func(rule Rule, want int) {
		t.Helper()
		got := matchRules(r, map[string]Rule{"career:1": rule}, false)
		if len(got) != want {
			t.Fatalf("rule matched incorrectly: %+v", rule)
		}
	}
	check(base, 1)
	for _, field := range []string{"house", "lagna", "nakshatra", "dasha", "approval", "draft", "demo"} {
		v := base
		switch field {
		case "house":
			v.House = 8
		case "lagna":
			v.LagnaSign = 7
		case "nakshatra":
			v.Nakshatra = 26
		case "dasha":
			v.DashaLord = "Sun"
		case "approval":
			v.ApprovedAt = nil
		case "draft":
			v.Status = "draft"
		case "demo":
			v.Demo = true
		}
		check(v, 0)
	}
	a := testApp(t)
	cookie, csrf := authorize(t, a, "admin", "admin")
	for _, v := range []Rule{{ID: "bad", Version: 1, Hindi: "परीक्षण व्याख्या सामग्री", Status: "draft", House: 2}, {ID: "bad", Version: 1, Hindi: "परीक्षण व्याख्या सामग्री", Status: "draft", Category: "unsupported"}, {ID: "bad", Version: 1, Hindi: "परीक्षण व्याख्या सामग्री", Status: "draft", Nakshatra: 28}} {
		if got := request(t, a, "POST", "/api/admin/rules", v, cookie, csrf); got.Code != 422 {
			t.Fatalf("invalid conditions accepted: %d %s", got.Code, got.Body)
		}
	}
}

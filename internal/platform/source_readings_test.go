package platform

import (
	"astroapp/backend/internal/gurujitext"
	"astroapp/backend/internal/numerology"
	"astroapp/backend/internal/swisseph"
	"bytes"
	"context"
	"encoding/json"
	"html"
	"os"
	"strings"
	"testing"
)

// TestGurujiMangalSavedPresentation recalculates legacy views without changing saved chart evidence or purchase snapshots.
func TestGurujiMangalSavedPresentation(t *testing.T) {
	r := &Report{Chart: &swisseph.Kundali{Lagna: swisseph.ZodiacPosition{Longitude: 100, RashiNumber: 4}, Planets: []swisseph.Graha{
		{Name: "Moon", ZodiacPosition: swisseph.ZodiacPosition{Longitude: 340, RashiNumber: 12}},
		{Name: "Mars", ZodiacPosition: swisseph.ZodiacPosition{Longitude: 160, RashiNumber: 6}},
	}}, Details: &KundliExpansion{MangalMethod: "legacy", Mangal: swisseph.MangalAssessment{BaseCount: 3, Level: "none"}}, Checks: []PlacementCheck{{Code: "mangal-lagna", House: 2, Matches: true}}}
	before, _ := json.Marshal(r)
	view := sourceOnlyReport(r)
	if view.Details.Mangal.Classification != "partial" || view.Details.MangalMethod != swisseph.MangalMethod || view.Checks[0].House != 3 || view.Checks[0].Matches {
		t.Fatalf("legacy Mangal escaped recalculation: %+v", view)
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("presentation mutated the saved chart")
	}
}

// TestSourceSelections checks original conditional branches, actual planet houses and both repeated Chaturdashi articles.
func TestSourceSelections(t *testing.T) {
	r := &Report{Chart: &swisseph.Kundali{Lagna: swisseph.ZodiacPosition{RashiNumber: 2}, Planets: []swisseph.Graha{{Name: "Moon", ZodiacPosition: swisseph.ZodiacPosition{RashiNumber: 5, NakshatraNumber: 10}, House: 4}, {Name: "Sun", House: 4}}}, Details: &KundliExpansion{Birth: swisseph.BirthDetails{HinduWeekday: 1, Limbs: []swisseph.PanchangPeriod{{Number: 29}}}}}
	entries := sourceReadings(r)
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Category]++
		if entry.Category == "भावफल" && entry.House > 0 && entry.House != 4 {
			t.Fatal("wrong house passage", entry)
		}
		found := false
		for _, original := range gurujitext.Lookup(entry.Category, entry.Key) {
			if original == entry {
				found = true
			}
		}
		if !found {
			t.Fatal("generated source prose")
		}
	}
	if counts["लग्न"] != 1 || counts["राशिफल"] != 1 || counts["नक्षत्र"] != 1 || counts["वारफल"] != 1 || counts["तिथिफल"] != 2 || counts["ग्रहों का फला देश"] != 2 {
		t.Fatal("incorrect source matching", counts)
	}
	r.Birth.Unknown = true
	if len(sourceReadings(r)) != 0 {
		t.Fatal("unknown-time chart invented source matches")
	}
}

// TestSourceTithiBoundaries distinguishes both fortnight endings and keeps the original duplicated Chaturdashi.
func TestSourceTithiBoundaries(t *testing.T) {
	for _, tc := range []struct {
		number int
		key    string
		count  int
	}{{14, "चतुर्दशी", 2}, {15, "पूर्णिमा", 1}, {16, "प्रतिपदा", 1}, {29, "चतुर्दशी", 2}, {30, "अमावस्या", 1}} {
		r := &Report{Chart: &swisseph.Kundali{}, Details: &KundliExpansion{Birth: swisseph.BirthDetails{Limbs: []swisseph.PanchangPeriod{{Number: tc.number}}}}}
		count := 0
		for _, entry := range sourceReadings(r) {
			if entry.Category == "तिथिफल" {
				count++
				if entry.Key != tc.key {
					t.Fatal("wrong fortnight mapping", tc, entry.Key)
				}
			}
		}
		if count != tc.count {
			t.Fatal("lost tithi source", tc, count)
		}
	}
}

// TestCurrentViewsExcludeLegacyProse checks browser/API copies and both PDF languages without mutating the purchase.
func TestCurrentViewsExcludeLegacyProse(t *testing.T) {
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	if target := os.Getenv("SOURCE_READING_TEST_OUTPUT"); target != "" {
		fixture, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(target, fixture, 0600); err != nil {
			t.Fatal(err)
		}
	}
	r.Rules = []Rule{{Hindi: "FORBIDDEN-LEGACY-RULE", English: "FORBIDDEN-ENGLISH-RULE", Mantra: "FORBIDDEN-REMEDY", Gemstone: "FORBIDDEN-GEMSTONE"}}
	r.GurujiFaladesh = &GurujiFaladesh{Analysis: &FaladeshSynthesis{Topics: []FaladeshTopic{{Conclusion: GurujiWording{Hindi: "FORBIDDEN-SUMMARY"}}}}}
	r.Details.RajyogReading = &RajyogReading{Summary: GurujiWording{Hindi: "FORBIDDEN-YOGA"}}
	r.Numerology.Extended = &numerology.ExtendedReading{Profile: numerology.Profile{Guidance: []numerology.Guidance{{Text: numerology.Bilingual{Hindi: "FORBIDDEN-PROFILE"}}}}}
	before, _ := json.Marshal(r)
	view := sourceOnlyReport(r)
	encoded, _ := json.Marshal(view)
	if bytes.Contains(encoded, []byte("FORBIDDEN-")) || len(view.SourceReadings) == 0 {
		t.Fatal("legacy prose in current response")
	}
	renderer := ChromePDF{FontPath: a.Config.FontPath, Origin: a.Config.Origin}
	for _, language := range []string{"hi", "en"} {
		markup, err := renderer.documentLanguage(&Order{ID: "source-only-test", Snapshot: r}, language)
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Contains(markup, []byte("FORBIDDEN-")) {
			t.Fatal("legacy prose in PDF", language)
		}
		for _, entry := range view.SourceReadings {
			if !strings.Contains(html.UnescapeString(string(markup)), entry.Text) {
				t.Fatal("original passage was omitted or rewritten", entry.Title, language)
			}
		}
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("stored purchase snapshot changed")
	}
}

// TestUnknownTimeNumerologyUsesOnlyPDFText preserves date arithmetic and missing-source behavior without chart assumptions.
func TestUnknownTimeNumerologyUsesOnlyPDFText(t *testing.T) {
	a := testApp(t)
	b := testBirth()
	b.Date = "2000-01-01"
	b.Unknown = true
	r, err := calculate(a.Config, b, map[string]Rule{"unwanted": {Hindi: "UNWANTED", Status: "approved"}}, a.Now())
	if err != nil || r.Numerology == nil || r.Chart != nil || len(r.Rules) != 0 {
		t.Fatal("date-only report invalid", err)
	}
	if r.Numerology.Bhagyank.Value != 4 {
		t.Fatal("changed date arithmetic")
	}
	for _, reference := range r.Numerology.SourceReadings {
		if reference.Category != "मूलांक" && reference.Category != "भाग्यांक" {
			t.Fatal("non-source number meaning")
		}
	}
	for _, entry := range (pdfLocale{"hi"}).numerologySection(&numerology.Reading{Mulank: numerology.Number{Value: 1}, Bhagyank: numerology.Number{Value: 3}}).Blocks {
		if strings.Contains(entry.Note, "भाग्यांक 3 का सम्बन्ध") {
			t.Fatal("generated missing-number fallback")
		}
	}
}

// TestLegacyCustomerResponsesFilterProse verifies both saved-report endpoints without rewriting the stored record.
func TestLegacyCustomerResponsesFilterProse(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "source-owner")
	r := saveTestReport(t, a, "source-owner")
	if err := a.Store.Update(context.Background(), func(s *State) error {
		saved := s.Reports[r.ID]
		saved.Rules = []Rule{{Hindi: "FORBIDDEN-CUSTOMER-RULE"}}
		saved.GurujiFaladesh = &GurujiFaladesh{Readings: []GurujiReading{{GurujiPlacement: GurujiPlacement{HindiExcerpt: "FORBIDDEN-OLD-READING"}}}}
		saved.SourceReadings = nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cookie, _ := authorize(t, a, "source-owner", "customer")
	for _, path := range []string{"/api/reports/" + r.ID, "/api/me"} {
		response := request(t, a, "GET", path, nil, cookie, "")
		if response.Code != 200 || strings.Contains(response.Body.String(), "FORBIDDEN-") || !strings.Contains(response.Body.String(), "source_readings") {
			t.Fatal("legacy response leaked prose", path, response.Code)
		}
	}
	if err := a.Store.View(context.Background(), func(s *State) error {
		if s.Reports[r.ID].Rules[0].Hindi != "FORBIDDEN-CUSTOMER-RULE" || s.Reports[r.ID].SourceReadings != nil {
			t.Fatal("presentation rewrote saved history")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

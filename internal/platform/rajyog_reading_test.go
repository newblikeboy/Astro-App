package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestRajyogRealChartReadings covers the three reported charts and a real
// negative control; it also checks that PDF chapters use the saved reader text.
func TestRajyogRealChartReadings(t *testing.T) {
	a := testApp(t)
	for _, tc := range []struct {
		name, date, clock, place string
		groups                   int
		participants             []string
	}{
		{"Raja Babu", "1995-11-29", "12:35", "geonames-1273491", 2, []string{"Sun", "Mercury", "Mars", "Venus"}},
		{"Purnia", "1996-11-26", "16:00", "geonames-1259166", 2, []string{"Mars", "Sun", "Moon"}},
		{"Third chart", "1997-08-28", "16:00", "delhi", 1, []string{"Mercury", "Sun"}},
		{"Negative control", "2001-09-21", "22:30", "delhi", 0, []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := calculate(a.Config, Birth{Name: tc.name, Date: tc.date, Time: tc.clock, PlaceID: tc.place}, seed().Rules, a.Now())
			if err != nil {
				t.Fatal(err)
			}
			d := r.Details
			if d == nil || d.RajyogReading == nil || d.Qualifications.Method != swisseph.YogaQualificationMethod {
				t.Fatal("missing current pair-specific reading")
			}
			global := d.Qualifications.Yogas[len(d.Qualifications.Yogas)-1]
			if !reflect.DeepEqual(global.Planets, tc.participants) {
				t.Fatalf("participants %v want %v", global.Planets, tc.participants)
			}
			if d.YogaChecks.Rajyog.Matches != (tc.groups > 0) || tc.groups > 0 && len(d.RajyogReading.Paragraphs) != tc.groups {
				t.Fatal("incorrect match or double-counted planet relationship")
			}
			before, _ := json.Marshal(r)
			for _, language := range []string{"hi", "en"} {
				l := pdfLocale{language}
				book := l.book(r, time.UTC, nil)
				found := false
				for _, card := range book.Yogas {
					if card.Title != l.text("Rajyog · Kendra–Trikona relationships") {
						continue
					}
					found = true
					if card.Conclusion != l.gurujiWording(d.RajyogReading.Summary) {
						t.Fatal("PDF summary differs from API")
					}
					for _, p := range d.RajyogReading.Paragraphs {
						text := l.gurujiWording(p)
						if !strings.Contains(card.Meaning, text) || strings.Contains(text, "rajyog.") || strings.Contains(text, "{v") {
							t.Fatalf("PDF paragraph missing or untranslated: %s", text)
						}
					}
				}
				if !found {
					t.Fatal("PDF Rajyog chapter missing")
				}
			}
			after, _ := json.Marshal(r)
			if string(before) != string(after) {
				t.Fatal("presentation mutated report")
			}
			if tc.name == "Raja Babu" {
				if !strings.Contains(d.RajyogReading.Summary.English, "Some have strength support") || !strings.Contains(d.RajyogReading.Paragraphs[1].English, "Neither participant") {
					t.Fatal("Mercury weakness contaminated Mars–Venus")
				}
			}
			if tc.name == "Purnia" && strings.Contains(d.RajyogReading.Paragraphs[0].English, "Jupiter") {
				t.Fatal("unrelated weak Jupiter contaminated actual Sun–Mars pair")
			}
			if tc.name == "Third chart" && !strings.Contains(d.RajyogReading.Paragraphs[0].English, "work with opportunities") {
				t.Fatal("duplicate 7th/10th ownership lost career theme")
			}
		})
	}
}

// TestRajyogReaderKeepsHistoricalAssessment distinguishes missing evidence
// from absence and never fabricates a new reader for an old qualification method.
func TestRajyogReaderKeepsHistoricalAssessment(t *testing.T) {
	checks := &swisseph.NatalYogaChecks{Rajyog: swisseph.RajyogAssessment{Matches: true}}
	for _, q := range []*swisseph.YogaQualifications{nil, {Method: "m4y-phaladeepika7-neechabhanga-strength-v1"}, {Method: swisseph.YogaQualificationMethod}} {
		if buildRajyogReading(checks, q) != nil {
			t.Fatal("fabricated a per-pair reading without evidence")
		}
	}
}

// TestRajyogTopicEvidenceDoesNotInventRelationships prevents a non-matching
// house lord from receiving yoga context merely because a different pair matched.
func TestRajyogTopicEvidenceDoesNotInventRelationships(t *testing.T) {
	r := &Report{Details: &KundliExpansion{YogaChecks: &swisseph.NatalYogaChecks{Rajyog: swisseph.RajyogAssessment{
		Matches: true,
		Pairs: []swisseph.RajyogPair{
			{KendraLord: "Sun", TrikonaLord: "Saturn", Links: []string{"conjunction"}},
			{KendraLord: "Mercury", TrikonaLord: "Jupiter", Links: []string{}},
		},
	}}}}
	topic := &FaladeshTopic{}
	addSynthesisYogas(r, synthesisTopicSpec{ID: "career"}, topic, map[string]bool{"Mercury": true})
	if len(topic.Evidence) != 0 {
		t.Fatal("unrelated planet received an invented yoga relationship")
	}
	addSynthesisYogas(r, synthesisTopicSpec{ID: "career"}, topic, map[string]bool{"Sun": true})
	if len(topic.Evidence) != 1 {
		t.Fatal("lost actual matched relationship")
	}
}

// TestRajyogTopicQualificationsRemainIndependent checks pair-specific evidence
// IDs and removes duplicate lordship combinations without changing the snapshot.
func TestRajyogTopicQualificationsRemainIndependent(t *testing.T) {
	q := &swisseph.YogaQualifications{Method: swisseph.YogaQualificationMethod,
		Yogas: []swisseph.YogaQualification{{Code: "kendra-trikona-rajyog", Status: "qualified"}},
		RajyogPairs: []swisseph.RajyogPairQualification{
			{Pair: swisseph.RajyogPair{KendraLord: "Mars", TrikonaLord: "Venus", Links: []string{"conjunction"}}, Qualification: swisseph.YogaQualification{Code: "kendra-trikona-rajyog", BasePresent: true, Status: "supported", Planets: []string{"Mars", "Venus"}}},
			{Pair: swisseph.RajyogPair{KendraLord: "Sun", TrikonaLord: "Mercury", Links: []string{"conjunction"}}, Qualification: swisseph.YogaQualification{Code: "kendra-trikona-rajyog", BasePresent: true, Status: "qualified", Planets: []string{"Sun", "Mercury"}}},
		},
	}
	q.RajyogPairs = append(q.RajyogPairs, q.RajyogPairs[0])
	before, _ := json.Marshal(q)
	got := readingYogaQualifications(q)
	if len(got) != 2 || got[0].Status != "supported" || got[1].Status != "qualified" || got[0].Code == got[1].Code {
		t.Fatal("pair capacity mixed or duplicate ownership counted twice")
	}
	after, _ := json.Marshal(q)
	if string(before) != string(after) {
		t.Fatal("altered retained qualification")
	}
}

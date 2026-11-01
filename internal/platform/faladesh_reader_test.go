package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestMatchingExceptionsReachStatelessAPI verifies separate active assessments
// without changing the core score or persisting either participant's chart.
func TestMatchingExceptionsReachStatelessAPI(t *testing.T) {
	a := testApp(t)
	var before, after []byte
	_ = a.Store.View(context.Background(), func(s *State) error { before, _ = json.Marshal(s); return nil })
	out, failure := a.calculateMatching(context.Background(), MatchingRequest{Bride: testBirth(), Groom: Birth{Name: "Matching example", Date: "1992-02-29", Time: "18:15:23", PlaceID: "mumbai"}})
	if failure != nil {
		t.Fatal(failure)
	}
	if out.Exceptions == nil || len(out.Exceptions.Nadi.Rules) != 5 || len(out.Exceptions.Bhakoot.Rules) != 4 {
		t.Fatal("exceptions missing from response")
	}
	base, e := swisseph.CalculateMatching(*out.Bride.Chart, *out.Groom.Chart)
	if e != nil || base.Total != out.Total || !reflect.DeepEqual(base.Factors, out.Factors) {
		t.Fatal("base score changed", e)
	}
	for _, d := range []swisseph.MatchingDosha{out.Exceptions.Nadi, out.Exceptions.Bhakoot} {
		if !d.Active {
			for _, c := range out.Cautions {
				if c == "nadi_same" && d.Code == "nadi" || c == "bhakoot_adverse" && d.Code == "bhakoot" {
					t.Fatal("cancelled caution remains active")
				}
			}
		}
	}
	_ = a.Store.View(context.Background(), func(s *State) error { after, _ = json.Marshal(s); return nil })
	if string(before) != string(after) {
		t.Fatal("matching persisted calculation")
	}
	if dir := os.Getenv("ADVANCED_QA_DIR"); dir != "" {
		if e := os.MkdirAll(dir, 0700); e != nil {
			t.Fatal(e)
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		if e := os.WriteFile(filepath.Join(dir, "matching.json"), data, 0600); e != nil {
			t.Fatal(e)
		}
	}
}

// integratedTopic selects the new canonical subject, failing on missing output.
func integratedTopic(t *testing.T, r *Report, category string) FaladeshTopic {
	t.Helper()
	out := buildIntegratedFaladesh(r)
	if out == nil {
		t.Fatal("missing integrated reading")
	}
	for _, topic := range out.Topics {
		if topic.Category == category {
			return topic
		}
	}
	t.Fatal("missing subject", category)
	return FaladeshTopic{}
}

// TestIntegratedReaderSnapshotAndPDFParity checks both real example charts,
// native inputs, complete retained cause IDs and identical PDF/browser prose.
// Optional fixture output is synthetic local QA, never a customer-store write.
func TestIntegratedReaderSnapshotAndPDFParity(t *testing.T) {
	a := testApp(t)
	asOf := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		id    string
		birth Birth
	}{{"purnia", Birth{Name: "Purnia reading example", Date: "1996-11-26", Time: "16:00", PlaceID: "geonames-1259166"}}, {"raja", Birth{Name: "Raja Babu", Date: "1995-11-29", Time: "12:35", PlaceID: "geonames-1273491"}}} {
		t.Run(tc.id, func(t *testing.T) {
			r, e := calculate(a.Config, tc.birth, nil, asOf)
			if e != nil {
				t.Fatal(e)
			}
			s := r.GurujiFaladesh.Analysis
			if s == nil || s.Version != integratedFaladeshVersion || s.ReaderVersion != readerContractVersion || len(s.Topics) != 9 {
				t.Fatal("new version not retained")
			}
			if r.Details.Shadbala == nil || len(r.Details.Shadbala.Planets) != 7 || r.Details.Qualifications == nil || r.Transits.Assessment == nil {
				t.Fatal("missing native assessment")
			}
			before, _ := json.Marshal(r)
			for _, topic := range s.Topics {
				if len(topic.Reader) < 4 {
					t.Fatalf("subject lacks detail: %s %d", topic.Category, len(topic.Reader))
				}
				if topic.Category == "general" && len(topic.Reader) != 4 {
					t.Fatal("General must remain four personal-direction paragraphs")
				}
				for _, p := range topic.Reader {
					if strings.TrimSpace(p.Text.Hindi) == "" || strings.TrimSpace(p.Text.English) == "" || readerTechnical.MatchString(p.Text.Hindi) || readerTechnical.MatchString(p.Text.English) {
						t.Fatalf("technical or missing public paragraph %s: %+v", topic.Category, p)
					}
					if topic.Category == "general" && p.Scope == "period" {
						t.Fatal("General imports other tabs' period reading")
					}
					if len(p.EvidenceIDs) == 0 {
						t.Fatalf("untraceable paragraph %s", p.ID)
					}
					for _, id := range p.EvidenceIDs {
						found := false
						for _, e := range topic.Evidence {
							found = found || e.ID == id
						}
						if !found {
							t.Fatal("invented cause", id)
						}
					}
				}
				if topic.Remedy != nil && topic.Remedy.Status == "selected" && (topic.Remedy.Source == nil || len(topic.Remedy.EvidenceIDs) == 0 || topic.Remedy.Planet == "") {
					t.Fatal("remedy lacks selection basis/source")
				}
			}
			for _, language := range []string{"en", "hi"} {
				l := pdfLocale{language}
				book := l.book(r, time.UTC, l.sections(r, time.UTC))
				if len(book.Topics) != 9 {
					t.Fatal("canonical reading missing from main PDF chapters")
				}
				doc, e := (ChromePDF{FontPath: a.Config.FontPath, Origin: a.Config.Origin}).documentLanguage(&Order{ID: "advanced-qa", Snapshot: r, Demo: true}, language)
				if e != nil {
					t.Fatal(e)
				}
				for i, topic := range s.Topics {
					got := l.faladeshParagraphs(topic)
					if !reflect.DeepEqual(got, book.Topics[i].Paragraphs) {
						t.Fatal("PDF chapter differs")
					}
					for j, p := range topic.Reader {
						want := l.gurujiWording(p.Text)
						if got[j] != want || !strings.Contains(string(doc), html.EscapeString(want)) {
							t.Fatalf("PDF differs from retained reader: %s %s %s", tc.id, language, p.ID)
						}
					}
				}
			}
			after, _ := json.Marshal(r)
			if string(before) != string(after) {
				t.Fatal("render mutated report")
			}
			var restored Report
			if e := json.Unmarshal(before, &restored); e != nil || !reflect.DeepEqual(restored.GurujiFaladesh.Analysis, s) {
				t.Fatal("canonical reader lost on snapshot roundtrip", e)
			}
			if directory := os.Getenv("ADVANCED_QA_DIR"); directory != "" {
				dir := filepath.Join(directory, tc.id)
				if e := os.MkdirAll(dir, 0700); e != nil {
					t.Fatal(e)
				}
				pretty, _ := json.MarshalIndent(r, "", "  ")
				if e := os.WriteFile(filepath.Join(dir, "report.json"), pretty, 0600); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}

// TestIntegratedReaderFactorsActuallyChangeFindings changes independent
// calculated inputs while holding positions fixed, proving they affect prose.
func TestIntegratedReaderFactorsActuallyChangeFindings(t *testing.T) {
	r := synthesisScenario()
	r.Details.Shadbala = &swisseph.Shadbala{}
	r.Details.Qualifications = &swisseph.YogaQualifications{}
	for _, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"} {
		r.Details.Shadbala.Planets = append(r.Details.Shadbala.Planets, swisseph.ShadbalaPlanet{Planet: name, Total: 500, Required: 300, Ratio: 1.5, MeetsRequirement: true})
	}
	base := integratedTopic(t, r, "career")
	for i := range r.Details.Shadbala.Planets {
		if r.Details.Shadbala.Planets[i].Planet == "Saturn" {
			r.Details.Shadbala.Planets[i].MeetsRequirement = false
			r.Details.Shadbala.Planets[i].Total = 150
			r.Details.Shadbala.Planets[i].Ratio = .5
		}
	}
	weak := integratedTopic(t, r, "career")
	if reflect.DeepEqual(base.Reader, weak.Reader) || weak.Remedy.Status != "selected" {
		t.Fatal("computed weakness has no effect")
	}
	r.Details.Vargas = []swisseph.Varga{{Factor: 9, Planets: []swisseph.DivisionalPlacement{{Name: "Saturn", RashiNumber: 1, House: 1}}}}
	divisional := integratedTopic(t, r, "career")
	if reflect.DeepEqual(weak.Reader, divisional.Reader) {
		t.Fatal("varga condition has no effect")
	}
	r.Details.Qualifications.Yogas = []swisseph.YogaQualification{{Code: "mahapurusha-Saturn", BasePresent: true, Status: "qualified", Planets: []string{"Saturn"}}}
	yoga := integratedTopic(t, r, "career")
	if reflect.DeepEqual(divisional.Reader, yoga.Reader) {
		t.Fatal("yoga qualification has no effect")
	}
	for i := range r.Details.Analysis.Conditions {
		if r.Details.Analysis.Conditions[i].Name == "Sun" {
			r.Details.Analysis.Conditions[i].Dignity = "Debilitation sign"
		}
	}
	dispositor := integratedTopic(t, r, "education")
	hasDispositor := false
	for _, v := range dispositor.Resolution {
		hasDispositor = hasDispositor || v.Factor == "foundation" && v.Direction == "attention"
	}
	if !hasDispositor {
		t.Fatal("strained dispositor ignored")
	}
}

// TestRemedySelectionIsConditionalAndRoleSafe prevents a blanket mandatory
// remedy list, false approvals and automatic strengthening of a Marak planet.
func TestRemedySelectionIsConditionalAndRoleSafe(t *testing.T) {
	r := synthesisScenario()
	topic := FaladeshTopic{Category: "finance"}
	if got := selectTopicRemedy(r, &topic, []string{"Jupiter"}); got.Status != "not_needed" || got.Planet != "" {
		t.Fatal("remedy invented without cause")
	}
	name := ""
	for _, p := range r.Chart.Planets {
		if gurujiRole(r.Chart.Lagna.RashiNumber, p.Name) == "marak" {
			name = p.Name
			break
		}
	}
	if name == "" {
		t.Fatal("test requires a board Marak")
	}
	appendSynthesisEvidence(&topic, "shadbala", name, 0, "attention", "checked weak capacity", "checked weak capacity")
	got := selectTopicRemedy(r, &topic, []string{name})
	if got.Status != "selected" || got.Approach != "balance" || got.Source == nil || len(got.ApprovedRules) != 0 {
		t.Fatal("role/source/approval conflated")
	}
	if strings.Contains(strings.ToLower(got.Practice.English), "gemstone") {
		t.Fatal("automatic gemstone strengthening")
	}
	nodeTopic := FaladeshTopic{Category: "career"}
	appendSynthesisEvidence(&nodeTopic, "conjunction-Saturn", "Rahu", 10, "attention", "checked whole-sign conjunction", "checked whole-sign conjunction")
	node := selectTopicRemedy(r, &nodeTopic, []string{"Saturn"})
	if node.Planet != "Rahu" || node.Approach != "balance" || gurujiRole(r.Chart.Lagna.RashiNumber, "Rahu") != "unlisted" {
		t.Fatal("actual node context ignored or invented board role")
	}
}

// TestReaderReferenceUsesBirthplaceDate prevents UTC midnight formatting from
// disagreeing with the website's birthplace-local saved reference date.
func TestReaderReferenceUsesBirthplaceDate(t *testing.T) {
	r := synthesisScenario()
	r.Details.Shadbala = &swisseph.Shadbala{Method: swisseph.ShadbalaMethod}
	r.Details.Qualifications = &swisseph.YogaQualifications{}
	for _, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"} {
		r.Details.Shadbala.Planets = append(r.Details.Shadbala.Planets, swisseph.ShadbalaPlanet{Planet: name, Total: 500, Required: 300, Ratio: 1.5, MeetsRequirement: true})
	}
	r.Dasha.AsOf = time.Date(2026, 11, 1, 23, 0, 0, 0, time.UTC)
	topic := integratedTopic(t, r, "career")
	found := false
	for _, p := range topic.Reader {
		if p.ID == "reader-period-career" {
			found = true
			if !strings.Contains(p.Text.Hindi, "02-11-2026") || !strings.Contains(p.Text.English, "02 Nov 2026") {
				t.Fatal("UTC date leaked into saved local reading")
			}
		}
	}
	if !found {
		t.Fatal("missing linked period")
	}
}

// TestReadingActorsUseOwnStrength prevents a weak Karak aspector from being
// counted as fully supportive and a strong own-sign Marak dispositor from
// becoming automatically favourable merely because of sign dignity.
func TestReadingActorsUseOwnStrength(t *testing.T) {
	r := synthesisScenario()
	r.Details.Shadbala = &swisseph.Shadbala{Method: swisseph.ShadbalaMethod}
	r.Details.Qualifications = &swisseph.YogaQualifications{}
	for _, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"} {
		r.Details.Shadbala.Planets = append(r.Details.Shadbala.Planets, swisseph.ShadbalaPlanet{Planet: name, Total: 500, Required: 300, Ratio: 1.5, MeetsRequirement: true})
	}
	no := false
	for i := range r.Details.Analysis.Conditions {
		r.Details.Analysis.Conditions[i].Combust = &no
		if r.Details.Analysis.Conditions[i].Name == "Mars" {
			r.Details.Analysis.Conditions[i].Dignity = "Own sign"
		}
	}
	if gurujiRole(2, "Mercury") != "karak" || gurujiRole(2, "Mars") != "marak" {
		t.Fatal("fixture must follow the supplied Taurus board row")
	}
	var topics []FaladeshTopic
	for _, weak := range []bool{false, true} {
		for i := range r.Details.Shadbala.Planets {
			if r.Details.Shadbala.Planets[i].Planet == "Mercury" {
				r.Details.Shadbala.Planets[i].MeetsRequirement = !weak
			}
		}
		topic := FaladeshTopic{Category: "career"}
		appendSynthesisEvidence(&topic, "aspect", "Mercury", 11, "context", "actual full aspect", "actual full aspect")
		appendSynthesisEvidence(&topic, "dispositor", "Mars", 9, "context", "own-sign dispositor", "own-sign dispositor")
		addAdvancedEvidence(r, &topic)
		topics = append(topics, topic)
		for _, e := range topic.Evidence {
			if e.Kind == "dispositor" && e.Direction == "support" {
				t.Fatal("own-sign Marak was automatically counted favourable")
			}
			if e.Kind == "aspect" && e.Direction == "support" && weak {
				t.Fatal("aspector's own weakness ignored")
			}
			if e.Kind == "aspect" && e.Direction != "support" && !weak {
				t.Fatal("known capable Karak support lost")
			}
		}
	}
	if reflect.DeepEqual(topics[0].Resolution, topics[1].Resolution) {
		t.Fatal("actor strength has no influence on resolution")
	}
}

package platform

import (
	"astroapp/backend/internal/swisseph"
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// TestYogaConclusionsRespectAssessmentState prevents false subtype names, lost cancellation and leaked localization keys.
func TestYogaConclusionsRespectAssessmentState(t *testing.T) {
	devanagari := regexp.MustCompile(`[\x{0900}-\x{097f}]`)
	for _, language := range []string{"hi", "en"} {
		l := pdfLocale{language}
		for _, tc := range []struct{ status, key string }{{"present", "kaal-sarp.present"}, {"absent", "kaal-sarp.absent"}, {"boundary", "kaal-sarp.boundary"}} {
			k := swisseph.KaalSarpAssessment{Status: tc.status, RahuHouse: 9, KetuHouse: 3}
			before, _ := json.Marshal(k)
			text := l.kaalSarpConclusion(k)
			want := l.conclusion(tc.key, l.text("kaal-sarp.type.9"))
			if !strings.HasPrefix(text, want) || strings.Contains(text, "{v") || strings.Contains(text, "kaal-sarp.") {
				t.Fatalf("%s/%s: %s", language, tc.status, text)
			}
			if tc.status != "present" && strings.Contains(text, l.text("kaal-sarp.type.9")) {
				t.Fatal("unconfirmed subtype appeared in conclusion")
			}
			if language == "en" && devanagari.MatchString(text) {
				t.Fatal("Hindi leaked into English conclusion")
			}
			after, _ := json.Marshal(k)
			if string(before) != string(after) {
				t.Fatal("presentation changed stored evidence")
			}
		}
		for _, tc := range []struct {
			base, active int
			key          string
		}{{0, 0, "mangal.absent"}, {2, 0, "mangal.cancelled"}, {3, 1, "mangal.present"}} {
			text := l.mangalConclusion(swisseph.MangalAssessment{BaseCount: tc.base, ActiveCount: tc.active})
			if !strings.HasPrefix(text, l.text(tc.key)) {
				t.Fatalf("lost cancellation distinction: %s", text)
			}
		}
	}
}

// TestYogaConclusionCatalogueCoversImplementedChecks requires bilingual meanings for every exposed base Yoga and all 12 names.
func TestYogaConclusionCatalogueCoversImplementedChecks(t *testing.T) {
	codes := []string{"mahapurusha-Mars", "mahapurusha-Mercury", "mahapurusha-Jupiter", "mahapurusha-Venus", "mahapurusha-Saturn", "Sun-neighbours-Vasi", "Sun-neighbours-Vesi", "Sun-neighbours-Ubhayachari", "Moon-neighbours-Anapha", "Moon-neighbours-Sunapha", "Moon-neighbours-Durudhara", "gajakesari-base", "budha-aditya-base", "mangal-lagna", "mangal-moon"}
	for _, language := range []string{"hi", "en"} {
		l := pdfLocale{language}
		for _, code := range codes {
			for _, matches := range []bool{true, false} {
				text := l.placementConclusion(code, matches)
				if strings.Contains(text, "yoga.") || strings.Contains(text, "placement.") || strings.Contains(text, "{v") {
					t.Fatalf("untranslated conclusion %s/%s: %s", language, code, text)
				}
			}
		}
		for house := 1; house <= 12; house++ {
			k := swisseph.KaalSarpAssessment{Status: "present", RahuHouse: house, KetuHouse: (house+5)%12 + 1}
			if text := l.kaalSarpConclusion(k); strings.Contains(text, "kaal-sarp.") || strings.Contains(text, "{v") {
				t.Fatal(text)
			}
		}
	}
}

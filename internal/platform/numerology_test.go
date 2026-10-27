package platform

import (
	"astroapp/backend/internal/numerology"
	"encoding/json"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// TestReportNumerologyWithoutBirthTime preserves date-only numbers in the saved report without inventing chart evidence.
func TestReportNumerologyWithoutBirthTime(t *testing.T) {
	a := testApp(t)
	birth := testBirth()
	birth.Date, birth.Time, birth.Unknown = "1995-11-29", "12:35", true
	r, err := calculate(a.Config, birth, nil, a.Now())
	if err != nil || r.Numerology == nil || r.Numerology.Mulank.Value != 2 || r.Numerology.Bhagyank.Value != 1 {
		t.Fatalf("missing date-only numerology: %+v, %v", r, err)
	}
	if r.Chart != nil || r.Details != nil || r.Birth.Time != "" {
		t.Fatal("date-only calculation fabricated a birth time or chart")
	}
	before, _ := json.Marshal(r)
	var restored Report
	if err := json.Unmarshal(before, &restored); err != nil || !reflect.DeepEqual(r.Numerology, restored.Numerology) {
		t.Fatal("numerology evidence lost in snapshot", err)
	}
	for _, language := range []string{"hi", "en"} {
		section := (pdfLocale{language}).numerologySection(restored.Numerology)
		if section.ID != "numerology" || len(section.Blocks) < 15 {
			t.Fatal("presentation lost date-only numerology")
		}
	}
	after, _ := json.Marshal(&restored)
	if string(before) != string(after) {
		t.Fatal("rendering changed saved evidence")
	}
}

// TestNumerologyBilingualPDF checks hand-worked arithmetic, all nine meanings, old reports and unchanged snapshots.
func TestNumerologyBilingualPDF(t *testing.T) {
	r, err := numerology.Calculate("1995-11-29")
	if err != nil {
		t.Fatal(err)
	}
	if got := numerologyWorking(r.Mulank); got != "2 + 9 = 11\n1 + 1 = 2" {
		t.Fatal("incorrect Mulank working", got)
	}
	if got := numerologyWorking(r.Bhagyank); got != "2 + 9 + 1 + 1 + 1 + 9 + 9 + 5 = 37\n3 + 7 = 10\n1 + 0 = 1" {
		t.Fatal("incorrect Bhagyank working", got)
	}
	for _, language := range []string{"hi", "en"} {
		l := pdfLocale{language}
		section := l.numerologySection(&r)
		encoded, _ := json.Marshal(section)
		if regexp.MustCompile(`numerology\.[a-z0-9-]+`).Match(encoded) || strings.Contains(string(encoded), "{v") {
			t.Fatal("untranslated summary", string(encoded))
		}
		if language == "en" && regexp.MustCompile(`[\x{0900}-\x{097f}]`).Match(encoded) {
			t.Fatal("untranslated English numerology")
		}
		old := l.numerologySection(nil)
		if len(old.Blocks) != 0 || old.Intro != l.text("numerology.missing") {
			t.Fatal("older report received an invented reading")
		}
		for value := 1; value <= 9; value++ {
			for _, kind := range []string{"mulank", "bhagyank"} {
				key := "numerology." + kind + "." + string(rune('0'+value))
				if text := l.text(key); text == key || text == "" {
					t.Fatal("missing number meaning", key, language)
				}
			}
			full, err := numerology.CalculateFull("2000-01-0"+string(rune('0'+value)), "Raja Babu")
			if err != nil {
				t.Fatal(err)
			}
			profile, _ := json.Marshal(l.numerologySection(&full))
			if regexp.MustCompile(`numerology\.[a-z0-9-]+`).Match(profile) || strings.Contains(string(profile), "{v") || (language == "en" && regexp.MustCompile(`[\x{0900}-\x{097f}]`).Match(profile)) {
				t.Fatal("incomplete profile translation", value, language)
			}
		}
	}
}

// TestExtendedNumerologySnapshots checks main-name calculation, ignored legacy spelling fields and immutable bilingual PDF evidence.
func TestExtendedNumerologySnapshots(t *testing.T) {
	a := testApp(t)
	birth := testBirth()
	birth.Name, birth.NumerologyName, birth.Date, birth.Unknown = "Raja Babu", "Mary Jane", "1995-11-29", true
	r, err := calculate(a.Config, birth, nil, a.Now())
	if err != nil || r.Numerology.Extended.Naamank.Number.Value != 7 || r.Numerology.Extended.Naamank.Spelling != "Raja Babu" || r.Birth.Name != "Raja Babu" || r.Birth.NumerologyName != "" {
		t.Fatal("main name was not used exclusively", err)
	}
	before, _ := json.Marshal(r)
	for _, language := range []string{"hi", "en"} {
		section := (pdfLocale{language}).numerologySection(r.Numerology)
		encoded, _ := json.Marshal(section)
		if regexp.MustCompile(`numerology\.[a-z0-9-]+`).Match(encoded) || strings.Contains(string(encoded), "{v1}") || len(section.Blocks) < 15 {
			t.Fatal("incomplete extension", language)
		}
		if language == "en" && regexp.MustCompile(`[\x{0900}-\x{097f}]`).Match(encoded) {
			t.Fatal("Hindi leaked into English profile")
		}
		for _, part := range r.Numerology.Extended.Profile.Guidance {
			if !strings.Contains(string(encoded), (pdfLocale{language}).numerologyWording(part.Title)) {
				t.Fatal("missing topic", part.ID)
			}
		}
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("rendering mutated snapshot")
	}
	birth.Name, birth.NumerologyName = "राजा बाबू", "Raja Babu"
	missing, err := calculate(a.Config, birth, nil, a.Now())
	if err != nil || missing.Numerology.Extended.Naamank != nil || missing.Numerology.Extended.NameStatus != "needs_latin_spelling" {
		t.Fatal("legacy spelling overrode the main name", err)
	}
	birth.Name = "Raja Babu"
	for _, spelling := range []string{"", "राजा बाबू", "Raja 123", "..", "A"} {
		birth.NumerologyName = spelling
		actual, err := calculate(a.Config, birth, nil, a.Now())
		if err != nil || actual.Numerology.Extended.Naamank.Number.Value != 7 || actual.Birth.NumerologyName != "" {
			t.Fatal("stale cached spelling affected main-name calculation", err)
		}
	}
}

package numerology

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestChaldeanNames checks independent full-name sums, case/separators, 9 totals and unsupported spellings.
func TestChaldeanNames(t *testing.T) {
	for _, tc := range []struct {
		name         string
		total, value int
	}{
		{"Raja Babu", 16, 7}, {"raja-babu", 16, 7}, {"RAJA.BABU", 16, 7},
		{"Mary Jane", 20, 2}, {"Abf", 11, 2}, {"John Smith", 35, 8}, {"Af", 9, 9},
	} {
		r, err := CalculateName(tc.name)
		if err != nil || r.Number.Steps[0] != tc.total || r.Number.Value != tc.value || len(r.Letters) != len(r.Number.Digits) {
			t.Fatalf("%s: %+v %v", tc.name, r, err)
		}
	}
	for _, name := range []string{"", "A", "...", "राजा बाबू", "Raja बाबू", "Raja 1", "José", "Raja@Babu", "ıra", "ſara", strings.Repeat("a", 101)} {
		if _, err := CalculateName(name); err == nil {
			t.Fatalf("accepted unsupported spelling %q", name)
		}
	}
}

// TestExtendedProfiles ensures calculations never acquire generated profiles, lucky dates or name meanings.
func TestExtendedProfiles(t *testing.T) {
	for n := 1; n <= 9; n++ {
		r, err := CalculateFull("2000-01-0"+string(rune('0'+n)), "Raja Babu")
		if err != nil || r.Extended == nil || r.Extended.Naamank.Number.Value != 7 {
			t.Fatal("name arithmetic changed", err)
		}
		e := r.Extended
		if len(e.Profile.Facts) != 0 || len(e.Profile.Guidance) != 0 || len(e.Profile.Yantra) != 0 || len(e.LuckyDates) != 0 || len(e.LuckyNumbers) != 0 || e.Naamank.Meaning.Hindi != "" || e.Naamank.Meaning.English != "" {
			t.Fatal("authored interpretation returned", n)
		}
		for _, entry := range r.SourceReadings {
			if entry.Category != "मूलांक" && entry.Category != "भाग्यांक" {
				t.Fatal("non-PDF number text")
			}
		}
	}
}

// TestExtendedEvidenceAndIsolation checks the supplied case, missing-script behaviour and no shared mutable catalogue data.
func TestExtendedEvidenceAndIsolation(t *testing.T) {
	r, err := CalculateFull("1995-11-29", "Raja Babu")
	if err != nil || r.Mulank.Value != 2 || r.Bhagyank.Value != 1 || r.Extended.Naamank.Number.Value != 7 {
		t.Fatal(r, err)
	}
	before, _ := json.Marshal(r)
	r.SourceReadings[0].Text = "changed"
	again, _ := CalculateFull("1995-11-29", "Raja Babu")
	after, _ := json.Marshal(again)
	if string(before) != string(after) {
		t.Fatal("profile shared mutable state")
	}
	missing, err := CalculateFull("1995-11-29", "राजा बाबू")
	if err != nil || missing.Extended.NameStatus != "needs_latin_spelling" || missing.Extended.Naamank != nil || missing.Mulank.Value != 2 {
		t.Fatal("unsupported script affected date reading", err)
	}
	if _, err := CalculateFull("1995-02-30", "Raja Babu"); err == nil {
		t.Fatal("invalid date admitted")
	}
}

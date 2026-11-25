package swisseph

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestGurujiMangalTextProvenance verifies archived bytes, original text and the 24 numbered headings without normalizing Guruji's words.
func TestGurujiMangalTextProvenance(t *testing.T) {
	var source struct {
		Source struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
			Bytes  int    `json:"bytes"`
		} `json:"source"`
		Original string                `json:"original_text"`
		Rules    []gurujiPariharSource `json:"rules"`
	}
	if err := json.Unmarshal(gurujiMangalPariharJSON, &source); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../" + source.Source.File)
	if err != nil || len(data) != source.Source.Bytes || string(data) != source.Original || fmt.Sprintf("%x", sha256.Sum256(data)) != source.Source.SHA256 {
		t.Fatal("source text changed", err)
	}
	if len(source.Rules) != 24 {
		t.Fatal("missing source rules")
	}
	for i, rule := range source.Rules {
		if rule.Number != i+1 || !bytes.Contains(data, []byte(rule.Heading)) || !bytes.Contains(data, []byte(rule.Body)) {
			t.Fatal("rewritten or missing source passage", i+1)
		}
	}
}

// pariharFixture supplies a raw Manglik placement with Jupiter away from Mars and no automatic source cancellation.
func pariharFixture() Kundali {
	c := mangalFixture()
	c.Lagna = zodiacPosition(160)
	setMangalFixture(&c, "Moon", 340)
	setMangalFixture(&c, "Jupiter", 222)
	return c
}

// pariharRule locates a numbered rule so tests assert source-specific effects rather than implementation ordering.
func pariharRule(t *testing.T, assessment MangalAssessment, number int) MangalRule {
	t.Helper()
	for _, rule := range assessment.Rules {
		if rule.SourceNumber == number {
			return rule
		}
	}
	t.Fatalf("missing source rule %d", number)
	return MangalRule{}
}

// TestGurujiExplicitParihar checks each unqualified cancellation independently, preserving both base indications and exact reasons.
func TestGurujiExplicitParihar(t *testing.T) {
	for _, tc := range []struct {
		name                              string
		mars, lagna, moon, jupiter, venus float64
		number                            int
	}{
		{"exalted", 280, 280, 100, 222, 160, 4},
		{"first_aries", 10, 10, 190, 222, 160, 17},
		{"fourth_scorpio", 220, 130, 40, 100, 160, 17},
		{"seventh_capricorn", 280, 100, 100, 222, 160, 17},
		{"eighth_cancer", 100, 250, 280, 200, 160, 17},
		{"twelfth_sagittarius", 250, 280, 70, 200, 160, 17},
		{"jupiter_fifth", 160, 160, 340, 40, 100, 24},
		{"jupiter_seventh", 160, 160, 340, 340, 100, 24},
		{"jupiter_ninth", 160, 160, 340, 280, 100, 24},
		{"moon_venus_second", 160, 160, 190, 222, 190, 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := pariharFixture()
			c.Lagna = zodiacPosition(tc.lagna)
			for name, longitude := range map[string]float64{"Mars": tc.mars, "Moon": tc.moon, "Jupiter": tc.jupiter, "Venus": tc.venus} {
				setMangalFixture(&c, name, longitude)
			}
			got, err := CalculateMangal(c)
			if err != nil || got.BaseClassification != "mangal" {
				t.Fatal(got, err)
			}
			rule := pariharRule(t, got, tc.number)
			if !rule.Met || rule.Evaluation != "met" || rule.Effect != "cancellation" || !got.References[0].Cancelled {
				t.Fatal("missing cancellation", got)
			}
			if tc.number == 24 && (len(rule.MatchedClauses) != 1 || !strings.Contains(rule.Text, rule.MatchedClauses[0]) || tc.name == "moon_venus_second" && !strings.HasPrefix(rule.MatchedClauses[0], "1.") || strings.HasPrefix(tc.name, "jupiter_") && !strings.HasPrefix(rule.MatchedClauses[0], "2.")) {
				t.Fatal("wrong source clause attached to cancellation", rule)
			}
			if tc.number != 17 && (got.Classification != "cancelled" || got.ActiveCount != 0) {
				t.Fatal("global cancellation incomplete", got)
			}
		})
	}
	// Source 17 is a natal house/sign exception, not a sign-only exception in every reference frame.
	c := pariharFixture()
	c.Lagna = zodiacPosition(10)
	setMangalFixture(&c, "Mars", 10)
	setMangalFixture(&c, "Moon", 190)
	setMangalFixture(&c, "Jupiter", 222)
	got, _ := CalculateMangal(c)
	if got.Classification != "partial" || !got.References[0].Cancelled || got.References[1].Cancelled || got.ActiveCount != 1 || got.BaseCount != 2 {
		t.Fatal("Lagna exception erased Moon indication", got)
	}
}

// TestGurujiConditionalPariharNeverAutoCancels checks conditional/disputed rules, conjunctions, nodes and one-planet false positives.
func TestGurujiConditionalPariharNeverAutoCancels(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*Kundali)
	}{
		{"own_sign", func(c *Kundali) {
			c.Lagna = zodiacPosition(190)
			setMangalFixture(c, "Mars", 10)
			setMangalFixture(c, "Moon", 190)
		}},
		{"jupiter_conjunction", func(c *Kundali) { setMangalFixture(c, "Jupiter", 162) }},
		{"moon_conjunction", func(c *Kundali) { setMangalFixture(c, "Moon", 162) }},
		{"mercury_aspect", func(c *Kundali) { setMangalFixture(c, "Mercury", 340) }},
		{"retrograde_combust", func(c *Kundali) { c.Planets[2].Speed = -.2; setMangalFixture(c, "Sun", 161) }},
		{"disputed_seventh_taurus", func(c *Kundali) {
			c.Lagna = zodiacPosition(220)
			setMangalFixture(c, "Mars", 40)
			setMangalFixture(c, "Moon", 220)
			setMangalFixture(c, "Jupiter", 100)
		}},
		{"disputed_eighth_aquarius", func(c *Kundali) {
			c.Lagna = zodiacPosition(100)
			setMangalFixture(c, "Mars", 310)
			setMangalFixture(c, "Moon", 100)
			setMangalFixture(c, "Jupiter", 40)
		}},
		{"rahu_conjunction", func(c *Kundali) { setMangalFixture(c, "Rahu", 162) }},
		{"moon_only_second", func(c *Kundali) { setMangalFixture(c, "Moon", 190); setMangalFixture(c, "Venus", 250) }},
		{"venus_only_second", func(c *Kundali) { setMangalFixture(c, "Venus", 190) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := pariharFixture()
			tc.mutate(&c)
			got, err := CalculateMangal(c)
			if err != nil || got.Classification != "mangal" || got.ActiveCount != got.BaseCount {
				t.Fatal("conditional evidence cancelled the dosha", got, err)
			}
		})
	}
	c := pariharFixture()
	c.Planets = c.Planets[:3]
	got, err := CalculateMangal(c)
	if err != nil || got.Classification != "mangal" || pariharRule(t, got, 24).Evaluation != "review" {
		t.Fatal("missing planets silently supplied", got, err)
	}
}

// TestGurujiPairBalanceDoesNotEraseNatal keeps pair matching separate from natal diagnoses and source passages.
func TestGurujiPairBalanceDoesNotEraseNatal(t *testing.T) {
	c := pariharFixture()
	out, err := CalculateMangalPair(c, c)
	if err != nil || out.Status != "pair_exception_found" || out.Bride.Classification != "mangal" || out.Groom.Classification != "mangal" || !out.Rules[0].Met {
		t.Fatal(out, err)
	}
	for _, rule := range out.Rules {
		if rule.SourceNumber != 1 && (rule.Effect != "review" || rule.Evaluation != "review") {
			t.Fatal("undefined compatibility threshold applied", rule)
		}
	}
}

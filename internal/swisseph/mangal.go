package swisseph

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
)

// MangalMethod versions Guruji's recorded base rule and the explicit cancellations in his supplied Hindi text.
const MangalMethod = "m4y-mangal-guruji-audio-parihar-2026-11-25-v2"

//go:embed data/guruji-mangal-rules.json
var gurujiMangalJSON []byte

type gurujiMangalConfig struct {
	Method     string   `json:"method"`
	References []string `json:"references"`
	Houses     []int    `json:"houses"`
}

var gurujiMangalRules = loadGurujiMangalRules()

// loadGurujiMangalRules validates the bundled source configuration before any chart is assessed.
func loadGurujiMangalRules() gurujiMangalConfig {
	var rules gurujiMangalConfig
	if err := json.Unmarshal(gurujiMangalJSON, &rules); err != nil || rules.Method != MangalMethod || len(rules.References) != 2 || rules.References[0] != "lagna" || rules.References[1] != "moon" {
		panic("invalid Guruji Mangal source configuration")
	}
	expected := []int{1, 4, 7, 8, 12}
	if len(rules.Houses) != len(expected) {
		panic("invalid Guruji Mangal houses")
	}
	for i, house := range expected {
		if rules.Houses[i] != house {
			panic("changed Guruji Mangal houses")
		}
	}
	return rules
}

type MangalRule struct {
	Code           string   `json:"code"`
	Met            bool     `json:"met"`
	Reference      string   `json:"reference"`
	SourcePage     int      `json:"source_pdf_page"`
	SourceNumber   int      `json:"source_rule_number,omitempty"`
	Heading        string   `json:"heading,omitempty"`
	Text           string   `json:"text,omitempty"`
	Effect         string   `json:"effect,omitempty"`
	Evaluation     string   `json:"evaluation,omitempty"`
	MatchedClauses []string `json:"matched_clauses,omitempty"`
}
type MangalReference struct {
	Reference  string   `json:"reference"`
	House      int      `json:"house"`
	Flagged    bool     `json:"flagged"`
	Cancelled  bool     `json:"cancelled"`
	Exceptions []string `json:"exceptions"`
}
type MangalAssessment struct {
	Classification     string            `json:"classification,omitempty"`
	BaseClassification string            `json:"base_classification,omitempty"`
	Strength           *MangalStrength   `json:"strength,omitempty"`
	References         []MangalReference `json:"references"`
	BaseCount          int               `json:"base_count"`
	ActiveCount        int               `json:"active_count"`
	BaseLevel          string            `json:"base_level"`
	Level              string            `json:"level"`
	SunSeparation      float64           `json:"sun_separation_degrees"`
	Rules              []MangalRule      `json:"rules"`
	ReviewConditions   []string          `json:"review_conditions"`
	NotEvaluated       []string          `json:"not_evaluated"`
}
type MangalPair struct {
	Method string           `json:"method"`
	Bride  MangalAssessment `json:"bride"`
	Groom  MangalAssessment `json:"groom"`
	Status string           `json:"status"`
	Rules  []MangalRule     `json:"rules"`
}

// Historical strength fields remain solely for decoding stored reports; the Guruji method does not compute them.
type MangalStrengthCondition struct {
	Planet      string   `json:"planet"`
	Eligible    bool     `json:"eligible"`
	Ratio       float64  `json:"ratio"`
	Confirmed   bool     `json:"confirmed"`
	Limitations []string `json:"limitations"`
}
type MangalStrength struct {
	Method     string                    `json:"method"`
	Conditions []MangalStrengthCondition `json:"conditions"`
}

// MangalSensitiveHouse checks only the five houses stated in Guruji's first recording.
func MangalSensitiveHouse(house int) bool {
	for _, h := range gurujiMangalRules.Houses {
		if house == h {
			return true
		}
	}
	return false
}

// relativeSign counts inclusive whole-sign distance between two zodiac positions.
func relativeSign(planet, reference int) int { return (planet-reference+12)%12 + 1 }

// mangalPositions extracts validated natal positions for Guruji's Lagna/Moon Mangal assessment.
func mangalPositions(chart Kundali) (map[string]Graha, error) {
	if !finite(chart.Lagna.Longitude) || chart.Lagna.Longitude < 0 || chart.Lagna.Longitude >= 360 {
		return nil, fmt.Errorf("invalid natal ascendant")
	}
	byName := map[string]Graha{}
	for _, p := range chart.Planets {
		if _, duplicate := byName[p.Name]; duplicate || !finite(p.Longitude) || p.Longitude < 0 || p.Longitude >= 360 || !finite(p.Speed) {
			return nil, fmt.Errorf("invalid or duplicate natal planet")
		}
		p.ZodiacPosition = zodiacPosition(p.Longitude)
		byName[p.Name] = p
	}
	for _, name := range []string{"Moon", "Mars"} {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("missing natal %s", name)
		}
	}
	return byName, nil
}

// CalculateMangal keeps Guruji's recorded base placement separate from supported text cancellations and conditional mitigations.
// A remaining Lagna indication is Manglik; a remaining Moon-only indication is partial. Legacy levels encode classes, not strength.
func CalculateMangal(chart Kundali) (MangalAssessment, error) {
	positions, err := mangalPositions(chart)
	if err != nil {
		return MangalAssessment{}, err
	}
	mars := positions["Mars"]
	lagna := zodiacPosition(chart.Lagna.Longitude).RashiNumber
	out := MangalAssessment{References: []MangalReference{}, Rules: []MangalRule{}, ReviewConditions: []string{}, NotEvaluated: []string{}, Classification: "none", BaseLevel: "none", Level: "none"}
	if sun, ok := positions["Sun"]; ok {
		separation := math.Abs(mars.Longitude - sun.Longitude)
		out.SunSeparation = math.Min(separation, 360-separation)
	}
	signs := map[string]int{"lagna": lagna, "moon": positions["Moon"].RashiNumber}
	for _, name := range gurujiMangalRules.References {
		h := relativeSign(mars.RashiNumber, signs[name])
		ref := MangalReference{Reference: name, House: h, Flagged: MangalSensitiveHouse(h), Exceptions: []string{}}
		if ref.Flagged {
			out.BaseCount++
			out.ActiveCount++
		}
		out.References = append(out.References, ref)
	}
	if out.References[0].Flagged {
		out.Classification, out.Level = "mangal", "high"
	} else if out.References[1].Flagged {
		out.Classification, out.Level = "partial", "low"
	}
	out.BaseLevel = out.Level
	out.BaseClassification = out.Classification
	applyGurujiMangalParihar(&out, positions, lagna)
	return out, nil
}

// CalculateMangalPair records partner balance separately so no partner's natal diagnosis or Guna score is erased.
func CalculateMangalPair(bride, groom Kundali) (MangalPair, error) {
	b, err := CalculateMangal(bride)
	if err != nil {
		return MangalPair{}, fmt.Errorf("bride: %w", err)
	}
	g, err := CalculateMangal(groom)
	if err != nil {
		return MangalPair{}, fmt.Errorf("groom: %w", err)
	}
	out := MangalPair{Method: MangalMethod, Bride: b, Groom: g, Rules: []MangalRule{}, Status: "different_active_levels"}
	if b.Classification == "none" && g.Classification == "none" {
		out.Status = "no_active_flags"
	} else if b.Classification == g.Classification {
		out.Status = "similar_active_levels"
	}
	applyGurujiMangalPairParihar(&out, bride, groom)
	return out, nil
}

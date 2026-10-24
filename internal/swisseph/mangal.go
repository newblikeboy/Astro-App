package swisseph

import (
	"fmt"
	"math"
)

// A declared six-house extension of AIFAS Horoscope Matching (2021), chapter12,
// PDFpp85,87-89. House2 is included as the regional extension documented at
// https://www.astrosage.com/free/mangal-dosha-report.asp . All reference frames,
// conjunctions and aspects here are whole-sign. This is not a strength system.
const MangalMethod = "m4y-mangal-aifas2021-six-house-triad-v1"

type MangalRule struct {
	Code       string `json:"code"`
	Met        bool   `json:"met"`
	Reference  string `json:"reference"`
	SourcePage int    `json:"source_pdf_page"`
}
type MangalReference struct {
	Reference  string   `json:"reference"`
	House      int      `json:"house"`
	Flagged    bool     `json:"flagged"`
	Cancelled  bool     `json:"cancelled"`
	Exceptions []string `json:"exceptions"`
}
type MangalAssessment struct {
	References       []MangalReference `json:"references"`
	BaseCount        int               `json:"base_count"`
	ActiveCount      int               `json:"active_count"`
	BaseLevel        string            `json:"base_level"`
	Level            string            `json:"level"`
	SunSeparation    float64           `json:"sun_separation_degrees"`
	Rules            []MangalRule      `json:"rules"`
	ReviewConditions []string          `json:"review_conditions"`
	NotEvaluated     []string          `json:"not_evaluated"`
}
type MangalPair struct {
	Method string           `json:"method"`
	Bride  MangalAssessment `json:"bride"`
	Groom  MangalAssessment `json:"groom"`
	Status string           `json:"status"`
	Rules  []MangalRule     `json:"rules"`
}

// mangalFlag classifies whether Mars occupies a declared sensitive house.
func mangalFlag(h int) bool { return h == 1 || h == 2 || h == 4 || h == 7 || h == 8 || h == 12 }

// relativeSign counts the inclusive whole-sign distance between two zodiac positions.
func relativeSign(planet, reference int) int { return (planet-reference+12)%12 + 1 }

// mangalLevel summarizes the Mangal assessment's level from its recorded reference evidence.
func mangalLevel(count int) string {
	if count == 0 {
		return "none"
	}
	if count == 1 {
		return "low"
	}
	return "high"
}

// mangalPositions extracts the Lagna, Moon, Venus and Mars positions required by the Mangal assessment.
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
	for _, name := range []string{"Sun", "Moon", "Mars", "Jupiter", "Venus", "Saturn", "Rahu"} {
		if _, ok := byName[name]; !ok {
			return nil, fmt.Errorf("missing natal %s", name)
		}
	}
	return byName, nil
}

// mangalHouseException looks up the declared Mars sign/house exception without hiding the raw placement.
func mangalHouseException(sign, house int) bool {
	switch house {
	case 1:
		return sign == 1 || sign == 10
	case 4:
		return sign == 1 || sign == 4 || sign == 8
	case 7:
		return sign == 1 || sign == 2 || sign == 4 || sign == 10
	case 8:
		return sign == 11 || sign == 12
	case 12:
		return sign == 1 || sign == 2 || sign == 3 || sign == 6 || sign == 7 || sign == 8 || sign == 9
	}
	return false
}

// CalculateMangal keeps raw geometry and selected cancellations separately.
// Severity is an explicitly defined coverage count:0none,1low,2/3high.
// It does not alter Ashtakoota points or predict any person's behaviour/health.
func CalculateMangal(chart Kundali) (MangalAssessment, error) {
	positions, err := mangalPositions(chart)
	if err != nil {
		return MangalAssessment{}, err
	}
	mars, sun, jupiter := positions["Mars"], positions["Sun"], positions["Jupiter"]
	lagna := zodiacPosition(chart.Lagna.Longitude).RashiNumber
	out := MangalAssessment{References: []MangalReference{}, Rules: []MangalRule{}, ReviewConditions: []string{}, NotEvaluated: []string{"full_planetary_strength", "all_benefic_malefic_distribution", "unspecified_high_guna_cancellation", "age_based_exceptions"}}
	sep := math.Abs(mars.Longitude - sun.Longitude)
	out.SunSeparation = math.Min(sep, 360-sep)
	global := []string{}
	// rule appends one labelled Mangal rule and its supporting condition to the assessment.
	rule := func(code string, met bool, page int) {
		out.Rules = append(out.Rules, MangalRule{code, met, "all", page})
		if met {
			global = append(global, code)
		}
	}
	rule("mars_own_or_exalted", mars.RashiNumber == 1 || mars.RashiNumber == 8 || mars.RashiNumber == 10, 87)
	rule("mars_debilitated", mars.RashiNumber == 4, 87)
	rule("mars_retrograde", mars.Speed < 0, 87)
	// The17degree inclusive Mars combustion limit follows the AIFAS publisher's
	// Astrological Assessment of Timing of Childbirth, combustion table.
	rule("mars_combust", out.SunSeparation <= 17, 87)
	rule("mars_conjunct_jupiter", mars.RashiNumber == jupiter.RashiNumber, 88)
	rule("mars_conjunct_moon", mars.RashiNumber == positions["Moon"].RashiNumber, 88)
	rule("mars_conjunct_rahu", mars.RashiNumber == positions["Rahu"].RashiNumber, 88)
	jupiterAspect := relativeSign(mars.RashiNumber, jupiter.RashiNumber)
	rule("jupiter_aspects_seventh_mars", relativeSign(mars.RashiNumber, lagna) == 7 && (jupiterAspect == 5 || jupiterAspect == 7 || jupiterAspect == 9), 88)
	for _, ref := range []struct {
		name string
		sign int
	}{{"lagna", lagna}, {"moon", positions["Moon"].RashiNumber}, {"venus", positions["Venus"].RashiNumber}} {
		h := relativeSign(mars.RashiNumber, ref.sign)
		check := MangalReference{Reference: ref.name, House: h, Flagged: mangalFlag(h), Exceptions: []string{}}
		exception := check.Flagged && mangalHouseException(mars.RashiNumber, h)
		sourcePage := 87
		if mars.RashiNumber == 10 && (h == 1 || h == 7) || mars.RashiNumber == 12 && h == 8 {
			sourcePage = 89
		}
		out.Rules = append(out.Rules, MangalRule{"sign_house_exception", exception, ref.name, sourcePage})
		if check.Flagged {
			out.BaseCount++
			check.Exceptions = append(check.Exceptions, global...)
			if exception {
				check.Exceptions = append(check.Exceptions, "sign_house_exception")
			}
			check.Cancelled = len(check.Exceptions) > 0
			if !check.Cancelled {
				out.ActiveCount++
			}
		}
		out.References = append(out.References, check)
	}
	// A sign/angular condition is only a candidate: the source additionally
	// requires strength. Never silently treat that uncomputed requirement as met.
	for _, name := range []string{"Jupiter", "Venus"} {
		p := positions[name]
		h := relativeSign(p.RashiNumber, lagna)
		dignified := (name == "Jupiter" && (p.RashiNumber == 4 || p.RashiNumber == 9 || p.RashiNumber == 12)) || (name == "Venus" && (p.RashiNumber == 2 || p.RashiNumber == 7 || p.RashiNumber == 12))
		if (h == 1 || h == 7) && dignified {
			out.ReviewConditions = append(out.ReviewConditions, "benefic_strength_"+name)
		}
	}
	out.BaseLevel, out.Level = mangalLevel(out.BaseCount), mangalLevel(out.ActiveCount)
	return out, nil
}

// CalculateMangalPair compares both participants' separate Mangal evidence and applicable cancellation rules.
func CalculateMangalPair(bride, groom Kundali) (MangalPair, error) {
	b, err := CalculateMangal(bride)
	if err != nil {
		return MangalPair{}, fmt.Errorf("bride: %w", err)
	}
	g, err := CalculateMangal(groom)
	if err != nil {
		return MangalPair{}, fmt.Errorf("groom: %w", err)
	}
	out := MangalPair{Method: MangalMethod, Bride: b, Groom: g, Rules: []MangalRule{}}
	for _, pair := range []struct {
		role         string
		owner, other Kundali
		assessment   MangalAssessment
	}{{"bride", bride, groom, b}, {"groom", groom, bride, g}} {
		p, _ := mangalPositions(pair.owner)
		q, _ := mangalPositions(pair.other)
		h := relativeSign(p["Mars"].RashiNumber, zodiacPosition(pair.owner.Lagna.Longitude).RashiNumber)
		otherLagna := zodiacPosition(pair.other.Lagna.Longitude).RashiNumber
		saturn := relativeSign(q["Saturn"].RashiNumber, otherLagna)
		rahu := relativeSign(q["Rahu"].RashiNumber, otherLagna)
		northFlag := h == 1 || h == 4 || h == 7 || h == 8 || h == 12
		otherSaturnFlag := saturn == 1 || saturn == 4 || saturn == 7 || saturn == 8 || saturn == 12
		upachaya := false
		for _, name := range []string{"Mars", "Saturn", "Rahu"} {
			v := relativeSign(q[name].RashiNumber, otherLagna)
			upachaya = upachaya || v == 3 || v == 6 || v == 11
		}
		active := pair.assessment.ActiveCount > 0
		out.Rules = append(out.Rules,
			MangalRule{"other_saturn_or_rahu_same_house", active && mangalFlag(h) && (saturn == h || rahu == h), pair.role, 88},
			MangalRule{"other_saturn_in_mangal_house", active && northFlag && otherSaturnFlag, pair.role, 88},
			MangalRule{"other_malefic_in_upachaya", active && upachaya, pair.role, 88})
	}
	if b.ActiveCount == 0 && g.ActiveCount == 0 {
		out.Status = "no_active_flags"
	} else if b.Level == g.Level {
		out.Status = "similar_active_levels"
	} else {
		out.Status = "different_active_levels"
		for _, rule := range out.Rules {
			if rule.Met {
				out.Status = "pair_exception_found"
				break
			}
		}
	}
	return out, nil
}

package swisseph

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/guruji-mangal-parihar.json
var gurujiMangalPariharJSON []byte

type gurujiPariharSource struct {
	Number  int    `json:"number"`
	Heading string `json:"heading"`
	Body    string `json:"body"`
}

var gurujiMangalParihar = loadGurujiMangalParihar()

// loadGurujiMangalParihar checks source numbering before exposing verbatim Hindi rules as calculation evidence.
func loadGurujiMangalParihar() []gurujiPariharSource {
	var source struct {
		Rules []gurujiPariharSource `json:"rules"`
	}
	if json.Unmarshal(gurujiMangalPariharJSON, &source) != nil || len(source.Rules) != 24 {
		panic("invalid Guruji Mangal parihar source")
	}
	for i, rule := range source.Rules {
		if rule.Number != i+1 || rule.Heading == "" || rule.Body == "" {
			panic("incomplete Guruji Mangal parihar source")
		}
	}
	return source.Rules
}

// gurujiPariharRule attaches exact source wording and distinguishes an observed condition from a confirmed cancellation.
func gurujiPariharRule(number int, reference, effect, evaluation string, met bool) MangalRule {
	source := gurujiMangalParihar[number-1]
	return MangalRule{Code: fmt.Sprintf("guruji-parihar-%d", number), Met: met, Reference: reference, SourceNumber: number, Heading: source.Heading, Text: source.Body, Effect: effect, Evaluation: evaluation}
}

// gurujiObservedRule records an objectively testable condition; missing supporting planets require review, never a zero-longitude substitute.
func gurujiObservedRule(number int, reference, effect string, known, met bool) MangalRule {
	evaluation := "not_met"
	if !known {
		evaluation = "review"
	} else if met {
		evaluation = "met"
	}
	return gurujiPariharRule(number, reference, effect, evaluation, known && met)
}

// gurujiSpecialMangalHouse checks only the five explicit house/sign combinations in source rule 17.
func gurujiSpecialMangalHouse(house, sign int) bool {
	return map[int]int{1: 1, 4: 8, 7: 10, 8: 4, 12: 9}[house] == sign
}

// applyGurujiMangalParihar applies only unqualified cancellations (4, 17 and the explicit clauses of 24).
// Conditional, disputed and strength-dependent passages remain supporting evidence or review, without invented thresholds.
func applyGurujiMangalParihar(out *MangalAssessment, positions map[string]Graha, lagna int) {
	if out.BaseCount == 0 {
		return
	}
	mars := positions["Mars"]
	jupiter, hasJupiter := positions["Jupiter"]
	mercury, hasMercury := positions["Mercury"]
	venus, hasVenus := positions["Venus"]
	rahu, hasRahu := positions["Rahu"]
	moon := positions["Moon"]
	jupiterAspect := hasJupiter && grahaAspectSign("Jupiter", jupiter.RashiNumber, mars.RashiNumber)
	secondMoonVenus := hasVenus && relativeSign(moon.RashiNumber, lagna) == 2 && relativeSign(venus.RashiNumber, lagna) == 2
	navamsa := navamsaSign(mars.Longitude)
	lagnaHouse := out.References[0].House
	for number := 3; number <= 19; number++ {
		rule := gurujiPariharRule(number, "natal", "review", "review", false)
		switch number {
		case 3:
			rule = gurujiObservedRule(number, "natal", "mitigation", true, mars.RashiNumber == 1 || mars.RashiNumber == 8)
		case 4:
			rule = gurujiObservedRule(number, "all", "cancellation", true, mars.RashiNumber == 10)
		case 5:
			rule = gurujiObservedRule(number, "d9", "mitigation", true, navamsa == 1 || navamsa == 8 || navamsa == 10)
		case 6:
			rule = gurujiObservedRule(number, "natal", "mitigation", hasJupiter, jupiterAspect)
		case 7:
			rule = gurujiObservedRule(number, "natal", "mitigation", hasJupiter, jupiter.RashiNumber == mars.RashiNumber)
		case 8:
			rule = gurujiObservedRule(number, "natal", "mitigation", true, moon.RashiNumber == mars.RashiNumber)
		case 9:
			rule = gurujiObservedRule(number, "natal", "mitigation", hasMercury, mercury.RashiNumber == mars.RashiNumber || grahaAspectSign("Mercury", mercury.RashiNumber, mars.RashiNumber))
		case 10, 11:
			// Presence alone does not establish the source's unspecified "strong" requirement.
			house := 1
			if number == 11 {
				house = 7
			}
			rule.Met = hasJupiter && relativeSign(jupiter.RashiNumber, lagna) == house || hasVenus && relativeSign(venus.RashiNumber, lagna) == house
			if hasJupiter && hasVenus && !rule.Met {
				rule.Evaluation = "not_met"
			}
		case 12:
			rule.Met = mars.RashiNumber == 4
			if !rule.Met {
				rule.Evaluation = "not_met"
			}
		case 13:
			// No enemy-sign list is supplied in this text; do not silently import a former rule table.
		case 14:
			// "Extremely near the Sun" has no angular cutoff in the supplied source.
		case 15:
			rule.Met = mars.Speed < 0
			if !rule.Met {
				rule.Evaluation = "not_met"
			}
		case 16:
			lord, known := positions[matchingLords[matchingSignLords[mars.RashiNumber-1]]]
			house := relativeSign(lord.RashiNumber, lagna)
			rule.Met = known && (house == 1 || house == 4 || house == 7 || house == 10 || house == 5 || house == 9)
			if known && !rule.Met {
				rule.Evaluation = "not_met"
			}
		case 17:
			rule = gurujiObservedRule(number, "lagna", "cancellation", true, gurujiSpecialMangalHouse(lagnaHouse, mars.RashiNumber))
		case 18:
			// The source explicitly calls these disputed, not a universal automatic cancellation.
			rule.Met = lagnaHouse == 7 && mars.RashiNumber == 2 || lagnaHouse == 8 && mars.RashiNumber == 11
			if !rule.Met {
				rule.Evaluation = "not_met"
			}
		}
		out.Rules = append(out.Rules, rule)
	}
	out.Rules = append(out.Rules, gurujiPariharRule(23, "natal", "review", "review", false))
	// Rule 24 states two clear cancellations; its unspecified node clauses stay explicitly unevaluated.
	rule24 := gurujiObservedRule(24, "all", "cancellation", hasJupiter && hasVenus, jupiterAspect || secondMoonVenus)
	if jupiterAspect || secondMoonVenus {
		rule24.Met, rule24.Evaluation = true, "met"
	}
	// Identify the actual clause: the source's rule-24 heading names Moon/Venus even when its Jupiter clause applies.
	for _, line := range strings.Split(rule24.Text, "\n") {
		line = strings.TrimSpace(line)
		if secondMoonVenus && strings.HasPrefix(line, "1. ") || jupiterAspect && strings.HasPrefix(line, "2. ") {
			rule24.MatchedClauses = append(rule24.MatchedClauses, line)
		}
	}
	out.Rules = append(out.Rules, rule24)
	out.NotEvaluated = append(out.NotEvaluated, "guruji-parihar-24-node-clauses", "guruji-parihar-8-other-lunar-influences")
	if hasRahu && (kendraFrom(rahu.RashiNumber, lagna) || rahu.RashiNumber == mars.RashiNumber) {
		out.ReviewConditions = append(out.ReviewConditions, "guruji-parihar-24-node-clauses")
	}
	for _, rule := range out.Rules {
		if rule.Evaluation == "review" {
			out.ReviewConditions = append(out.ReviewConditions, rule.Code)
		}
		if rule.Effect != "cancellation" || rule.Evaluation != "met" {
			continue
		}
		for i := range out.References {
			ref := &out.References[i]
			if ref.Flagged && (rule.Reference == "all" || rule.Reference == ref.Reference) {
				ref.Cancelled = true
				ref.Exceptions = append(ref.Exceptions, rule.Code)
			}
		}
	}
	out.ActiveCount = 0
	for _, ref := range out.References {
		if ref.Flagged && !ref.Cancelled {
			out.ActiveCount++
		}
	}
	switch {
	case out.References[0].Flagged && !out.References[0].Cancelled:
		out.Classification, out.Level = "mangal", "high"
	case out.References[1].Flagged && !out.References[1].Cancelled:
		out.Classification, out.Level = "partial", "low"
	default:
		out.Classification, out.Level = "cancelled", "none"
	}
}

// applyGurujiMangalPairParihar balances matching active classes under rule 1 without changing either natal assessment.
// Rule 2's equivalence and rules 20-22's friendship, other yogas and sufficient-point cutoffs need Guruji's definitions.
func applyGurujiMangalPairParihar(out *MangalPair, bride, groom Kundali) {
	if out.Bride.ActiveCount == 0 && out.Groom.ActiveCount == 0 {
		return
	}
	balanced := out.Bride.ActiveCount > 0 && out.Groom.ActiveCount > 0 && out.Bride.Classification == out.Groom.Classification
	out.Rules = append(out.Rules, gurujiObservedRule(1, "pair", "balance", true, balanced))
	if balanced {
		out.Status = "pair_exception_found"
	}
	for _, number := range []int{2, 20, 21, 22} {
		rule := gurujiPariharRule(number, "pair", "review", "review", false)
		if number == 2 {
			// Only the explicitly named three malefics are observed; "other equivalent" planets are undefined.
			for i, chart := range []Kundali{bride, groom} {
				other := out.Groom
				if i == 1 {
					other = out.Bride
				}
				if other.ActiveCount == 0 {
					continue
				}
				lagna := zodiacPosition(chart.Lagna.Longitude).RashiNumber
				for _, planet := range chart.Planets {
					if (planet.Name == "Saturn" || planet.Name == "Rahu" || planet.Name == "Ketu") && MangalSensitiveHouse(relativeSign(zodiacPosition(planet.Longitude).RashiNumber, lagna)) {
						rule.Met = true
					}
				}
			}
		}
		out.Rules = append(out.Rules, rule)
	}
}

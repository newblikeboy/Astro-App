package swisseph

import "fmt"

// MangalStrengthMethod adds the source's explicitly strong Jupiter/Venus
// exception to the unchanged six-house base assessment (AIFAS PDF p.87).
const MangalStrengthMethod = "m4y-mangal-aifas2021-shadbala-v2"

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

// QualifyMangalStrength resolves the formerly uncomputed strong-benefic
// candidate using all six strength groups, combustion and planetary war.
// Natural Jupiter/Venus in this source's cancellation rule do not reclassify
// their separate ascendant-specific board roles. Base placements never change.
func QualifyMangalStrength(chart Kundali, base MangalAssessment, bala Shadbala, analysis PlanetAnalysis) (MangalAssessment, error) {
	positions, e := mangalPositions(chart)
	if e != nil {
		return MangalAssessment{}, e
	}
	if len(bala.Planets) != 7 || bala.Method != ShadbalaMethod {
		return MangalAssessment{}, fmt.Errorf("complete native strength required")
	}
	seen := map[string]bool{}
	for _, b := range bala.Planets {
		if seen[b.Planet] || matchingSignLordsIndex(b.Planet) < 0 || !finite(b.Ratio) {
			return MangalAssessment{}, fmt.Errorf("invalid native strength")
		}
		seen[b.Planet] = true
	}
	out := base
	out.References = append([]MangalReference{}, base.References...)
	out.Rules = append([]MangalRule{}, base.Rules...)
	out.ReviewConditions = []string{}
	out.NotEvaluated = []string{}
	for _, v := range base.NotEvaluated {
		if v != "full_planetary_strength" {
			out.NotEvaluated = append(out.NotEvaluated, v)
		}
	}
	out.Strength = &MangalStrength{Method: MangalStrengthMethod, Conditions: []MangalStrengthCondition{}}
	lagna := zodiacPosition(chart.Lagna.Longitude).RashiNumber
	for _, name := range []string{"Jupiter", "Venus"} {
		p := positions[name]
		h := relativeSign(p.RashiNumber, lagna)
		dignified := p.RashiNumber == exaltationSigns[matchingSignLordsIndex(name)] || matchingLords[matchingSignLords[p.RashiNumber-1]] == name
		condition := MangalStrengthCondition{Planet: name, Eligible: (h == 1 || h == 7) && dignified, Limitations: []string{}}
		found := false
		for _, b := range bala.Planets {
			if b.Planet == name {
				found = true
				condition.Ratio = b.Ratio
				if !finite(b.Ratio) || !b.MeetsRequirement {
					condition.Limitations = append(condition.Limitations, "below_shadbala")
				}
			}
		}
		if !found {
			return MangalAssessment{}, fmt.Errorf("missing benefic strength")
		}
		conditionFound := false
		for _, c := range analysis.Conditions {
			if c.Name != name {
				continue
			}
			conditionFound = true
			if c.Combust != nil && *c.Combust {
				condition.Limitations = append(condition.Limitations, "combust")
			}
		}
		if !conditionFound {
			return MangalAssessment{}, fmt.Errorf("missing benefic condition")
		}
		unresolved := false
		for _, w := range bala.Wars {
			if w.First != name && w.Second != name {
				continue
			}
			if w.Status == "unresolved" {
				unresolved = true
				condition.Limitations = append(condition.Limitations, "unresolved_war")
			} else if w.Winner != name {
				condition.Limitations = append(condition.Limitations, "war_defeat")
			}
		}
		condition.Confirmed = condition.Eligible && len(condition.Limitations) == 0
		if condition.Eligible && unresolved {
			out.ReviewConditions = append(out.ReviewConditions, "benefic_strength_"+name)
		}
		code := "strong_benefic_" + name
		out.Rules = append(out.Rules, MangalRule{Code: code, Met: condition.Confirmed, Reference: "all", SourcePage: 87})
		if condition.Confirmed {
			for i := range out.References {
				ref := &out.References[i]
				if !ref.Flagged {
					continue
				}
				ref.Exceptions = append(append([]string{}, ref.Exceptions...), code)
				ref.Cancelled = true
			}
		}
		out.Strength.Conditions = append(out.Strength.Conditions, condition)
	}
	out.ActiveCount = 0
	for _, ref := range out.References {
		if ref.Flagged && !ref.Cancelled {
			out.ActiveCount++
		}
	}
	out.Level = mangalLevel(out.ActiveCount)
	return out, nil
}

// matchingSignLordsIndex resolves a classical name without giving nodes
// invented sign ownership or a Shadbala index.
func matchingSignLordsIndex(name string) int {
	for i, v := range matchingLords {
		if v == name {
			return i
		}
	}
	return -1
}

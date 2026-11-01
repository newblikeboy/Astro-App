package swisseph

import "fmt"

const YogaQualificationMethod = "m4y-phaladeepika7-neechabhanga-strength-v2"

type CancellationRule struct {
	Code      string   `json:"code"`
	Met       bool     `json:"met"`
	Planets   []string `json:"planets"`
	Reference string   `json:"reference"`
}
type Neechabhanga struct {
	Planet      string             `json:"planet"`
	Debilitated bool               `json:"debilitated"`
	Cancelled   bool               `json:"cancelled"`
	Rules       []CancellationRule `json:"rules"`
}
type YogaQualification struct {
	Code           string   `json:"code"`
	BasePresent    bool     `json:"base_present"`
	Status         string   `json:"status"`
	DominantPlanet string   `json:"dominant_planet,omitempty"`
	Planets        []string `json:"planets"`
	Limitations    []string `json:"limitations"`
}
type YogaQualifications struct {
	Method       string                    `json:"method"`
	Neechabhanga []Neechabhanga            `json:"neechabhanga"`
	Yogas        []YogaQualification       `json:"yogas"`
	RajyogPairs  []RajyogPairQualification `json:"rajyog_pairs"`
}

type RajyogPairQualification struct {
	Pair          RajyogPair        `json:"pair"`
	Qualification YogaQualification `json:"qualification"`
}

// kendraFrom counts whole signs from a named reference, never Chalit houses.
func kendraFrom(sign, reference int) bool {
	h := relativeSign(sign, reference)
	return h == 1 || h == 4 || h == 7 || h == 10
}

// grahaAspectSign tests the same classical whole-sign aspects used by the natal
// engine; the nodes have no invented aspects in this convention.
func grahaAspectSign(name string, from, to int) bool {
	d := relativeSign(to, from)
	if d == 7 && name != "Rahu" && name != "Ketu" {
		return true
	}
	return name == "Mars" && (d == 4 || d == 8) || name == "Jupiter" && (d == 5 || d == 9) || name == "Saturn" && (d == 3 || d == 10)
}

// CalculateYogaQualifications tests Phaladeepika 7.26-29's four distinct
// debility-cancellation conditions. Dignity remains debilitated, even when
// cancellation is present. Every supported base yoga receives its own strength
// qualifications; a partial obstacle is not falsely called a missing yoga.
func CalculateYogaQualifications(chart Kundali, analysis PlanetAnalysis, bala Shadbala, checks NatalYogaChecks) (YogaQualifications, error) {
	out := YogaQualifications{Method: YogaQualificationMethod, Neechabhanga: []Neechabhanga{}, Yogas: []YogaQualification{}, RajyogPairs: []RajyogPairQualification{}}
	if len(chart.Planets) != 9 || len(bala.Planets) != 7 || !finite(chart.Lagna.Longitude) || chart.Lagna.Longitude < 0 || chart.Lagna.Longitude >= 360 {
		return out, fmt.Errorf("complete natal chart and strength required")
	}
	positions := map[string]Graha{}
	for _, p := range chart.Planets {
		if _, ok := positions[p.Name]; ok || !finite(p.Longitude) || p.Longitude < 0 || p.Longitude >= 360 {
			return out, fmt.Errorf("invalid yoga planet")
		}
		p.ZodiacPosition = zodiacPosition(p.Longitude)
		positions[p.Name] = p
	}
	for _, name := range skyNames {
		if _, ok := positions[name]; !ok {
			return out, fmt.Errorf("missing yoga planet")
		}
	}
	lagna, moon := zodiacPosition(chart.Lagna.Longitude).RashiNumber, positions["Moon"].RashiNumber
	cancelled := map[string]bool{}
	for i, name := range matchingLords {
		p := positions[name]
		r := Neechabhanga{Planet: name, Debilitated: p.RashiNumber == (exaltationSigns[i]+5)%12+1, Rules: []CancellationRule{}}
		dispositor := matchingLords[matchingSignLords[p.RashiNumber-1]]
		exaltLord := matchingLords[matchingSignLords[exaltationSigns[i]-1]]
		d, e := positions[dispositor], positions[exaltLord]
		r.Rules = []CancellationRule{
			{"dispositor_kendra_lagna_or_moon", kendraFrom(d.RashiNumber, lagna) || kendraFrom(d.RashiNumber, moon), []string{dispositor}, "Phaladeepika 7.26/29"},
			{"exaltation_sign_lord_kendra_lagna_or_moon", kendraFrom(e.RashiNumber, lagna) || kendraFrom(e.RashiNumber, moon), []string{exaltLord}, "Phaladeepika 7.26/29"},
			{"dispositor_and_exaltation_lord_mutual_kendra", kendraFrom(d.RashiNumber, e.RashiNumber), []string{dispositor, exaltLord}, "Phaladeepika 7.27"},
			{"dispositor_aspects_debilitated_planet", grahaAspectSign(dispositor, d.RashiNumber, p.RashiNumber), []string{dispositor}, "Phaladeepika 7.28"},
		}
		if r.Debilitated {
			for _, rule := range r.Rules {
				r.Cancelled = r.Cancelled || rule.Met
			}
		}
		cancelled[name] = r.Cancelled
		out.Neechabhanga = append(out.Neechabhanga, r)
	}
	strength := map[string]ShadbalaPlanet{}
	for _, r := range bala.Planets {
		if _, duplicate := strength[r.Planet]; duplicate || !finite(r.Ratio) {
			return out, fmt.Errorf("invalid yoga strength")
		}
		strength[r.Planet] = r
	}
	for _, name := range matchingLords {
		if _, ok := strength[name]; !ok {
			return out, fmt.Errorf("missing yoga strength")
		}
	}
	// qualify computes the same qualifications for the existing named patterns
	// and lord relationships, with no interpretation of an absent yoga as harm.
	qualify := func(code string, present bool, names []string) YogaQualification {
		r := YogaQualification{Code: code, BasePresent: present, Status: "absent", Planets: []string{}, Limitations: []string{}}
		if !present {
			return r
		}
		r.Status = "supported"
		best := -1e9
		seen := map[string]bool{}
		for _, name := range names {
			if seen[name] {
				continue
			}
			seen[name] = true
			r.Planets = append(r.Planets, name)
			b, ok := strength[name]
			if !ok {
				continue
			}
			if b.Ratio > best {
				best = b.Ratio
				r.DominantPlanet = name
			}
			if !b.MeetsRequirement {
				r.Limitations = append(r.Limitations, "below_shadbala:"+name)
			}
			for _, c := range analysis.Conditions {
				if c.Name == name {
					if c.Combust != nil && *c.Combust {
						r.Limitations = append(r.Limitations, "combust:"+name)
					}
					if c.Dignity == "Debilitation sign" && !cancelled[name] {
						r.Limitations = append(r.Limitations, "uncancelled_debility:"+name)
					}
				}
			}
			for _, war := range bala.Wars {
				if war.First != name && war.Second != name {
					continue
				}
				if war.Status == "unresolved" {
					r.Limitations = append(r.Limitations, "unresolved_war:"+name)
				} else if war.Winner != name {
					r.Limitations = append(r.Limitations, "war_defeat:"+name)
				}
			}
		}
		if len(r.Limitations) > 0 {
			r.Status = "qualified"
		}
		return r
	}
	for _, p := range analysis.Patterns {
		out.Yogas = append(out.Yogas, qualify(p.Code, p.Matches, p.Planets))
	}
	out.Yogas = append(out.Yogas, qualify("gajakesari-base", kendraFrom(positions["Jupiter"].RashiNumber, moon), []string{"Moon", "Jupiter"}))
	out.Yogas = append(out.Yogas, qualify("budha-aditya-base", positions["Sun"].RashiNumber == positions["Mercury"].RashiNumber, []string{"Sun", "Mercury"}))
	rajPlanets := []string{}
	for _, p := range checks.Rajyog.Pairs {
		// Pairs records every tested house combination, including non-matches.
		// An unrelated weak lord must not qualify a relationship that actually formed.
		if len(p.Links) == 0 {
			continue
		}
		rajPlanets = append(rajPlanets, p.KendraLord, p.TrikonaLord)
		out.RajyogPairs = append(out.RajyogPairs, RajyogPairQualification{p, qualify("kendra-trikona-rajyog", true, []string{p.KendraLord, p.TrikonaLord})})
	}
	// Single-planet Yogakaraka lordship remains separate from a matched two-lord yoga.
	out.Yogas = append(out.Yogas, qualify("kendra-trikona-rajyog", checks.Rajyog.Matches, rajPlanets))
	return out, nil
}

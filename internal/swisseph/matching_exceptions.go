package swisseph

import "fmt"

// MatchingExceptionsMethod fixes the AIFAS 2021 pp.63-64/71 exception convention;
// exceptions qualify the assessment and never silently change the base 36 points.
const MatchingExceptionsMethod = "m4y-ashtakoota-aifas2021-exceptions-v1"

type MatchingExceptionRule struct {
	Code       string `json:"code"`
	Met        bool   `json:"met"`
	SourcePage int    `json:"source_pdf_page"`
}

type MatchingDosha struct {
	Code        string                  `json:"code"`
	Status      string                  `json:"status"`
	BasePresent bool                    `json:"base_present"`
	Active      bool                    `json:"active"`
	Rules       []MatchingExceptionRule `json:"rules"`
}

type MatchingExceptions struct {
	Method  string        `json:"method"`
	Nadi    MatchingDosha `json:"nadi"`
	Bhakoot MatchingDosha `json:"bhakoot"`
}

// classicalFriends uses only the declared natural relationships for matching
// exceptions; Guruji's ascendant-specific role classification is a separate input.
func classicalFriends(a, b int) bool {
	return a != b && naturalRelations[a][b] >= 0 && naturalRelations[b][a] >= 0 && (naturalRelations[a][b] > 0 || naturalRelations[b][a] > 0)
}

// CalculateMatchingExceptions retains each tested rule, its page and raw dosha
// independently. The source's contradictory directional 5/9 paragraph and an
// ambiguous lord-1/7 sentence are not used to invent additional cancellations.
func CalculateMatchingExceptions(bride, groom Kundali) (MatchingExceptions, error) {
	b, err := matchingMoon(bride)
	if err != nil {
		return MatchingExceptions{}, err
	}
	g, err := matchingMoon(groom)
	if err != nil {
		return MatchingExceptions{}, err
	}
	bp, gp := 0, 0
	for _, p := range bride.Planets {
		if p.Name == "Moon" {
			bp = zodiacPosition(p.Longitude).Pada
		}
	}
	for _, p := range groom.Planets {
		if p.Name == "Moon" {
			gp = zodiacPosition(p.Longitude).Pada
		}
	}
	if bp == 0 || gp == 0 {
		return MatchingExceptions{}, fmt.Errorf("missing natal Moon pada")
	}
	out := MatchingExceptions{Method: MatchingExceptionsMethod, Nadi: MatchingDosha{Code: "nadi", BasePresent: b.nadi == g.nadi, Rules: []MatchingExceptionRule{}}, Bhakoot: MatchingDosha{Code: "bhakoot", Rules: []MatchingExceptionRule{}}}
	distance := relativeSign(g.sign+1, b.sign+1)
	out.Bhakoot.BasePresent = distance == 2 || distance == 12 || distance == 5 || distance == 9 || distance == 6 || distance == 8
	// add records all conditions even when the raw dosha is absent; a met
	// exception only cancels an actually present indication.
	add := func(d *MatchingDosha, code string, met bool, page int) {
		d.Rules = append(d.Rules, MatchingExceptionRule{code, met, page})
	}
	add(&out.Nadi, "same_star_different_pada", b.nak == g.nak && bp != gp, 80)
	add(&out.Nadi, "different_stars_same_sign_lord", b.nak != g.nak && b.sign != g.sign && b.lord == g.lord, 80)
	add(&out.Nadi, "different_stars_same_sign", b.nak != g.nak && b.sign == g.sign, 80)
	exempt := map[int]bool{3: true, 4: true, 5: true, 6: true, 8: true, 18: true, 22: true, 26: true, 27: true}
	add(&out.Nadi, "both_stars_in_exception_list", exempt[b.nak+1] && exempt[g.nak+1], 80)
	beneficLords := map[int]bool{3: true, 4: true, 5: true}
	add(&out.Nadi, "both_sign_lords_mercury_jupiter_venus", beneficLords[b.lord] && beneficLords[g.lord], 80)
	add(&out.Bhakoot, "same_moon_sign_lord", b.lord == g.lord, 73)
	add(&out.Bhakoot, "friendly_moon_sign_lords", classicalFriends(b.lord, g.lord), 73)
	add(&out.Bhakoot, "two_twelve_groom_even_bride_odd", (distance == 2 || distance == 12) && (g.sign+1)%2 == 0 && (b.sign+1)%2 == 1, 72)
	add(&out.Bhakoot, "six_eight_groom_sixth_even_bride_odd", distance == 6 && (g.sign+1)%2 == 0 && (b.sign+1)%2 == 1, 73)
	for _, d := range []*MatchingDosha{&out.Nadi, &out.Bhakoot} {
		d.Status = "absent"
		if !d.BasePresent {
			continue
		}
		d.Status, d.Active = "present", true
		for _, rule := range d.Rules {
			if rule.Met {
				d.Status, d.Active = "cancelled", false
			}
		}
	}
	return out, nil
}

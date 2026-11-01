package swisseph

import "fmt"

// TransitAssessmentMethod is the classical lunar-sign Gochar/Vedha table in
// PVR Narasimha Rao, Vedic Astrology: An Integrated Approach, table 63 (p.347).
const TransitAssessmentMethod = "m4y-lunar-gochar-vedha-v1"

type TransitAssessment struct {
	Method  string             `json:"method"`
	Planets []TransitJudgement `json:"planets"`
}

type TransitJudgement struct {
	Planet         string   `json:"planet"`
	FromMoon       int      `json:"from_moon"`
	FromLagna      int      `json:"from_lagna"`
	Tara           int      `json:"tara"`
	TaraFavourable bool     `json:"tara_favourable"`
	BaseFavourable bool     `json:"base_favourable"`
	VedhaHouse     int      `json:"vedha_house,omitempty"`
	BlockedBy      []string `json:"blocked_by"`
	Status         string   `json:"status"`
}

var lunarVedha = map[string]map[int]int{
	"Sun":     {3: 9, 6: 12, 10: 4, 11: 5},
	"Moon":    {1: 5, 3: 9, 6: 12, 7: 2, 10: 4, 11: 8},
	"Mars":    {3: 12, 6: 9, 11: 5},
	"Mercury": {2: 5, 4: 3, 6: 9, 8: 1, 10: 8, 11: 12},
	"Jupiter": {2: 12, 5: 4, 7: 3, 9: 10, 11: 8},
	"Venus":   {1: 8, 2: 7, 3: 1, 4: 10, 5: 9, 8: 5, 9: 11, 11: 6, 12: 3},
	"Saturn":  {3: 12, 6: 9, 11: 5},
}

// CalculateTransitAssessment combines current positions with natal Moon/Lagna
// without relabelling natal houses or assigning unlisted node Vedha rules.
// Tara remains independent of sign-Vedha status and is never a promised event.
func CalculateTransitAssessment(natal, current Kundali) (TransitAssessment, error) {
	if len(current.Planets) != 9 {
		return TransitAssessment{}, fmt.Errorf("complete current sky required")
	}
	b, err := matchingMoon(natal)
	if err != nil {
		return TransitAssessment{}, err
	}
	if !finite(natal.Lagna.Longitude) || natal.Lagna.Longitude < 0 || natal.Lagna.Longitude >= 360 {
		return TransitAssessment{}, fmt.Errorf("invalid natal ascendant")
	}
	lagna := zodiacPosition(natal.Lagna.Longitude).RashiNumber
	positions := map[string]ZodiacPosition{}
	for _, p := range current.Planets {
		if _, ok := positions[p.Name]; ok || !finite(p.Longitude) || p.Longitude < 0 || p.Longitude >= 360 {
			return TransitAssessment{}, fmt.Errorf("invalid current planet")
		}
		positions[p.Name] = zodiacPosition(p.Longitude)
	}
	for _, name := range skyNames {
		if _, ok := positions[name]; !ok {
			return TransitAssessment{}, fmt.Errorf("missing current %s", name)
		}
	}
	out := TransitAssessment{Method: TransitAssessmentMethod, Planets: []TransitJudgement{}}
	for _, name := range skyNames {
		p := positions[name]
		h := relativeSign(p.RashiNumber, b.sign+1)
		tara := ((p.NakshatraNumber-(b.nak+1)+27)%27)%9 + 1
		r := TransitJudgement{Planet: name, FromMoon: h, FromLagna: relativeSign(p.RashiNumber, lagna), Tara: tara, TaraFavourable: tara != 1 && tara != 3 && tara != 5 && tara != 7, BlockedBy: []string{}, Status: "not_assigned"}
		if table, ok := lunarVedha[name]; ok {
			r.Status = "attention"
			if vedha, good := table[h]; good {
				r.BaseFavourable, r.VedhaHouse = true, vedha
				for _, other := range skyNames {
					if other == name || (name == "Sun" && other == "Saturn") || (name == "Saturn" && other == "Sun") || (name == "Moon" && other == "Mercury") || (name == "Mercury" && other == "Moon") {
						continue
					}
					if relativeSign(positions[other].RashiNumber, b.sign+1) == vedha {
						r.BlockedBy = append(r.BlockedBy, other)
					}
				}
				r.Status = "support"
				if len(r.BlockedBy) > 0 {
					r.Status = "obstructed"
				}
			}
		}
		out.Planets = append(out.Planets, r)
	}
	return out, nil
}

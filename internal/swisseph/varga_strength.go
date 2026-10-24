package swisseph

import "fmt"

type VargaStrengthPart struct {
	Factor       int     `json:"factor"`
	Sign         int     `json:"sign"`
	Owner        string  `json:"owner"`
	Relationship string  `json:"relationship"`
	Weight       float64 `json:"weight"`
	Points       float64 `json:"points"`
	Contribution float64 `json:"contribution"`
}
type VargaStrength struct {
	Planet  string              `json:"planet"`
	Total   float64             `json:"total"`
	Maximum float64             `json:"maximum"`
	Parts   []VargaStrengthPart `json:"parts"`
}

// Shodashavarga Vimshopaka weights sum to 20. This measures varga ownership /
// compound friendship only. It is deliberately not labelled full Shadbala.
var shodashaWeights = map[int]float64{1: 3.5, 2: 1, 3: 1, 4: .5, 7: .5, 9: 3, 10: .5, 12: .5, 16: 2, 20: .5, 24: .5, 27: .5, 30: 1, 40: .5, 45: .5, 60: 4}

// CalculateVargaStrength calculates the declared seven-graha Vimshopaka scores with auditable varga weights.
func CalculateVargaStrength(vargas []Varga, analysis PlanetAnalysis) ([]VargaStrength, error) {
	if len(vargas) != 16 {
		return nil, fmt.Errorf("sixteen vargas required")
	}
	rows := []VargaStrength{}
	points := map[string]float64{"Great enemy": 5, "Enemy": 7, "Neutral": 10, "Friend": 15, "Great friend": 18}
	for _, planet := range matchingLords {
		row := VargaStrength{Planet: planet, Maximum: 20, Parts: []VargaStrengthPart{}}
		sumWeight := 0.0
		seen := map[int]bool{}
		for _, v := range vargas {
			weight, ok := shodashaWeights[v.Factor]
			if !ok || seen[v.Factor] {
				return nil, fmt.Errorf("invalid varga weights")
			}
			seen[v.Factor] = true
			sumWeight += weight
			sign := 0
			for _, p := range v.Planets {
				if p.Name == planet {
					sign = p.RashiNumber
				}
			}
			if sign < 1 || sign > 12 {
				return nil, fmt.Errorf("missing varga planet")
			}
			owner := matchingLords[matchingSignLords[sign-1]]
			relation := "Own sign"
			score := 20.0
			if owner != planet {
				relation = ""
				for _, r := range analysis.Friendships {
					if r.From == planet && r.To == owner {
						relation = r.Compound
					}
				}
				var exists bool
				score, exists = points[relation]
				if !exists {
					return nil, fmt.Errorf("missing compound friendship")
				}
			}
			contribution := weight * score / 20
			row.Total += contribution
			row.Parts = append(row.Parts, VargaStrengthPart{v.Factor, sign, owner, relation, weight, score, contribution})
		}
		if sumWeight != 20 || row.Total < 0 || row.Total > 20 {
			return nil, fmt.Errorf("invalid Vimshopaka score")
		}
		rows = append(rows, row)
	}
	return rows, nil
}

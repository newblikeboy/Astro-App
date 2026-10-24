package swisseph

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// Traditional numeric benefic-place tables; provenance and totals are recorded
// in docs/CALCULATIONS.md. Eight contributors include Lagna, never Rahu/Ketu.
//
//go:embed data/ashtakavarga.json
var ashtakavargaData []byte

type BAV struct {
	Planet       string     `json:"planet"`
	Points       [12]int    `json:"points"`
	Total        int        `json:"total"`
	Contributors [8][12]int `json:"contributors"`
}
type Ashtakavarga struct {
	Method  string   `json:"method"`
	Rows    []BAV    `json:"rows"`
	Sarva   [12]int  `json:"sarva"`
	Total   int      `json:"total"`
	Sources []string `json:"sources"`
}

// CalculateAshtakavarga builds BAV, SAV and Prastara tables from the chart under the declared contributor rules.
func CalculateAshtakavarga(chart Kundali) (Ashtakavarga, error) {
	out := Ashtakavarga{Method: "Parashari unreduced Bhinna/Prastara Ashtakavarga; SAV is seven planetary BAVs, excludes separate Lagna BAV; total 337", Rows: []BAV{}, Sources: append(append([]string{}, matchingLords[:]...), "Lagna")}
	var rules [7][8][]int
	if err := json.Unmarshal(ashtakavargaData, &rules); err != nil {
		return out, err
	}
	positions := [8]int{}
	positions[7] = chart.Lagna.RashiNumber
	for i, name := range matchingLords {
		for _, p := range chart.Planets {
			if p.Name == name {
				positions[i] = p.RashiNumber
			}
		}
	}
	for _, sign := range positions {
		if sign < 1 || sign > 12 {
			return out, fmt.Errorf("missing Ashtakavarga contributor")
		}
	}
	totals := []int{48, 49, 39, 54, 56, 52, 39}
	for target, rule := range rules {
		row := BAV{Planet: matchingLords[target]}
		for source, houses := range rule {
			for _, house := range houses {
				if house < 1 || house > 12 {
					return out, fmt.Errorf("invalid Ashtakavarga rule")
				}
				sign := (positions[source] + house - 2) % 12
				if row.Contributors[source][sign] != 0 {
					return out, fmt.Errorf("duplicate Ashtakavarga rule")
				}
				row.Contributors[source][sign] = 1
				row.Points[sign]++
				row.Total++
				out.Sarva[sign]++
				out.Total++
			}
		}
		if row.Total != totals[target] {
			return out, fmt.Errorf("invalid BAV total")
		}
		out.Rows = append(out.Rows, row)
	}
	if out.Total != 337 {
		return out, fmt.Errorf("invalid Sarvashtakavarga total")
	}
	return out, nil
}

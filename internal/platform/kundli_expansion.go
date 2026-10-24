package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"time"
)

type KundliExpansion struct {
	Method       string                    `json:"method"`
	Birth        swisseph.BirthDetails     `json:"birth"`
	Chalit       swisseph.Chalit           `json:"chalit"`
	Vargas       []swisseph.Varga          `json:"vargas"`
	Analysis     swisseph.PlanetAnalysis   `json:"analysis"`
	Ashtakavarga swisseph.Ashtakavarga     `json:"ashtakavarga"`
	Yogini       swisseph.Yogini           `json:"yogini"`
	Mangal       swisseph.MangalAssessment `json:"mangal"`
	MangalMethod string                    `json:"mangal_method"`
}

// expandKundli assembles the report's birth Panchang, vargas, planetary analysis and supplementary timelines.
func expandKundli(c Config, chart swisseph.Kundali, at, asOf time.Time, zone string) (*KundliExpansion, error) {
	birth, chalit, err := swisseph.CalculateBirthDetails(context.Background(), c.EpheDir, zone, at, chart)
	if err != nil {
		return nil, err
	}
	vargas, err := swisseph.CalculateVargas(chart)
	if err != nil {
		return nil, err
	}
	ashtaka, err := swisseph.CalculateAshtakavarga(chart)
	if err != nil {
		return nil, err
	}
	var moonLongitude float64
	for _, p := range chart.Planets {
		if p.Name == "Moon" {
			moonLongitude = p.Longitude
		}
	}
	yogini, err := swisseph.CalculateYogini(moonLongitude, at, asOf)
	if err != nil {
		return nil, err
	}
	analysis := swisseph.CalculatePlanetAnalysis(chart)
	strength, err := swisseph.CalculateVargaStrength(vargas, analysis)
	if err != nil {
		return nil, err
	}
	analysis.VargaStrength = strength
	mangal, err := swisseph.CalculateMangal(chart)
	if err != nil {
		return nil, err
	}
	return &KundliExpansion{Method: "m4y-kundli-details-v1", Birth: birth, Chalit: chalit, Vargas: vargas, Analysis: analysis, Ashtakavarga: ashtaka, Yogini: yogini, Mangal: mangal, MangalMethod: swisseph.MangalMethod}, nil
}

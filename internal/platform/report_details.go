package platform

import (
	"astroapp/backend/internal/swisseph"
	"time"
)

type PlacementCheck struct {
	Code    string `json:"code"`
	Matches bool   `json:"matches"`
	House   int    `json:"house,omitempty"`
}
type TransitPosition struct {
	swisseph.Graha
	FromMoon int `json:"from_moon"`
}
type TransitSnapshot struct {
	AsOf     time.Time         `json:"as_of"`
	Planets  []TransitPosition `json:"planets"`
	SadeSati string            `json:"sade_sati"`
}

// placementChecks derives the report's labelled placement patterns from complete, validated chart references.
func placementChecks(chart swisseph.Kundali) []PlacementCheck {
	byName := map[string]swisseph.Graha{}
	for _, p := range chart.Planets {
		byName[p.Name] = p
	}
	// relative counts inclusive zodiac signs between two known grahas, returning zero when a reference is absent.
	relative := func(planet, ref string) int {
		p, pOK := byName[planet]
		r, rOK := byName[ref]
		if !pOK || !rOK || p.RashiNumber < 1 || p.RashiNumber > 12 || r.RashiNumber < 1 || r.RashiNumber > 12 {
			return 0
		}
		return (p.RashiNumber-r.RashiNumber+12)%12 + 1
	}
	// angular identifies the four angular houses used by the placement-pattern rules.
	angular := func(h int) bool { return h == 1 || h == 4 || h == 7 || h == 10 }
	// mangal identifies the declared Mars-sensitive houses used by the placement checks.
	mangal := func(h int) bool { return h == 1 || h == 2 || h == 4 || h == 7 || h == 8 || h == 12 }
	mars := 0
	if p, ok := byName["Mars"]; ok && p.RashiNumber >= 1 && p.RashiNumber <= 12 && chart.Lagna.RashiNumber >= 1 && chart.Lagna.RashiNumber <= 12 {
		mars = (p.RashiNumber-chart.Lagna.RashiNumber+12)%12 + 1
	}
	moonMars := relative("Mars", "Moon")
	return []PlacementCheck{
		{Code: "mangal-lagna", Matches: mangal(mars), House: mars},
		{Code: "mangal-moon", Matches: mangal(moonMars), House: moonMars},
		{Code: "gajakesari-base", Matches: angular(relative("Jupiter", "Moon")), House: relative("Jupiter", "Moon")},
		{Code: "budha-aditya-base", Matches: relative("Sun", "Mercury") == 1},
	}
}

// transitSnapshot records current grahas and their houses relative to natal Lagna and Moon at report creation.
func transitSnapshot(natal, current swisseph.Kundali, at time.Time) TransitSnapshot {
	moon := 0
	for _, p := range natal.Planets {
		if p.Name == "Moon" {
			moon = p.RashiNumber
		}
	}
	out := TransitSnapshot{AsOf: at.UTC(), Planets: []TransitPosition{}, SadeSati: "none"}
	for _, p := range current.Planets {
		if p.RashiNumber < 1 || p.RashiNumber > 12 {
			continue
		}
		p.House = 0
		if natal.Lagna.RashiNumber >= 1 && natal.Lagna.RashiNumber <= 12 {
			p.House = (p.RashiNumber-natal.Lagna.RashiNumber+12)%12 + 1
		}
		fromMoon := 0
		if moon >= 1 && moon <= 12 {
			fromMoon = (p.RashiNumber-moon+12)%12 + 1
		}
		out.Planets = append(out.Planets, TransitPosition{p, fromMoon})
		if p.Name == "Saturn" {
			switch fromMoon {
			case 12:
				out.SadeSati = "rising"
			case 1:
				out.SadeSati = "peak"
			case 2:
				out.SadeSati = "setting"
			}
		}
	}
	return out
}

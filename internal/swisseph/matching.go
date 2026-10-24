package swisseph

import "fmt"

// MatchingMethod identifies one North Indian base Ashtakoota convention.
// No Nadi/Bhakoot cancellation, same-star exception or bonus is applied.
const MatchingMethod = "m4y-ashtakoota-aifas2021-base-v1"

type MatchingFactor struct {
	Code       string         `json:"code"`
	Name       string         `json:"name"`
	Score      float64        `json:"score"`
	Max        float64        `json:"max"`
	BrideValue string         `json:"bride_value"`
	GroomValue string         `json:"groom_value"`
	DetailCode string         `json:"detail_code,omitempty"`
	Evidence   map[string]int `json:"evidence,omitempty"`
}

type Matching struct {
	Method   string           `json:"method"`
	Total    float64          `json:"total"`
	Max      float64          `json:"max"`
	Factors  []MatchingFactor `json:"factors"`
	Cautions []string         `json:"cautions"`
}

// The factual category/point tables below follow AIFAS, Horoscope Matching,
// edited by Arun Bansal, third edition (December 2021), chapters 4-11 and
// printed pp.111-114 (PDF pp.119-122):
// https://www.futuresamachar.com/download/horoscope-matching-325.pdf
// They describe a traditional scoring convention, not demonstrated personal,
// medical or marital outcomes. No source prose or third-party code is used.
// All directed matrices are indexed [bride][groom].
var matchingSigns = [...]string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"}
var matchingVarnas = [...]string{"Shudra", "Vaishya", "Kshatriya", "Brahmin"}
var matchingVashyas = [...]string{"Quadruped", "Human", "Aquatic", "Wild animal", "Insect"}
var matchingTaras = [...]string{"Janma", "Sampat", "Vipat", "Kshema", "Pratyari", "Sadhaka", "Vadha", "Mitra", "Atimitra"}
var matchingYonis = [...]string{"Horse", "Elephant", "Sheep", "Serpent", "Dog", "Cat", "Rat", "Cow", "Buffalo", "Tiger", "Deer", "Monkey", "Mongoose", "Lion"}
var matchingLords = [...]string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"}
var matchingGanas = [...]string{"Deva", "Manushya", "Rakshasa"}
var matchingNadis = [...]string{"Adi", "Madhya", "Antya"}

var matchingSignLords = [...]int{2, 5, 3, 1, 0, 3, 5, 2, 4, 6, 6, 4}
var matchingNakYoni = [...]int{0, 1, 2, 3, 3, 4, 5, 2, 5, 6, 6, 7, 8, 9, 8, 9, 10, 10, 4, 11, 12, 11, 13, 0, 13, 7, 1}
var matchingNakGana = [...]int{0, 1, 2, 1, 0, 1, 0, 0, 2, 2, 1, 1, 0, 2, 0, 2, 0, 2, 2, 1, 1, 0, 2, 2, 1, 1, 0}
var matchingNakNadi = [...]int{0, 1, 2, 2, 1, 0, 0, 1, 2, 2, 1, 0, 0, 1, 2, 2, 1, 0, 0, 1, 2, 2, 1, 0, 0, 1, 2}

var matchingVashyaPoints = [5][5]float64{
	{2, 1, 1, 0, 1},
	{1, 2, 1, 0, 1},
	{1, 1, 2, 1, 1},
	{0, 0, 1, 2, 0},
	{1, 1, 1, 0, 2},
}

// AIFAS prints Tiger->Cow as 1 in one cell, but Cow->Tiger as 0. Its explicit
// reciprocal-enemy table and worked descriptions identify Cow/Tiger as a
// great-enemy pair. Use 0 in both directions; this editorial correction is
// part of the versioned convention, rather than silently mixing matrices.
var matchingYoniPoints = [14][14]float64{
	{4, 2, 2, 3, 2, 2, 2, 1, 0, 1, 3, 3, 2, 1},
	{2, 4, 3, 3, 2, 2, 2, 2, 3, 1, 2, 3, 2, 0},
	{2, 3, 4, 2, 1, 2, 1, 3, 3, 1, 2, 0, 3, 1},
	{3, 3, 2, 4, 2, 1, 1, 1, 1, 2, 2, 2, 0, 2},
	{2, 2, 1, 2, 4, 2, 1, 2, 2, 1, 0, 2, 1, 1},
	{2, 2, 2, 1, 2, 4, 0, 2, 2, 1, 3, 3, 2, 1},
	{2, 2, 1, 1, 1, 0, 4, 2, 2, 2, 2, 2, 1, 2},
	{1, 2, 3, 1, 2, 2, 2, 4, 3, 0, 3, 2, 2, 1},
	{0, 3, 3, 1, 2, 2, 2, 3, 4, 1, 2, 2, 2, 1},
	{1, 1, 1, 2, 1, 1, 2, 0, 1, 4, 1, 1, 2, 1},
	{3, 2, 2, 2, 0, 3, 2, 3, 2, 1, 4, 2, 2, 1},
	{3, 3, 0, 2, 2, 3, 2, 2, 2, 1, 2, 4, 3, 2},
	{2, 2, 3, 0, 1, 2, 1, 2, 2, 2, 2, 3, 4, 2},
	{1, 0, 1, 2, 1, 1, 2, 1, 1, 1, 1, 2, 2, 4},
}

// Combined natural relationships of the Moon-sign lords. This is neither
// temporary friendship from natal house separations nor fivefold friendship.
var matchingMaitriPoints = [7][7]float64{
	{5, 5, 5, 4, 5, 0, 0},
	{5, 5, 4, 1, 4, 0.5, 0.5},
	{5, 4, 5, 0.5, 5, 3, 0.5},
	{4, 1, 0.5, 5, 0.5, 5, 4},
	{5, 4, 5, 0.5, 5, 0.5, 3},
	{0, 0.5, 3, 5, 0.5, 5, 5},
	{0, 0.5, 0.5, 4, 3, 5, 5},
}

var matchingGanaPoints = [3][3]float64{
	{6, 5, 1},
	{6, 6, 0},
	{1, 0, 6},
}

type matchingMoonCategories struct {
	sign, nak, varna, vashya, yoni, lord, gana, nadi int
}

// matchingMoon extracts and validates the unrounded Moon position used by Ashtakoota calculations.
func matchingMoon(chart Kundali) (matchingMoonCategories, error) {
	var out matchingMoonCategories
	var moon *Graha
	for i := range chart.Planets {
		if chart.Planets[i].Name == "Moon" {
			if moon != nil {
				return out, fmt.Errorf("multiple natal Moon positions")
			}
			moon = &chart.Planets[i]
		}
	}
	if moon == nil || !finite(moon.Longitude) || moon.Longitude < 0 || moon.Longitude >= 360 {
		return out, fmt.Errorf("a finite natal Moon longitude in [0,360) is required")
	}
	// Derive every category from the unrounded physical longitude. Cached
	// sign/star/pada labels are deliberately not trusted as separate inputs.
	position := zodiacPosition(moon.Longitude)
	out.sign, out.nak = position.RashiNumber-1, position.NakshatraNumber-1
	out.varna = [...]int{2, 1, 0, 3}[out.sign%4]
	out.vashya = [...]int{0, 0, 1, 2, 3, 1, 1, 4, 1, 0, 1, 2}[out.sign]
	if out.sign == 8 && moon.Longitude >= 255 {
		out.vashya = 0 // Sagittarius: human first half, quadruped second half.
	}
	if out.sign == 9 && moon.Longitude >= 285 {
		out.vashya = 2 // Capricorn: quadruped first half, aquatic second half.
	}
	out.yoni, out.lord = matchingNakYoni[out.nak], matchingSignLords[out.sign]
	out.gana, out.nadi = matchingNakGana[out.nak], matchingNakNadi[out.nak]
	return out, nil
}

// marriageTara is an inclusive count. Janma (1) is permitted in this marriage
// convention; unlike the daily Tarabalam tool, only 3, 5 and 7 lose points.
func marriageTara(from, to int) int { return ((to-from+27)%27)%9 + 1 }

// favourableMarriageTara identifies favourable inclusive Nakshatra-count remainders in the declared Tara convention.
func favourableMarriageTara(tara int) bool {
	return tara != 3 && tara != 5 && tara != 7
}

// CalculateMatching returns the eight base factors out of 36. Bride/groom
// ordering is meaningful for the traditional Varna and Gana tables. Cautions
// describe table outcomes, not medical findings or guaranteed life events.
// Only one valid natal Moon per chart is needed by these eight factors; the
// API is responsible for deriving charts from validated known birth details.
func CalculateMatching(bride Kundali, groom Kundali) (Matching, error) {
	b, err := matchingMoon(bride)
	if err != nil {
		return Matching{}, fmt.Errorf("bride: %w", err)
	}
	g, err := matchingMoon(groom)
	if err != nil {
		return Matching{}, fmt.Errorf("groom: %w", err)
	}
	out := Matching{Method: MatchingMethod, Max: 36, Factors: make([]MatchingFactor, 0, 8), Cautions: []string{}}
	// add appends a scored Ashtakoota factor with its maximum, values and calculation evidence.
	add := func(code, name string, score, max float64, brideValue, groomValue, detail string) {
		out.Factors = append(out.Factors, MatchingFactor{Code: code, Name: name, Score: score, Max: max, BrideValue: brideValue, GroomValue: groomValue, DetailCode: detail})
		out.Total += score
	}
	varna, varnaDetail := 0.0, "varna_bride_rank_higher"
	if g.varna >= b.varna {
		varna, varnaDetail = 1, "varna_groom_rank_at_least_bride"
	}
	add("varna", "Varna", varna, 1, matchingVarnas[b.varna], matchingVarnas[g.varna], varnaDetail)
	add("vashya", "Vashya", matchingVashyaPoints[b.vashya][g.vashya], 2, matchingVashyas[b.vashya], matchingVashyas[g.vashya], "vashya_table")
	// BrideValue describes the bride's Tara counted from the groom's star;
	// GroomValue describes the groom's Tara counted from the bride's star.
	brideTara, groomTara := marriageTara(g.nak, b.nak), marriageTara(b.nak, g.nak)
	tara := 0.0
	for _, n := range []int{brideTara, groomTara} {
		if favourableMarriageTara(n) {
			tara += 1.5
		}
	}
	taraDetail := "tara_one_favourable"
	if tara == 3 {
		taraDetail = "tara_both_favourable"
	}
	add("tara", "Tara", tara, 3, fmt.Sprintf("%s (%d)", matchingTaras[brideTara-1], brideTara), fmt.Sprintf("%s (%d)", matchingTaras[groomTara-1], groomTara), taraDetail)
	out.Factors[len(out.Factors)-1].Evidence = map[string]int{"bride_count": (b.nak-g.nak+27)%27 + 1, "groom_count": (g.nak-b.nak+27)%27 + 1, "bride_class": brideTara, "groom_class": groomTara}
	yoni := matchingYoniPoints[b.yoni][g.yoni]
	yoniDetail := "yoni_table"
	if b.yoni == g.yoni {
		yoniDetail = "yoni_same"
	} else if yoni == 0 {
		yoniDetail = "yoni_great_enemy"
		out.Cautions = append(out.Cautions, "yoni_enemy")
	}
	add("yoni", "Yoni", yoni, 4, matchingYonis[b.yoni], matchingYonis[g.yoni], yoniDetail)
	maitri := matchingMaitriPoints[b.lord][g.lord]
	add("maitri", "Graha Maitri", maitri, 5, matchingLords[b.lord], matchingLords[g.lord], "maitri_natural_lords")
	if maitri <= 0.5 {
		out.Cautions = append(out.Cautions, "maitri_adverse")
	}
	gana := matchingGanaPoints[b.gana][g.gana]
	add("gana", "Gana", gana, 6, matchingGanas[b.gana], matchingGanas[g.gana], "gana_table")
	if gana <= 1 {
		out.Cautions = append(out.Cautions, "gana_adverse")
	}
	bhakoot, bhakootDetail := 7.0, "bhakoot_non_adverse"
	distance := (g.sign-b.sign+12)%12 + 1
	switch distance {
	case 2, 12:
		bhakoot, bhakootDetail = 0, "bhakoot_adverse_2_12"
	case 5, 9:
		bhakoot, bhakootDetail = 0, "bhakoot_adverse_5_9"
	case 6, 8:
		bhakoot, bhakootDetail = 0, "bhakoot_adverse_6_8"
	}
	add("bhakoot", "Bhakoot", bhakoot, 7, matchingSigns[b.sign], matchingSigns[g.sign], bhakootDetail)
	out.Factors[len(out.Factors)-1].Evidence = map[string]int{"bride_to_groom": distance, "groom_to_bride": (b.sign-g.sign+12)%12 + 1}
	if bhakoot == 0 {
		out.Cautions = append(out.Cautions, "bhakoot_adverse")
	}
	nadi, nadiDetail := 8.0, "nadi_different"
	if b.nadi == g.nadi {
		nadi, nadiDetail = 0, "nadi_same"
		out.Cautions = append(out.Cautions, "nadi_same")
	}
	add("nadi", "Nadi", nadi, 8, matchingNadis[b.nadi], matchingNadis[g.nadi], nadiDetail)
	return out, nil
}

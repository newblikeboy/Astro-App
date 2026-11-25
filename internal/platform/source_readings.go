package platform

import (
	"astroapp/backend/internal/gurujitext"
	"astroapp/backend/internal/swisseph"
	"strconv"
)

// sourceReadings selects complete original Hindi passages using recorded chart factors.
// Conditional house passages retain both branches; no strength score chooses a prediction.
func sourceReadings(r *Report) []gurujitext.Reference {
	out := []gurujitext.Reference{}
	if r == nil || r.Birth.Unknown || r.Chart == nil {
		return out
	}
	signs := []string{"मेष", "वृषभ", "मिथुन", "कर्क", "सिंह", "कन्या", "तुला", "वृश्चिक", "धनु", "मकर", "कुंभ", "मीन"}
	if n := r.Chart.Lagna.RashiNumber; n >= 1 && n <= 12 {
		key := signs[n-1]
		if n == 2 {
			key = "वृष"
		}
		out = append(out, gurujitext.Lookup("लग्न", key)...)
	}
	stars := []string{"अश्विनी", "भरणी", "कृत्तिका", "रोहिणी", "मृगशिरा", "आर्द्रा", "पुनर्वसु", "पुष्य", "आश्लेषा", "मघा", "पूर्वाफाल्गुनी", "उत्तराफाल्गुनी", "हस्त", "चित्रा", "स्वाती", "विशाखा", "अनुराधा", "ज्येष्ठा", "मूल", "पूर्वाषाढ़ा", "उत्तराषाढ़ा", "श्रवण", "धनिष्ठा", "शतभिषा", "पूर्वाभाद्रपद", "उत्तराभाद्रपद", "रेवती"}
	planets := map[string]string{"Sun": "सूर्य", "Moon": "चंद्रमा", "Mars": "मंगल", "Mercury": "बुध", "Jupiter": "गुरु", "Venus": "शुक्र", "Saturn": "शनि", "Rahu": "राहु", "Ketu": "केतु"}
	seenHouses := map[int]bool{}
	for _, p := range r.Chart.Planets {
		if p.Name == "Moon" {
			if p.RashiNumber >= 1 && p.RashiNumber <= 12 {
				out = append(out, gurujitext.Lookup("राशिफल", signs[p.RashiNumber-1])...)
			}
			if p.NakshatraNumber >= 1 && p.NakshatraNumber <= 27 {
				out = append(out, gurujitext.Lookup("नक्षत्र", stars[p.NakshatraNumber-1])...)
			}
		}
		if name := planets[p.Name]; name != "" && p.House >= 1 && p.House <= 12 {
			for _, entry := range gurujitext.Lookup("भावफल", name) {
				if entry.House == p.House || entry.Kind == "introduction" {
					out = append(out, entry)
				}
			}
			out = append(out, gurujitext.Lookup("उपाय", name)...)
			if !seenHouses[p.House] {
				out = append(out, gurujitext.Lookup("ग्रहों का फला देश", strconv.Itoa(p.House))...)
				seenHouses[p.House] = true
			}
		}
	}
	if r.Details != nil {
		days := []string{"रविवार", "सोमवार", "मंगलवार", "बुधवार", "गुरुवार", "शुक्रवार", "शनिवार"}
		if day := r.Details.Birth.HinduWeekday; day >= 0 && day <= 6 {
			out = append(out, gurujitext.Lookup("वारफल", days[day])...)
		}
		if len(r.Details.Birth.Limbs) > 0 {
			n := r.Details.Birth.Limbs[0].Number
			tithis := []string{"प्रतिपदा", "द्वितीया", "तृतीया", "चतुर्थी", "पंचमी", "षष्ठी", "सप्तमी", "अष्टमी", "नवमी", "दशमी", "एकादशी", "द्वादशी", "त्रयोदशी", "चतुर्दशी"}
			key := ""
			if n == 15 {
				key = "पूर्णिमा"
			} else if n == 30 {
				key = "अमावस्या"
			} else if n >= 1 && n <= 29 {
				key = tithis[(n-1)%15]
			}
			if key != "" {
				out = append(out, gurujitext.Lookup("तिथिफल", key)...)
			}
		}
	}
	return out
}

// sourceOnlyReport builds a presentation copy without changing stored reports or purchased snapshots.
// Authored legacy narratives and arbitrary rule prose are excluded from current customer output.
func sourceOnlyReport(r *Report) *Report {
	if r == nil {
		return nil
	}
	out := *r
	out.GurujiFaladesh = nil
	out.Rules = []Rule{}
	out.SourceReadings = sourceReadings(r)
	if r.Chart != nil && !r.Birth.Unknown {
		out.Checks = placementChecks(*r.Chart)
	}
	if r.Numerology != nil {
		n := *r.Numerology
		n.SourceReadings = append(gurujitext.Lookup("मूलांक", strconv.Itoa(n.Mulank.Value)), gurujitext.Lookup("भाग्यांक", strconv.Itoa(n.Bhagyank.Value))...)
		n.Extended = nil
		out.Numerology = &n
	}
	if r.Details != nil {
		details := *r.Details
		details.RajyogReading = nil
		if r.Chart != nil {
			if mangal, err := swisseph.CalculateMangal(*r.Chart); err == nil {
				details.Mangal, details.MangalMethod = mangal, swisseph.MangalMethod
			}
		}
		out.Details = &details
	}
	return &out
}

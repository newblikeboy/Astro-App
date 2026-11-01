package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
)

// yogaQualificationNote reads only retained participant qualifications; old
// purchases receive no newly calculated caveat or changed yoga judgement.
func (l pdfLocale) yogaQualificationNote(details *KundliExpansion, code string) string {
	if details == nil || details.Qualifications == nil {
		return ""
	}
	for _, y := range details.Qualifications.Yogas {
		if y.Code == code && y.BasePresent && y.Status == "qualified" {
			return l.gurujiWording(GurujiWording{"यह योग मौजूद है, लेकिन उससे जुड़े ग्रहों की कमजोरी या दूसरी रुकावटों से उसका पूरा फायदा मिलने में मेहनत लग सकती है।", "This yoga is present, but weaknesses or other obstacles affecting its participants may limit how fully its potential is realised."})
		}
	}
	return ""
}

// advancedStrengthSection prints retained strength results in the existing
// appendix, keeping detailed calculation evidence outside the reader chapters.
func (l pdfLocale) advancedStrengthSection(b *swisseph.Shadbala, q *swisseph.YogaQualifications) pdfSection {
	w := l.gurujiWording
	section := pdfSection{ID: "advanced-strength", Title: w(GurujiWording{"ग्रहबल और नीचभंग का आकलन", "Planetary strength and debility cancellation"})}
	summary := pdfBlock{Title: w(GurujiWording{"ग्रह की क्षमता", "Planetary capacity"}), Headers: []string{w(GurujiWording{"ग्रह", "Planet"}), w(GurujiWording{"कुल बल", "Total virupas"}), w(GurujiWording{"अपेक्षित", "Required"}), w(GurujiWording{"मानक पूरा", "Meets requirement"})}}
	parts := pdfBlock{Title: w(GurujiWording{"षड्बल के छह भाग · विरूप", "Six strength groups · virupas"}), Headers: []string{w(GurujiWording{"ग्रह", "Planet"}), w(GurujiWording{"स्थान", "Position"}), w(GurujiWording{"दिशा", "Direction"}), w(GurujiWording{"समय", "Time"}), w(GurujiWording{"गति", "Motion"}), w(GurujiWording{"नैसर्गिक", "Natural"}), w(GurujiWording{"दृष्टि", "Aspect"})}}
	for _, p := range b.Planets {
		summary.Rows = append(summary.Rows, []string{l.text(p.Planet), fmt.Sprintf("%.2f", p.Total), fmt.Sprintf("%.0f", p.Required), l.yes(p.MeetsRequirement)})
		parts.Rows = append(parts.Rows, []string{l.text(p.Planet), fmt.Sprintf("%.2f", p.Sthana), fmt.Sprintf("%.2f", p.Dig), fmt.Sprintf("%.2f", p.Kala), fmt.Sprintf("%.2f", p.Cheshta), fmt.Sprintf("%.2f", p.Naisargika), fmt.Sprintf("%.2f", p.Drik)})
	}
	section.Blocks = append(section.Blocks, summary, parts)
	if q != nil {
		for _, n := range q.Neechabhanga {
			if n.Debilitated {
				section.Blocks = append(section.Blocks, pdfBlock{Title: l.text(n.Planet), Note: w(GurujiWording{"नीच राशि; जाँचे गए नीचभंग नियम से कमजोरी का शमन: ", "Debilitated sign; mitigation under the checked cancellation rules: "}) + l.yes(n.Cancelled)})
			}
		}
	}
	return section
}

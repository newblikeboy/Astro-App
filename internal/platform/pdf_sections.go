package platform

import (
	"astroapp/backend/internal/swisseph"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"sort"
	"time"
)

//go:embed data/pdf-hindi.json
var pdfHindiJSON []byte

var pdfHindi = func() map[string]string {
	var labels map[string]string
	if err := json.Unmarshal(pdfHindiJSON, &labels); err != nil {
		panic("invalid bundled PDF labels")
	}
	return labels
}()

type pdfBlock struct {
	Title, Note string
	Headers     []string
	Rows        [][]string
	Chart       template.HTML
}

type pdfSection struct {
	ID, Title, Intro string
	Blocks           []pdfBlock
}

// pdfText translates calculation labels only; customer names and approved remedy prose bypass this helper.
func pdfText(value string) string {
	if label, ok := map[string]string{"none": "कोई नहीं", "low": "कम", "high": "अधिक", "Lagna": "लग्न", "lagna": "लग्न", "moon": "चंद्र", "venus": "शुक्र", "all": "सभी संदर्भ", "draft": "प्रारूप", "approved": "स्वीकृत", "full_planetary_strength": "पूर्ण ग्रह बल", "all_benefic_malefic_distribution": "सभी शुभ और अशुभ ग्रहों का वितरण", "unspecified_high_guna_cancellation": "अनिर्दिष्ट उच्च गुण आधारित अपवाद", "age_based_exceptions": "आयु आधारित अपवाद", "benefic_strength_Jupiter": "गुरु के बल की समीक्षा", "benefic_strength_Venus": "शुक्र के बल की समीक्षा", "benefic_strength_Mercury": "बुध के बल की समीक्षा", "benefic_strength_Moon": "चंद्र के बल की समीक्षा"}[value]; ok {
		return label
	}
	if label, ok := pdfHindi[value]; ok {
		return label
	}
	return matchingText("hi", value)
}

// pdfSign names an already classified sign and never rounds or recalculates its boundary.
func pdfSign(sign int) string {
	if sign < 1 || sign > 12 {
		return "—"
	}
	return []string{"मेष", "वृषभ", "मिथुन", "कर्क", "सिंह", "कन्या", "तुला", "वृश्चिक", "धनु", "मकर", "कुंभ", "मीन"}[sign-1]
}

// pdfYes labels recorded boolean evidence, without implying an outcome prediction.
func pdfYes(value bool) string {
	if value {
		return "हाँ"
	}
	return "नहीं"
}

// pdfNames localizes a list of calculation labels while preserving its order.
func pdfNames(values []string) string {
	out := ""
	for _, value := range values {
		if out != "" {
			out += " · "
		}
		out += pdfText(value)
	}
	if out == "" {
		return "—"
	}
	return out
}

// pdfDate formats a stored instant in the birthplace zone using Hindi month names and minute precision.
func pdfDate(value time.Time, zone *time.Location) string {
	if value.IsZero() {
		return "—"
	}
	at := value.In(zone)
	month := []string{"जनवरी", "फ़रवरी", "मार्च", "अप्रैल", "मई", "जून", "जुलाई", "अगस्त", "सितंबर", "अक्टूबर", "नवंबर", "दिसंबर"}[int(at.Month())-1]
	return fmt.Sprintf("%02d %s %d · %s", at.Day(), month, at.Year(), at.Format("15:04 MST"))
}

// pdfVargaBlock presents divisional signs and houses without inventing astronomical longitudes.
func pdfVargaBlock(title, method string, lagna int, planets []swisseph.DivisionalPlacement) pdfBlock {
	var signs [12]int
	for i := range signs {
		signs[i] = (lagna+i-1)%12 + 1
	}
	b := pdfBlock{Title: title, Note: "लग्न: " + pdfSign(lagna) + " · " + method, Headers: []string{"ग्रह", "राशि", "भाव", "वर्गोत्तम D9"}, Chart: pdfPlacementChart(signs, planets)}
	for _, p := range planets {
		vargottama := "—"
		// Vargottama is defined here only for the recorded D9 comparison.
		if title == "नवांश · D9" {
			vargottama = pdfYes(p.Vargottama)
		}
		b.Rows = append(b.Rows, []string{pdfText(p.Name), pdfSign(p.RashiNumber), fmt.Sprint(p.House), vargottama})
	}
	return b
}

// pdfPeriodBlocks retains every snapshotted parent and Antardasha in finite printable tables.
// Five-level current periods are printed separately; arbitrary interactive branches are not a static document.
func pdfPeriodBlocks(periods []swisseph.DashaPeriod, zone *time.Location, yogini []swisseph.YoginiLord) []pdfBlock {
	blocks := []pdfBlock{}
	for _, p := range periods {
		name := pdfText(p.Lord)
		for _, y := range yogini {
			if y.Lord == p.Lord {
				name = pdfText(y.Name) + " · " + name
			}
		}
		b := pdfBlock{Title: name, Note: pdfDate(p.Start, zone) + " से " + pdfDate(p.End, zone), Headers: []string{"अंतर्दशा", "आरंभ", "समाप्ति"}}
		for _, child := range p.Antardashas {
			b.Rows = append(b.Rows, []string{pdfText(child.Lord), pdfDate(child.Start, zone), pdfDate(child.End, zone)})
		}
		blocks = append(blocks, b)
	}
	return blocks
}

// pdfSections builds complete printable views of available snapshot sections without database reads or native recalculation.
// Older purchases remain unchanged: unavailable expanded fields get an explicit availability note.
func pdfSections(r *Report, zone *time.Location) []pdfSection {
	sections := []pdfSection{}
	d := r.Details
	if d == nil {
		sections = append(sections, pdfSection{ID: "birth-details", Title: "जन्म के विस्तृत विवरण", Intro: "इस पुराने सुरक्षित संस्करण में विस्तृत जन्म पंचांग, भाव चलित और वर्ग गणनाएँ उपलब्ध नहीं हैं। नए विवरण के लिए वेबसाइट पर नई कुंडली बनाएँ।"})
	} else {
		b := d.Birth
		weekdays := []string{"रविवार", "सोमवार", "मंगलवार", "बुधवार", "गुरुवार", "शुक्रवार", "शनिवार"}
		facts := pdfBlock{Title: "जन्म पंचांग और अवकहड़ा चक्र", Headers: []string{"विवरण", "स्थिति"}, Rows: [][]string{
			{"जन्म क्षण", pdfDate(b.At, zone)}, {"नागरिक वार", weekdays[b.CivilWeekday%7]}, {"हिंदू वार", weekdays[b.HinduWeekday%7]}, {"पक्ष", pdfText(b.Paksha)},
			{"लग्न / लग्नेश", pdfSign(r.Chart.Lagna.RashiNumber) + " / " + pdfText(b.Chakra.LagnaLord)}, {"चंद्र राशीश", pdfText(b.Chakra.RashiLord)}, {"नक्षत्र स्वामी", pdfText(b.Chakra.NakshatraLord)},
			{"वर्ण", pdfText(b.Chakra.Varna)}, {"वश्य", pdfText(b.Chakra.Vashya)}, {"योनि", pdfText(b.Chakra.Yoni)}, {"गण", pdfText(b.Chakra.Gana)}, {"नाड़ी", pdfText(b.Chakra.Nadi)},
			{"जन्म तारीख का सूर्योदय", pdfDate(b.Sunrise.Round(time.Minute), zone)}, {"जन्म तारीख का सूर्यास्त", pdfDate(b.Sunset.Round(time.Minute), zone)}, {"जन्म का हिंदू दिन", pdfDate(b.DayStart.Round(time.Minute), zone) + " से " + pdfDate(b.DayEnd.Round(time.Minute), zone)},
		}}
		limbs := pdfBlock{Title: "जन्म क्षण के पंचांग अंग", Note: "सीमाएँ निकटतम मिनट तक हैं। हिंदू वार सूर्योदय पर बदलता है; यह जन्म क्षण का पंचांग है।", Headers: []string{"अंग", "नाम", "आरंभ", "समाप्ति"}}
		for i, p := range b.Limbs {
			limbs.Rows = append(limbs.Rows, []string{[]string{"तिथि", "नक्षत्र", "योग", "करण"}[min(i, 3)], pdfText(p.Name), pdfDate(p.Start.Round(time.Minute), zone), pdfDate(p.End.Round(time.Minute), zone)})
		}
		sections = append(sections, pdfSection{ID: "birth-details", Title: "जन्म के विस्तृत विवरण", Blocks: []pdfBlock{facts, limbs}})

		chalit := pdfBlock{Title: "भाव चलित", Note: d.Chalit.Method, Headers: []string{"ग्रह", "राशि", "जन्म भाव", "चलित भाव"}}
		var signs [12]int
		placements := []swisseph.DivisionalPlacement{}
		for _, h := range d.Chalit.Houses {
			if h.House >= 1 && h.House <= 12 {
				signs[h.House-1] = int(math.Floor(h.Middle/30)) + 1
			}
		}
		for _, p := range d.Chalit.Planets {
			chalit.Rows = append(chalit.Rows, []string{pdfText(p.Name), pdfSign(p.RashiNumber), fmt.Sprint(p.NatalHouse), fmt.Sprint(p.House)})
			placements = append(placements, swisseph.DivisionalPlacement{Name: p.Name, House: p.House})
		}
		chalit.Chart = pdfPlacementChart(signs, placements)
		boundaries := pdfBlock{Title: "भाव की सीमाएँ", Headers: []string{"भाव", "आरंभ °", "मध्य °", "समाप्ति °"}}
		for _, h := range d.Chalit.Houses {
			boundaries.Rows = append(boundaries.Rows, []string{fmt.Sprint(h.House), fmt.Sprintf("%.6f", h.Start), fmt.Sprintf("%.6f", h.Middle), fmt.Sprintf("%.6f", h.End)})
		}
		charts := pdfSection{ID: "charts", Title: "चंद्र, भाव चलित और वर्ग कुंडलियाँ", Intro: "वर्ग कुंडलियाँ जन्म राशि के विभाजन हैं। उच्च वर्गों के लिए जन्म समय की शुद्धता विशेष रूप से आवश्यक है।", Blocks: []pdfBlock{chalit, boundaries}}
		for _, p := range r.Chart.Planets {
			if p.Name == "Moon" {
				moon := []swisseph.DivisionalPlacement{}
				for _, g := range r.Chart.Planets {
					moon = append(moon, swisseph.DivisionalPlacement{Name: g.Name, RashiNumber: g.RashiNumber, House: (g.RashiNumber-p.RashiNumber+12)%12 + 1})
				}
				charts.Blocks = append(charts.Blocks, pdfVargaBlock("चंद्र कुंडली", "चंद्र राशि से राशि आधारित भाव", p.RashiNumber, moon))
			}
		}
		for _, v := range d.Vargas {
			if v.Factor == 1 {
				continue
			}
			title := fmt.Sprintf("%s · D%d", pdfText(v.Name), v.Factor)
			if v.Factor == 9 {
				title = "नवांश · D9"
			}
			charts.Blocks = append(charts.Blocks, pdfVargaBlock(title, v.Method, v.LagnaSign, v.Planets))
		}
		sections = append(sections, charts)

		conditions := pdfBlock{Title: "ग्रह अवस्था और अस्त स्थिति", Headers: []string{"ग्रह / राशीश", "राशि स्थिति", "बालादि", "सूर्य से अंतर", "अस्त / सीमा"}}
		for _, p := range d.Analysis.Conditions {
			combustion := "लागू नहीं"
			if p.Combust != nil && p.CombustLimit != nil {
				combustion = pdfYes(*p.Combust) + fmt.Sprintf(" / %.2f°", *p.CombustLimit)
			}
			conditions.Rows = append(conditions.Rows, []string{pdfText(p.Name) + " / " + pdfText(p.SignLord), pdfText(p.Dignity), pdfText(p.Baladi), fmt.Sprintf("%.3f°", p.SunSeparation), combustion})
		}
		aspects := pdfBlock{Title: "ग्रह दृष्टि", Headers: []string{"ग्रह", "दृष्टि दूरी", "राशि / भाव", "दृष्ट राशि के ग्रह"}}
		for _, p := range d.Analysis.Aspects {
			aspects.Rows = append(aspects.Rows, []string{pdfText(p.Planet), fmt.Sprint(p.Distance), pdfSign(p.Sign) + " / " + fmt.Sprint(p.House), pdfNames(p.Occupants)})
		}
		friends := pdfBlock{Title: "नैसर्गिक, तात्कालिक और पंचधा मैत्री", Note: "संबंध दिशात्मक हैं; विपरीत दिशा का परिणाम अलग हो सकता है।", Headers: []string{"ग्रह", "दूसरा ग्रह", "नैसर्गिक", "तात्कालिक", "पंचधा"}}
		for _, p := range d.Analysis.Friendships {
			friends.Rows = append(friends.Rows, []string{pdfText(p.From), pdfText(p.To), pdfText(p.Natural), pdfText(p.Temporary), pdfText(p.Compound)})
		}
		analysis := pdfSection{ID: "analysis", Title: "ग्रह विश्लेषण और वर्ग बल", Intro: pdfText(d.Analysis.Method), Blocks: []pdfBlock{conditions, aspects, friends}}
		for _, s := range d.Analysis.VargaStrength {
			block := pdfBlock{Title: fmt.Sprintf("%s · विंशोपक बल %.3f / %.0f", pdfText(s.Planet), s.Total, s.Maximum), Note: "यह वर्ग स्वामित्व और मैत्री आधारित विंशोपक बल है; पूर्ण षड्बल नहीं।", Headers: []string{"वर्ग", "राशि / स्वामी", "संबंध", "भार", "अंक", "योगदान"}}
			for _, p := range s.Parts {
				block.Rows = append(block.Rows, []string{fmt.Sprintf("D%d", p.Factor), pdfSign(p.Sign) + " / " + pdfText(p.Owner), pdfText(p.Relationship), fmt.Sprintf("%.2f", p.Weight), fmt.Sprintf("%.2f", p.Points), fmt.Sprintf("%.3f", p.Contribution)})
			}
			analysis.Blocks = append(analysis.Blocks, block)
		}
		sections = append(sections, analysis)

		ashtaka := pdfSection{ID: "ashtakavarga", Title: "अष्टकवर्ग", Intro: "अशोधित भिन्न अष्टकवर्ग, सर्वाष्टकवर्ग और प्रस्तार। राशि क्रम: 1 मेष, 2 वृषभ, 3 मिथुन, 4 कर्क, 5 सिंह, 6 कन्या, 7 तुला, 8 वृश्चिक, 9 धनु, 10 मकर, 11 कुंभ, 12 मीन।"}
		headers := []string{"ग्रह"}
		for i := 1; i <= 12; i++ {
			headers = append(headers, fmt.Sprint(i))
		}
		headers = append(headers, "योग")
		summary := pdfBlock{Title: "BAV / SAV सारणी", Headers: headers}
		for _, p := range d.Ashtakavarga.Rows {
			row := []string{pdfText(p.Planet)}
			for _, n := range p.Points {
				row = append(row, fmt.Sprint(n))
			}
			row = append(row, fmt.Sprint(p.Total))
			summary.Rows = append(summary.Rows, row)
		}
		sarva := []string{"सर्व"}
		for _, n := range d.Ashtakavarga.Sarva {
			sarva = append(sarva, fmt.Sprint(n))
		}
		sarva = append(sarva, fmt.Sprint(d.Ashtakavarga.Total))
		summary.Rows = append(summary.Rows, sarva)
		ashtaka.Blocks = append(ashtaka.Blocks, summary)
		for _, p := range d.Ashtakavarga.Rows {
			block := pdfBlock{Title: pdfText(p.Planet) + " · प्रस्तार", Headers: headers[:13]}
			for i, contribution := range p.Contributors {
				row := []string{pdfText(d.Ashtakavarga.Sources[i])}
				for _, n := range contribution {
					row = append(row, fmt.Sprint(n))
				}
				block.Rows = append(block.Rows, row)
			}
			ashtaka.Blocks = append(ashtaka.Blocks, block)
		}
		sections = append(sections, ashtaka)

		yogas := pdfSection{ID: "yogas", Title: "योग और दोष का आकलन", Intro: "ये घोषित गणना पद्धति के स्थिति संकेत हैं। व्यक्तिगत फलादेश और उपाय आरंभ में अलग दिए गए हैं।"}
		patterns := pdfBlock{Title: "योग स्थितियाँ", Headers: []string{"योग", "स्थिति मिली", "ग्रह", "गणना का आधार"}}
		for _, p := range d.Analysis.Patterns {
			patterns.Rows = append(patterns.Rows, []string{pdfText(p.Name), pdfYes(p.Matches), pdfNames(p.Planets), pdfText(p.Condition)})
		}
		yogas.Blocks = append(yogas.Blocks, patterns)
		m := d.Mangal
		mangal := pdfBlock{Title: "मंगल आकलन", Note: fmt.Sprintf("प्रारंभिक स्तर: %s · अपवाद के बाद: %s · सक्रिय संदर्भ: %d / %d · सूर्य से अंतर %.3f°", pdfText(m.BaseLevel), pdfText(m.Level), m.ActiveCount, m.BaseCount, m.SunSeparation), Headers: []string{"संदर्भ", "मंगल का भाव", "संकेत", "अपवाद से निरस्त", "अपवाद"}}
		for _, p := range m.References {
			exceptions := []string{}
			for _, code := range p.Exceptions {
				exceptions = append(exceptions, pdfMangalRule(code))
			}
			mangal.Rows = append(mangal.Rows, []string{pdfText(p.Reference), fmt.Sprint(p.House), pdfYes(p.Flagged), pdfYes(p.Cancelled), pdfNames(exceptions)})
		}
		yogas.Blocks = append(yogas.Blocks, mangal)
		rules := pdfBlock{Title: "मंगल नियम और समीक्षा", Headers: []string{"नियम", "मिला", "संदर्भ / स्रोत पृष्ठ"}}
		for _, p := range m.Rules {
			rules.Rows = append(rules.Rows, []string{pdfMangalRule(p.Code), pdfYes(p.Met), pdfText(p.Reference) + fmt.Sprintf(" / %d", p.SourcePage)})
		}
		for _, p := range m.ReviewConditions {
			rules.Rows = append(rules.Rows, []string{pdfMangalRule(p), "समीक्षा", "—"})
		}
		for _, p := range m.NotEvaluated {
			rules.Rows = append(rules.Rows, []string{pdfText(p), "गणना नहीं हुई", "—"})
		}
		yogas.Blocks = append(yogas.Blocks, rules)
		sections = append(sections, yogas)
	}
	if r.Navamsa != nil && d == nil {
		v := r.Navamsa
		sections = append(sections, pdfSection{ID: "charts", Title: "नवांश कुंडली", Blocks: []pdfBlock{pdfVargaBlock("नवांश · D9", "राशि आधारित भाव", v.LagnaSign, v.Planets)}})
	}
	if len(r.Checks) > 0 {
		block := pdfBlock{Title: "मूल स्थिति संकेत", Headers: []string{"संकेत", "मिला", "संदर्भ से भाव"}}
		for _, p := range r.Checks {
			name := map[string]string{"mangal-lagna": "लग्न से मंगल", "mangal-moon": "चंद्र से मंगल", "gajakesari-base": "गजकेसरी का मूल संकेत", "budha-aditya-base": "बुध-आदित्य का मूल संकेत"}[p.Code]
			if name == "" {
				name = p.Code
			}
			house := "—"
			if p.House > 0 {
				house = fmt.Sprint(p.House)
			}
			block.Rows = append(block.Rows, []string{name, pdfYes(p.Matches), house})
		}
		if d != nil {
			for i := range sections {
				if sections[i].ID == "yogas" {
					sections[i].Blocks = append(sections[i].Blocks, block)
				}
			}
		} else {
			sections = append(sections, pdfSection{ID: "checks", Title: "मूल ग्रह स्थिति जाँच", Blocks: []pdfBlock{block}})
		}
	}
	if r.Dasha != nil {
		v := r.Dasha
		section := pdfSection{ID: "dasha", Title: "विंशोत्तरी दशा", Intro: fmt.Sprintf("संदर्भ समय: %s · जन्म दशा: %s · शेष %.3f वर्ष · वर्ष %.2f दिन। वर्तमान महादशा / अंतर्दशा: %s / %s। पहली महादशा जन्म से पहले शुरू हो सकती है। अंतर्दशाएँ पूरे मूल काल से गिनी जाती हैं।", pdfDate(v.AsOf, zone), pdfText(v.BirthLord), v.BalanceYears, v.YearDays, pdfText(v.CurrentLord), pdfText(v.CurrentSubLord))}
		current := pdfBlock{Title: "संदर्भ समय पर सक्रिय पाँच स्तर", Headers: []string{"स्तर", "स्वामी", "आरंभ", "समाप्ति"}}
		for _, p := range v.CurrentPath {
			level := "—"
			if p.Level >= 1 && p.Level <= 5 {
				level = []string{"महादशा", "अंतर्दशा", "प्रत्यंतर", "सूक्ष्म", "प्राण"}[p.Level-1]
			}
			current.Rows = append(current.Rows, []string{level, pdfText(p.Lord), pdfDate(p.Start, zone), pdfDate(p.End, zone)})
		}
		if len(current.Rows) > 0 {
			section.Blocks = append(section.Blocks, current)
		}
		section.Blocks = append(section.Blocks, pdfPeriodBlocks(v.Periods, zone, nil)...)
		sections = append(sections, section)
	}
	if d != nil {
		v := d.Yogini
		section := pdfSection{ID: "yogini", Title: "योगिनी दशा", Intro: fmt.Sprintf("आठ योगिनियाँ, 36 वर्ष का चक्र। संदर्भ: %s · जन्म स्वामी: %s · जन्म शेष %.3f वर्ष · वर्तमान: %s / %s।", pdfDate(v.AsOf, zone), pdfText(v.BirthLord), v.BalanceYears, pdfText(v.CurrentLord), pdfText(v.CurrentSubLord))}
		section.Blocks = pdfPeriodBlocks(v.Periods, zone, v.Lords)
		sections = append(sections, section)
	}
	if r.Transits != nil {
		v := r.Transits
		block := pdfBlock{Title: "रिपोर्ट के समय ग्रह गोचर", Headers: []string{"ग्रह", "राशि / अंश", "नक्षत्र / पद", "लग्न से भाव", "चंद्र से भाव", "वक्री"}}
		for _, p := range v.Planets {
			block.Rows = append(block.Rows, []string{pdfText(p.Name), pdfSign(p.RashiNumber) + " / " + p.DMS, pdfText(p.Nakshatra) + fmt.Sprintf(" / %d", p.Pada), fmt.Sprint(p.House), fmt.Sprint(p.FromMoon), pdfYes(p.Retrograde)})
		}
		status := map[string]string{"none": "साढ़ेसाती सक्रिय नहीं", "rising": "साढ़ेसाती का पहला चरण", "peak": "साढ़ेसाती का मध्य चरण", "setting": "साढ़ेसाती का अंतिम चरण"}[v.SadeSati]
		for _, p := range v.Planets {
			if p.Name == "Saturn" && (p.FromMoon == 4 || p.FromMoon == 8) {
				status += " · ढैय्या का स्थिति संकेत"
			}
		}
		sections = append(sections, pdfSection{ID: "transits", Title: "गोचर और शनि स्थिति", Intro: pdfDate(v.AsOf, zone) + " · " + status + "। यह सुरक्षित समय का गोचर है। तारीख बदलकर साढ़ेसाती/ढैय्या की पूरी समयरेखा और अन्य दशा शाखाएँ वेबसाइट पर देखें।", Blocks: []pdfBlock{block}})
	}
	reference := pdfBlock{Title: "गणना सेटिंग्स", Headers: []string{"संदर्भ", "मान"}}
	// Templates sort map keys; this list also makes the appendix stable for review.
	for _, key := range pdfSortedKeys(r.Chart.Settings) {
		reference.Rows = append(reference.Rows, []string{pdfText(key), pdfText(r.Chart.Settings[key])})
	}
	if d != nil {
		reference.Rows = append(reference.Rows, []string{"लाहिड़ी अयनांश", fmt.Sprintf("%.6f°", d.Birth.Ayanamsa)}, []string{"जूलियन दिन UT1", fmt.Sprintf("%.8f", d.Birth.JulianDayUT1)}, []string{"विस्तार पद्धति", d.Method}, []string{"मंगल पद्धति", d.MangalMethod}, []string{"अष्टकवर्ग पद्धति", d.Ashtakavarga.Method}, []string{"योगिनी पद्धति", d.Yogini.Method})
		for _, key := range pdfSortedKeys(d.Birth.Settings) {
			reference.Rows = append(reference.Rows, []string{pdfText(key), pdfText(d.Birth.Settings[key])})
		}
	}
	sections = append(sections, pdfSection{ID: "reference", Title: "गणना संदर्भ और सीमाएँ", Intro: "Swiss Ephemeris " + r.Chart.SwissVersion + " · " + r.Engine, Blocks: []pdfBlock{reference}})
	return sections
}

// pdfMangalRule shares the matching report's approved factual descriptions of implemented Mangal rules.
func pdfMangalRule(code string) string {
	if label, ok := matchingMangalRuleText[code]; ok {
		return pdfText(label)
	}
	return pdfText(code)
}

// pdfSortedKeys keeps reference tables stable across renders of the same purchased snapshot.
func pdfSortedKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

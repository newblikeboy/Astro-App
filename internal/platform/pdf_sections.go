package platform

import (
	"astroapp/backend/internal/swisseph"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"sort"
	"strings"
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
	Title, Note  string
	KeepTogether bool
	Headers      []string
	Rows         [][]string
	Chart        template.HTML
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

// date formats a stored instant with localized month names and minute precision in the birthplace zone.
func (l pdfLocale) date(value time.Time, zone *time.Location) string {
	if value.IsZero() {
		return "—"
	}
	at := value.In(zone)
	month := []string{l.text("जनवरी"), l.text("फ़रवरी"), l.text("मार्च"), l.text("अप्रैल"), l.text("मई"), l.text("जून"), l.text("जुलाई"), l.text("अगस्त"), l.text("सितंबर"), l.text("अक्टूबर"), l.text("नवंबर"), l.text("दिसंबर")}[int(at.Month())-1]
	return fmt.Sprintf("%02d %s %d · %s", at.Day(), month, at.Year(), at.Format("15:04 MST"))
}

// vargaBlock presents localized divisional signs and houses without inventing astronomical longitudes.
func (l pdfLocale) vargaBlock(title, method string, lagna int, planets []swisseph.DivisionalPlacement) pdfBlock {
	var signs [12]int
	for i := range signs {
		signs[i] = (lagna+i-1)%12 + 1
	}
	b := pdfBlock{Title: title, Note: l.text("लग्न: ") + l.sign(lagna) + " · " + method, Headers: []string{l.text("ग्रह"), l.text("राशि"), l.text("भाव"), l.text("वर्गोत्तम D9")}, Chart: l.placementChart(signs, planets)}
	for _, p := range planets {
		vargottama := "—"
		// Vargottama is defined here only for the recorded D9 comparison.
		if title == l.text("नवांश · D9") {
			vargottama = l.yes(p.Vargottama)
		}
		b.Rows = append(b.Rows, []string{l.text(p.Name), l.sign(p.RashiNumber), fmt.Sprint(p.House), vargottama})
	}
	return b
}

// periodBlocks retains every snapshotted parent and Antardasha in localized printable tables.
// Five-level current periods are printed separately; arbitrary interactive branches are not a static document.
func (l pdfLocale) periodBlocks(periods []swisseph.DashaPeriod, zone *time.Location, yogini []swisseph.YoginiLord) []pdfBlock {
	blocks := []pdfBlock{}
	for _, p := range periods {
		name := l.text(p.Lord)
		for _, y := range yogini {
			if y.Lord == p.Lord {
				name = l.text(y.Name) + " · " + name
			}
		}
		b := pdfBlock{Title: name, Note: l.date(p.Start, zone) + l.text(" से ") + l.date(p.End, zone), Headers: []string{l.text("अंतर्दशा"), l.text("आरंभ"), l.text("समाप्ति")}}
		for _, child := range p.Antardashas {
			b.Rows = append(b.Rows, []string{l.text(child.Lord), l.date(child.Start, zone), l.date(child.End, zone)})
		}
		blocks = append(blocks, b)
	}
	return blocks
}

// sections builds localized printable views of snapshot sections without database reads or native recalculation.
// Older purchases remain unchanged: unavailable expanded fields get an explicit availability note.
func (l pdfLocale) sections(r *Report, zone *time.Location) []pdfSection {
	sections := []pdfSection{l.numerologySection(r.Numerology)}
	d := r.Details
	if d == nil {
		sections = append(sections, pdfSection{ID: "birth-details", Title: l.text("जन्म के विस्तृत विवरण"), Intro: l.text("इस पुराने सुरक्षित संस्करण में विस्तृत जन्म पंचांग, भाव चलित और वर्ग गणनाएँ उपलब्ध नहीं हैं। नए विवरण के लिए वेबसाइट पर नई कुंडली बनाएँ।")})
	} else {
		b := d.Birth
		weekdays := []string{l.text("रविवार"), l.text("सोमवार"), l.text("मंगलवार"), l.text("बुधवार"), l.text("गुरुवार"), l.text("शुक्रवार"), l.text("शनिवार")}
		facts := pdfBlock{Title: l.text("जन्म पंचांग और अवकहड़ा चक्र"), Headers: []string{l.text("विवरण"), l.text("स्थिति")}, Rows: [][]string{
			{l.text("जन्म क्षण"), l.date(b.At, zone)}, {l.text("नागरिक वार"), weekdays[b.CivilWeekday%7]}, {l.text("हिंदू वार"), weekdays[b.HinduWeekday%7]}, {l.text("पक्ष"), l.text(b.Paksha)},
			{l.text("लग्न / लग्नेश"), l.sign(r.Chart.Lagna.RashiNumber) + " / " + l.text(b.Chakra.LagnaLord)}, {l.text("चंद्र राशीश"), l.text(b.Chakra.RashiLord)}, {l.text("नक्षत्र स्वामी"), l.text(b.Chakra.NakshatraLord)},
			{l.text("वर्ण"), l.text(b.Chakra.Varna)}, {l.text("वश्य"), l.text(b.Chakra.Vashya)}, {l.text("योनि"), l.text(b.Chakra.Yoni)}, {l.text("गण"), l.text(b.Chakra.Gana)}, {l.text("नाड़ी"), l.text(b.Chakra.Nadi)},
			{l.text("जन्म तारीख का सूर्योदय"), l.date(b.Sunrise.Round(time.Minute), zone)}, {l.text("जन्म तारीख का सूर्यास्त"), l.date(b.Sunset.Round(time.Minute), zone)}, {l.text("जन्म का हिंदू दिन"), l.date(b.DayStart.Round(time.Minute), zone) + l.text(" से ") + l.date(b.DayEnd.Round(time.Minute), zone)},
		}}
		limbs := pdfBlock{Title: l.text("जन्म क्षण के पंचांग अंग"), Note: l.text("सीमाएँ निकटतम मिनट तक हैं। हिंदू वार सूर्योदय पर बदलता है; यह जन्म क्षण का पंचांग है।"), Headers: []string{l.text("अंग"), l.text("नाम"), l.text("आरंभ"), l.text("समाप्ति")}}
		for i, p := range b.Limbs {
			limbs.Rows = append(limbs.Rows, []string{[]string{l.text("तिथि"), l.text("नक्षत्र"), l.text("योग"), l.text("करण")}[min(i, 3)], l.text(p.Name), l.date(p.Start.Round(time.Minute), zone), l.date(p.End.Round(time.Minute), zone)})
		}
		sections = append(sections, pdfSection{ID: "birth-details", Title: l.text("जन्म के विस्तृत विवरण"), Blocks: []pdfBlock{facts, limbs}})

		chalit := pdfBlock{Title: l.text("भाव चलित"), Note: d.Chalit.Method, Headers: []string{l.text("ग्रह"), l.text("राशि"), l.text("जन्म भाव"), l.text("चलित भाव")}}
		var signs [12]int
		placements := []swisseph.DivisionalPlacement{}
		for _, h := range d.Chalit.Houses {
			if h.House >= 1 && h.House <= 12 {
				signs[h.House-1] = int(math.Floor(h.Middle/30)) + 1
			}
		}
		for _, p := range d.Chalit.Planets {
			chalit.Rows = append(chalit.Rows, []string{l.text(p.Name), l.sign(p.RashiNumber), fmt.Sprint(p.NatalHouse), fmt.Sprint(p.House)})
			placements = append(placements, swisseph.DivisionalPlacement{Name: p.Name, House: p.House})
		}
		chalit.Chart = l.placementChart(signs, placements)
		boundaries := pdfBlock{Title: l.text("भाव की सीमाएँ"), Headers: []string{l.text("भाव"), l.text("आरंभ °"), l.text("मध्य °"), l.text("समाप्ति °")}}
		for _, h := range d.Chalit.Houses {
			boundaries.Rows = append(boundaries.Rows, []string{fmt.Sprint(h.House), fmt.Sprintf("%.6f", h.Start), fmt.Sprintf("%.6f", h.Middle), fmt.Sprintf("%.6f", h.End)})
		}
		charts := pdfSection{ID: "charts", Title: l.text("चंद्र, भाव चलित और वर्ग कुंडलियाँ"), Intro: l.text("वर्ग कुंडलियाँ जन्म राशि के विभाजन हैं। उच्च वर्गों के लिए जन्म समय की शुद्धता विशेष रूप से आवश्यक है।"), Blocks: []pdfBlock{chalit, boundaries}}
		for _, p := range r.Chart.Planets {
			if p.Name == "Moon" {
				moon := []swisseph.DivisionalPlacement{}
				for _, g := range r.Chart.Planets {
					moon = append(moon, swisseph.DivisionalPlacement{Name: g.Name, RashiNumber: g.RashiNumber, House: (g.RashiNumber-p.RashiNumber+12)%12 + 1})
				}
				charts.Blocks = append(charts.Blocks, l.vargaBlock(l.text("चंद्र कुंडली"), l.text("चंद्र राशि से राशि आधारित भाव"), p.RashiNumber, moon))
			}
		}
		for _, v := range d.Vargas {
			if v.Factor == 1 {
				continue
			}
			title := fmt.Sprintf("%s · D%d", l.text(v.Name), v.Factor)
			if v.Factor == 9 {
				title = l.text("नवांश · D9")
			}
			charts.Blocks = append(charts.Blocks, l.vargaBlock(title, v.Method, v.LagnaSign, v.Planets))
		}
		sections = append(sections, charts)

		conditions := pdfBlock{Title: l.text("ग्रह अवस्था और अस्त स्थिति"), Headers: []string{l.text("ग्रह / राशीश"), l.text("राशि स्थिति"), l.text("बालादि"), l.text("सूर्य से अंतर"), l.text("अस्त / सीमा")}}
		for _, p := range d.Analysis.Conditions {
			combustion := l.text("लागू नहीं")
			if p.Combust != nil && p.CombustLimit != nil {
				combustion = l.yes(*p.Combust) + fmt.Sprintf(" / %.2f°", *p.CombustLimit)
			}
			conditions.Rows = append(conditions.Rows, []string{l.text(p.Name) + " / " + l.text(p.SignLord), l.text(p.Dignity), l.text(p.Baladi), fmt.Sprintf("%.3f°", p.SunSeparation), combustion})
		}
		aspects := pdfBlock{Title: l.text("ग्रह दृष्टि"), Headers: []string{l.text("ग्रह"), l.text("दृष्टि दूरी"), l.text("राशि / भाव"), l.text("दृष्ट राशि के ग्रह")}}
		for _, p := range d.Analysis.Aspects {
			aspects.Rows = append(aspects.Rows, []string{l.text(p.Planet), fmt.Sprint(p.Distance), l.sign(p.Sign) + " / " + fmt.Sprint(p.House), l.names(p.Occupants)})
		}
		friends := pdfBlock{Title: l.text("नैसर्गिक, तात्कालिक और पंचधा मैत्री"), Note: l.text("संबंध दिशात्मक हैं; विपरीत दिशा का परिणाम अलग हो सकता है।"), Headers: []string{l.text("ग्रह"), l.text("दूसरा ग्रह"), l.text("नैसर्गिक"), l.text("तात्कालिक"), l.text("पंचधा")}}
		for _, p := range d.Analysis.Friendships {
			friends.Rows = append(friends.Rows, []string{l.text(p.From), l.text(p.To), l.text(p.Natural), l.text(p.Temporary), l.text(p.Compound)})
		}
		analysis := pdfSection{ID: "analysis", Title: l.text("ग्रह विश्लेषण और वर्ग बल"), Intro: l.text(d.Analysis.Method), Blocks: []pdfBlock{conditions, aspects, friends}}
		for _, s := range d.Analysis.VargaStrength {
			block := pdfBlock{Title: fmt.Sprintf(l.text("%s · विंशोपक बल %.3f / %.0f"), l.text(s.Planet), s.Total, s.Maximum), Note: l.text("यह वर्ग स्वामित्व और मैत्री आधारित विंशोपक बल है; पूर्ण षड्बल नहीं।"), Headers: []string{l.text("वर्ग"), l.text("राशि / स्वामी"), l.text("संबंध"), l.text("भार"), l.text("अंक"), l.text("योगदान")}}
			for _, p := range s.Parts {
				block.Rows = append(block.Rows, []string{fmt.Sprintf("D%d", p.Factor), l.sign(p.Sign) + " / " + l.text(p.Owner), l.text(p.Relationship), fmt.Sprintf("%.2f", p.Weight), fmt.Sprintf("%.2f", p.Points), fmt.Sprintf("%.3f", p.Contribution)})
			}
			analysis.Blocks = append(analysis.Blocks, block)
		}
		sections = append(sections, analysis)

		ashtaka := pdfSection{ID: "ashtakavarga", Title: l.text("अष्टकवर्ग"), Intro: l.text("अशोधित भिन्न अष्टकवर्ग, सर्वाष्टकवर्ग और प्रस्तार। राशि क्रम: 1 मेष, 2 वृषभ, 3 मिथुन, 4 कर्क, 5 सिंह, 6 कन्या, 7 तुला, 8 वृश्चिक, 9 धनु, 10 मकर, 11 कुंभ, 12 मीन।")}
		headers := []string{l.text("ग्रह")}
		for i := 1; i <= 12; i++ {
			headers = append(headers, fmt.Sprint(i))
		}
		headers = append(headers, l.text("योग"))
		summary := pdfBlock{Title: l.text("BAV / SAV सारणी"), Headers: headers}
		for _, p := range d.Ashtakavarga.Rows {
			row := []string{l.text(p.Planet)}
			for _, n := range p.Points {
				row = append(row, fmt.Sprint(n))
			}
			row = append(row, fmt.Sprint(p.Total))
			summary.Rows = append(summary.Rows, row)
		}
		sarva := []string{l.text("सर्व")}
		for _, n := range d.Ashtakavarga.Sarva {
			sarva = append(sarva, fmt.Sprint(n))
		}
		sarva = append(sarva, fmt.Sprint(d.Ashtakavarga.Total))
		summary.Rows = append(summary.Rows, sarva)
		ashtaka.Blocks = append(ashtaka.Blocks, summary)
		for _, p := range d.Ashtakavarga.Rows {
			block := pdfBlock{Title: l.text(p.Planet) + l.text(" · प्रस्तार"), Headers: headers[:13]}
			for i, contribution := range p.Contributors {
				row := []string{l.text(d.Ashtakavarga.Sources[i])}
				for _, n := range contribution {
					row = append(row, fmt.Sprint(n))
				}
				block.Rows = append(block.Rows, row)
			}
			ashtaka.Blocks = append(ashtaka.Blocks, block)
		}
		sections = append(sections, ashtaka)

		yogas := pdfSection{ID: "yogas", Title: l.text("योग और दोष का आकलन"), Intro: l.text("yoga.intro") + " " + l.text("yoga.scope")}
		yogas.Blocks = append(yogas.Blocks, l.natalYogaBlocks(d.YogaChecks)...)
		patterns := []pdfBlock{}
		for _, p := range d.Analysis.Patterns {
			patterns = append(patterns, pdfBlock{Title: l.text("yoga.name." + p.Code), Note: l.text("Conclusion") + ": " + l.placementConclusion(p.Code, p.Matches), KeepTogether: true, Headers: []string{l.text("ग्रह"), l.text("गणना का आधार")}, Rows: [][]string{{l.names(p.Planets), l.text(p.Condition)}}})
		}
		m := d.Mangal
		mangal := pdfBlock{Title: l.text("मंगल आकलन"), Note: l.text("Conclusion") + ": " + l.mangalConclusion(m) + "\n\n" + l.text("Calculation details") + ": " + fmt.Sprintf(l.text("प्रारंभिक स्तर: %s · अपवाद के बाद: %s · सक्रिय संदर्भ: %d / %d · सूर्य से अंतर %.3f°"), l.text(m.BaseLevel), l.text(m.Level), m.ActiveCount, m.BaseCount, m.SunSeparation), Headers: []string{l.text("संदर्भ"), l.text("मंगल का भाव"), l.text("संकेत"), l.text("अपवाद से निरस्त"), l.text("अपवाद")}}
		for _, p := range m.References {
			exceptions := []string{}
			for _, code := range p.Exceptions {
				exceptions = append(exceptions, l.mangalRule(code))
			}
			mangal.Rows = append(mangal.Rows, []string{l.text(p.Reference), fmt.Sprint(p.House), l.yes(p.Flagged), l.yes(p.Cancelled), l.names(exceptions)})
		}
		yogas.Blocks = append(yogas.Blocks, mangal)
		rules := pdfBlock{Title: l.text("मंगल नियम और समीक्षा"), Headers: []string{l.text("नियम"), l.text("मिला"), l.text("संदर्भ / स्रोत पृष्ठ")}}
		for _, p := range m.Rules {
			rules.Rows = append(rules.Rows, []string{l.mangalRule(p.Code), l.yes(p.Met), l.text(p.Reference) + fmt.Sprintf(" / %d", p.SourcePage)})
		}
		for _, p := range m.ReviewConditions {
			rules.Rows = append(rules.Rows, []string{l.mangalRule(p), l.text("समीक्षा"), "—"})
		}
		for _, p := range m.NotEvaluated {
			rules.Rows = append(rules.Rows, []string{l.text(p), l.text("गणना नहीं हुई"), "—"})
		}
		yogas.Blocks = append(yogas.Blocks, rules)
		yogas.Blocks = append(yogas.Blocks, patterns...)
		sections = append(sections, yogas)
	}
	if r.Navamsa != nil && d == nil {
		v := r.Navamsa
		sections = append(sections, pdfSection{ID: "charts", Title: l.text("नवांश कुंडली"), Blocks: []pdfBlock{l.vargaBlock(l.text("नवांश · D9"), l.text("राशि आधारित भाव"), v.LagnaSign, v.Planets)}})
	}
	if len(r.Checks) > 0 {
		block := pdfBlock{Title: l.text("मूल स्थिति संकेत"), Headers: []string{l.text("संकेत"), l.text("Conclusion"), l.text("संदर्भ से भाव")}}
		for _, p := range r.Checks {
			if d != nil && strings.HasPrefix(p.Code, "mangal-") {
				continue // The full assessment above already explains cancellation and remaining indications.
			}
			name := map[string]string{"mangal-lagna": l.text("लग्न से मंगल"), "mangal-moon": l.text("चंद्र से मंगल"), "gajakesari-base": l.text("गजकेसरी का मूल संकेत"), "budha-aditya-base": l.text("बुध-आदित्य का मूल संकेत")}[p.Code]
			if name == "" {
				name = p.Code
			}
			house := "—"
			if p.House > 0 {
				house = fmt.Sprint(p.House)
			}
			block.Rows = append(block.Rows, []string{name, l.placementConclusion(p.Code, p.Matches), house})
		}
		if d != nil {
			for i := range sections {
				if sections[i].ID == "yogas" {
					sections[i].Blocks = append(sections[i].Blocks, block)
				}
			}
		} else {
			sections = append(sections, pdfSection{ID: "checks", Title: l.text("मूल ग्रह स्थिति जाँच"), Blocks: []pdfBlock{block}})
		}
	}
	if r.Dasha != nil {
		v := r.Dasha
		section := pdfSection{ID: "dasha", Title: l.text("विंशोत्तरी दशा"), Intro: fmt.Sprintf(l.text("संदर्भ समय: %s · जन्म दशा: %s · शेष %.3f वर्ष · वर्ष %.2f दिन। वर्तमान महादशा / अंतर्दशा: %s / %s। पहली महादशा जन्म से पहले शुरू हो सकती है। अंतर्दशाएँ पूरे मूल काल से गिनी जाती हैं।"), l.date(v.AsOf, zone), l.text(v.BirthLord), v.BalanceYears, v.YearDays, l.text(v.CurrentLord), l.text(v.CurrentSubLord))}
		current := pdfBlock{Title: l.text("संदर्भ समय पर सक्रिय पाँच स्तर"), Headers: []string{l.text("स्तर"), l.text("स्वामी"), l.text("आरंभ"), l.text("समाप्ति")}}
		for _, p := range v.CurrentPath {
			level := "—"
			if p.Level >= 1 && p.Level <= 5 {
				level = []string{l.text("महादशा"), l.text("अंतर्दशा"), l.text("प्रत्यंतर"), l.text("सूक्ष्म"), l.text("प्राण")}[p.Level-1]
			}
			current.Rows = append(current.Rows, []string{level, l.text(p.Lord), l.date(p.Start, zone), l.date(p.End, zone)})
		}
		if len(current.Rows) > 0 {
			section.Blocks = append(section.Blocks, current)
		}
		section.Blocks = append(section.Blocks, l.periodBlocks(v.Periods, zone, nil)...)
		sections = append(sections, section)
	}
	if d != nil {
		v := d.Yogini
		section := pdfSection{ID: "yogini", Title: l.text("योगिनी दशा"), Intro: fmt.Sprintf(l.text("आठ योगिनियाँ, 36 वर्ष का चक्र। संदर्भ: %s · जन्म स्वामी: %s · जन्म शेष %.3f वर्ष · वर्तमान: %s / %s।"), l.date(v.AsOf, zone), l.text(v.BirthLord), v.BalanceYears, l.text(v.CurrentLord), l.text(v.CurrentSubLord))}
		section.Blocks = l.periodBlocks(v.Periods, zone, v.Lords)
		sections = append(sections, section)
	}
	if r.Transits != nil {
		v := r.Transits
		block := pdfBlock{Title: l.text("रिपोर्ट के समय ग्रह गोचर"), Headers: []string{l.text("ग्रह"), l.text("राशि / अंश"), l.text("नक्षत्र / पद"), l.text("लग्न से भाव"), l.text("चंद्र से भाव"), l.text("वक्री")}}
		for _, p := range v.Planets {
			block.Rows = append(block.Rows, []string{l.text(p.Name), l.sign(p.RashiNumber) + " / " + p.DMS, l.text(p.Nakshatra) + fmt.Sprintf(" / %d", p.Pada), fmt.Sprint(p.House), fmt.Sprint(p.FromMoon), l.yes(p.Retrograde)})
		}
		status := map[string]string{"none": l.text("साढ़ेसाती सक्रिय नहीं"), "rising": l.text("साढ़ेसाती का पहला चरण"), "peak": l.text("साढ़ेसाती का मध्य चरण"), "setting": l.text("साढ़ेसाती का अंतिम चरण")}[v.SadeSati]
		for _, p := range v.Planets {
			if p.Name == "Saturn" && (p.FromMoon == 4 || p.FromMoon == 8) {
				status += l.text(" · ढैय्या का स्थिति संकेत")
			}
		}
		sections = append(sections, pdfSection{ID: "transits", Title: l.text("गोचर और शनि स्थिति"), Intro: l.date(v.AsOf, zone) + " · " + status + l.text("। यह सुरक्षित समय का गोचर है। तारीख बदलकर साढ़ेसाती/ढैय्या की पूरी समयरेखा और अन्य दशा शाखाएँ वेबसाइट पर देखें।"), Blocks: []pdfBlock{block}})
	}
	reference := pdfBlock{Title: l.text("गणना सेटिंग्स"), Headers: []string{l.text("संदर्भ"), l.text("मान")}}
	// Templates sort map keys; this list also makes the appendix stable for review.
	for _, key := range pdfSortedKeys(r.Chart.Settings) {
		reference.Rows = append(reference.Rows, []string{l.text(key), l.text(r.Chart.Settings[key])})
	}
	if d != nil {
		reference.Rows = append(reference.Rows, []string{l.text("लाहिड़ी अयनांश"), fmt.Sprintf("%.6f°", d.Birth.Ayanamsa)}, []string{l.text("जूलियन दिन UT1"), fmt.Sprintf("%.8f", d.Birth.JulianDayUT1)}, []string{l.text("विस्तार पद्धति"), d.Method}, []string{l.text("मंगल पद्धति"), d.MangalMethod}, []string{l.text("अष्टकवर्ग पद्धति"), d.Ashtakavarga.Method}, []string{l.text("योगिनी पद्धति"), d.Yogini.Method})
		for _, key := range pdfSortedKeys(d.Birth.Settings) {
			reference.Rows = append(reference.Rows, []string{l.text(key), l.text(d.Birth.Settings[key])})
		}
	}
	sections = append(sections, pdfSection{ID: "reference", Title: l.text("गणना संदर्भ और सीमाएँ"), Intro: "Swiss Ephemeris " + r.Chart.SwissVersion + " · " + r.Engine, Blocks: []pdfBlock{reference}})
	return sections
}

// mangalRule localizes the shared factual descriptions of implemented Mangal rules.
func (l pdfLocale) mangalRule(code string) string {
	if label, ok := matchingMangalRuleText[code]; ok {
		return l.text(label)
	}
	return l.text(code)
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

// pdfDate formats an original Hindi PDF timestamp in the stored birthplace timezone.
func pdfDate(value time.Time, zone *time.Location) string { return (pdfLocale{"hi"}).date(value, zone) }

// pdfSections preserves the original Hindi section builder for background jobs and older callers.
func pdfSections(r *Report, zone *time.Location) []pdfSection {
	return (pdfLocale{"hi"}).sections(r, zone)
}

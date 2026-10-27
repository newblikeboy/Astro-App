package platform

import (
	"fmt"
	"strings"
	"time"
)

type pdfFact struct{ Label, Value, Meaning string }
type pdfReadingCard struct{ Title, Conclusion, Meaning string }
type pdfTopic struct {
	ID, Title, Intro string
	Rules            []Rule
}
type pdfBookAppendix struct {
	pdfSection
	Number string
}
type pdfBook struct {
	Portrait      []pdfFact
	Topics        []pdfTopic
	Yogas         []pdfReadingCard
	AbsentYogas   []string
	YogaReference string
	Numerology    pdfSection
	Periods       pdfSection
	Appendices    []pdfBookAppendix
}

// book organizes immutable calculation evidence into readable chapters and a complete technical appendix.
// It performs no astronomy, database lookup or replacement of approved personal prose.
func (l pdfLocale) book(r *Report, zone *time.Location, sections []pdfSection) pdfBook {
	book := pdfBook{Numerology: l.numerologySection(r.Numerology), Topics: l.bookTopics(r.Rules), YogaReference: "sources", Periods: pdfSection{ID: "time-reading", Title: l.text("book.periods"), Intro: l.text("book.periods-intro")}}
	if r.Chart != nil {
		book.Portrait = append(book.Portrait, pdfFact{l.text("Ascendant"), l.sign(r.Chart.Lagna.RashiNumber), l.text("book.lagna-meaning")})
		for _, p := range r.Chart.Planets {
			if p.Name == "Moon" {
				book.Portrait = append(book.Portrait, pdfFact{l.text("Moon sign"), l.sign(p.RashiNumber), l.text("book.moon-meaning")}, pdfFact{l.text("Birth nakshatra"), l.text(p.Nakshatra) + " · " + l.text("Pada") + " " + fmt.Sprint(p.Pada), l.text("book.star-meaning")})
			}
		}
		for _, p := range r.Chart.Planets {
			if p.Name == "Sun" {
				book.Portrait = append(book.Portrait, pdfFact{l.text("Sun sign"), l.sign(p.RashiNumber), l.text("book.sun-meaning")})
			}
		}
	}
	if d := r.Details; d != nil {
		if checks := d.YogaChecks; checks != nil {
			kaalStatus, kaalMeaning := l.kaalSarpReading(checks.KaalSarp)
			book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Kaal Sarp Dosha"), kaalStatus, kaalMeaning + "\n\n" + l.text("kaal-sarp.definition")})
			status, meaning := "rajyog.absent", "rajyog.absent-meaning"
			if checks.Rajyog.Matches {
				status, meaning = "rajyog.present", "rajyog.meaning"
			}
			book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Rajyog · Kendra–Trikona relationships"), l.text(status), l.text(meaning)})
			for _, lord := range checks.Rajyog.Yogakarakas {
				book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Yogakaraka lordship"), l.conclusion("yogakaraka.present", l.text(lord.Planet)), l.text("yogakaraka.meaning")})
			}
		} else {
			book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Kaal Sarp Dosha & Rajyog"), l.text("Recreate this Kundli from its birth details to include Kaal Sarp Dosha and Rajyog checks. This saved report has not been assessed for them."), ""})
		}
		mangalStatus, mangalMeaning := l.mangalReading(d.Mangal)
		book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Mangal analysis"), mangalStatus, mangalMeaning})
		for _, p := range d.Analysis.Patterns {
			if !p.Matches {
				book.AbsentYogas = append(book.AbsentYogas, l.text("yoga.name."+p.Code))
				continue
			}
			status, meaning := l.placementReading(p.Code, true)
			book.Yogas = append(book.Yogas, pdfReadingCard{l.text("yoga.name." + p.Code), status, meaning})
		}
	} else {
		book.Yogas = append(book.Yogas, pdfReadingCard{l.text("Kaal Sarp Dosha & Rajyog"), l.text("Recreate this Kundli from its birth details to include Kaal Sarp Dosha and Rajyog checks. This saved report has not been assessed for them."), ""})
	}
	for _, p := range r.Checks {
		if r.Details != nil && strings.HasPrefix(p.Code, "mangal-") {
			continue
		}
		if !p.Matches {
			book.AbsentYogas = append(book.AbsentYogas, l.text("yoga.name."+p.Code)+" · "+l.text("book.base-check"))
			continue
		}
		status, meaning := l.placementReading(p.Code, true)
		book.Yogas = append(book.Yogas, pdfReadingCard{l.text("yoga.name." + p.Code), status, meaning})
	}
	if r.Dasha == nil {
		book.Periods.Intro = l.text("book.periods-unavailable")
	} else {
		d := r.Dasha
		summary := l.text("book.periods-summary-unavailable")
		if d.CurrentLord != "" && d.CurrentSubLord != "" {
			summary = l.conclusion("book.periods-conclusion", l.text(d.CurrentLord), l.text(d.CurrentSubLord)) + "\n\n" + l.text("book.periods-meaning")
		}
		book.Periods.Blocks = append(book.Periods.Blocks, pdfBlock{Title: l.text("Conclusion"), Note: summary, KeepTogether: true})
		for _, section := range sections {
			if section.ID == "dasha" && len(section.Blocks) > 0 && len(d.CurrentPath) > 0 {
				book.Periods.Blocks = append(book.Periods.Blocks, section.Blocks[0])
			}
		}
		book.Periods.Blocks = append(book.Periods.Blocks, pdfBlock{Title: l.text("गणना का संदर्भ:"), Note: l.date(d.AsOf, zone) + "\n" + l.text("book.snapshot-note"), KeepTogether: true})
	}
	for _, section := range sections {
		if section.ID == "yogas" || section.ID == "checks" {
			book.YogaReference = section.ID
		}
		if section.ID == "numerology" {
			continue
		}
		book.Appendices = append(book.Appendices, pdfBookAppendix{section, fmt.Sprintf("A%02d", len(book.Appendices)+2)})
	}
	return book
}

// bookTopics groups copied rule values into life subjects while retaining every rule's wording and within-topic order.
func (l pdfLocale) bookTopics(rules []Rule) []pdfTopic {
	aliases := map[string]string{"marriage": "relationships", "health": "wellbeing", "debt": "finance", "": "general"}
	grouped := map[string][]Rule{}
	order := []string{"general", "personality", "education", "career", "finance", "relationships", "family", "wellbeing", "dasha", "spiritual"}
	for _, rule := range rules {
		category := rule.Category
		if alias, ok := aliases[category]; ok {
			category = alias
		}
		if _, ok := grouped[category]; !ok {
			known := false
			for _, value := range order {
				if value == category {
					known = true
					break
				}
			}
			if !known {
				order = append(order, category)
			}
		}
		grouped[category] = append(grouped[category], rule)
	}
	topics := []pdfTopic{}
	for _, category := range order {
		if len(grouped[category]) == 0 {
			continue
		}
		title, intro := l.text("book.topic."+category), l.text("book.context."+category)
		if title == "book.topic."+category {
			title, intro = l.text(pdfCategory(category)), l.text("book.context.general")
		}
		topics = append(topics, pdfTopic{ID: fmt.Sprintf("topic-%02d", len(topics)+1), Title: title, Intro: intro, Rules: grouped[category]})
	}
	return topics
}

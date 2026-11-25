package platform

import (
	"astroapp/backend/internal/gurujitext"
	"fmt"
	"time"
)

type pdfFact struct{ Label, Value, Meaning string }
type pdfReadingCard struct{ Title, Conclusion, Meaning string }
type pdfTopic struct {
	ID, Title, Intro string
	Category         string
	Rules            []Rule
	Paragraphs       []string
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

// sourcePassageLocator formats the catalogue's filename and exact inclusive page range.
func sourcePassageLocator(entry gurujitext.Reference) string {
	pages := fmt.Sprint(entry.Pages[0])
	if entry.Pages[1] != entry.Pages[0] {
		pages += "–" + fmt.Sprint(entry.Pages[1])
	}
	return entry.File + " · पृष्ठ " + pages
}

// book presents only PDF-origin prose and recorded calculations, including for historical purchase snapshots.
// The stored snapshot itself is never changed or used as an authored-prose fallback.
func (l pdfLocale) book(original *Report, zone *time.Location, sections []pdfSection) pdfBook {
	r := sourceOnlyReport(original)
	book := pdfBook{Numerology: l.numerologySection(r.Numerology), YogaReference: "sources", Periods: pdfSection{ID: "time-reading", Title: l.text("book.periods")}}
	groups := map[string]int{}
	for _, entry := range r.SourceReadings {
		index, exists := groups[entry.Category]
		if !exists {
			index = len(book.Topics)
			groups[entry.Category] = index
			book.Topics = append(book.Topics, pdfTopic{ID: fmt.Sprintf("source-topic-%d", index+1), Category: entry.Category, Title: entry.Category})
		}
		book.Topics[index].Paragraphs = append(book.Topics[index].Paragraphs, entry.Text+"\n\n"+sourcePassageLocator(entry))
	}
	if r.Chart != nil {
		book.Portrait = append(book.Portrait, pdfFact{Label: l.text("Ascendant"), Value: l.sign(r.Chart.Lagna.RashiNumber)})
		for _, p := range r.Chart.Planets {
			if p.Name == "Moon" {
				book.Portrait = append(book.Portrait, pdfFact{Label: l.text("Moon sign"), Value: l.sign(p.RashiNumber)}, pdfFact{Label: l.text("Birth nakshatra"), Value: l.text(p.Nakshatra) + " · " + l.text("Pada") + " " + fmt.Sprint(p.Pada)})
			}
			if p.Name == "Sun" {
				book.Portrait = append(book.Portrait, pdfFact{Label: l.text("Sun sign"), Value: l.sign(p.RashiNumber)})
			}
		}
	}
	if d := r.Details; d != nil {
		if checks := d.YogaChecks; checks != nil {
			book.Yogas = append(book.Yogas, pdfReadingCard{Title: l.text("Kaal Sarp Dosha"), Conclusion: l.kaalSarpConclusion(checks.KaalSarp)})
			state := "rajyog.absent"
			if checks.Rajyog.Matches {
				state = "rajyog.present"
			}
			book.Yogas = append(book.Yogas, pdfReadingCard{Title: l.text("Rajyog · Kendra–Trikona relationships"), Conclusion: l.text(state)})
		}
		book.Yogas = append(book.Yogas, pdfReadingCard{Title: l.text("Mangal analysis"), Conclusion: l.mangalConclusion(d.Mangal)})
		for _, p := range d.Analysis.Patterns {
			book.Yogas = append(book.Yogas, pdfReadingCard{Title: l.text("yoga.name." + p.Code), Conclusion: l.placementConclusion(p.Code, p.Matches)})
		}
	}
	if r.Dasha != nil {
		for _, section := range sections {
			if section.ID == "dasha" && len(section.Blocks) > 0 {
				book.Periods.Blocks = append(book.Periods.Blocks, section.Blocks[0])
			}
		}
		book.Periods.Blocks = append(book.Periods.Blocks, pdfBlock{Title: l.text("गणना का संदर्भ:"), Note: l.date(r.Dasha.AsOf, zone), KeepTogether: true})
	}
	for _, section := range sections {
		if section.ID == "numerology" {
			continue
		}
		if section.ID == "yogas" || section.ID == "checks" {
			book.YogaReference = section.ID
		}
		book.Appendices = append(book.Appendices, pdfBookAppendix{section, fmt.Sprintf("A%02d", len(book.Appendices)+2)})
	}
	return book
}

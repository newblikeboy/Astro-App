package platform

import (
	"astroapp/backend/internal/numerology"
	"fmt"
	"html"
	"html/template"
	"strconv"
	"strings"
)

// numerologyWorking formats only saved arithmetic steps, never inferring missing numbers from a purchased birth date.
func numerologyWorking(number numerology.Number) string {
	working := []string{}
	for i, total := range number.Steps {
		terms := []string{}
		if i == 0 {
			for _, digit := range number.Digits {
				terms = append(terms, strconv.Itoa(digit))
			}
		} else {
			for _, digit := range strconv.Itoa(number.Steps[i-1]) {
				terms = append(terms, string(digit))
			}
		}
		step := strconv.Itoa(total)
		if len(terms) > 1 {
			step = strings.Join(terms, " + ") + " = " + step
		}
		working = append(working, step)
	}
	return strings.Join(working, "\n")
}

// numerologySection prints the snapshotted date numbers and shared meanings in the requested PDF language.
func (l pdfLocale) numerologySection(reading *numerology.Reading) pdfSection {
	section := pdfSection{ID: "numerology", Title: l.text("Numerology"), Intro: l.text("numerology.intro")}
	if reading == nil {
		section.Intro = l.text("numerology.missing")
		return section
	}
	summary := l.conclusion("numerology.summary", fmt.Sprint(reading.Mulank.Value), fmt.Sprint(reading.Bhagyank.Value))
	if reading.Extended != nil && reading.Extended.Naamank != nil {
		summary += " " + strings.ReplaceAll(l.text("numerology.name-summary"), "{v1}", strconv.Itoa(reading.Extended.Naamank.Number.Value))
	}
	meaning := l.text("numerology.summary-meaning")
	if reading.Extended != nil {
		meaning = l.text("numerology.extended-intro")
	}
	section.Blocks = append(section.Blocks, pdfBlock{Title: l.text("Conclusion"), Note: summary + " " + meaning, KeepTogether: true})
	for _, row := range []struct {
		kind, title string
		number      numerology.Number
	}{{"mulank", "Mulank", reading.Mulank}, {"bhagyank", "Bhagyank", reading.Bhagyank}} {
		section.Blocks = append(section.Blocks, pdfBlock{Title: l.text(row.title) + " · " + fmt.Sprint(row.number.Value), Note: l.text("Number ruler") + ": " + l.text(row.number.Planet) + "\n" + l.text(fmt.Sprintf("numerology.%s.%d", row.kind, row.number.Value)) + "\n\n" + l.text("Calculation details") + ":\n" + numerologyWorking(row.number), KeepTogether: true})
	}
	if reading.Extended != nil {
		section.Blocks = append(section.Blocks, l.numerologyExtension(reading.Extended)...)
	} else {
		section.Blocks = append(section.Blocks, pdfBlock{Title: l.text("numerology.profile-title"), Note: l.text("numerology.extended-missing"), KeepTogether: true})
	}
	date := reading.BirthDate
	if parts := strings.Split(date, "-"); len(parts) == 3 {
		date = parts[2] + "-" + parts[1] + "-" + parts[0]
	}
	method := l.text("numerology.method")
	if reading.Extended != nil {
		method += "\n\n" + l.text("numerology.profile-method") + "\n" + reading.Extended.Method
	}
	section.Blocks = append(section.Blocks, pdfBlock{Title: l.text("Birth calculation reference"), Note: l.text("Birth date") + ": " + date + "\n" + method + "\n\n" + l.text("numerology.scope"), KeepTogether: true})
	return section
}

// numerologyWording selects only the requested language from snapshotted original profile prose.
func (l pdfLocale) numerologyWording(text numerology.Bilingual) string {
	if l.language == "en" {
		return text.English
	}
	return text.Hindi
}

// numerologyExtension presents stored name evidence, correspondences and topic readings without recalculating an old purchase.
func (l pdfLocale) numerologyExtension(ext *numerology.ExtendedReading) []pdfBlock {
	blocks := []pdfBlock{}
	if named := ext.Naamank; named != nil {
		terms := []string{}
		for i, letter := range named.Letters {
			terms = append(terms, letter+" = "+strconv.Itoa(named.Number.Digits[i]))
		}
		blocks = append(blocks, pdfBlock{Title: l.text("Naamank") + " · " + strconv.Itoa(named.Number.Value), Note: l.text("numerology.name-used") + ": " + named.Spelling + "\n" + l.text("Number ruler") + ": " + l.text(named.Number.Planet) + "\n\n" + l.numerologyWording(named.Meaning) + "\n\n" + l.text("Calculation details") + ":\n" + strings.Join(terms, " · ") + "\n" + numerologyWorking(named.Number), KeepTogether: true})
	} else {
		blocks = append(blocks, pdfBlock{Title: l.text("Naamank"), Note: l.text("numerology.name-missing"), KeepTogether: true})
	}
	facts := pdfBlock{Title: l.text("numerology.profile-title"), Note: strings.ReplaceAll(l.text("numerology.profile-basis"), "{v1}", strconv.Itoa(ext.Profile.Number)) + "\n\n" + l.text("numerology.correspondence-note"), Headers: []string{l.text("numerology.fact-label"), l.text("numerology.fact-value")}}
	for _, row := range []struct {
		key     string
		numbers []int
	}{{"numerology.lucky-numbers", ext.LuckyNumbers}, {"numerology.lucky-dates", ext.LuckyDates}} {
		values := []string{}
		for _, value := range row.numbers {
			values = append(values, strconv.Itoa(value))
		}
		facts.Rows = append(facts.Rows, []string{l.text(row.key), strings.Join(values, ", ")})
	}
	for _, fact := range ext.Profile.Facts {
		facts.Rows = append(facts.Rows, []string{l.numerologyWording(fact.Label), l.numerologyWording(fact.Value)})
	}
	blocks = append(blocks, facts)
	for _, part := range ext.Profile.Guidance {
		block := pdfBlock{Title: l.numerologyWording(part.Title), Note: l.numerologyWording(part.Text), KeepTogether: true}
		if part.ID == "yantra" {
			block.Chart = pdfNumerologyYantra(ext.Profile.Yantra, l.text("numerology.yantra-caption"))
		}
		blocks = append(blocks, block)
	}
	blocks = append(blocks, pdfBlock{Title: l.text("Calculation details"), Note: l.text("numerology.name-method"), KeepTogether: true})
	return blocks
}

// pdfNumerologyYantra draws the stored 3x3 devotional grid as a labelled vector image rather than a column-headed data table.
func pdfNumerologyYantra(grid [][]int, label string) template.HTML {
	if len(grid) != 3 {
		return ""
	}
	var out strings.Builder
	fmt.Fprintf(&out, `<svg class="numerology-yantra" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 210 210" role="img" aria-label="%s"><title>%s</title>`, html.EscapeString(label), html.EscapeString(label))
	out.WriteString(`<rect x="1" y="1" width="208" height="208" fill="#fffaf0" stroke="#b48b54"/><path d="M70 1V209M140 1V209M1 70H209M1 140H209" fill="none" stroke="#b48b54"/>`)
	for i, row := range grid {
		if len(row) != 3 {
			return ""
		}
		for j, value := range row {
			fmt.Fprintf(&out, `<text x="%d" y="%d" text-anchor="middle" fill="#6f2839" font-size="21">%d</text>`, j*70+35, i*70+44, value)
		}
	}
	out.WriteString(`</svg>`)
	return template.HTML(out.String()) // Only escaped labels, fixed SVG markup and integers enter this image.
}

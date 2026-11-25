package platform

import (
	"astroapp/backend/internal/gurujitext"
	"astroapp/backend/internal/numerology"
	"fmt"
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

// numerologySection prints recorded arithmetic and full Hindi PDF passages in either interface language.
// Missing source numbers remain absent; no old profile or generated interpretation is used.
func (l pdfLocale) numerologySection(reading *numerology.Reading) pdfSection {
	section := pdfSection{ID: "numerology", Title: l.text("Numerology")}
	if reading == nil {
		section.Intro = l.text("numerology.missing")
		return section
	}
	for _, row := range []struct {
		category string
		number   numerology.Number
	}{{"मूलांक", reading.Mulank}, {"भाग्यांक", reading.Bhagyank}} {
		section.Blocks = append(section.Blocks, pdfBlock{Title: row.category + " · " + fmt.Sprint(row.number.Value), Note: numerologyWorking(row.number), KeepTogether: true})
		entries := gurujitext.Lookup(row.category, strconv.Itoa(row.number.Value))
		for _, entry := range entries {
			section.Blocks = append(section.Blocks, pdfBlock{Title: entry.Title, Note: entry.Text + "\n\n" + sourcePassageLocator(entry)})
		}
		if len(entries) == 0 {
			section.Blocks = append(section.Blocks, pdfBlock{Note: "इस अंक का फल दी गई PDF में उपलब्ध नहीं है।"})
		}
	}
	return section
}

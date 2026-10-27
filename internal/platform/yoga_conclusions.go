package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"strings"
)

// conclusion formats the shared application-owned summary without modifying stored interpretations.
func (l pdfLocale) conclusion(key string, values ...string) string {
	text := l.text(key)
	for i, value := range values {
		text = strings.ReplaceAll(text, fmt.Sprintf("{v%d}", i+1), value)
	}
	return text
}

// placementConclusion describes a saved base configuration; absence never implies an adverse result.
func (l pdfLocale) placementConclusion(code string, matches bool) string {
	status, meaning := l.placementReading(code, matches)
	return status + " " + meaning
}

// placementReading separates the recorded decision from its meaning for readable chapter cards.
func (l pdfLocale) placementReading(code string, matches bool) (string, string) {
	status, meaning := "placement.absent", "placement.absent-meaning"
	if matches {
		status, meaning = "placement.present", "yoga.meaning."+code
	}
	if strings.HasSuffix(code, "-base") || strings.HasPrefix(code, "mangal-") {
		status = strings.Replace(status, "placement.", "placement.base-", 1)
	}
	return l.conclusion(status, l.text("yoga.name."+code)), l.text(meaning)
}

// mangalConclusion keeps absent placement, cancellation and remaining indications distinct.
func (l pdfLocale) mangalConclusion(m swisseph.MangalAssessment) string {
	status, meaning := l.mangalReading(m)
	return status + " " + meaning
}

// mangalReading separates the assessed state and explanation without changing cancellation rules.
func (l pdfLocale) mangalReading(m swisseph.MangalAssessment) (string, string) {
	status, meaning := "mangal.absent", "mangal.absent-meaning"
	if m.ActiveCount > 0 {
		status, meaning = "mangal.present", "mangal.meaning"
	} else if m.BaseCount > 0 {
		status, meaning = "mangal.cancelled", "mangal.cancelled-meaning"
	}
	return l.text(status), l.text(meaning)
}

// kaalSarpConclusion names only confirmed enclosures, using recorded whole-sign houses for older snapshots too.
func (l pdfLocale) kaalSarpConclusion(k swisseph.KaalSarpAssessment) string {
	status, meaning := l.kaalSarpReading(k)
	return status + " " + meaning
}

// kaalSarpReading separates the confirmed/absent/undecided conclusion from general traditional meaning.
func (l pdfLocale) kaalSarpReading(k swisseph.KaalSarpAssessment) (string, string) {
	switch k.Status {
	case "present":
		status := l.text("kaal-sarp.present-unnamed")
		theme := ""
		if k.RahuHouse >= 1 && k.RahuHouse <= 12 && k.KetuHouse == (k.RahuHouse+5)%12+1 {
			status = l.conclusion("kaal-sarp.present", l.text(fmt.Sprintf("kaal-sarp.type.%d", k.RahuHouse)))
			theme = " " + l.text(fmt.Sprintf("kaal-sarp.theme.%d", k.RahuHouse))
		}
		return status, l.text("kaal-sarp.meaning") + theme
	case "absent":
		return l.text("kaal-sarp.absent"), l.text("kaal-sarp.absent-meaning")
	default:
		return l.text("kaal-sarp.boundary"), l.text("kaal-sarp.boundary-meaning")
	}
}

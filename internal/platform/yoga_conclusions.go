package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"strings"
)

// conclusion formats shared calculation status labels without writing an outcome interpretation.
func (l pdfLocale) conclusion(key string, values ...string) string {
	text := l.text(key)
	for i, value := range values {
		text = strings.ReplaceAll(text, fmt.Sprintf("{v%d}", i+1), value)
	}
	return text
}

// placementConclusion reports only whether the recorded configuration is present.
func (l pdfLocale) placementConclusion(code string, matches bool) string {
	status, meaning := l.placementReading(code, matches)
	return status + " " + meaning
}

// placementReading returns the recorded configuration state without an authored outcome.
func (l pdfLocale) placementReading(code string, matches bool) (string, string) {
	status := "placement.absent"
	if matches {
		status = "placement.present"
	}
	if strings.HasSuffix(code, "-base") || strings.HasPrefix(code, "mangal-") {
		status = strings.Replace(status, "placement.", "placement.base-", 1)
	}
	return l.conclusion(status, l.text("yoga.name."+code)), ""
}

// mangalConclusion keeps absent placement, cancellation and remaining indications distinct.
func (l pdfLocale) mangalConclusion(m swisseph.MangalAssessment) string {
	status, meaning := l.mangalReading(m)
	return status + " " + meaning
}

// mangalReading returns the recorded assessment state without a forecast.
func (l pdfLocale) mangalReading(m swisseph.MangalAssessment) (string, string) {
	if m.Classification == "mangal" || m.Classification == "partial" || m.Classification == "none" || m.Classification == "cancelled" {
		return l.text("mangal.guruji." + m.Classification), ""
	}
	status := "mangal.absent"
	if m.ActiveCount > 0 {
		status = "mangal.present"
	} else if m.BaseCount > 0 {
		status = "mangal.cancelled"
	}
	return l.text(status), ""
}

// kaalSarpConclusion names only confirmed geometric enclosures from recorded houses.
func (l pdfLocale) kaalSarpConclusion(k swisseph.KaalSarpAssessment) string {
	status, meaning := l.kaalSarpReading(k)
	return status + " " + meaning
}

// kaalSarpReading returns only the confirmed, absent or boundary calculation state.
func (l pdfLocale) kaalSarpReading(k swisseph.KaalSarpAssessment) (string, string) {
	switch k.Status {
	case "present":
		status := l.text("kaal-sarp.present-unnamed")
		if k.RahuHouse >= 1 && k.RahuHouse <= 12 && k.KetuHouse == (k.RahuHouse+5)%12+1 {
			status = l.conclusion("kaal-sarp.present", l.text(fmt.Sprintf("kaal-sarp.type.%d", k.RahuHouse)))
		}
		return status, ""
	case "absent":
		return l.text("kaal-sarp.absent"), ""
	default:
		return l.text("kaal-sarp.boundary"), ""
	}
}

package platform

import (
	_ "embed"
	"encoding/json"
	"html/template"
	"sort"
	"strings"
	"time"
)

//go:embed data/pdf-english.json
var pdfEnglishJSON []byte

var pdfEnglish = func() map[string]string {
	labels := map[string]string{}
	// Reuse existing application-owned calculation vocabulary deterministically.
	keys := make([]string, 0, len(pdfHindi))
	for en := range pdfHindi {
		keys = append(keys, en)
	}
	sort.Strings(keys)
	for _, en := range keys {
		labels[pdfHindi[en]] = en
	}
	var authored map[string]string
	if err := json.Unmarshal(pdfEnglishJSON, &authored); err != nil {
		panic("invalid English PDF labels")
	}
	for hi, en := range authored {
		labels[hi] = en
	}
	return labels
}()

type pdfLocale struct{ language string }

// names translates a list of calculation labels without touching stored customer prose.
func (l pdfLocale) names(values []string) string {
	labels := make([]string, 0, len(values))
	for _, value := range values {
		labels = append(labels, l.text(value))
	}
	if len(labels) == 0 {
		return "—"
	}
	return strings.Join(labels, " · ")
}

// text translates only exact application-owned labels; stored names, prose and remedies bypass it.
func (l pdfLocale) text(value string) string {
	wording, found := kundliCopy[value]
	if !found {
		wording, found = kundliCopy[pdfEnglish[value]]
	}
	if found {
		if l.language == "en" {
			return wording.English
		}
		return wording.Hindi
	}
	if l.language == "en" {
		if translated, ok := pdfEnglish[value]; ok {
			return translated
		}
		return value
	}
	return pdfText(value)
}

// sign names a recorded zodiac sign in the requested PDF language without reclassifying it.
func (l pdfLocale) sign(value int) string {
	if l.language != "en" {
		return pdfSign(value)
	}
	if value < 1 || value > 12 {
		return "—"
	}
	return []string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"}[value-1]
}

// yes formats boolean evidence in the selected language without asserting an outcome.
func (l pdfLocale) yes(value bool) string {
	if l.language != "en" {
		return pdfYes(value)
	}
	if value {
		return "Yes"
	}
	return "No"
}

// templateFunctions binds translation and date helpers to this request, keeping simultaneous languages isolated.
func (l pdfLocale) templateFunctions(zone *time.Location) template.FuncMap {
	return template.FuncMap{
		"clock": pdfClockMinutes, "instant": pdfMinuteInstant, "money": pdfMoney, "join": strings.Join,
		"birthDate": pdfBirthDate,
		"date":      func(v time.Time) string { return l.date(v, zone) },
		"label":     l.text, "sign": l.sign, "t": l.text,
		"category": func(value string) string { return l.text(pdfCategory(value)) },
		"ruleText": func(rule Rule) string {
			if l.language == "en" && strings.TrimSpace(rule.English) != "" {
				return rule.English
			}
			return rule.Hindi
		},
		"hasEnglish": func(rule Rule) bool { return strings.TrimSpace(rule.English) != "" },
	}
}

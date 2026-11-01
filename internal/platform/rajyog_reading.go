package platform

import (
	"astroapp/backend/internal/swisseph"
	"sort"
	"strconv"
	"strings"
)

const rajyogReadingMethod = "m4y-pair-specific-rajyog-reader-v1"

type RajyogReading struct {
	Method     string          `json:"method"`
	Summary    GurujiWording   `json:"summary"`
	Paragraphs []GurujiWording `json:"paragraphs"`
}

// readingYogaQualifications keeps each matched Rajyog pair independent when
// linking capacity to a life topic; legacy snapshots retain their saved assessment.
func readingYogaQualifications(q *swisseph.YogaQualifications) []swisseph.YogaQualification {
	if q == nil {
		return nil
	}
	if q.Method != swisseph.YogaQualificationMethod {
		return q.Yogas
	}
	out := []swisseph.YogaQualification{}
	for _, y := range q.Yogas {
		if y.Code != "kendra-trikona-rajyog" {
			out = append(out, y)
		}
	}
	seen := map[string]bool{}
	for _, matched := range q.RajyogPairs {
		if len(matched.Pair.Links) == 0 {
			continue
		}
		names := []string{matched.Pair.KendraLord, matched.Pair.TrikonaLord}
		sort.Strings(names)
		key := strings.Join(names, "-")
		if seen[key] {
			continue
		}
		seen[key] = true
		y := matched.Qualification
		y.Code += "-" + key
		out = append(out, y)
	}
	return out
}

// rajyogWording freezes the same catalogue-based paragraph in both languages for web and PDF.
func rajyogWording(key string, values ...GurujiWording) GurujiWording {
	var hi, en []string
	for _, value := range values {
		hi = append(hi, value.Hindi)
		en = append(en, value.English)
	}
	return GurujiWording{(pdfLocale{"hi"}).conclusion(key, hi...), (pdfLocale{"en"}).conclusion(key, en...)}
}

// joinRajyogWording combines only the recorded relationship and its own qualifications.
func joinRajyogWording(parts []GurujiWording) GurujiWording {
	var hi, en []string
	for _, part := range parts {
		hi = append(hi, part.Hindi)
		en = append(en, part.English)
	}
	return GurujiWording{strings.Join(hi, " "), strings.Join(en, " ")}
}

// buildRajyogReading groups identical planet relationships without counting different
// lordships twice. Each relationship keeps its own strength; unrelated lords and
// separate Yogakaraka ownership never supply an obstacle or a success claim.
func buildRajyogReading(checks *swisseph.NatalYogaChecks, qualifications *swisseph.YogaQualifications) *RajyogReading {
	if checks == nil || qualifications == nil || qualifications.Method != swisseph.YogaQualificationMethod {
		return nil
	}
	out := &RajyogReading{Method: rajyogReadingMethod, Paragraphs: []GurujiWording{}}
	if !checks.Rajyog.Matches {
		out.Summary = rajyogWording("rajyog.absent")
		out.Paragraphs = append(out.Paragraphs, rajyogWording("rajyog.absent-meaning"))
		return out
	}
	type group struct {
		assessment swisseph.YogaQualification
		pairs      []swisseph.RajyogPair
	}
	groups := []group{}
	indices := map[string]int{}
	for _, matched := range qualifications.RajyogPairs {
		p := matched.Pair
		if len(p.Links) == 0 || !matched.Qualification.BasePresent {
			continue
		}
		names := []string{p.KendraLord, p.TrikonaLord}
		sort.Strings(names)
		key := strings.Join(names, ":")
		index, exists := indices[key]
		if !exists {
			index = len(groups)
			indices[key] = index
			groups = append(groups, group{assessment: matched.Qualification})
		}
		groups[index].pairs = append(groups[index].pairs, p)
	}
	if len(groups) == 0 {
		return nil
	}
	supported := 0
	for _, g := range groups {
		p := g.pairs[0]
		parts := []GurujiWording{}
		for _, link := range p.Links {
			parts = append(parts, rajyogWording("rajyog.link."+link, rajyogWording(p.KendraLord), rajyogWording(p.TrikonaLord)))
		}
		// A ninth/tenth relationship is described once even when the same lord
		// also owns the seventh. Theme selection never changes the yoga decision.
		theme := "personal"
		for _, pair := range g.pairs {
			if pair.KendraHouse == 10 {
				theme = "career"
				if pair.TrikonaHouse == 9 {
					theme = "career-opportunity"
					break
				}
			} else if theme == "personal" && pair.KendraHouse == 4 {
				theme = "stability"
			} else if theme == "personal" && pair.KendraHouse == 7 {
				theme = "cooperation"
			}
		}
		parts = append(parts, rajyogWording("rajyog.theme."+theme))
		if g.assessment.Status == "supported" {
			supported++
			parts = append(parts, rajyogWording("rajyog.strength.supported"))
		} else {
			for _, limitation := range g.assessment.Limitations {
				code, planet, ok := strings.Cut(limitation, ":")
				if ok {
					parts = append(parts, rajyogWording("rajyog.limit."+code, rajyogWording(planet)))
				}
			}
		}
		out.Paragraphs = append(out.Paragraphs, joinRajyogWording(parts))
	}
	state := "mixed"
	if supported == 0 {
		state = "qualified"
	} else if supported == len(groups) {
		state = "supported"
	}
	count := strconv.Itoa(len(groups))
	key := "rajyog.summary." + state
	if len(groups) == 1 {
		key += "-single"
	}
	out.Summary = rajyogWording(key, GurujiWording{count, count})
	return out
}

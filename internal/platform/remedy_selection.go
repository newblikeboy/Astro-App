package platform

import "strings"

// selectTopicRemedy ranks actual subject-linked obstacles, not all nine general
// remedy passages. Rank expresses review priority, never outcome probability.
// A strong Marak is never relabelled Karak or automatically strengthened.
func selectTopicRemedy(r *Report, topic *FaladeshTopic, names []string) *RemedyRecommendation {
	out := &RemedyRecommendation{Method: "m4y-subject-source-remedy-v1", Status: "not_needed", Approach: "practical", EvidenceIDs: []string{}}
	best := 0
	major, sub := findingActivePeriods(r)
	candidates := append([]string{}, names...)
	for _, e := range topic.Evidence {
		if e.Direction == "attention" && strings.HasPrefix(e.Kind, "conjunction-") && (e.Planet == "Rahu" || e.Planet == "Ketu") && !synthesisContainsPlanet(candidates, e.Planet) {
			candidates = append(candidates, e.Planet)
		}
	}
	for i, name := range candidates {
		ids := []string{}
		priority := 0
		for _, e := range topic.Evidence {
			if e.Planet != name || e.Direction != "attention" {
				continue
			}
			switch e.Kind {
			case "shadbala", "combustion", "neechabhanga", "planet-war", "lord-demand":
				ids = append(ids, e.ID)
				priority += 2
			default:
				if strings.HasPrefix(e.Kind, "conjunction-") && (name == "Rahu" || name == "Ketu") {
					ids = append(ids, e.ID)
					priority++
				}
			}
		}
		if len(ids) == 0 {
			continue
		}
		if i == 0 {
			priority += 3
		}
		if major != nil && sub != nil && (major.Lord == name || sub.Lord == name) {
			priority++
		}
		if priority <= best {
			continue
		}
		best = priority
		out.Status, out.Planet = "selected", name
		out.EvidenceIDs = ids
	}
	if out.Status != "selected" {
		return out
	}
	out.Approach = "balance"
	if gurujiRole(r.Chart.Lagna.RashiNumber, out.Planet) == "karak" {
		out.Approach = "support"
	}
	out.Reason = GurujiWording{"यदि यही परेशानी आपके अनुभव में भी आ रही है, तो एक आसान अभ्यास नियमित रखें।", "If this concern is also present in your experience, choose one manageable practice you can follow regularly."}
	out.Practice = remedyPractices[out.Planet]
	for _, remedy := range gurujiReferences.Remedies {
		if remedy.Planet == out.Planet {
			source := remedy.Source
			out.Source = &source
			break
		}
	}
	for _, rule := range r.Rules {
		if rule.Planet == out.Planet && normalizeRemedyCategory(rule.Category) == topic.Category && rule.Status == "approved" && rule.Approver != "" && rule.ApprovedAt != nil && !rule.Demo {
			out.ApprovedRules = append(out.ApprovedRules, rule)
		}
	}
	return out
}

// normalizeRemedyCategory accepts historical category aliases without mutating
// the approved record or treating uncategorised advice as every subject.
func normalizeRemedyCategory(category string) string {
	switch category {
	case "":
		return "general"
	case "marriage":
		return "relationships"
	case "health":
		return "wellbeing"
	case "debt":
		return "finance"
	}
	return category
}

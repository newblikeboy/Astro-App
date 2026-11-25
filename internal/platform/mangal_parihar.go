package platform

import (
	"astroapp/backend/internal/swisseph"
	"strings"
)

// mangalEvidenceBlocks adds source passages for observed conditions and a compact list of rules requiring review.
func (l pdfLocale) mangalEvidenceBlocks(rules []swisseph.MangalRule) []pdfBlock {
	blocks := []pdfBlock{}
	pending := pdfBlock{Title: l.text("mangal.parihar.review"), Note: l.text("mangal.parihar.review-note"), Headers: []string{l.text("mangal.parihar.title")}}
	for _, rule := range rules {
		if rule.SourceNumber == 0 {
			continue
		}
		if !rule.Met {
			if rule.Evaluation == "review" {
				pending.Rows = append(pending.Rows, []string{rule.Heading})
			}
			continue
		}
		state := rule.Effect
		if rule.Evaluation == "review" {
			state = "review"
		}
		rows := [][]string{{l.text("mangal.parihar." + state)}}
		if len(rule.MatchedClauses) > 0 {
			rows = append(rows, []string{l.text("mangal.parihar.basis") + ": " + strings.Join(rule.MatchedClauses, "\n")})
		}
		blocks = append(blocks, pdfBlock{Title: rule.Heading, Note: rule.Text, Headers: []string{l.text("Result")}, Rows: rows})
	}
	if len(pending.Rows) > 0 {
		blocks = append(blocks, pending)
	}
	return blocks
}

// mangalRuleState localizes only the computed status label; Guruji's supplied Hindi prose stays unchanged.
func (l pdfLocale) mangalRuleState(rule swisseph.MangalRule) string {
	state := rule.Effect
	if rule.Evaluation == "review" {
		state = "review"
	}
	return l.text("mangal.parihar." + state)
}

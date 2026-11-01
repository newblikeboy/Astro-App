package platform

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const integratedFaladeshVersion = "m4y-cross-factor-faladesh-v5"
const readerContractVersion = "m4y-reader-paragraphs-v1"

//go:embed data/reader-copy.json
var readerCopyJSON []byte

//go:embed data/reader-factor-copy.json
var readerFactorCopyJSON []byte

//go:embed data/remedy-practices.json
var remedyPracticesJSON []byte

type ReaderParagraph struct {
	ID          string        `json:"id"`
	Scope       string        `json:"scope"`
	Text        GurujiWording `json:"text"`
	EvidenceIDs []string      `json:"evidence_ids"`
}
type ReadingResolution struct {
	Factor      string   `json:"factor"`
	Direction   string   `json:"direction"`
	EvidenceIDs []string `json:"evidence_ids"`
	Rule        string   `json:"rule"`
}
type RemedyRecommendation struct {
	Method        string         `json:"method"`
	Status        string         `json:"status"`
	Planet        string         `json:"planet,omitempty"`
	Approach      string         `json:"approach"`
	Reason        GurujiWording  `json:"reason"`
	Practice      GurujiWording  `json:"practice"`
	EvidenceIDs   []string       `json:"evidence_ids"`
	Source        *GurujiLocator `json:"source,omitempty"`
	ApprovedRules []Rule         `json:"approved_rules,omitempty"`
}

type readerCopyCatalog struct {
	Version     string                              `json:"version"`
	Headlines   map[string][]GurujiWording          `json:"headlines"`
	Activities  []GurujiWording                     `json:"activities"`
	Capacity    map[string]map[string]GurujiWording `json:"capacity"`
	SimpleHindi map[string]string                   `json:"simple_hindi"`
	General     struct {
		Direction      []GurujiWording          `json:"direction"`
		Axes           []string                 `json:"axes"`
		Strength       map[string]GurujiWording `json:"strength"`
		Priority       map[string]GurujiWording `json:"priority"`
		Balances       map[string]GurujiWording `json:"balances"`
		SaturnBalance  GurujiWording            `json:"saturn_balance"`
		SaturnPriority GurujiWording            `json:"saturn_priority"`
		CapacityCare   GurujiWording            `json:"capacity_care"`
	} `json:"general"`
}

var readerCopy = loadReaderCopy()
var readerFactors = loadReaderFactors()
var remedyPractices = loadRemedyPractices()
var readerSentences = regexp.MustCompile(`[^.!?।]+[.!?।]*`)
var readerTechnical = regexp.MustCompile(`(?i)लग्न|कारक ग्रह|मारक|ग्रह की|ग्रहों|भाव[- ]संबंध|राशि-कुंडली|वर्ग-कुंडली|इस संबंध का फल दूसरे|\bD[0-9]+\b|ascendant|\bKarak\b|\bMarak\b|combust|divisional|natal house|sign placement|needs to be read alongside`)

// loadReaderCopy rejects incomplete generated editorial input at startup.
func loadReaderCopy() readerCopyCatalog {
	var out readerCopyCatalog
	if json.Unmarshal(readerCopyJSON, &out) != nil || out.Version == "" || len(out.General.Direction) != 12 || len(out.General.Axes) != 12 || len(out.Activities) != 12 {
		panic("invalid canonical reader copy")
	}
	for _, id := range []string{"personality", "education", "career", "finance", "relationships", "family", "wellbeing"} {
		if len(out.Headlines[id]) != 12 {
			panic("missing subject reader copy")
		}
		for _, kind := range []string{"care", "strength"} {
			if out.Capacity[kind][id].Hindi == "" || out.Capacity[kind][id].English == "" {
				panic("incomplete capacity copy")
			}
		}
	}
	if len(out.General.Balances) != 36 {
		panic("incomplete personal balance copy")
	}
	return out
}

// loadReaderFactors validates every category's original supporting and opposing
// consequences; factor prose is application interpretation, not forged approval.
func loadReaderFactors() map[string]map[string]GurujiWording {
	var out map[string]map[string]GurujiWording
	if json.Unmarshal(readerFactorCopyJSON, &out) != nil {
		panic("invalid reader factor copy")
	}
	for id := range readerCopy.Headlines {
		for _, key := range []string{"support", "care", "mixed", "help", "pressure", "yoga", "action"} {
			v := out[id][key]
			if v.Hindi == "" || v.English == "" {
				panic("incomplete reader factor copy")
			}
		}
	}
	return out
}

// loadRemedyPractices validates nine gentle, optional practices paraphrased from
// the supplied remedy pages; the selector never invents gemstone instructions.
func loadRemedyPractices() map[string]GurujiWording {
	var out map[string]GurujiWording
	if json.Unmarshal(remedyPracticesJSON, &out) != nil || len(out) != 9 {
		panic("invalid remedy practices")
	}
	for name := range synthesisHindiPlanets {
		if out[name].Hindi == "" || out[name].English == "" {
			panic("missing remedy practice")
		}
	}
	return out
}

// readerJoin appends bilingual prose without translating personal names/mantras.
func readerJoin(parts ...GurujiWording) GurujiWording {
	hi, en := []string{}, []string{}
	for _, p := range parts {
		if strings.TrimSpace(p.Hindi) != "" {
			hi = append(hi, strings.TrimSpace(p.Hindi))
		}
		if strings.TrimSpace(p.English) != "" {
			en = append(en, strings.TrimSpace(p.English))
		}
	}
	return GurujiWording{strings.Join(hi, " "), strings.Join(en, " ")}
}

// readerPlain removes old technical sentences, preserves uncertainty and keeps
// the same longest-first Hindi simplification used by the retained browser view.
func readerPlain(value GurujiWording, dropOpening bool) GurujiWording {
	result := []string{}
	for _, text := range []string{value.Hindi, value.English} {
		sentences := []string{}
		for _, sentence := range readerSentences.FindAllString(text, -1) {
			if !readerTechnical.MatchString(sentence) {
				sentences = append(sentences, strings.TrimSpace(sentence))
			}
		}
		if dropOpening && len(sentences) > 1 {
			sentences = sentences[1:]
		}
		result = append(result, strings.Join(sentences, " "))
	}
	keys := make([]string, 0, len(readerCopy.SimpleHindi))
	for key := range readerCopy.SimpleHindi {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(keys[i]) == len(keys[j]) {
			return keys[i] < keys[j]
		}
		return len(keys[i]) > len(keys[j])
	})
	for _, key := range keys {
		result[0] = strings.ReplaceAll(result[0], key, readerCopy.SimpleHindi[key])
	}
	return GurujiWording{result[0], result[1]}
}

// readerEvidence returns stable IDs from the topic's actual retained ledger.
func readerEvidence(topic *FaladeshTopic, kinds []string, planets ...string) []string {
	ids := []string{}
	for _, e := range topic.Evidence {
		kindMatch, planetMatch := len(kinds) == 0, len(planets) == 0
		for _, kind := range kinds {
			kindMatch = kindMatch || e.Kind == kind || strings.HasSuffix(kind, "*") && strings.HasPrefix(e.Kind, strings.TrimSuffix(kind, "*"))
		}
		for _, name := range planets {
			planetMatch = planetMatch || e.Planet == name
		}
		if kindMatch && planetMatch {
			ids = append(ids, e.ID)
		}
	}
	return ids
}

// addReaderParagraph validates all cause IDs and deduplicates complete sentences
// across the category; the same stored paragraphs feed browser and PDF.
func addReaderParagraph(topic *FaladeshTopic, id, scope string, text GurujiWording, ids []string) {
	for _, cause := range ids {
		found := false
		for _, e := range topic.Evidence {
			found = found || e.ID == cause
		}
		if !found {
			return
		}
	}
	text = readerPlain(text, false)
	seenHI, seenEN := map[string]bool{}, map[string]bool{}
	for _, p := range topic.Reader {
		for _, s := range readerSentences.FindAllString(p.Text.Hindi, -1) {
			seenHI[strings.TrimSpace(s)] = true
		}
		for _, s := range readerSentences.FindAllString(p.Text.English, -1) {
			seenEN[strings.TrimSpace(s)] = true
		}
	}
	parts := []string{}
	for index, value := range []string{text.Hindi, text.English} {
		seen := seenHI
		if index == 1 {
			seen = seenEN
		}
		sentences := []string{}
		for _, s := range readerSentences.FindAllString(value, -1) {
			s = strings.Join(strings.Fields(s), " ")
			if s != "" && !seen[s] {
				sentences = append(sentences, s)
				seen[s] = true
			}
		}
		parts = append(parts, strings.Join(sentences, " "))
	}
	if parts[0] == "" || parts[1] == "" {
		return
	}
	unique := []string{}
	seen := map[string]bool{}
	for _, v := range ids {
		if !seen[v] {
			unique = append(unique, v)
			seen[v] = true
		}
	}
	topic.Reader = append(topic.Reader, ReaderParagraph{id, scope, GurujiWording{parts[0], parts[1]}, unique})
}

// buildIntegratedFaladesh preserves the supported legacy ledger, adds the new
// calculation qualifications, and freezes a single bilingual reader contract.
func buildIntegratedFaladesh(r *Report) *FaladeshSynthesis {
	out := buildFaladeshSynthesis(r)
	if out == nil || r.Details.Shadbala == nil || r.Details.Qualifications == nil {
		return out
	}
	out.Version, out.ReaderVersion = integratedFaladeshVersion, readerContractVersion
	out.Coverage = append(out.Coverage, "six-component Shadbala with subcomponents", "Phaladeepika debility-cancellation rules", "yoga strength qualifications", "personal lunar Gochar/Vedha", "ranked source-linked remedy selection", "shared bilingual reader snapshot")
	out.Limits = []GurujiWording{{"फलादेश में जीवन के रुझान और ध्यान देने योग्य बातें दी गई हैं। इससे किसी घटना, बीमारी या परिणाम की पुष्टि नहीं होती।", "The reading describes tendencies and priorities, not verified events, diagnoses or guaranteed outcomes."}, {"गणना घोषित परंपरा के नियमों के अनुसार है। किसी दूसरी परंपरा का अलग नियम अपने-आप इसमें नहीं मिलाया गया है।", "Calculations follow the declared conventions; other schools are not silently mixed into them."}}
	for i := range out.Topics {
		topic := &out.Topics[i]
		if topic.Category == "general" {
			// General is a separate personal-direction reading. Rebuilding its
			// ledger from the personal topic removes imported work/money summaries.
			for _, p := range out.Topics {
				if p.Category == "personality" {
					topic.Evidence = append([]FaladeshEvidence{}, p.Evidence...)
					break
				}
			}
			topic.Findings = nil
			topic.NextPeriod = nil
			topic.PeriodRelevant = false
		}
		addAdvancedEvidence(r, topic)
		if topic.Category == "general" {
			buildGeneralReader(r, topic)
		} else if topic.Category == "dasha" {
			buildDashaReader(r, topic)
		} else {
			buildSubjectReader(r, topic)
		}
		if len(topic.Reader) > 0 {
			topic.Conclusion = topic.Reader[0].Text
			topic.Potential = topic.Reader[len(topic.Reader)-1].Text
		}
	}
	return out
}

// buildGeneralReader retains four personal-direction paragraphs and qualifies
// the resource/balance from actual strength and cancellation, without other tabs.
func buildGeneralReader(r *Report, topic *FaladeshTopic) {
	name := synthesisLord(r.Chart.Lagna.RashiNumber, 1)
	anchor, moon := findingPlanet(r, name), findingPlanet(r, "Moon")
	if anchor == nil || moon == nil {
		return
	}
	focus, need := readerCopy.General.Axes[anchor.House-1], readerCopy.General.Axes[moon.House-1]
	personal := readerEvidence(topic, []string{"placement"}, name)
	combined := append(append([]string{}, personal...), readerEvidence(topic, []string{"placement"}, "Moon")...)
	resource := readerCopy.General.Strength[focus]
	capacity := readerEvidence(topic, []string{"shadbala", "neechabhanga", "combustion", "dignity"}, name)
	attention := false
	environmentCare := false
	for _, resolution := range topic.Resolution {
		if (resolution.Factor == "capacity" || resolution.Factor == "foundation") && (resolution.Direction == "attention" || resolution.Direction == "mixed") {
			attention = true
			capacity = append(capacity, resolution.EvidenceIDs...)
		}
		if resolution.Factor == "environment" && (resolution.Direction == "attention" || resolution.Direction == "mixed") {
			environmentCare = true
			combined = append(combined, resolution.EvidenceIDs...)
		}
	}
	if attention {
		resource = readerJoin(resource, readerCopy.General.CapacityCare)
	}
	balance, action := readerCopy.General.Balances[focus+"/"+need], readerCopy.General.Priority[focus]
	if saturn := findingPlanet(r, "Saturn"); saturn != nil && saturn.House == anchor.House && saturn.House == moon.House {
		balance, action = readerCopy.General.SaturnBalance, readerCopy.General.SaturnPriority
		combined = append(combined, readerEvidence(topic, []string{"placement"}, "Saturn")...)
	}
	if environmentCare {
		balance = readerJoin(balance, readerFactors["personality"]["pressure"])
	}
	addReaderParagraph(topic, "reader-general-direction", "natal", readerCopy.General.Direction[anchor.House-1], personal)
	addReaderParagraph(topic, "reader-general-strength", "natal", resource, append(personal, capacity...))
	addReaderParagraph(topic, "reader-general-balance", "natal", balance, combined)
	topic.Remedy = selectTopicRemedy(r, topic, []string{name, "Moon"})
	if topic.Remedy.Status == "selected" {
		action = readerJoin(action, topic.Remedy.Reason, topic.Remedy.Practice)
		combined = append(combined, topic.Remedy.EvidenceIDs...)
	}
	addReaderParagraph(topic, "reader-general-priority", "action", action, combined)
}

// buildSubjectReader connects every distinct subject owner to its native D1
// destination, then adds supported cross-factor qualifications and practical help.
func buildSubjectReader(r *Report, topic *FaladeshTopic) {
	var spec synthesisTopicSpec
	for _, s := range synthesisSpecs {
		if s.ID == topic.Category {
			spec = s
		}
	}
	if spec.Primary == 0 {
		return
	}
	primary := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
	p := findingPlanet(r, primary)
	if p == nil {
		return
	}
	for _, f := range topic.Findings {
		if f.Kind == "principal-lord" {
			addReaderParagraph(topic, "reader-conclusion-"+topic.Category, "natal", readerJoin(readerCopy.Headlines[topic.Category][p.House-1], readerPlain(f.Text, true)), f.EvidenceIDs)
		}
	}
	for _, f := range topic.Findings {
		if f.Kind == "occupants" {
			addReaderParagraph(topic, "reader-occupants-"+topic.Category, "natal", readerPlain(f.Text, false), f.EvidenceIDs)
		}
	}
	seen := map[string]bool{primary: true}
	names := []string{primary}
	for _, house := range spec.Houses {
		name := synthesisLord(r.Chart.Lagna.RashiNumber, house)
		if seen[name] {
			continue
		}
		seen[name] = true
		names = append(names, name)
		secondary := findingPlanet(r, name)
		if secondary == nil {
			continue
		}
		f := lordHouseFinding(r, spec, topic, house, "secondary-lord")
		if secondary.House == p.House {
			continue
		}
		addReaderParagraph(topic, fmt.Sprintf("reader-secondary-%s-%d", topic.Category, house), "natal", readerJoin(readerCopy.Headlines[topic.Category][secondary.House-1], readerPlain(f.Text, true)), f.EvidenceIDs)
	}
	addResolvedParagraphs(r, topic)
	addPersonalPeriodReader(r, topic, names)
	topic.Remedy = selectTopicRemedy(r, topic, names)
	action := detailedFindingAction(spec, p.House)
	if len(readerSentences.FindAllString(action.Hindi, -1)) < 2 {
		action = readerFactors[topic.Category]["action"]
	}
	ids := readerEvidence(topic, []string{"placement"}, names...)
	if topic.Remedy.Status == "selected" {
		action = readerJoin(action, topic.Remedy.Reason, topic.Remedy.Practice)
		ids = append(ids, topic.Remedy.EvidenceIDs...)
	}
	addReaderParagraph(topic, "reader-action-"+topic.Category, "action", action, ids)
}

// buildDashaReader chooses the strongest direct house-owner connection of each
// running lord, preserving the saved reference and actual parent/sub boundaries.
func buildDashaReader(r *Report, topic *FaladeshTopic) {
	major, sub := findingActivePeriods(r)
	if major == nil || sub == nil {
		return
	}
	addReaderParagraph(topic, "reader-dasha-conclusion", "period", GurujiWording{fmt.Sprintf("इस समय %s महादशा में %s अंतर्दशा चल रही है। इस दौर में नीचे बताई गई बातों पर ध्यान देना उपयोगी रहेगा।", synthesisHindiPlanets[major.Lord], synthesisHindiPlanets[sub.Lord]), fmt.Sprintf("The running period is %s Mahadasha with %s Antardasha. The subjects below deserve attention during this interval.", major.Lord, sub.Lord)}, readerEvidence(topic, []string{"period-window"}))
	seen := map[string]bool{}
	for _, name := range []string{major.Lord, sub.Lord} {
		if seen[name] {
			continue
		}
		seen[name] = true
		best := synthesisTopicSpec{}
		score := 0
		for _, spec := range synthesisSpecs {
			if spec.ID == "general" || spec.ID == "dasha" {
				continue
			}
			weight := 0
			for _, h := range spec.Houses {
				if synthesisLord(r.Chart.Lagna.RashiNumber, h) == name {
					weight = 2
				}
			}
			if synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary) == name {
				weight = 3
			}
			if weight > score {
				score, best = weight, spec
			}
		}
		p := findingPlanet(r, name)
		if p == nil {
			continue
		}
		text := readerCopy.Activities[p.House-1]
		text = GurujiWording{"इस दौर में " + text.Hindi + " पर ध्यान देना जरूरी रहेगा।", "This period draws attention to " + text.English + "."}
		if best.Primary > 0 {
			text = readerJoin(text, readerCopy.Headlines[best.ID][p.House-1])
		}
		addReaderParagraph(topic, "reader-dasha-"+name, "period", text, readerEvidence(topic, []string{"placement"}, name))
	}
	addResolvedParagraphs(r, topic)
	addPersonalPeriodReader(r, topic, []string{major.Lord, sub.Lord})
	topic.Remedy = selectTopicRemedy(r, topic, []string{major.Lord, sub.Lord})
	action := GurujiWording{"जो बात अभी वास्तव में परेशानी दे रही है, उसी को पहले सँभालें। एक साथ कई उपाय शुरू करने के बजाय एक आसान कदम नियमित करें।", "Address the concern that is actually present first. Follow one manageable step regularly rather than starting several practices together."}
	if topic.Remedy.Status == "selected" {
		action = readerJoin(action, topic.Remedy.Reason, topic.Remedy.Practice)
	}
	addReaderParagraph(topic, "reader-dasha-action", "action", action, append(readerEvidence(topic, []string{"placement"}, major.Lord, sub.Lord), topic.Remedy.EvidenceIDs...))
}

package platform

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

//go:embed data/faladesh-synthesis-copy.json
var faladeshSynthesisCopyJSON []byte

type FaladeshEvidence struct {
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Planet    string        `json:"planet"`
	House     int           `json:"house,omitempty"`
	Direction string        `json:"direction"`
	Text      GurujiWording `json:"text"`
}
type FaladeshPoint struct {
	Tone        string        `json:"tone"`
	Text        GurujiWording `json:"text"`
	EvidenceIDs []string      `json:"evidence_ids"`
}
type FaladeshWindow struct {
	Lord  string        `json:"lord"`
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Text  GurujiWording `json:"text"`
}
type FaladeshTopic struct {
	Reader         []ReaderParagraph     `json:"reader,omitempty"`
	Remedy         *RemedyRecommendation `json:"remedy_recommendation,omitempty"`
	Resolution     []ReadingResolution   `json:"resolution,omitempty"`
	Category       string                `json:"category"`
	Assessment     string                `json:"assessment"`
	Conclusion     GurujiWording         `json:"conclusion"`
	Present        GurujiWording         `json:"present"`
	PeriodRelevant bool                  `json:"period_relevant,omitempty"`
	Potential      GurujiWording         `json:"potential"`
	Findings       []FaladeshFinding     `json:"findings,omitempty"`
	MainPoints     []FaladeshPoint       `json:"main_points"`
	Evidence       []FaladeshEvidence    `json:"evidence"`
	RelatedPlanets []string              `json:"related_planets"`
	NextPeriod     *FaladeshWindow       `json:"next_period,omitempty"`
}
type FaladeshSynthesis struct {
	ReaderVersion string          `json:"reader_version,omitempty"`
	Version       string          `json:"version"`
	AsOf          time.Time       `json:"as_of"`
	Topics        []FaladeshTopic `json:"topics"`
	Coverage      []string        `json:"coverage"`
	Limits        []GurujiWording `json:"limits"`
}
type synthesisTopicSpec struct {
	ID      string
	Houses  []int
	Primary int
	Karakas []string
	Varga   int
}
type synthesisTopicCopy struct {
	Support     GurujiWording `json:"support"`
	Attention   GurujiWording `json:"attention"`
	Mixed       GurujiWording `json:"mixed"`
	Context     GurujiWording `json:"context"`
	Opportunity GurujiWording `json:"opportunity"`
	Caution     GurujiWording `json:"caution"`
	Action      GurujiWording `json:"action"`
}

var synthesisSpecs = []synthesisTopicSpec{
	{"general", []int{1, 2, 4, 5, 7, 9, 10, 11}, 1, []string{"Moon"}, 9},
	{"personality", []int{1, 3, 5}, 1, []string{"Sun", "Moon", "Mercury"}, 9},
	{"education", []int{4, 5, 9}, 5, []string{"Mercury", "Jupiter"}, 24},
	{"career", []int{6, 10, 11}, 10, []string{"Sun", "Saturn"}, 10},
	{"finance", []int{2, 11}, 2, []string{"Jupiter", "Venus"}, 2},
	{"relationships", []int{5, 7}, 7, []string{"Venus", "Jupiter"}, 9},
	{"family", []int{2, 3, 4}, 4, []string{"Moon"}, 4},
	{"wellbeing", []int{1, 6, 8, 12}, 1, []string{"Sun", "Moon"}, 9},
	{"dasha", nil, 1, nil, 9},
}
var synthesisSignLords = []string{"Mars", "Venus", "Mercury", "Moon", "Sun", "Mercury", "Venus", "Mars", "Jupiter", "Saturn", "Saturn", "Jupiter"}
var synthesisHindiPlanets = map[string]string{"Sun": "सूर्य", "Moon": "चंद्रमा", "Mars": "मंगल", "Mercury": "बुध", "Jupiter": "गुरु", "Venus": "शुक्र", "Saturn": "शनि", "Rahu": "राहु", "Ketu": "केतु"}
var synthesisCopy = loadSynthesisCopy()

// loadSynthesisCopy validates the versioned editorial vocabulary; these are application interpretations, not Guruji approval records.
func loadSynthesisCopy() map[string]synthesisTopicCopy {
	out := map[string]synthesisTopicCopy{}
	if err := json.Unmarshal(faladeshSynthesisCopyJSON, &out); err != nil {
		panic("invalid combined Faladesh copy")
	}
	for _, spec := range synthesisSpecs {
		v, ok := out[spec.ID]
		if !ok {
			panic("incomplete combined Faladesh copy")
		}
		for _, wording := range []GurujiWording{v.Support, v.Attention, v.Mixed, v.Context, v.Opportunity, v.Caution, v.Action} {
			if wording.Hindi == "" || wording.English == "" {
				panic("incomplete combined Faladesh copy")
			}
		}
	}
	return out
}

// synthesisContains tests explicit membership; an empty house is not interpreted as an absent life subject.
func synthesisContains(values []int, value int) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// synthesisLord resolves ordinary D1 sign ownership; nodes receive no invented sign ownership.
func synthesisLord(lagna, house int) string { return synthesisSignLords[(lagna+house-2)%12] }

// appendSynthesisEvidence adds a stable, deduplicated fact with a separately declared interpretive direction.
func appendSynthesisEvidence(topic *FaladeshTopic, kind, planet string, house int, direction string, hi, en string) {
	id := fmt.Sprintf("%s:%s:%d", kind, planet, house)
	for _, item := range topic.Evidence {
		if item.ID == id {
			return
		}
	}
	topic.Evidence = append(topic.Evidence, FaladeshEvidence{id, kind, planet, house, direction, GurujiWording{hi, en}})
}

// buildFaladeshSynthesis snapshots supported cross-factor interpretations only after the report's complete calculations exist.
// This bounded rule set is not full Shadbala, a probability model, a factual life-event detector, or personal remedy approval.
func buildFaladeshSynthesis(r *Report) *FaladeshSynthesis {
	if r.Chart == nil || r.Details == nil || r.GurujiFaladesh == nil || r.Birth.Unknown {
		return nil
	}
	if r.Chart.Lagna.RashiNumber < 1 || r.Chart.Lagna.RashiNumber > 12 {
		return nil
	}
	seen := map[string]bool{}
	for _, p := range r.Chart.Planets {
		if seen[p.Name] || p.RashiNumber < 1 || p.RashiNumber > 12 || p.House < 1 || p.House > 12 {
			return nil
		}
		seen[p.Name] = true
	}
	for name := range synthesisHindiPlanets {
		if !seen[name] {
			return nil
		}
	}
	out := &FaladeshSynthesis{Version: "m4y-cross-factor-faladesh-v4", AsOf: r.Created, Topics: []FaladeshTopic{}, Coverage: []string{"D1 lordship and occupants", "whole-sign conjunctions and graha drishti", "native sign dignity and combustion", "dispositor condition", "D9 and subject varga context", "Chalit differences", "Ashtakavarga context", "recorded yoga configurations and cancellations", "dated Vimshottari major/sub periods", "dated transit snapshot"}, Limits: []GurujiWording{{"जन्म-कुंडली और दशा के संकेत संभावनाएँ हैं; वास्तविक घटनाओं की पुष्टि नहीं।", "Natal and dasha indications are possibilities, not verified life events."}, {"यह घोषित नियमों का संयुक्त आकलन है; पूर्ण षड्बल, सभी दोषभंग, गोचर-वेद्ध और घटना की निश्चित तारीख इसमें शामिल नहीं हैं।", "This combines the declared rules; full Shadbala, every cancellation, transit Vedha and exact event dates are not included."}, {"उपाय का व्यक्तिगत चयन गुरुजी के आकलन से हो; यहाँ कोई रत्न या अनुष्ठान अनिवार्य नहीं बताया गया है।", "Personal remedy selection requires an individual assessment; no gemstone or ritual is declared mandatory here."}}}
	for _, spec := range synthesisSpecs {
		out.Topics = append(out.Topics, synthesizeFaladeshTopic(r, spec))
	}
	composeDetailedOverall(r, out)
	return out
}

// synthesisSubject names the life subject explicitly so causes do not inherit another category's generic source interpretation.
func synthesisSubject(category string) GurujiWording {
	return map[string]GurujiWording{
		"general": {"जीवन की दिशा", "overall direction"}, "personality": {"स्वभाव", "personality"},
		"education": {"पढ़ाई", "studies"}, "career": {"करियर", "career"}, "finance": {"धन और बचत", "money and savings"},
		"relationships": {"संबंध", "relationships"}, "family": {"घर-परिवार", "home and family"},
		"wellbeing": {"दिनचर्या और विश्राम", "routine and rest"}, "dasha": {"दशा से जुड़े विषय", "the period's connected subjects"},
	}[category]
}

// synthesisEffect connects a chart cause to a complete subject-specific consequence, avoiding vague phrases such as a demanding personality.
func synthesisEffect(category, direction string) GurujiWording {
	if direction == "support" {
		return map[string]GurujiWording{
			"general":       {"इससे अपनी जिम्मेदारियाँ सँभालने की क्षमता को सहारा मिलता है।", "This supports your ability to handle responsibilities."},
			"personality":   {"इससे आत्मविश्वास और स्पष्ट अभिव्यक्ति को सहारा मिलता है।", "This supports confidence and clear expression."},
			"education":     {"इससे विषय समझने और पढ़ाई में आगे बढ़ने की क्षमता को सहारा मिलता है।", "This supports understanding a subject and making progress in your studies."},
			"career":        {"इससे काम की जिम्मेदारी सँभालने और पेशेवर पहचान बनाने को सहारा मिलता है।", "This supports taking responsibility at work and building a professional identity."},
			"finance":       {"इससे आय और बचत सँभालने की क्षमता को सहारा मिलता है।", "This supports managing income and savings."},
			"relationships": {"इससे आपसी सहयोग और संबंध निभाने की क्षमता को सहारा मिलता है।", "This supports cooperation and sustaining a relationship."},
			"family":        {"इससे घरेलू जिम्मेदारियाँ सँभालने और परिवार को सहयोग देने की क्षमता को सहारा मिलता है।", "This supports handling home responsibilities and contributing to family life."},
			"wellbeing":     {"इससे नियमित दिनचर्या बनाए रखने की क्षमता को सहारा मिलता है।", "This supports maintaining a regular routine."},
			"dasha":         {"इससे दशा से जुड़े विषयों में अपनी तैयारी का लाभ लेने को सहारा मिलता है।", "This supports using your preparation in the period's connected subjects."},
		}[category]
	}
	return map[string]GurujiWording{
		"general":       {"इसी वजह से कुछ जिम्मेदारियाँ निभाने में अधिक मेहनत या समायोजन लग सकता है।", "This can mean extra effort or adjustment in handling some responsibilities."},
		"personality":   {"इसी वजह से अपनी बात कहने या फैसले पर टिके रहने में कठिनाई हो सकती है।", "This may make it harder to express your views or stay with a decision."},
		"education":     {"इसी वजह से पढ़ाई में निरंतरता बनाए रखने में अतिरिक्त प्रयास लग सकता है।", "This may require extra effort to keep your studies consistent."},
		"career":        {"इसी वजह से मेहनत को पेशेवर प्रगति में बदलने में अतिरिक्त प्रयास लग सकता है।", "This may require extra effort to turn hard work into professional progress."},
		"finance":       {"इसी वजह से कमाई को बचत में बदलने पर अधिक ध्यान देना पड़ सकता है।", "This may require closer attention to turning earnings into savings."},
		"relationships": {"इसी वजह से आपसी समझ और अपेक्षाओं में तालमेल पर अधिक ध्यान देना पड़ सकता है।", "This may require more care with mutual understanding and expectations."},
		"family":        {"इसी वजह से घरेलू जिम्मेदारियों और सहयोग में तालमेल पर अधिक ध्यान देना पड़ सकता है।", "This may require more care in balancing family responsibilities and support."},
		"wellbeing":     {"इसी वजह से जिम्मेदारियों के बीच विश्राम का समय निकालने में अतिरिक्त प्रयास लग सकता है।", "This may make it harder to find time for rest alongside responsibilities."},
		"dasha":         {"इसी वजह से इस दशा से जुड़े फैसलों में अधिक सावधानी उपयोगी रहेगी।", "This makes extra care useful in decisions connected to this period."},
	}[category]
}

// synthesisCause describes an actual directional fact in the current subject, leaving the complete technical wording in references.
func synthesisCause(r *Report, spec synthesisTopicSpec, e FaladeshEvidence) GurujiWording {
	nameHI, nameEN := synthesisHindiPlanets[e.Planet], e.Planet
	owned := []string{}
	for _, house := range spec.Houses {
		if synthesisLord(r.Chart.Lagna.RashiNumber, house) == e.Planet {
			owned = append(owned, fmt.Sprint(house))
		}
	}
	subject := synthesisSubject(spec.ID)
	if len(owned) > 0 {
		nameHI = subject.Hindi + " से जुड़े भाव (" + strings.Join(owned, ", ") + ") के स्वामी " + nameHI
		nameEN += ", ruling house " + strings.Join(owned, ", ") + " for " + subject.English + ","
	} else if synthesisContainsPlanet(spec.Karakas, e.Planet) {
		nameHI = subject.Hindi + " के कारक " + nameHI
		nameEN += ", a significator for " + subject.English + ","
	}
	switch e.Kind {
	case "lord-demand":
		return GurujiWording{fmt.Sprintf("%s भाव %d में हैं।", nameHI, e.House), fmt.Sprintf("%s occupies house %d.", nameEN, e.House)}
	case "combustion":
		return GurujiWording{nameHI + " सूर्य के निकट होने से अस्त हैं।", nameEN + " is combust due to proximity to the Sun."}
	case "dignity":
		return e.Text
	}
	// Keep only the factual sentence; qualifications and cancellations remain available in the full evidence list.
	return GurujiWording{strings.SplitAfter(e.Text.Hindi, "।")[0], strings.SplitAfter(e.Text.English, ".")[0]}
}

// synthesisContainsPlanet checks membership without giving nodes invented ownership or significator roles.
func synthesisContainsPlanet(planets []string, name string) bool {
	for _, planet := range planets {
		if planet == name {
			return true
		}
	}
	return false
}

// synthesizeFaladeshTopic evaluates all relevant owners, occupants, significators and aspectors; opposing evidence remains visible instead of being cancelled by a numerical score.
func synthesizeFaladeshTopic(r *Report, spec synthesisTopicSpec) FaladeshTopic {
	topic := FaladeshTopic{Category: spec.ID, Evidence: []FaladeshEvidence{}, RelatedPlanets: []string{}, MainPoints: []FaladeshPoint{}}
	lagna := r.Chart.Lagna.RashiNumber
	primary := synthesisLord(lagna, spec.Primary)
	relevant := map[string]bool{primary: true}
	focal := map[string]bool{primary: true}
	for _, h := range spec.Houses {
		relevant[synthesisLord(lagna, h)] = true
		focal[synthesisLord(lagna, h)] = true
	}
	for _, name := range spec.Karakas {
		relevant[name] = true
		focal[name] = true
	}
	if spec.ID == "general" {
		for _, p := range r.Chart.Planets {
			relevant[p.Name] = true
		}
	}
	if spec.ID == "dasha" {
		relevant = map[string]bool{}
		focal = map[string]bool{}
		if r.Dasha != nil {
			relevant[r.Dasha.CurrentLord] = true
			relevant[r.Dasha.CurrentSubLord] = true
			focal[r.Dasha.CurrentLord] = true
			focal[r.Dasha.CurrentSubLord] = true
		}
	}
	for _, p := range r.Chart.Planets {
		if synthesisContains(spec.Houses, p.House) {
			relevant[p.Name] = true
		}
	}
	for _, a := range r.Details.Analysis.Aspects {
		connected := synthesisContains(spec.Houses, a.House)
		for _, name := range a.Occupants {
			connected = connected || focal[name]
		}
		if connected {
			relevant[a.Planet] = true
		}
	}
	for _, p := range r.Chart.Planets {
		if !relevant[p.Name] {
			continue
		}
		topic.RelatedPlanets = append(topic.RelatedPlanets, p.Name)
		hi := synthesisHindiPlanets[p.Name]
		owned := []string{}
		for h := 1; h <= 12; h++ {
			if synthesisLord(lagna, h) == p.Name && synthesisContains(spec.Houses, h) {
				owned = append(owned, fmt.Sprint(h))
			}
		}
		prefixHI, prefixEN := "", ""
		if len(owned) > 0 {
			prefixHI = "भाव " + strings.Join(owned, ", ") + " के स्वामी "
			prefixEN = "Lord of house " + strings.Join(owned, ", ") + ": "
		}
		appendSynthesisEvidence(&topic, "placement", p.Name, p.House, "context", fmt.Sprintf("%s%s जन्म में भाव %d और राशि %d में हैं।", prefixHI, hi, p.House, p.RashiNumber), fmt.Sprintf("%s%s is in natal house %d, sign %d.", prefixEN, p.Name, p.House, p.RashiNumber))
		for _, reading := range r.GurujiFaladesh.Readings {
			if reading.Planet == p.Name {
				roleHI := map[string]string{"karak": "कारक", "marak": "मारक", "neutral": "सम", "unlisted": "असूचीबद्ध"}[reading.Role]
				appendSynthesisEvidence(&topic, "teacher-role", p.Name, p.House, "context", fmt.Sprintf("गुरुजी के दिए लग्न-नोट में %s की श्रेणी %s है। इस श्रेणी को ग्रहबल या अंतिम शुभ/अशुभ फल के बराबर नहीं माना गया।", hi, roleHI), fmt.Sprintf("Guruji's supplied ascendant note lists %s as %s. This category is kept separate from strength and the final outcome.", p.Name, reading.Role))
			}
		}
		role := gurujiRole(lagna, p.Name)
		status := gurujiSignStatus(p.Name, p.RashiNumber)
		effect := gurujiRoleEffect(role, status)
		direction := "context"
		if focal[p.Name] {
			direction = effect.Direction
		}
		roleHI := map[string]string{"karak": "कारक", "marak": "मारक", "neutral": "सम", "unlisted": "बोर्ड में नहीं दी गई"}[role]
		label := gurujiSignLabel(status)
		appendSynthesisEvidence(&topic, "dignity", p.Name, p.House, direction, fmt.Sprintf("%s की लग्न-भूमिका %s है और राशि %d में स्थिति %s है।", hi, roleHI, p.RashiNumber, label.Hindi), fmt.Sprintf("%s has board role %s and sign %d status: %s.", p.Name, role, p.RashiNumber, label.English))
		appendSynthesisEvidence(&topic, "role-sign", p.Name, p.House, "context", hi+": "+effect.Explanation.Hindi, p.Name+": "+effect.Explanation.English)
		if p.Name == primary && spec.ID != "dasha" && synthesisContains([]int{6, 8, 12}, p.House) && p.House != spec.Primary {
			appendSynthesisEvidence(&topic, "lord-demand", p.Name, p.House, "attention", fmt.Sprintf("मुख्य भावेश %s का भाव %d से संबंध अतिरिक्त मेहनत, समायोजन या खर्च पर ध्यान माँगता है; अकेले इससे असफलता तय नहीं होती।", hi, p.House), fmt.Sprintf("The main house lord %s is in house %d, calling for attention to effort, adjustment or resources.", p.Name, p.House))
		}
		for _, c := range r.Details.Analysis.Conditions {
			if c.Name != p.Name {
				continue
			}
			if c.Combust != nil && *c.Combust {
				direction := "context"
				if focal[p.Name] {
					direction = "attention"
				}
				appendSynthesisEvidence(&topic, "combustion", p.Name, p.House, direction, hi+" सूर्य से दहन की जाँची गई सीमा के भीतर हैं। अस्त होना, लग्न की भूमिका और राशि की उच्च/नीच स्थिति अलग तथ्य हैं।", p.Name+" is within the checked combustion limit. Combustion, ascendant role and sign dignity are separate facts.")
			}
			if p.Name == primary && c.SignLord != p.Name {
				for _, d := range r.Details.Analysis.Conditions {
					if d.Name == c.SignLord {
						appendSynthesisEvidence(&topic, "dispositor", d.Name, p.House, "context", fmt.Sprintf("मुख्य भावेश की राशि के स्वामी %s की स्थिति %s है; भावेश को अकेले नहीं पढ़ा गया है।", synthesisHindiPlanets[d.Name], map[string]string{"Own sign": "स्वराशि", "Exaltation sign": "उच्च राशि", "Debilitation sign": "नीच राशि", "Friend sign": "मित्र राशि", "Enemy sign": "शत्रु राशि", "Neutral sign": "सम राशि", "Not assigned": "अवर्गीकृत"}[d.Dignity]), fmt.Sprintf("The main lord's dispositor %s has %s; the house lord is not read in isolation.", d.Name, d.Dignity))
					}
				}
			}
		}
		if focal[p.Name] {
			for _, q := range r.Chart.Planets {
				if q.Name != p.Name && q.RashiNumber == p.RashiNumber {
					direction := "context"
					if q.Name == "Rahu" || q.Name == "Ketu" {
						direction = "attention"
					}
					appendSynthesisEvidence(&topic, "conjunction-"+p.Name, q.Name, p.House, direction, fmt.Sprintf("%s और %s एक जन्म राशि में हैं। यह राशि-युति है; इसे अकेले निश्चित दोष या घटना नहीं कहा गया है।", hi, synthesisHindiPlanets[q.Name]), fmt.Sprintf("%s and %s share a natal sign. This is a whole-sign conjunction, not a standalone Dosha or event verdict.", p.Name, q.Name))
				}
			}
		}
		if p.Retrograde {
			appendSynthesisEvidence(&topic, "retrograde", p.Name, p.House, "context", hi+" वक्री हैं; वक्रता को अपने-आप कमजोरी या बाधा नहीं माना गया है।", p.Name+" is retrograde; retrogradation is not automatically treated as weakness.")
		}
	}
	for _, a := range r.Details.Analysis.Aspects {
		connected := synthesisContains(spec.Houses, a.House)
		for _, name := range a.Occupants {
			connected = connected || focal[name]
		}
		if !connected {
			continue
		}
		direction := "context"
		if a.House == spec.Primary && a.Planet == primary {
			direction = "support"
		}
		appendSynthesisEvidence(&topic, "aspect", a.Planet, a.House, direction, fmt.Sprintf("%s की पूर्ण ग्रह-दृष्टि भाव %d पर है।", synthesisHindiPlanets[a.Planet], a.House), fmt.Sprintf("%s casts a full graha aspect on house %d.", a.Planet, a.House))
	}
	addSynthesisDivisions(r, spec, &topic, primary)
	addSynthesisYogas(r, spec, &topic, relevant)
	timingRelevant := map[string]bool{}
	if spec.ID == "dasha" {
		for name := range focal {
			timingRelevant[name] = true
		}
	} else {
		for _, house := range spec.Houses {
			timingRelevant[synthesisLord(lagna, house)] = true
		}
	}
	for _, p := range r.Chart.Planets {
		if synthesisContains(spec.Houses, p.House) {
			timingRelevant[p.Name] = true
		}
	}
	finalizeSynthesisTopic(r, spec, &topic, timingRelevant)
	buildDetailedFaladeshTopic(r, spec, &topic)
	return topic
}

// addSynthesisDivisions records D9/subject-varga agreement, Chalit shifts and actual bindus without inventing probability thresholds or calling Vimshopaka full Shadbala.
func addSynthesisDivisions(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic, primary string) {
	for _, v := range r.Details.Vargas {
		if v.Factor != 9 && v.Factor != spec.Varga {
			continue
		}
		for _, p := range v.Planets {
			if p.Name != primary {
				continue
			}
			text := fmt.Sprintf("D%d में मुख्य भावेश %s राशि %d, भाव %d में हैं।", v.Factor, synthesisHindiPlanets[p.Name], p.RashiNumber, p.House)
			en := fmt.Sprintf("In D%d, the main lord %s occupies sign %d, house %d.", v.Factor, p.Name, p.RashiNumber, p.House)
			if p.Vargottama && v.Factor == 9 {
				text += " D1 और D9 की राशि समान है (वर्गोत्तम); यह अकेले शुभ फल की गारंटी नहीं।"
				en += " The D1/D9 signs agree (Vargottama); this alone does not guarantee a favourable outcome."
			}
			status := gurujiSignStatus(p.Name, p.RashiNumber)
			effect := gurujiRoleEffect(gurujiRole(r.Chart.Lagna.RashiNumber, p.Name), status)
			direction := effect.Direction
			label := gurujiSignLabel(status)
			text += " " + label.Hindi + ": " + effect.Explanation.Hindi
			en += " " + label.English + ": " + effect.Explanation.English
			appendSynthesisEvidence(topic, fmt.Sprintf("varga-%d", v.Factor), p.Name, p.House, direction, text, en)
		}
	}
	for _, p := range r.Details.Chalit.Planets {
		if p.Name == primary && p.House != p.NatalHouse {
			appendSynthesisEvidence(topic, "chalit", p.Name, p.House, "context", fmt.Sprintf("भाव-चलित में %s भाव %d में हैं, जबकि D1 में भाव %d में हैं। दोनों पद्धतियों को मिलाकर एक ही स्थान नहीं माना गया।", synthesisHindiPlanets[p.Name], p.House, p.NatalHouse), fmt.Sprintf("Chalit places %s in house %d versus D1 house %d; these house systems are kept distinct.", p.Name, p.House, p.NatalHouse))
		}
	}
	sign := (r.Chart.Lagna.RashiNumber + spec.Primary - 2) % 12
	if len(r.Details.Ashtakavarga.Sarva) == 12 {
		appendSynthesisEvidence(topic, "ashtakavarga", primary, spec.Primary, "context", fmt.Sprintf("मुख्य भाव की राशि में सर्वाष्टकवर्ग के %d बिंदु हैं; इन्हें प्रतिशत संभावना नहीं बनाया गया है।", r.Details.Ashtakavarga.Sarva[sign]), fmt.Sprintf("The main house's sign has %d Sarvashtakavarga bindus; these are not converted into outcome probabilities.", r.Details.Ashtakavarga.Sarva[sign]))
	}
}

// addSynthesisYogas includes matched configurations and explicit Mangal cancellation, with no negative inference from an absent yoga.
func addSynthesisYogas(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic, relevant map[string]bool) {
	for _, p := range r.Details.Analysis.Patterns {
		if !p.Matches {
			continue
		}
		connected := false
		for _, name := range p.Planets {
			connected = connected || relevant[name]
		}
		if connected {
			appendSynthesisEvidence(topic, "yoga-"+p.Code, "", 0, "context", p.Name+" की मूल ग्रहस्थिति मिलती है; उसका पूरा फल ग्रहबल और दशा से सीमित रहता है।", p.Name+" has its base configuration; full effects remain qualified by strength and dasha.")
		}
	}
	if r.Details.YogaChecks != nil && r.Details.YogaChecks.Rajyog.Matches {
		for _, pair := range r.Details.YogaChecks.Rajyog.Pairs {
			if len(pair.Links) == 0 {
				continue
			}
			if relevant[pair.KendraLord] || relevant[pair.TrikonaLord] {
				appendSynthesisEvidence(topic, "rajyog", pair.KendraLord, pair.KendraHouse, "context", "संबंधित ग्रहों का केंद्र–त्रिकोण संबंध मिला है; इससे निश्चित पद या धन की घोषणा नहीं की गई है।", "A connected Kendra–Trikona relationship is present; it does not promise a specific position or wealth.")
			}
		}
	}
	if spec.ID == "relationships" || spec.ID == "general" {
		m := r.Details.Mangal
		if m.BaseCount > 0 {
			hi, en := "जाँचे गए मंगलदोष संकेत दोषभंग के बाद शेष हैं; इससे वैवाहिक परेशानी निश्चित नहीं होती।", "Checked Mangal indications remain after cancellation assessment; marital problems are not certain."
			if m.ActiveCount == 0 {
				hi, en = "मंगलदोष की मूल स्थिति है, पर जाँचे गए नियमों से दोषभंग होता है; इसे सक्रिय बाधा नहीं गिना गया है।", "A base Mangal placement is cancelled by the checked rules and is not counted as an active obstacle."
			}
			appendSynthesisEvidence(topic, "mangal", "Mars", 0, "context", hi, en)
		}
	}
}

// finalizeSynthesisTopic composes qualified present/future narratives and traceable remedy relevance; neutral or supportive charts never receive an invented mandatory obstacle.
func finalizeSynthesisTopic(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic, relevant map[string]bool) {
	support, attention := []FaladeshEvidence{}, []FaladeshEvidence{}
	for _, e := range topic.Evidence {
		if e.Direction == "support" {
			support = append(support, e)
		}
		if e.Direction == "attention" {
			attention = append(attention, e)
		}
	}
	copy := synthesisCopy[spec.ID]
	topic.Assessment = "context"
	topic.Conclusion = copy.Context
	topic.Potential = copy.Opportunity
	if len(support) > 0 {
		topic.Assessment = "supportive"
		topic.Conclusion = copy.Support
	}
	if len(attention) > 0 {
		topic.Assessment = "attention"
		topic.Conclusion = copy.Attention
		topic.Potential = copy.Caution
	}
	if len(support) > 0 && len(attention) > 0 {
		topic.Assessment = "mixed"
		topic.Conclusion = copy.Mixed
		topic.Potential = GurujiWording{copy.Opportunity.Hindi + " " + copy.Caution.Hindi, copy.Opportunity.English + " " + copy.Caution.English}
	}
	for _, group := range []struct {
		values []FaladeshEvidence
		tone   string
	}{{attention, "attention"}, {support, "support"}} {
		if len(group.values) == 0 {
			continue
		}
		primary := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
		// Explain the subject's principal lord before secondary owners or significators, preserving order for ties.
		sort.SliceStable(group.values, func(i, j int) bool { return group.values[i].Planet == primary && group.values[j].Planet != primary })
		selected := group.values[:min(2, len(group.values))]
		hi, en, ids := []string{}, []string{}, []string{}
		for _, e := range selected {
			cause := synthesisCause(r, spec, e)
			hi = append(hi, cause.Hindi)
			en = append(en, cause.English)
			ids = append(ids, e.ID)
		}
		effect := synthesisEffect(spec.ID, group.tone)
		topic.MainPoints = append(topic.MainPoints, FaladeshPoint{group.tone, GurujiWording{strings.Join(hi, " ") + " " + effect.Hindi, strings.Join(en, " ") + " " + effect.English}, ids})
	}
	if len(topic.MainPoints) == 0 {
		primary := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
		if spec.ID == "dasha" && r.Dasha != nil {
			primary = r.Dasha.CurrentLord
		}
		for _, e := range topic.Evidence {
			if e.Kind == "placement" && e.Planet == primary {
				topic.MainPoints = append(topic.MainPoints, FaladeshPoint{"context", e.Text, []string{e.ID}})
				break
			}
		}
	}
	topic.Present = GurujiWording{"इस विषय के लिए वर्तमान दशा का स्पष्ट संबंध नहीं मिला है।", "No clear connection to the running period was found for this subject."}
	if r.Dasha != nil {
		at := r.Dasha.AsOf
		major, sub := "", ""
		for _, m := range r.Dasha.Periods {
			if !at.Before(m.Start) && at.Before(m.End) && m.Lord == r.Dasha.CurrentLord {
				major = m.Lord
				for _, s := range m.Antardashas {
					if !at.Before(s.Start) && at.Before(s.End) && s.Lord == r.Dasha.CurrentSubLord {
						sub = s.Lord
					}
				}
			}
		}
		if major != "" && sub != "" {
			// Search all retained major periods so the next related interval is
			// not lost at a Mahadasha boundary; reject malformed child ranges.
			for _, m := range r.Dasha.Periods {
				for _, s := range m.Antardashas {
					if !s.Start.Before(m.Start) && !s.End.After(m.End) && s.End.After(s.Start) && s.Start.After(at) && s.Start.Before(at.AddDate(1, 0, 0)) && relevant[s.Lord] && (topic.NextPeriod == nil || s.Start.Before(topic.NextPeriod.Start)) {
						topic.NextPeriod = &FaladeshWindow{s.Lord, s.Start, s.End, GurujiWording{"अगली संबंधित अंतर्दशा में इस विषय के निर्णयों पर ध्यान देना उपयोगी रहेगा। यह घटना की निश्चित तारीख नहीं है।", "The next related Antardasha is a useful review window for this subject, not a certain event date."}}
					}
				}
			}
			links := []string{}
			if relevant[major] {
				links = append(links, major)
			}
			if sub != major && relevant[sub] {
				links = append(links, sub)
			}
			if len(links) > 0 {
				his := []string{}
				for _, name := range links {
					his = append(his, synthesisHindiPlanets[name])
				}
				loc, err := time.LoadLocation(r.Place.Zone)
				if err != nil {
					loc = time.UTC
				}
				date := at.In(loc).Format("02 Jan 2006")
				topic.PeriodRelevant = true
				topic.Present = GurujiWording{fmt.Sprintf("%s पर %s महादशा और %s अंतर्दशा चल रही है। %s का संबंध %s से है।", date, synthesisHindiPlanets[major], synthesisHindiPlanets[sub], strings.Join(his, " और "), synthesisSubject(spec.ID).Hindi), fmt.Sprintf("At the report reference %s, the period is %s / %s. %s connects with %s.", date, major, sub, strings.Join(links, " and "), synthesisSubject(spec.ID).English)}
			}
		}
	}
	if r.Transits != nil {
		for _, p := range r.Transits.Planets {
			if relevant[p.Name] && (p.Name == "Saturn" || p.Name == "Jupiter") {
				appendSynthesisEvidence(topic, "transit", p.Name, p.House, "context", fmt.Sprintf("%s के गोचर-संदर्भ पर %s चंद्रमा से %dवें स्थान में हैं। वेध/पूर्ण गोचर-फल जाँचे बिना इसे शुभ या अशुभ घटना नहीं बनाया गया।", r.Transits.AsOf.Format("02 Jan 2006"), synthesisHindiPlanets[p.Name], p.FromMoon), fmt.Sprintf("At the transit snapshot %s, %s is %d signs from the natal Moon. No favourable/adverse event is inferred without full transit/Vedha assessment.", r.Transits.AsOf.Format("02 Jan 2006"), p.Name, p.FromMoon))
			}
		}
	}
	// Give one practical next step; the conclusion already states the favourable and demanding possibilities.
	topic.Potential = copy.Action
}

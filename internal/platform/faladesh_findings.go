package platform

import (
	"astroapp/backend/internal/swisseph"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed data/faladesh-findings.json
var faladeshFindingJSON []byte

type FaladeshFinding struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	Scope         string        `json:"scope"`
	Tone          string        `json:"tone"`
	Title         GurujiWording `json:"title"`
	Text          GurujiWording `json:"text"`
	Basis         GurujiWording `json:"basis"`
	Qualification GurujiWording `json:"qualification"`
	EvidenceIDs   []string      `json:"evidence_ids"`
	SourceIDs     []string      `json:"source_ids"`
}

type faladeshFindingLibrary struct {
	Version    string `json:"version"`
	LordHouses map[string][]struct {
		House int           `json:"house"`
		Text  GurujiWording `json:"text"`
	} `json:"lord_houses"`
	OccupantThemes map[string]map[string]GurujiWording `json:"occupant_themes"`
}

var findingLibrary = loadFaladeshFindings()
var findingSigns = []GurujiWording{{"मेष", "Aries"}, {"वृषभ", "Taurus"}, {"मिथुन", "Gemini"}, {"कर्क", "Cancer"}, {"सिंह", "Leo"}, {"कन्या", "Virgo"}, {"तुला", "Libra"}, {"वृश्चिक", "Scorpio"}, {"धनु", "Sagittarius"}, {"मकर", "Capricorn"}, {"कुम्भ", "Aquarius"}, {"मीन", "Pisces"}}

// loadFaladeshFindings validates all seven subjects, twelve destination houses and nine occupant themes before serving any detailed reading.
func loadFaladeshFindings() faladeshFindingLibrary {
	var out faladeshFindingLibrary
	if json.Unmarshal(faladeshFindingJSON, &out) != nil || out.Version == "" {
		panic("invalid Faladesh finding library")
	}
	for _, category := range []string{"personality", "education", "career", "finance", "relationships", "family", "wellbeing"} {
		rows := out.LordHouses[category]
		if len(rows) != 12 || len(out.OccupantThemes[category]) != 9 {
			panic("incomplete Faladesh finding coverage")
		}
		for i, row := range rows {
			if row.House != i+1 || row.Text.Hindi == "" || row.Text.English == "" {
				panic("invalid Faladesh house finding")
			}
		}
		for _, theme := range out.OccupantThemes[category] {
			if theme.Hindi == "" || theme.English == "" {
				panic("incomplete Faladesh occupant theme")
			}
		}
	}
	return out
}

// findingPlanet resolves a supplied D1 position without computing, moving or substituting a Chalit planet.
func findingPlanet(r *Report, name string) *swisseph.Graha {
	for i := range r.Chart.Planets {
		if r.Chart.Planets[i].Name == name {
			return &r.Chart.Planets[i]
		}
	}
	return nil
}

// findingEvidenceID resolves an actual retained fact; missing facts never receive fabricated provenance IDs.
func findingEvidenceID(topic *FaladeshTopic, kind, planet string) string {
	for _, e := range topic.Evidence {
		if e.Kind == kind && e.Planet == planet {
			return e.ID
		}
	}
	return ""
}

// appendFaladeshFinding keeps a bounded, deduplicated narrative only when every stated cause belongs to this topic's evidence.
func appendFaladeshFinding(topic *FaladeshTopic, f FaladeshFinding) {
	if len(topic.Findings) >= 5 || f.Text.Hindi == "" || f.Text.English == "" || len(f.EvidenceIDs) == 0 {
		return
	}
	for _, id := range f.EvidenceIDs {
		found := false
		for _, e := range topic.Evidence {
			found = found || e.ID == id
		}
		if !found {
			return
		}
	}
	for i := range topic.Findings {
		old := &topic.Findings[i]
		if old.ID == f.ID {
			return
		}
		if old.Text != f.Text {
			continue
		}
		// Distinct subject owners in the same destination retain both causes without repeating their identical thematic paragraph.
		if old.Kind == "principal-lord" && f.Kind == "secondary-lord" {
			old.Basis.Hindi += " " + f.Basis.Hindi
			old.Basis.English += " " + f.Basis.English
			if f.Qualification.Hindi != "" && f.Qualification != old.Qualification {
				planet := ""
				for _, id := range f.EvidenceIDs {
					if strings.HasPrefix(id, "placement:") {
						planet = strings.Split(id, ":")[1]
						break
					}
				}
				old.Qualification.Hindi = strings.TrimSpace(old.Qualification.Hindi + " " + synthesisHindiPlanets[planet] + ": " + f.Qualification.Hindi)
				old.Qualification.English = strings.TrimSpace(old.Qualification.English + " " + planet + ": " + f.Qualification.English)
			}
			for _, id := range f.EvidenceIDs {
				found := false
				for _, existing := range old.EvidenceIDs {
					found = found || existing == id
				}
				if !found {
					old.EvidenceIDs = append(old.EvidenceIDs, id)
				}
			}
		}
		return
	}
	topic.Findings = append(topic.Findings, f)
}

// findingJoin formats bilingual lists without inventing a causal relationship between their items.
func findingJoin(values []GurujiWording) GurujiWording {
	hi, en := []string{}, []string{}
	for _, v := range values {
		hi = append(hi, v.Hindi)
		en = append(en, v.English)
	}
	if len(values) < 2 {
		return GurujiWording{strings.Join(hi, ""), strings.Join(en, "")}
	}
	return GurujiWording{strings.Join(hi[:len(hi)-1], ", ") + " और " + hi[len(hi)-1], strings.Join(en[:len(en)-1], ", ") + " and " + en[len(en)-1]}
}

// findingQualification keeps the board role/sign capacity and checked combustion distinct from the thematic house interpretation.
func findingQualification(r *Report, name string) GurujiWording {
	p := findingPlanet(r, name)
	if p == nil {
		return GurujiWording{}
	}
	status := gurujiSignStatus(name, p.RashiNumber)
	role := gurujiRole(r.Chart.Lagna.RashiNumber, name)
	hi, en := []string{}, []string{}
	switch {
	case role == "karak" && status.Exalted:
		hi = append(hi, "कारक ग्रह की उच्च राशि इस क्षमता को सहारा देती है।")
		en = append(en, "Its Karak role and exaltation support this capacity.")
	case role == "karak" && status.OwnSign:
		hi = append(hi, "कारक ग्रह की स्वगृही स्थिति इस क्षमता को सहारा देती है।")
		en = append(en, "Its Karak role and own-sign placement support this capacity.")
	case role == "karak" && status.Debilitated:
		hi = append(hi, "यह कारक ग्रह नीच राशि में है, इसलिए संबंधित क्षमता का उपयोग करने में अतिरिक्त प्रयास लग सकता है।")
		en = append(en, "This Karak is debilitated, so using the connected capacity may take extra effort.")
	case role == "marak":
		hi = append(hi, "लग्न के लिए यह ग्रह मारक है; इसलिए इसकी शक्ति के साथ इस भूमिका का ध्यान रखना भी जरूरी है।")
		en = append(en, "Its ascendant role is Marak, so its role needs consideration alongside its strength.")
	case role == "neutral":
		hi = append(hi, "लग्न के लिए यह ग्रह सम है। इस संबंध का फल दूसरे सहायक और बाधक संकेतों के साथ समझना होगा।")
		en = append(en, "Its ascendant role is neutral. This connection needs to be read alongside the other supporting and demanding conditions.")
	}
	for _, c := range r.Details.Analysis.Conditions {
		if c.Name == name && c.Combust != nil && *c.Combust {
			hi = append(hi, synthesisHindiPlanets[name]+" सूर्य के निकट अस्त भी हैं। इस कारण संबंधित क्षमता को व्यक्त करने में अतिरिक्त स्पष्टता और तैयारी उपयोगी रहेगी।")
			en = append(en, name+" is also within the calculated combustion limit. Extra clarity and preparation may help in expressing the connected capacity.")
		}
	}
	return GurujiWording{strings.Join(hi, " "), strings.Join(en, " ")}
}

// findingSources links editorial interpretations to the supplied house/role references without marking them as Guruji-approved quotations.
func findingSources() []string { return []string{"house-significations", "lagna-whiteboard"} }

// findingFirstSentence keeps the opening conclusion short while the complete interpretation remains in its detailed finding.
func findingFirstSentence(text GurujiWording) GurujiWording {
	return GurujiWording{strings.TrimSpace(strings.SplitAfter(text.Hindi, "।")[0]), strings.TrimSpace(strings.SplitAfter(text.English, ".")[0])}
}

// lordHouseFinding interprets a particular subject owner's actual destination, merging shared owners such as wealth and gains instead of duplicating their paragraph.
func lordHouseFinding(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic, sourceHouse int, kind string) FaladeshFinding {
	name := synthesisLord(r.Chart.Lagna.RashiNumber, sourceHouse)
	p := findingPlanet(r, name)
	if p == nil {
		return FaladeshFinding{}
	}
	owned := []string{}
	for _, house := range spec.Houses {
		if synthesisLord(r.Chart.Lagna.RashiNumber, house) == name {
			owned = append(owned, fmt.Sprint(house))
		}
	}
	label := synthesisSubject(spec.ID)
	text := findingLibrary.LordHouses[spec.ID][p.House-1].Text
	basis := GurujiWording{fmt.Sprintf("%s से जुड़े भाव %s के स्वामी %s, %s राशि में भाव %d में हैं।", label.Hindi, strings.Join(owned, ", "), synthesisHindiPlanets[name], findingSigns[p.RashiNumber-1].Hindi, p.House), fmt.Sprintf("%s rules house %s for %s and occupies %s, D1 house %d.", name, strings.Join(owned, ", "), label.English, findingSigns[p.RashiNumber-1].English, p.House)}
	if len(owned) > 1 {
		basis.English = strings.Replace(basis.English, "rules house ", "rules houses ", 1)
	}
	roleNames := map[string]GurujiWording{"karak": {"कारक", "Karak"}, "marak": {"मारक", "Marak"}, "neutral": {"सम", "Neutral"}, "unlisted": {"सूची में नहीं", "Not listed"}}
	role, status := roleNames[gurujiRole(r.Chart.Lagna.RashiNumber, name)], gurujiSignLabel(gurujiSignStatus(name, p.RashiNumber))
	basis.Hindi += " लग्न के लिए भूमिका: " + role.Hindi + "; राशि की स्थिति: " + status.Hindi + "।"
	basis.English += " Ascendant role: " + role.English + "; sign status: " + status.English + "."
	title := GurujiWording{label.Hindi + " की मुख्य दिशा", "The main pattern in " + label.English}
	if kind == "secondary-lord" {
		title = GurujiWording{label.Hindi + " का दूसरा महत्वपूर्ण संबंध", "Another important connection in " + label.English}
	}
	ids := []string{findingEvidenceID(topic, "placement", name), findingEvidenceID(topic, "dignity", name)}
	if id := findingEvidenceID(topic, "combustion", name); id != "" {
		ids = append(ids, id)
	}
	return FaladeshFinding{ID: fmt.Sprintf("lord-%s-%d-in-%d", spec.ID, sourceHouse, p.House), Kind: kind, Scope: "natal", Tone: "context", Title: title, Text: text, Basis: basis, Qualification: findingQualification(r, name), EvidenceIDs: ids, SourceIDs: findingSources()}
}

// occupantFinding interprets all occupants of the principal subject house, or a defined secondary subject house when it is empty, without importing their generic natal paragraphs.
func occupantFinding(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic) FaladeshFinding {
	house := spec.Primary
	occupants := []string{}
	for _, h := range append([]int{spec.Primary}, spec.Houses...) {
		for _, p := range r.Chart.Planets {
			if p.House == h {
				occupants = append(occupants, p.Name)
			}
		}
		if len(occupants) > 0 {
			house = h
			break
		}
	}
	if len(occupants) == 0 {
		return FaladeshFinding{}
	}
	themes, names := []GurujiWording{}, []GurujiWording{}
	ids := []string{}
	for _, name := range occupants {
		themes = append(themes, findingLibrary.OccupantThemes[spec.ID][name])
		names = append(names, GurujiWording{synthesisHindiPlanets[name], name})
		ids = append(ids, findingEvidenceID(topic, "placement", name))
	}
	joined, named, subject := findingJoin(themes), findingJoin(names), synthesisSubject(spec.ID)
	text := GurujiWording{fmt.Sprintf("%s में %s जैसे पहलू महत्वपूर्ण हो सकते हैं।", subject.Hindi, joined.Hindi), fmt.Sprintf("Your %s may place emphasis on %s.", subject.English, joined.English)}
	followup := map[string]GurujiWording{
		"personality":   {"इससे अपनी बात रखने और निर्णय लेने के तरीके में एक से अधिक जरूरतें साथ उभर सकती हैं। किसी प्रतिक्रिया को समझते समय परिस्थिति और अपनी वास्तविक जरूरत दोनों देखें।", "Different needs may influence how you express yourself and make decisions. Consider the situation alongside your actual needs when reflecting on a response."},
		"education":     {"समझने, अभ्यास करने और मार्गदर्शन लेने के तरीके को इन जरूरतों के अनुसार ढालना उपयोगी हो सकता है। पढ़ाई में किस तरीके से वास्तविक सुधार आता है, इसे अभ्यास के परिणाम से परखें।", "Adapting how you learn, practise and seek guidance may be useful. Judge a study method by the improvement it produces in actual practice."},
		"career":        {"एक ही पदनाम से अधिक, काम की वास्तविक जिम्मेदारी और उसमें आपके योगदान का तरीका महत्वपूर्ण रहेगा। भूमिका चुनते समय देखें कि इन क्षमताओं का उपयोग कहाँ होता है।", "The actual responsibilities and how you contribute may matter more than a job title alone. Assess where these capacities are used when considering a role."},
		"finance":       {"कमाई की संभावना, धन को सँभालने का तरीका और खर्च की प्राथमिकता अलग प्रश्न हैं। इनका अलग हिसाब रखने से आर्थिक स्थिति अधिक साफ समझ आएगी।", "Earning opportunities, managing money and spending priorities are separate questions. Reviewing each separately can make your financial position clearer."},
		"relationships": {"आपसी अपेक्षाएँ और उन्हें व्यक्त करने का तरीका समझना महत्वपूर्ण रहेगा। किसी एक मतभेद से पूरे संबंध का निष्कर्ष निकालने के बजाय बार-बार सामने आने वाली जरूरत पहचानें।", "Understanding expectations and how they are expressed can matter. Identify recurring needs rather than judging an entire relationship from one disagreement."},
		"family":        {"घर में भावनात्मक जरूरत और व्यावहारिक जिम्मेदारी हमेशा एक जैसी नहीं होती। किसे किस प्रकार के सहयोग की जरूरत है, यह स्पष्ट करना तालमेल में मदद कर सकता है।", "Emotional needs and practical responsibilities at home are not always the same. Clarifying what kind of support each person needs may help coordination."},
		"wellbeing":     {"व्यस्तता के बीच विश्राम, नियमित भोजन और अपनी ऊर्जा का ध्यान रखने की जरूरत को देखें। यह दिनचर्या की पढ़त है, किसी बीमारी का निदान नहीं।", "Consider rest, regular meals and energy levels alongside a busy schedule. This is a reading about routine, not a medical diagnosis."},
	}[spec.ID]
	text.Hindi += " " + followup.Hindi
	text.English += " " + followup.English
	if spec.ID == "career" && house == 10 && synthesisContainsPlanet(occupants, "Sun") && synthesisContainsPlanet(occupants, "Mercury") && synthesisContainsPlanet(occupants, "Jupiter") {
		text = GurujiWording{"आपके काम में जिम्मेदारी लेना, जानकारी समझना और उसे दूसरों के लिए उपयोगी बनाना—इन तीनों का मेल महत्वपूर्ण हो सकता है। विश्लेषण, प्रशिक्षण, प्रशासन या सलाह जैसे कार्य इस दिशा के उदाहरण हैं। केवल ग्रहों के इस मेल से किसी एक पेशे को निश्चित नहीं माना गया है।", "Your work may combine responsibility, understanding information and making it useful to others. Analysis, training, administration or advisory tasks illustrate this direction; the combination does not establish one fixed profession."}
	}
	if spec.ID == "personality" && synthesisContainsPlanet(occupants, "Moon") && synthesisContainsPlanet(occupants, "Saturn") {
		text = GurujiWording{"भावनात्मक प्रतिक्रिया और अपने ऊपर नियंत्रण रखने की जरूरत साथ रह सकती हैं। भीतर किसी बात का असर होने पर भी आप पहले उसे सँभालकर फिर व्यक्त करना चाह सकते हैं। इसे हमेशा समझ लिया जाना कठिन हो सकता है; अपनी जरूरत सीधे बताना और हर जिम्मेदारी अकेले न उठाना उपयोगी रहेगा।", "Emotional responses may coexist with a need for self-restraint. You may prefer to manage a feeling before expressing it, which other people may not always recognise. Stating your needs directly and sharing responsibilities can be useful."}
	}
	basis := GurujiWording{fmt.Sprintf("%s राशि-कुंडली के भाव %d में स्थित हैं।", named.Hindi, house), fmt.Sprintf("%s occupy natal D1 house %d.", named.English, house)}
	if len(occupants) == 1 {
		basis.English = strings.Replace(basis.English, " occupy ", " occupies ", 1)
	}
	return FaladeshFinding{ID: fmt.Sprintf("occupants-%s-%d", spec.ID, house), Kind: "occupants", Scope: "natal", Tone: "context", Title: GurujiWording{subject.Hindi + " में ग्रहों का प्रभाव", "Planetary influences in " + subject.English}, Text: text, Basis: basis, EvidenceIDs: ids, SourceIDs: findingSources()}
}

// conditionFinding explains one most-relevant calculated capacity qualification, including a combust main-house occupant instead of ignoring it as generic context.
func conditionFinding(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic) FaladeshFinding {
	// A calculated divisional conflict qualifies D1 support; it is not concealed by the principal lord's own-sign label.
	for _, e := range topic.Evidence {
		if e.Direction != "attention" || !strings.HasPrefix(e.Kind, "varga-") {
			continue
		}
		return FaladeshFinding{ID: "capacity-" + spec.ID + "-" + e.Kind, Kind: "capacity", Scope: "natal", Tone: "attention", Title: GurujiWording{"सहायक संकेत के साथ एक सीमा भी देखें", "A qualification alongside the supporting indications"}, Text: GurujiWording{"राशि-कुंडली का सहारा होने पर भी संबंधित वर्ग-कुंडली में ग्रह की स्थिति कमजोर है। इसलिए इस विषय की क्षमता को परिणाम में बदलने के लिए तैयारी और परिस्थिति का ध्यान जरूरी रहेगा। वर्ग-कुंडली की इस स्थिति से अकेले असफलता का निष्कर्ष नहीं निकलता।", "Even with support in D1, the connected divisional placement is weaker. Preparation and circumstances still matter in turning capacity into results. The divisional condition alone does not establish failure."}, Basis: e.Text, EvidenceIDs: []string{e.ID}, SourceIDs: []string{"lagna-whiteboard"}}
	}
	primary := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
	ordered := []string{primary}
	for _, p := range r.Chart.Planets {
		if p.House == spec.Primary && p.Name != primary {
			ordered = append(ordered, p.Name)
		}
	}
	for _, h := range spec.Houses {
		if lord := synthesisLord(r.Chart.Lagna.RashiNumber, h); lord != primary {
			ordered = append(ordered, lord)
		}
	}
	ordered = append(ordered, spec.Karakas...)
	// Conditions that need care are explained before a secondary supporting capacity; roles are never converted to outcome scores.
	for _, want := range []string{"attention", "support"} {
		for _, name := range ordered {
			// House-owner qualifications already explained in this reading do not become duplicate capacity findings.
			explained := name == primary
			for _, f := range topic.Findings {
				if f.Kind != "principal-lord" && f.Kind != "secondary-lord" {
					continue
				}
				for _, id := range f.EvidenceIDs {
					explained = explained || strings.HasPrefix(id, "placement:"+name+":")
				}
			}
			if explained {
				continue
			}
			p := findingPlanet(r, name)
			if p == nil {
				continue
			}
			status := gurujiSignStatus(name, p.RashiNumber)
			effect := gurujiRoleEffect(gurujiRole(r.Chart.Lagna.RashiNumber, name), status)
			kind, direction := "dignity", effect.Direction
			for _, c := range r.Details.Analysis.Conditions {
				if c.Name == name && c.Combust != nil && *c.Combust {
					kind, direction = "combustion", "attention"
				}
			}
			if direction != want {
				continue
			}
			id := findingEvidenceID(topic, kind, name)
			if id == "" {
				continue
			}
			text := effect.Explanation
			title := GurujiWording{"इस क्षमता को कहाँ सहारा मिलता है", "Where this capacity has support"}
			if want == "attention" {
				title = GurujiWording{"किस बात में अतिरिक्त सावधानी उपयोगी है", "Where extra care may be useful"}
			}
			if kind == "combustion" {
				subject := synthesisSubject(spec.ID)
				text = GurujiWording{subject.Hindi + " में अपनी तैयारी और बात साफ रखने पर अतिरिक्त ध्यान उपयोगी रह सकता है। संबंधित क्षमता को व्यक्त करने में कठिनाई आए तो पहले देखें कि अपनी जरूरत या जिम्मेदारी कहाँ स्पष्ट नहीं हो पा रही।", "Extra preparation and clear expression may be useful in " + subject.English + ". If expressing the connected capacity is difficult, first review where needs or responsibilities are unclear."}
			}
			basis := ""
			basisEN := ""
			for _, e := range topic.Evidence {
				if e.ID == id {
					basis = e.Text.Hindi
					basisEN = e.Text.English
				}
			}
			return FaladeshFinding{ID: "capacity-" + spec.ID + "-" + name + "-" + kind, Kind: "capacity", Scope: "natal", Tone: want, Title: title, Text: text, Basis: GurujiWording{basis, basisEN}, EvidenceIDs: []string{id}, SourceIDs: []string{"lagna-whiteboard", "bhav-upay"}}
		}
	}
	return FaladeshFinding{}
}

// periodLordFinding explains the actual running lord through its subject ownership and natal placement, without assigning those themes to verified events.
func periodLordFinding(r *Report, topic *FaladeshTopic, lord string, sub bool) FaladeshFinding {
	major, minor := findingActivePeriods(r)
	if major == nil || minor == nil {
		return FaladeshFinding{}
	}
	period := major
	label := GurujiWording{"महादशा", "Mahadasha"}
	if sub {
		period = minor
		label = GurujiWording{"अंतर्दशा", "Antardasha"}
	}
	if period.Lord != lord {
		return FaladeshFinding{}
	}
	p := findingPlanet(r, lord)
	if p == nil {
		return FaladeshFinding{}
	}
	parts := []GurujiWording{}
	owned := []string{}
	for h := 1; h <= 12; h++ {
		if synthesisLord(r.Chart.Lagna.RashiNumber, h) == lord {
			owned = append(owned, fmt.Sprint(h))
		}
	}
	for _, spec := range synthesisSpecs {
		if spec.ID == "general" || spec.ID == "dasha" || spec.ID == "wellbeing" {
			continue
		}
		if synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary) == lord {
			parts = append(parts, findingLibrary.LordHouses[spec.ID][p.House-1].Text)
			if len(parts) == 2 {
				break
			}
		}
	}
	if len(parts) == 0 {
		for _, ref := range r.GurujiFaladesh.HouseReferences {
			if ref.House == p.House {
				parts = append(parts, GurujiWording{"इस ग्रह की जन्मस्थिति " + ref.Meaning.Hindi + " से जुड़े विषयों का संदर्भ देती है। इन विषयों को वास्तविक परिस्थिति के साथ समझना उपयोगी रहेगा।", "This planet's natal house context concerns " + ref.Meaning.English + ". Review these themes alongside your actual circumstances."})
				break
			}
		}
	}
	text := GurujiWording{}
	for _, part := range parts {
		text.Hindi += part.Hindi + " "
		text.English += part.English + " "
	}
	text.Hindi = strings.TrimSpace(text.Hindi)
	text.English = strings.TrimSpace(text.English)
	basis := GurujiWording{fmt.Sprintf("चल रही %s %s के ग्रह जन्म-कुंडली में भाव %d, %s राशि में हैं। स्वामित्व वाले भाव: %s।", synthesisHindiPlanets[lord], label.Hindi, p.House, findingSigns[p.RashiNumber-1].Hindi, strings.Join(owned, ", ")), fmt.Sprintf("The running %s %s lord occupies natal D1 house %d, %s. Owned houses: %s.", lord, label.English, p.House, findingSigns[p.RashiNumber-1].English, strings.Join(owned, ", "))}
	if len(owned) == 0 {
		basis.Hindi = strings.Replace(basis.Hindi, "स्वामित्व वाले भाव: ।", "इस पद्धति में कोई राशि-स्वामित्व नहीं दिया गया है।", 1)
		basis.English = strings.Replace(basis.English, "Owned houses: .", "No sign ownership is assigned in this method.", 1)
	}
	ids := []string{findingEvidenceID(topic, "placement", lord), findingEvidenceID(topic, "period-window", major.Lord)}
	return FaladeshFinding{ID: "period-lord-" + label.English + "-" + lord, Kind: "period-lord", Scope: "period", Tone: "context", Title: GurujiWording{synthesisHindiPlanets[lord] + " " + label.Hindi + " में किन विषयों को देखें", "Subjects to review in " + lord + " " + label.English}, Text: text, Basis: basis, Qualification: findingQualification(r, lord), EvidenceIDs: ids, SourceIDs: findingSources()}
}

// findingActivePeriods requires matching names and actual start-inclusive/end-exclusive boundaries at the saved report instant.
func findingActivePeriods(r *Report) (*swisseph.DashaPeriod, *swisseph.DashaPeriod) {
	if r.Dasha == nil {
		return nil, nil
	}
	at := r.Dasha.AsOf
	for i := range r.Dasha.Periods {
		m := &r.Dasha.Periods[i]
		if m.Lord != r.Dasha.CurrentLord || at.Before(m.Start) || !at.Before(m.End) {
			continue
		}
		for j := range m.Antardashas {
			s := &m.Antardashas[j]
			if s.Lord == r.Dasha.CurrentSubLord && !at.Before(s.Start) && at.Before(s.End) {
				return m, s
			}
		}
	}
	return nil, nil
}

// periodFinding links verified current lords to exact ownership/occupancy and existing subject themes; it never assigns an event or a probability to a period.
func periodFinding(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic) FaladeshFinding {
	major, sub := findingActivePeriods(r)
	if major == nil || sub == nil {
		return FaladeshFinding{}
	}
	items := []GurujiWording{}
	connected := []GurujiWording{}
	ids := []string{}
	seen := map[string]bool{}
	for _, name := range []string{major.Lord, sub.Lord} {
		if seen[name] {
			continue
		}
		seen[name] = true
		p := findingPlanet(r, name)
		if p == nil {
			continue
		}
		owned := []string{}
		for h := 1; h <= 12; h++ {
			if synthesisLord(r.Chart.Lagna.RashiNumber, h) == name && (spec.ID == "dasha" || spec.ID == "general" || synthesisContains(spec.Houses, h)) {
				owned = append(owned, fmt.Sprint(h))
			}
		}
		if len(owned) == 0 && !synthesisContains(spec.Houses, p.House) && spec.ID != "dasha" {
			continue
		}
		item := GurujiWording{fmt.Sprintf("%s भाव %d में हैं", synthesisHindiPlanets[name], p.House), fmt.Sprintf("%s occupies house %d", name, p.House)}
		if len(owned) > 0 {
			item.Hindi += ", और संबंधित भाव " + strings.Join(owned, ", ") + " के स्वामी हैं"
			item.English += ", ruling connected house " + strings.Join(owned, ", ")
		}
		items = append(items, item)
		connected = append(connected, GurujiWording{synthesisHindiPlanets[name], name})
		if id := findingEvidenceID(topic, "placement", name); id != "" {
			ids = append(ids, id)
		}
	}
	if len(items) == 0 || len(ids) == 0 {
		return FaladeshFinding{}
	}
	joined, subject, names := findingJoin(items), synthesisSubject(spec.ID), findingJoin(connected)
	text := GurujiWording{fmt.Sprintf("चल रही %s महादशा और %s अंतर्दशा में %s का %s से सीधा भाव-संबंध है। इसलिए ऊपर बताई गई इस विषय की जरूरतें और निर्णय इस अवधि में अधिक ध्यान माँग सकते हैं।", synthesisHindiPlanets[major.Lord], synthesisHindiPlanets[sub.Lord], names.Hindi, subject.Hindi), fmt.Sprintf("During %s Mahadasha and %s Antardasha, the direct natal house connection comes through %s for %s. The needs and decisions described above may therefore deserve attention in this interval.", major.Lord, sub.Lord, names.English, subject.English)}
	if spec.ID == "dasha" {
		text = GurujiWording{fmt.Sprintf("इस समय %s महादशा के भीतर %s अंतर्दशा चल रही है। दोनों ग्रहों की जन्मस्थितियाँ एक साथ देखनी होंगी: एक ग्रह अवधि की व्यापक दिशा देता है और दूसरा उसी अवधि में जुड़े विषयों को सामने लाता है। नीचे बताए भाव-संबंध इस समय की समीक्षा के विषय हैं, जीवन की पुष्ट घटनाएँ नहीं।", synthesisHindiPlanets[major.Lord], synthesisHindiPlanets[sub.Lord]), fmt.Sprintf("The running period is %s Mahadasha with %s Antardasha. The major lord gives the wider period context while the sub lord connects its more immediate subjects. Their natal house connections provide areas to review, not verified life events.", major.Lord, sub.Lord)}
	}
	at := r.Dasha.AsOf.UTC().Format("2006-01-02T15:04:05Z")
	appendSynthesisEvidence(topic, "period-window", major.Lord, 0, "context", at+": "+major.Lord+" / "+sub.Lord, at+": "+major.Lord+" / "+sub.Lord)
	ids = append(ids, findingEvidenceID(topic, "period-window", major.Lord))
	return FaladeshFinding{ID: "period-" + spec.ID + "-" + major.Lord + "-" + sub.Lord, Kind: "period", Scope: "period", Tone: "context", Title: GurujiWording{"इस समय इन बातों पर ध्यान क्यों जाता है", "Why these subjects matter in the running period"}, Text: text, Basis: joined, EvidenceIDs: ids, SourceIDs: []string{"house-significations"}}
}

// detailedFindingAction selects a practical next step from the actual destination house rather than from an overall good/bad label.
func detailedFindingAction(spec synthesisTopicSpec, house int) GurujiWording {
	if spec.ID == "career" {
		switch house {
		case 7:
			return GurujiWording{"अपने साझा काम में ग्राहक, जिम्मेदारी और भुगतान की शर्तें स्पष्ट करें। किस तरह की भूमिका आपको उपयुक्त है, इसे अपने अनुभव के उदाहरणों से परखें।", "Clarify client expectations, responsibilities and payment terms in shared work. Use examples from your actual experience to assess which roles suit you."}
		case 11:
			return GurujiWording{"अपना काम सही लोगों तक पहुँचाने, कौशल का प्रमाण दिखाने और उसके बदले अपेक्षित लाभ स्पष्ट करने पर ध्यान दें। किसी बदलाव से पहले मौजूदा अवसर और वास्तविक जरूरत साथ देखें।", "Focus on showing evidence of your skills, reaching suitable people and clarifying the expected return for your contribution. Review actual opportunities and needs before a change."}
		case 12:
			return GurujiWording{"अपनी जिम्मेदारी, लगाए गए समय और बदले में मिलने वाले परिणाम का स्पष्ट हिसाब रखें। काम का कितना हिस्सा दिखाई नहीं दे रहा और उसे कैसे व्यक्त किया जाए, यह पहले समझें।", "Clarify responsibilities, time invested and expected outcomes. Identify any less visible contribution and how to communicate it clearly."}
		}
	}
	if spec.ID == "finance" && house == 10 {
		return GurujiWording{"अपनी आय को भूमिका, कौशल और भुगतान की शर्तों से जोड़कर देखें। कमाई के अवसर और वास्तव में उपलब्ध बचत का अलग हिसाब बनाएं; किसी बड़े आर्थिक फैसले का आधार वास्तविक जानकारी हो।", "Review income alongside your role, skills and payment terms. Keep earning opportunities separate from actual available savings and base major financial decisions on real information."}
	}
	if spec.ID == "education" && house == 10 {
		return GurujiWording{"पढ़ाई के लक्ष्य को किसी वास्तविक कौशल या परियोजना से जोड़ें। सीखी हुई बात को लागू करके देखें और जहाँ कमी मिले वहाँ शिक्षक या उपयुक्त प्रशिक्षण से मदद लें।", "Connect your study goal with a practical skill or project. Apply what you learn and seek suitable teaching or training where gaps appear."}
	}
	return synthesisCopy[spec.ID].Action
}

// buildDetailedFaladeshTopic replaces the verdict paragraph with distinct chart-linked findings while preserving the earlier calculation ledger and assessment.
func buildDetailedFaladeshTopic(r *Report, spec synthesisTopicSpec, topic *FaladeshTopic) {
	if spec.ID == "general" {
		return
	}
	topic.Findings = []FaladeshFinding{}
	if spec.ID == "dasha" {
		appendFaladeshFinding(topic, periodFinding(r, spec, topic))
		if len(topic.Findings) > 0 {
			topic.Conclusion = topic.Findings[0].Text
			major, sub := findingActivePeriods(r)
			appendFaladeshFinding(topic, periodLordFinding(r, topic, major.Lord, false))
			appendFaladeshFinding(topic, periodLordFinding(r, topic, sub.Lord, true))
		}
		return
	}
	root := lordHouseFinding(r, spec, topic, spec.Primary, "principal-lord")
	appendFaladeshFinding(topic, root)
	appendFaladeshFinding(topic, occupantFinding(r, spec, topic))
	primary := synthesisLord(r.Chart.Lagna.RashiNumber, spec.Primary)
	for _, h := range spec.Houses {
		if h != spec.Primary && synthesisLord(r.Chart.Lagna.RashiNumber, h) != primary {
			before := len(topic.Findings)
			appendFaladeshFinding(topic, lordHouseFinding(r, spec, topic, h, "secondary-lord"))
			if len(topic.Findings) > before {
				break
			}
		}
	}
	condition := conditionFinding(r, spec, topic)
	appendFaladeshFinding(topic, condition)
	appendFaladeshFinding(topic, periodFinding(r, spec, topic))
	if len(topic.Findings) > 0 {
		topic.Conclusion = root.Text
		for _, f := range topic.Findings {
			if f.Kind != "occupants" {
				continue
			}
			principalOccupants := true
			for _, id := range f.EvidenceIDs {
				for _, e := range topic.Evidence {
					if e.ID == id && e.House != spec.Primary {
						principalOccupants = false
					}
				}
			}
			if principalOccupants {
				opening, detail := findingFirstSentence(root.Text), findingFirstSentence(f.Text)
				topic.Conclusion = GurujiWording{opening.Hindi + " " + detail.Hindi, opening.English + " " + detail.English}
			}
		}
		for _, c := range r.Details.Analysis.Conditions {
			if c.Name == primary && c.Combust != nil && *c.Combust {
				topic.Conclusion.Hindi += " संबंधित क्षमता को प्रभावी ढंग से व्यक्त करने में अतिरिक्त तैयारी और स्पष्टता उपयोगी रहेगी।"
				topic.Conclusion.English += " Extra preparation and clarity may help express the connected capacity effectively."
			}
		}
		if condition.Tone == "attention" && condition.Text.Hindi != "" {
			qualified := findingFirstSentence(condition.Text)
			topic.Conclusion.Hindi += " " + qualified.Hindi
			topic.Conclusion.English += " " + qualified.English
		}
		p := findingPlanet(r, primary)
		if p != nil {
			topic.Potential = detailedFindingAction(spec, p.House)
		}
	}
}

// composeDetailedOverall builds a personal overview from independently grounded subject findings, retaining each imported cause in the General ledger.
func composeDetailedOverall(r *Report, out *FaladeshSynthesis) {
	var general *FaladeshTopic
	for i := range out.Topics {
		if out.Topics[i].Category == "general" {
			general = &out.Topics[i]
		}
	}
	if general == nil {
		return
	}
	general.Findings = []FaladeshFinding{}
	partsHI, partsEN := []string{}, []string{}
	for _, category := range []string{"personality", "career", "finance", "relationships"} {
		for _, topic := range out.Topics {
			if topic.Category != category || len(topic.Findings) == 0 {
				continue
			}
			f := topic.Findings[0]
			if category == "personality" {
				for _, candidate := range topic.Findings {
					if candidate.Kind == "occupants" && len(candidate.EvidenceIDs) > 1 {
						f = candidate
						break
					}
				}
			}
			f.ID = "overall-" + f.ID
			f.Title = map[string]GurujiWording{"personality": {"आपकी सोच और जीवन की दिशा", "Your outlook and personal direction"}, "career": {"काम में मुख्य रुझान", "The main work pattern"}, "finance": {"कमाई और आर्थिक स्थिरता", "Earning and financial stability"}, "relationships": {"निकट संबंधों की जरूरत", "Needs in close relationships"}}[category]
			for _, e := range topic.Evidence {
				found := false
				for _, old := range general.Evidence {
					found = found || old.ID == e.ID
				}
				if !found {
					general.Evidence = append(general.Evidence, e)
				}
			}
			appendFaladeshFinding(general, f)
			if category == "personality" || category == "career" {
				partsHI = append(partsHI, strings.SplitAfter(f.Text.Hindi, "।")[0])
				partsEN = append(partsEN, strings.SplitAfter(f.Text.English, ".")[0])
			}
		}
	}
	for _, spec := range synthesisSpecs {
		if spec.ID == "general" {
			appendFaladeshFinding(general, periodFinding(r, spec, general))
		}
	}
	general.Conclusion = GurujiWording{strings.Join(partsHI, " "), strings.Join(partsEN, " ")}
}

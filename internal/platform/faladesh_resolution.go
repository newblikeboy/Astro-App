package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"strings"
	"time"
)

// topicOwners identifies direct subject owners; General and Dasha deliberately
// use their own anchors rather than importing the other subjects' conclusions.
func topicOwners(r *Report, topic *FaladeshTopic) []string {
	if topic.Category == "general" {
		return []string{synthesisLord(r.Chart.Lagna.RashiNumber, 1), "Moon"}
	}
	if topic.Category == "dasha" {
		major, sub := findingActivePeriods(r)
		if major == nil || sub == nil {
			return nil
		}
		return []string{major.Lord, sub.Lord}
	}
	var names []string
	for _, s := range synthesisSpecs {
		if s.ID != topic.Category {
			continue
		}
		names = append(names, synthesisLord(r.Chart.Lagna.RashiNumber, s.Primary))
		for _, h := range s.Houses {
			n := synthesisLord(r.Chart.Lagna.RashiNumber, h)
			if !synthesisContainsPlanet(names, n) {
				names = append(names, n)
			}
		}
	}
	return names
}

// resolveEvidence preserves opposing causes instead of cancelling them with a
// made-up probability score. Context alone produces no favourable/adverse claim.
func resolveEvidence(topic *FaladeshTopic, factor, rule string, ids []string) {
	support, care := false, false
	for _, e := range topic.Evidence {
		if !synthesisContainsPlanet(ids, e.ID) {
			continue
		}
		support = support || e.Direction == "support"
		care = care || e.Direction == "attention"
	}
	direction := "context"
	if support {
		direction = "support"
	}
	if care {
		direction = "attention"
	}
	if care && support {
		direction = "mixed"
	}
	topic.Resolution = append(topic.Resolution, ReadingResolution{factor, direction, ids, rule})
}

// readingActorCapacity qualifies an aspector or dispositor by its own computed
// strength and obstacles. Unknown or weak capacity never becomes an invented
// favourable influence, and the immutable board role remains a separate fact.
func readingActorCapacity(r *Report, name string, cancelled map[string]bool) (bool, bool) {
	found, capable := false, false
	for _, b := range r.Details.Shadbala.Planets {
		if b.Planet == name {
			found = true
			capable = b.MeetsRequirement
		}
	}
	if !found {
		return false, false
	}
	conditionFound := false
	for _, c := range r.Details.Analysis.Conditions {
		if c.Name != name {
			continue
		}
		conditionFound = true
		if c.Combust == nil && name != "Sun" && name != "Moon" {
			return false, false
		}
		if c.Combust != nil && *c.Combust || c.Dignity == "Debilitation sign" && !cancelled[name] {
			capable = false
		}
	}
	if !conditionFound {
		return false, false
	}
	for _, war := range r.Details.Shadbala.Wars {
		if (war.First == name || war.Second == name) && (war.Status == "unresolved" || war.Winner != name) {
			capable = false
		}
	}
	return true, capable
}

// addAdvancedEvidence connects computed strength, cancellation, divisional,
// dispositor, aspect and BAV facts to explicit subject-specific resolution rules.
func addAdvancedEvidence(r *Report, topic *FaladeshTopic) {
	names := topicOwners(r, topic)
	if len(names) == 0 {
		return
	}
	primary := names[0]
	if topic.Category != "general" {
		if major, sub := findingActivePeriods(r); major != nil && sub != nil {
			for _, p := range []*swisseph.DashaPeriod{major, sub} {
				appendSynthesisEvidence(topic, "period-window", p.Lord, 0, "context", p.Start.Format(time.RFC3339)+" से "+p.End.Format(time.RFC3339), p.Start.Format(time.RFC3339)+" to "+p.End.Format(time.RFC3339))
			}
		}
	}
	if topic.Category == "general" {
		filtered := []FaladeshEvidence{}
		for _, e := range topic.Evidence {
			if (synthesisContainsPlanet(names, e.Planet) || e.Kind == "aspect" && e.House == 1) && !strings.HasPrefix(e.Kind, "period") && !strings.HasPrefix(e.Kind, "transit") && !strings.HasPrefix(e.Kind, "yoga") && e.Kind != "rajyog" {
				filtered = append(filtered, e)
			}
		}
		topic.Evidence = filtered
		topic.RelatedPlanets = append([]string{}, names...)
	}
	cancelled := map[string]bool{}
	for _, n := range r.Details.Qualifications.Neechabhanga {
		cancelled[n.Planet] = n.Cancelled
		if !n.Debilitated || !synthesisContainsPlanet(names, n.Planet) {
			continue
		}
		direction := "attention"
		if n.Cancelled {
			direction = "context"
		}
		appendSynthesisEvidence(topic, "neechabhanga", n.Planet, 0, direction, fmt.Sprintf("%s: नीच राशि बनी रहती है; जाँचे गए नीचभंग नियम: %t।", synthesisHindiPlanets[n.Planet], n.Cancelled), fmt.Sprintf("%s remains debilitated by sign; checked cancellation: %t.", n.Planet, n.Cancelled))
	}
	environmentActors, foundationActors := []string{}, []string{}
	for i := range topic.Evidence {
		e := &topic.Evidence[i]
		if e.Kind == "dignity" && cancelled[e.Planet] && e.Direction == "attention" {
			e.Direction = "context"
		}
		if e.Kind == "aspect" || strings.HasPrefix(e.Kind, "conjunction-") {
			if !synthesisContainsPlanet(environmentActors, e.Planet) {
				environmentActors = append(environmentActors, e.Planet)
			}
			role := gurujiRole(r.Chart.Lagna.RashiNumber, e.Planet)
			known, capable := readingActorCapacity(r, e.Planet, cancelled)
			if e.Planet != "Rahu" && e.Planet != "Ketu" {
				e.Direction = "context"
			}
			if known && capable && role == "karak" {
				e.Direction = "support"
			} else if known && capable && role == "marak" {
				e.Direction = "attention"
			}
		}
		if e.Kind == "dispositor" {
			if !synthesisContainsPlanet(foundationActors, e.Planet) {
				foundationActors = append(foundationActors, e.Planet)
			}
			known, capable := readingActorCapacity(r, e.Planet, cancelled)
			e.Direction = "context"
			for _, c := range r.Details.Analysis.Conditions {
				if c.Name != e.Planet {
					continue
				}
				if known && !capable {
					e.Direction = "attention"
				} else if known && capable && gurujiRole(r.Chart.Lagna.RashiNumber, c.Name) == "karak" && (c.Dignity == "Own sign" || c.Dignity == "Exaltation sign") {
					e.Direction = "support"
				}
			}
		}
	}
	for _, b := range r.Details.Shadbala.Planets {
		if !synthesisContainsPlanet(environmentActors, b.Planet) && !synthesisContainsPlanet(foundationActors, b.Planet) {
			continue
		}
		known, capable := readingActorCapacity(r, b.Planet, cancelled)
		appendSynthesisEvidence(topic, "actor-strength", b.Planet, 0, "context", fmt.Sprintf("सहायक संबंध में ग्रह का अपना षड्बल %.4f/%g; बाधाएँ सहित क्षमता की जाँच ज्ञात: %t, सक्षम: %t।", b.Total, b.Required, known, capable), fmt.Sprintf("Supporting actor's own strength %.4f/%g; qualification known: %t, capable after obstacles: %t.", b.Total, b.Required, known, capable))
	}
	for _, b := range r.Details.Shadbala.Planets {
		if !synthesisContainsPlanet(names, b.Planet) {
			continue
		}
		direction := "context"
		if !b.MeetsRequirement {
			direction = "attention"
		} else if gurujiRole(r.Chart.Lagna.RashiNumber, b.Planet) == "karak" {
			direction = "support"
		}
		appendSynthesisEvidence(topic, "shadbala", b.Planet, 0, direction, fmt.Sprintf("%s: षड्बल %.4f विरूप, अपेक्षित %.0f; बोर्ड की भूमिका %s रहती है।", synthesisHindiPlanets[b.Planet], b.Total, b.Required, gurujiRole(r.Chart.Lagna.RashiNumber, b.Planet)), fmt.Sprintf("%s: six-component strength %.4f virupas, requirement %.0f; board role remains %s.", b.Planet, b.Total, b.Required, gurujiRole(r.Chart.Lagna.RashiNumber, b.Planet)))
	}
	for _, war := range r.Details.Shadbala.Wars {
		for _, name := range names {
			if (war.First == name || war.Second == name) && (war.Status == "unresolved" || war.Winner != name) {
				appendSynthesisEvidence(topic, "planet-war", name, 0, "attention", "ग्रहयुद्ध में पराजय या अनिर्णीत स्थिति है; इसे क्षमता की सावधानी में रखा गया है।", "Planetary-war defeat or unresolved status qualifies capacity.")
			}
		}
	}
	for _, y := range readingYogaQualifications(r.Details.Qualifications) {
		if !y.BasePresent {
			continue
		}
		connected := false
		for _, name := range names {
			connected = connected || synthesisContainsPlanet(y.Planets, name)
		}
		if !connected || topic.Category == "general" {
			continue
		}
		direction := "context"
		if y.Status == "supported" {
			direction = "support"
		}
		appendSynthesisEvidence(topic, "qualified-yoga-"+y.Code, "", 0, direction, "मूल योग मिला है; बल की जाँच के बाद स्थिति: "+y.Status+"। इसे अकेले घटना या निश्चित उपलब्धि नहीं माना गया।", "The base yoga is present; strength qualification: "+y.Status+". It does not establish an event or guaranteed achievement.")
	}
	if p := findingPlanet(r, primary); p != nil {
		for _, row := range r.Details.Ashtakavarga.Rows {
			if row.Planet != primary {
				continue
			}
			points := row.Points[p.RashiNumber-1]
			direction := "context"
			if points > 4 {
				direction = "support"
			} else if points < 4 {
				direction = "attention"
			}
			appendSynthesisEvidence(topic, "bhinna-ashtakavarga", primary, p.House, direction, fmt.Sprintf("जन्म राशि में इस ग्रह के भिन्नाष्टकवर्ग में %d बिंदु हैं; 4 सम, उससे अधिक सहायक और कम पर अतिरिक्त सावधानी का संदर्भ है।", points), fmt.Sprintf("The planet's natal-sign BAV has %d rekhas; four is neutral, above four supportive, below four calls for care (PVR ch.12).", points))
		}
	}
	resolveEvidence(topic, "capacity", "primary owner first; actual six-component strength, combustion, sign, house demand and checked cancellation; role never changes", readerEvidence(topic, []string{"shadbala", "dignity", "combustion", "neechabhanga", "planet-war", "lord-demand"}, primary))
	environmentIDs := readerEvidence(topic, []string{"aspect", "conjunction-*"})
	if len(environmentActors) > 0 {
		environmentIDs = append(environmentIDs, readerEvidence(topic, []string{"actor-strength"}, environmentActors...)...)
	}
	foundationIDs := readerEvidence(topic, []string{"dispositor", "varga-*", "bhinna-ashtakavarga"})
	if len(foundationActors) > 0 {
		foundationIDs = append(foundationIDs, readerEvidence(topic, []string{"actor-strength"}, foundationActors...)...)
	}
	resolveEvidence(topic, "environment", "whole-sign conjunctions and incoming full aspects qualified by actor's own strength/obstacles; board role retained; opposing influences remain mixed", environmentIDs)
	resolveEvidence(topic, "foundation", "native dispositor's own strength/obstacles, D9/subject varga and planet's own BAV; no probability conversion or D1/Chalit mixing", foundationIDs)
	resolveEvidence(topic, "yoga", "present configurations only, qualified by actual participants' strength and obstacles", readerEvidence(topic, []string{"qualified-yoga-*"}))
	anySupport, anyCare := false, false
	for _, v := range topic.Resolution {
		anySupport = anySupport || v.Direction == "support" || v.Direction == "mixed"
		anyCare = anyCare || v.Direction == "attention" || v.Direction == "mixed"
	}
	topic.Assessment = "context"
	if anySupport {
		topic.Assessment = "supportive"
	}
	if anyCare {
		topic.Assessment = "attention"
	}
	if anySupport && anyCare {
		topic.Assessment = "mixed"
	}
}

// addResolvedParagraphs turns independent resolved causes into complete plain
// consequences. Support and pressure remain visible instead of selecting only
// the most flattering branch of a source passage.
func addResolvedParagraphs(r *Report, topic *FaladeshTopic) {
	category := topic.Category
	if category == "dasha" {
		category = "personality"
	}
	for _, resolution := range topic.Resolution {
		if resolution.Direction == "context" {
			continue
		}
		key := map[string]string{"support": "support", "attention": "care", "mixed": "mixed"}[resolution.Direction]
		if resolution.Factor == "environment" {
			if key == "support" {
				key = "help"
			} else if key == "care" {
				key = "pressure"
			}
		}
		if resolution.Factor == "yoga" {
			continue
		} // Named yoga consequences are composed separately below.
		text := readerFactors[category][key]
		if resolution.Factor == "foundation" {
			capacityKey := "strength"
			if resolution.Direction == "attention" || resolution.Direction == "mixed" {
				capacityKey = "care"
			}
			text = readerCopy.Capacity[capacityKey][category]
		}
		addReaderParagraph(topic, "reader-"+resolution.Factor+"-"+topic.Category, "natal", text, resolution.EvidenceIDs)
	}
	yogaIDs := readerEvidence(topic, []string{"qualified-yoga-*"})
	if len(yogaIDs) > 0 {
		text := readerFactors[category]["yoga"]
		qualified := false
		for _, y := range readingYogaQualifications(r.Details.Qualifications) {
			if y.Status == "qualified" && len(readerEvidence(topic, []string{"qualified-yoga-" + y.Code})) > 0 {
				qualified = true
			}
		}
		if qualified {
			text = readerJoin(text, GurujiWording{"फिर भी इसका पूरा फायदा तुरंत या बिना रुकावट मिले, ऐसा निष्कर्ष नहीं है। तैयारी और नियमित कोशिश की जरूरत बनी रहती है।", "This does not mean the full benefit will arrive immediately or without obstacles. Preparation and steady effort remain necessary."})
		}
		addReaderParagraph(topic, "reader-yoga-"+topic.Category, "natal", text, yogaIDs)
	}
	if topic.Category == "relationships" && r.Details.Mangal.BaseCount > 0 {
		text := GurujiWording{"मतभेद के समय तुरंत प्रतिक्रिया देने से बातचीत बिगड़ सकती है। जवाब देने से पहले बात पूरी सुनना और दोनों की जिम्मेदारी साफ रखना ज्यादा उपयोगी होगा। इस संकेत से अकेले विवाह में परेशानी निश्चित नहीं होती।", "An immediate reaction during disagreement can make a conversation harder. Hear the other person out and clarify each person's responsibilities. This indication alone does not establish marital trouble."}
		if r.Details.Mangal.ActiveCount == 0 {
			text = GurujiWording{"मंगल से जुड़ी शुरुआती आपत्ति को कम करने वाली स्थितियाँ भी मिली हैं। इसलिए केवल उस एक आपत्ति के कारण संबंध को खराब मानना ठीक नहीं होगा; आपसी समझ और व्यवहार को साथ देखें।", "Conditions that cancel the initial Mangal indication are also present. That initial flag alone should not make the relationship appear adverse; mutual understanding and actual behaviour still matter."}
		}
		addReaderParagraph(topic, "reader-relationship-mangal", "natal", text, readerEvidence(topic, []string{"mangal"}))
	}
}

// addPersonalPeriodReader uses only a valid saved parent/sub period and the
// subject's own linked planets. Natal conclusions never become dated events.
func addPersonalPeriodReader(r *Report, topic *FaladeshTopic, names []string) {
	major, sub := findingActivePeriods(r)
	if major == nil || sub == nil {
		return
	}
	linked := []string{}
	for _, name := range []string{major.Lord, sub.Lord} {
		if synthesisContainsPlanet(names, name) && !synthesisContainsPlanet(linked, name) {
			linked = append(linked, name)
		}
	}
	if len(linked) == 0 {
		return
	}
	ids := readerEvidence(topic, []string{"period-window", "placement"}, linked...)
	zone, err := time.LoadLocation(r.Place.Zone)
	if err != nil {
		zone = time.UTC
	}
	reference := r.Dasha.AsOf.In(zone)
	text := GurujiWording{fmt.Sprintf("%s की इस रिपोर्ट में यह विषय चल रहे समय से भी जुड़ता है। इसलिए इससे जुड़े फैसले टालने के बजाय अपनी जरूरत और उपलब्ध तैयारी के हिसाब से एक-एक कदम लेना उपयोगी रहेगा। अभी की परेशानी को हमेशा की स्थिति मानकर फैसला न करें।", reference.Format("02-01-2006")), fmt.Sprintf("At this report's reference date, %s, the running period also connects to this subject. Take decisions in manageable steps based on your needs and preparation. Avoid treating a present difficulty as a permanent condition.", reference.Format("02 Jan 2006"))}
	if r.Transits != nil && r.Transits.Assessment != nil {
		support, care, blocked := false, false, false
		for _, t := range r.Transits.Assessment.Planets {
			if !synthesisContainsPlanet(linked, t.Planet) || t.Status == "not_assigned" {
				continue
			}
			direction := "attention"
			if t.Status == "support" {
				direction = "support"
			}
			appendSynthesisEvidence(topic, "personal-transit", t.Planet, t.FromMoon, direction, fmt.Sprintf("चंद्रमा से %d; वेध स्थिति %s; तारा %d अलग रखा गया है।", t.FromMoon, t.Status, t.Tara), fmt.Sprintf("From natal Moon %d; Vedha status %s; Tara %d remains separate.", t.FromMoon, t.Status, t.Tara))
			support = support || t.Status == "support"
			care = care || t.Status == "attention"
			blocked = blocked || t.Status == "obstructed"
			taraDirection := "attention"
			if t.TaraFavourable {
				taraDirection = "support"
			}
			appendSynthesisEvidence(topic, "transit-tara", t.Planet, 0, taraDirection, fmt.Sprintf("जन्म नक्षत्र से तारा %d; राशि-वेध से अलग आकलन।", t.Tara), fmt.Sprintf("Tara %d from the natal lunar star, separate from sign Vedha.", t.Tara))
			care = care || !t.TaraFavourable
		}
		ids = append(ids, readerEvidence(topic, []string{"personal-transit"}, linked...)...)
		ids = append(ids, readerEvidence(topic, []string{"transit-tara"}, linked...)...)
		if blocked {
			text = readerJoin(text, GurujiWording{"मौका दिखने के बाद भी बीच में दूसरी जरूरत या जिम्मेदारी आ सकती है। इसलिए किसी सहमति को पूरा हुआ काम मानने से पहले अगली जरूरी बात की पुष्टि करें।", "An apparent opportunity may encounter a competing need or responsibility. Confirm the next necessary step before treating an agreement as a completed result."})
		} else if support && care {
			text = readerJoin(text, GurujiWording{"कुछ बातों में आगे बढ़ने का मौका है, जबकि कुछ में तैयारी पूरी करना जरूरी रहेगा। केवल उत्साह या डर के आधार पर बड़ा फैसला करने से बचें।", "There is room to move forward in some respects while others still need preparation. Avoid a major decision driven only by enthusiasm or fear."})
		} else if support {
			text = readerJoin(text, GurujiWording{"तैयारी को आगे बढ़ाने और जरूरी बातचीत शुरू करने के लिए सहारा मिलता है। मौका आए तो छोटा, साफ और पूरा किया जा सकने वाला कदम चुनें।", "There is support for advancing preparation and opening useful conversations. Choose a clear, manageable step when an opportunity arises."})
		} else if care {
			text = readerJoin(text, GurujiWording{"अभी जल्दबाजी से ज्यादा फायदा तैयारी पूरी करने में है। समय, जिम्मेदारी और अपनी सीमा साफ करके आगे बढ़ें।", "Completing preparation is more useful than rushing at this stage. Clarify timing, responsibilities and your limits before proceeding."})
		}
	}
	addReaderParagraph(topic, "reader-period-"+topic.Category, "period", text, ids)
}

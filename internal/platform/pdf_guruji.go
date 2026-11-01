package platform

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var faladeshProseSentences = regexp.MustCompile(`[^.!?।]+[.!?।]*`)

// faladeshParagraphs joins the retained findings and guidance beneath one category title, preserving distinct detail while removing repeated sentences.
func (l pdfLocale) faladeshParagraphs(topic FaladeshTopic) []string {
	if len(topic.Reader) > 0 {
		out := []string{}
		for _, p := range topic.Reader {
			out = append(out, l.gurujiWording(p.Text))
		}
		return out
	}
	sources := []string{}
	if len(topic.Findings) == 0 {
		sources = append(sources, l.gurujiWording(topic.Conclusion))
	}
	for _, finding := range topic.Findings {
		text := l.gurujiWording(finding.Text) + " " + l.gurujiWording(finding.Qualification)
		if finding.Kind == "period-lord" {
			text = l.gurujiWording(finding.Title) + ": " + text
		}
		sources = append(sources, text)
	}
	sources = append(sources, l.gurujiWording(topic.Potential))
	seen := map[string]bool{}
	paragraphs := []string{}
	for _, source := range sources {
		sentences := []string{}
		for _, sentence := range faladeshProseSentences.FindAllString(source, -1) {
			sentence = strings.Join(strings.Fields(sentence), " ")
			if sentence == "" || seen[sentence] {
				continue
			}
			seen[sentence] = true
			sentences = append(sentences, sentence)
		}
		if len(sentences) > 0 {
			paragraphs = append(paragraphs, strings.Join(sentences, " "))
		}
	}
	return paragraphs
}

// readerTopicTitle shares the new reading categories across PDF chapters; the
// preserved legacy renderer continues to use its original headings.
func (l pdfLocale) readerTopicTitle(category string) string {
	return l.gurujiWording(map[string]GurujiWording{"general": {"सामान्य फलादेश", "General reading"}, "personality": {"आपका स्वभाव", "Your nature"}, "education": {"पढ़ाई और सीखना", "Learning and direction"}, "career": {"काम और करियर", "Work and career"}, "finance": {"पैसे और बचत", "Money and resources"}, "relationships": {"प्यार और साझेदारी", "Love and partnership"}, "family": {"घर और परिवार", "Home and family"}, "wellbeing": {"रोज की आदतें और आराम", "Balance and routine"}, "dasha": {"दशा का फलादेश", "Dasha reading"}}[category])
}

// combinedFaladeshSection prints only the purchased cross-factor snapshot; older purchases do not acquire a live interpretation.
func (l pdfLocale) combinedFaladeshSection(snapshot *FaladeshSynthesis, zone *time.Location) pdfSection {
	section := pdfSection{ID: "combined-faladesh", Title: l.gurujiWording(GurujiWording{"संयुक्त फलादेश और मुख्य कारण", "Combined Faladesh and main reasons"}), Intro: snapshot.Version + " · " + l.date(snapshot.AsOf, zone)}
	titles := map[string]GurujiWording{"general": {"समग्र फलादेश", "Overall reading"}, "personality": {"स्वभाव", "Personality"}, "education": {"शिक्षा", "Learning"}, "career": {"करियर", "Career"}, "finance": {"धन", "Resources"}, "relationships": {"संबंध", "Relationships"}, "family": {"परिवार", "Family"}, "wellbeing": {"दिनचर्या और संतुलन", "Routine and balance"}, "dasha": {"दशा", "Dasha"}}
	for _, topic := range snapshot.Topics {
		if snapshot.ReaderVersion == readerContractVersion {
			section.Blocks = append(section.Blocks, pdfBlock{Title: l.readerTopicTitle(topic.Category), Note: strings.Join(l.faladeshParagraphs(topic), "\n\n")})
			continue
		}
		parts := []string{l.gurujiWording(topic.Conclusion), l.gurujiWording(GurujiWording{"इस समय का संकेत: ", "Present period: "}) + l.gurujiWording(topic.Present), l.gurujiWording(GurujiWording{"आगे की संभावना: ", "Looking ahead: "}) + l.gurujiWording(topic.Potential)}
		detailed := snapshot.Version == "m4y-cross-factor-faladesh-v4" && len(topic.Findings) > 0
		concise := detailed || snapshot.Version == "m4y-cross-factor-faladesh-v2" || snapshot.Version == "m4y-cross-factor-faladesh-v3"
		if concise {
			parts = []string{l.gurujiWording(topic.Conclusion)}
		}
		if detailed {
			parts = l.faladeshParagraphs(topic)
		} else {
			for _, point := range topic.MainPoints {
				parts = append(parts, l.gurujiWording(GurujiWording{"मुख्य बात: ", "Main point: "})+l.gurujiWording(point.Text))
			}
		}
		if concise && !detailed {
			if topic.Category == "dasha" {
				parts = append(parts, l.gurujiWording(topic.Present))
			}
			parts = append(parts, l.gurujiWording(GurujiWording{"आपके लिए अगला कदम: ", "Your next step: "})+l.gurujiWording(topic.Potential))
		}
		if topic.NextPeriod != nil && (!concise || topic.Category == "dasha") {
			p := topic.NextPeriod
			parts = append(parts, l.text(p.Lord)+": "+l.date(p.Start, zone)+" - "+l.date(p.End, zone)+"\n"+l.gurujiWording(p.Text))
		}
		section.Blocks = append(section.Blocks, pdfBlock{Title: l.gurujiWording(titles[topic.Category]), Note: strings.Join(parts, "\n\n")})
	}
	limits := []string{}
	for _, limit := range snapshot.Limits {
		limits = append(limits, l.gurujiWording(limit))
	}
	section.Blocks = append(section.Blocks, pdfBlock{Title: l.gurujiWording(GurujiWording{"पढ़त की सीमा", "Reading limits"}), Note: strings.Join(limits, "\n\n")})
	return section
}

// gurujiWording selects supplied bilingual display copy without translating original source passages.
func (l pdfLocale) gurujiWording(wording GurujiWording) string {
	if l.language == "en" {
		return wording.English
	}
	return wording.Hindi
}

// gurujiSource formats the inclusive page locator from the purchased snapshot, never a live catalogue lookup.
func (l pdfLocale) gurujiSource(snapshot *GurujiFaladesh, locator GurujiLocator) string {
	name := locator.SourceID
	for _, source := range snapshot.Sources {
		if source.ID == locator.SourceID {
			name = source.Filename
			break
		}
	}
	pages := fmt.Sprint(locator.PageStart)
	if locator.PageEnd != locator.PageStart {
		pages += "-" + fmt.Sprint(locator.PageEnd)
	}
	return name + " · " + l.gurujiWording(GurujiWording{"पृष्ठ ", "pp. "}) + pages
}

// gurujiSection prints matched source references and role/strength evidence without granting personal remedy approval.
// Hindi originals retain both conditional branches; no general house-time passage is applied as a current forecast.
func (l pdfLocale) gurujiSection(snapshot *GurujiFaladesh) pdfSection {
	section := pdfSection{ID: "guruji-references", Title: l.gurujiWording(GurujiWording{"गुरुजी के संदर्भों से भावफल", "Faladesh from Guruji's references"}), Intro: l.gurujiWording(GurujiWording{"पहले संक्षिप्त अर्थ, फिर सहायक और चुनौतीपूर्ण संभावनाएँ तथा उनके मूल संदर्भ पढ़ें। लग्न की कारक/मारक भूमिका और राशि की स्थिति अलग आधार हैं; अकेले इनसे अंतिम शुभ/अशुभ निर्णय नहीं दिया गया है। सामान्य उपाय और रत्न-संबंध व्यक्तिगत निर्देश नहीं हैं।", "Begin with the concise meaning, then the helpful and challenging possibilities and their original references. Ascendant roles and sign dignity are separate factors; they do not establish a final favourable/adverse judgement on their own. General practices and gemstone associations are not personal prescriptions."})}
	roleLabels := map[string]GurujiWording{"karak": {"कारक", "Karak"}, "marak": {"मारक", "Marak"}, "neutral": {"सम", "Neutral"}, "unlisted": {"भूमिका नहीं दी गई", "Role not listed"}}
	dignities := map[string]GurujiWording{"exalted": {"उच्च राशि", "Exaltation sign"}, "debilitated": {"नीच राशि", "Debilitation sign"}, "ordinary": {"उच्च/नीच राशि नहीं", "Neither exaltation nor debilitation sign"}}
	for _, reading := range snapshot.Readings {
		meaning := l.gurujiWording(reading.Summary)
		if reading.Narrative != nil {
			meaning = l.gurujiWording(reading.Narrative.Meaning) + "\n\n" + l.gurujiWording(GurujiWording{"सहायक संभावना: ", "Helpful possibility: "}) + l.gurujiWording(reading.Narrative.Support) + "\n\n" + l.gurujiWording(GurujiWording{"ध्यान देने योग्य: ", "Point to watch: "}) + l.gurujiWording(reading.Narrative.Caution)
		}
		evidence := l.gurujiWording(GurujiWording{"व्हाइटबोर्ड की भूमिका: ", "Whiteboard role: "}) + l.gurujiWording(roleLabels[reading.Role]) + " · " + l.gurujiWording(dignities[reading.Dignity])
		if reading.SignStatus != nil {
			evidence = l.gurujiWording(GurujiWording{"बोर्ड की लग्न-भूमिका: ", "Board ascendant role: "}) + l.gurujiWording(roleLabels[reading.Role]) + "\n" + l.gurujiWording(GurujiWording{"राशि संख्या: ", "Sign number: "}) + fmt.Sprint(reading.SignNumber) + " · " + l.gurujiWording(GurujiWording{"राशि-स्थिति: ", "Sign status: "}) + l.gurujiWording(gurujiSignLabel(*reading.SignStatus)) + "\n" + l.gurujiWording(GurujiWording{"D1 राशि-कुंडली का भाव: ", "D1 whole-sign house: "}) + fmt.Sprint(reading.House)
			if reading.RoleEffect != nil {
				evidence += "\n" + l.gurujiWording(reading.RoleEffect.Explanation)
			}
		}
		section.Blocks = append(section.Blocks, pdfBlock{Title: l.gurujiWording(reading.Title), Note: meaning + "\n\n" + evidence + "\n\n" + l.gurujiSource(snapshot, reading.Source) + "\n\n" + l.gurujiWording(GurujiWording{"मूल हिंदी संदर्भ:", "Original Hindi passage:"}) + "\n" + reading.HindiExcerpt})
		section.Blocks = append(section.Blocks, pdfBlock{Title: l.text(reading.Planet) + " · " + l.gurujiWording(GurujiWording{"सामान्य पारंपरिक उपाय का संदर्भ", "General traditional remedy reference"}), Note: l.gurujiSource(snapshot, reading.Remedy.Source) + "\n\n" + reading.Remedy.HindiExcerpt})
	}
	section.Blocks = append(section.Blocks, pdfBlock{Title: l.gurujiWording(GurujiWording{"संदर्भ पद्धति", "Reference method"}), Note: snapshot.Version + "\n" + l.gurujiWording(GurujiWording{"कारक/मारक तथा उच्च/नीच: गुरुजी का दिया हुआ व्हाइटबोर्ड। राहु-केतु की दो-राशि उच्च/नीच परंपरा इसी संदर्भ की है; उनकी लग्न-विशिष्ट कारक/मारक भूमिका बोर्ड में नहीं दी गई है। ग्रह बल, दृष्टि, युति, दहन और दशा का पूर्ण संयुक्त निर्णय इस वर्गीकरण में नहीं किया गया है।", "Role/dignity source: Guruji's supplied whiteboard. Rahu/Ketu follow its two-sign dignity convention; their ascendant roles are not listed. This classification does not combine complete strength, aspects, conjunctions, combustion or Dasha assessment."})})
	return section
}

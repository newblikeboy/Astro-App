package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"strings"
)

// pdfNatalYogaBlocks prints only stored Kaal Sarp and Rajyog evidence. Older
// purchases retain their original snapshot and explicitly say these checks are absent.
func (l pdfLocale) natalYogaBlocks(checks *swisseph.NatalYogaChecks) []pdfBlock {
	if checks == nil {
		return []pdfBlock{{Title: l.text("काल सर्प दोष और राजयोग"), Note: l.text("इस सुरक्षित रिपोर्ट में काल सर्प दोष और राजयोग की जाँच उपलब्ध नहीं है। नई गणना के लिए जन्म विवरण से कुंडली दोबारा बनाएँ।")}}
	}
	sideLabels := map[string]string{"rahu_to_ketu": l.text("राहु से केतु"), "ketu_to_rahu": l.text("केतु से राहु"), "node_boundary": l.text("राहु–केतु की सीमा पर"), "none": l.text("दोनों ओर ग्रह")}
	k := checks.KaalSarp
	kaal := pdfBlock{Title: l.text("काल सर्प दोष"), Note: l.text("Conclusion") + ": " + l.kaalSarpConclusion(k) + "\n\n" + l.text("kaal-sarp.definition") + "\n\n" + l.text("Calculation details") + ": " + fmt.Sprintf(l.text("राहु %.6f° (भाव %d), केतु %.6f° (भाव %d)"), k.RahuLongitude, k.RahuHouse, k.KetuLongitude, k.KetuHouse) + " · " + sideLabels[k.Arc], Headers: []string{l.text("ग्रह"), l.text("निरयण देशांतर"), l.text("राहु से आगे का कोण"), l.text("अर्धवृत्त")}}
	for _, p := range k.Planets {
		kaal.Rows = append(kaal.Rows, []string{l.text(p.Planet), fmt.Sprintf("%.6f°", p.Longitude), fmt.Sprintf("%.6f°", p.FromRahu), sideLabels[p.Side]})
	}
	status := l.text("rajyog.absent") + " " + l.text("rajyog.absent-meaning")
	if checks.Rajyog.Matches {
		status = l.text("rajyog.present") + " " + l.text("rajyog.meaning")
	}
	raj := pdfBlock{Title: l.text("राजयोग · केंद्र–त्रिकोण संबंध"), Note: l.text("Conclusion") + ": " + status + "\n\n" + l.text("Calculation details") + l.text("। केंद्र 1/4/7/10 और त्रिकोण 1/5/9 के अलग स्वामियों में युति, परस्पर पूर्ण ग्रह-दृष्टि या राशि परिवर्तन की जाँच। अंतिम योग-बल, भंग और दशा में सक्रियता का आकलन नहीं है।"), Headers: []string{l.text("केंद्र भाव / स्वामी"), l.text("त्रिकोण भाव / स्वामी"), l.text("स्वामियों की राशियाँ"), l.text("संबंध")}}
	linkLabels := map[string]string{"conjunction": l.text("एक राशि में युति"), "mutual_aspect": l.text("परस्पर पूर्ण दृष्टि"), "sign_exchange": l.text("राशि परिवर्तन")}
	for _, p := range checks.Rajyog.Pairs {
		links := []string{}
		for _, link := range p.Links {
			links = append(links, linkLabels[link])
		}
		label := l.text("जाँचा गया संबंध नहीं मिला")
		if len(links) > 0 {
			label = strings.Join(links, " · ")
		}
		raj.Rows = append(raj.Rows, []string{fmt.Sprintf("%d / %s", p.KendraHouse, l.text(p.KendraLord)), fmt.Sprintf("%d / %s", p.TrikonaHouse, l.text(p.TrikonaLord)), l.sign(p.KendraSign) + " / " + l.sign(p.TrikonaSign), label})
	}
	yogakaraka := pdfBlock{Title: l.text("योगकारक स्वामित्व"), Headers: []string{l.text("ग्रह"), l.text("केंद्र भाव"), l.text("त्रिकोण भाव")}}
	for _, p := range checks.Rajyog.Yogakarakas {
		yogakaraka.Note += l.conclusion("yogakaraka.present", l.text(p.Planet)) + " "
		yogakaraka.Rows = append(yogakaraka.Rows, []string{l.text(p.Planet), fmt.Sprint(p.KendraHouse), fmt.Sprint(p.TrikonaHouse)})
	}
	if len(yogakaraka.Rows) == 0 {
		yogakaraka.Note = l.text("yogakaraka.absent")
	} else {
		yogakaraka.Note += l.text("yogakaraka.meaning")
	}
	types := pdfBlock{Title: l.text("Twelve Kaal Sarp types"), KeepTogether: true, Headers: []string{l.text("Kaal Sarp type"), l.text("Rahu") + " / " + l.text("House"), l.text("Ketu") + " / " + l.text("House")}}
	for house := 1; house <= 12; house++ {
		types.Rows = append(types.Rows, []string{l.text(fmt.Sprintf("kaal-sarp.type.%d", house)), fmt.Sprint(house), fmt.Sprint((house+5)%12 + 1)})
	}
	return []pdfBlock{kaal, types, raj, yogakaraka}
}

// pdfNatalYogaBlocks retains Hindi output for existing purchased-report callers.
func pdfNatalYogaBlocks(checks *swisseph.NatalYogaChecks) []pdfBlock {
	return (pdfLocale{"hi"}).natalYogaBlocks(checks)
}

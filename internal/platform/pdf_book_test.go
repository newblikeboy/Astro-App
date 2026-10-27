package platform

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

// TestPDFBookPreservesTopicsAndEvidence checks complete appendix retention, readable summaries and unmodified personal text.
func TestPDFBookPreservesTopicsAndEvidence(t *testing.T) {
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Rules = []Rule{
		{ID: "marriage-first", Category: "marriage", Topic: "Saved topic A", Hindi: "पहली पंक्ति\nदूसरी पंक्ति", English: "Saved English A", Mantra: "मूल निर्देश"},
		{ID: "education", Category: "education", Hindi: "अध्ययन का मूल पाठ"},
		{ID: "relationship-second", Category: "relationships", Hindi: "सम्बन्ध का मूल पाठ"},
		{ID: "career", Category: "career", Hindi: "कार्य का मूल पाठ"},
		{ID: "unknown", Category: "custom-topic", Hindi: "विशेष विषय का मूल पाठ"},
	}
	before, _ := json.Marshal(r)
	for _, language := range []string{"hi", "en"} {
		l := pdfLocale{language}
		sections := l.sections(r, time.UTC)
		book := l.book(r, time.UTC, sections)
		if len(book.Portrait) != 4 || len(book.Appendices) != len(sections)-1 || len(book.Topics) != 4 || book.Numerology.ID != "numerology" || len(book.Periods.Blocks) < 2 {
			t.Fatal("book lost a reading or calculation section")
		}
		if book.Topics[0].Rules[0].ID != "education" || book.Topics[1].Rules[0].ID != "career" || len(book.Topics[2].Rules) != 2 || book.Topics[2].Rules[0].ID != "marriage-first" || book.Topics[2].Rules[1].ID != "relationship-second" {
			t.Fatal("topic grouping lost or reordered personal readings")
		}
		if book.Topics[2].Rules[0].Hindi != r.Rules[0].Hindi || book.Topics[2].Rules[0].Mantra != r.Rules[0].Mantra {
			t.Fatal("personal wording was replaced")
		}
		index := 0
		for _, section := range sections {
			if section.ID == "numerology" {
				continue
			}
			if !reflect.DeepEqual(section, book.Appendices[index].pdfSection) {
				t.Fatal("appendix evidence changed")
			}
			index++
		}
		if language == "en" {
			book.Topics = nil // Stored personal words intentionally keep their supplied language.
			owned, _ := json.Marshal(book)
			if regexp.MustCompile(`[\x{0900}-\x{097f}]`).Match(owned) || strings.Contains(string(owned), "book.") || strings.Contains(string(owned), "{v") {
				t.Fatal("untranslated book wording")
			}
		}
	}
	after, _ := json.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("book preparation mutated a saved report")
	}
}

// TestPDFBookMissingAssessmentRemainsUnknown prevents a redesigned chapter from inventing results on older snapshots.
func TestPDFBookMissingAssessmentRemainsUnknown(t *testing.T) {
	l := pdfLocale{"en"}
	r := &Report{}
	book := l.book(r, time.UTC, nil)
	if len(book.Yogas) != 1 || !strings.Contains(book.Yogas[0].Conclusion, "were not examined") || len(book.AbsentYogas) != 0 || book.Periods.Intro != l.text("book.periods-unavailable") || book.Numerology.Intro != l.text("numerology.missing") {
		t.Fatal("unavailable evidence became an absence or prediction")
	}
}

// TestPDFBookPreview renders an explicitly synthetic Hindi review copy without accounts, purchases or customer records.
func TestPDFBookPreview(t *testing.T) {
	if os.Getenv("RUN_PDF_TEST") != "1" {
		t.Skip("set RUN_PDF_TEST=1 with local Chrome installed")
	}
	a := testApp(t)
	birth := testBirth()
	birth.Name = "आदित्य शर्मा (नमूना)"
	r, err := calculate(a.Config, birth, nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Rules = []Rule{
		{ID: "sample-personality", Version: 1, Category: "personality", Topic: "स्वभाव और निर्णय की शैली", Demo: true, Status: "draft", Hindi: "व्यक्तित्व का विचार करते समय लग्न, चन्द्रराशि और ग्रहों के परस्पर सम्बन्ध को साथ समझा जाता है। सहज प्रवृत्ति मन की प्रतिक्रिया का परिचय देती है; व्यवहार में उसका रूप अनुभव, शिक्षा और अभ्यास से बनता है।\n\nअपनी बात स्पष्ट रखना उपयोगी है, पर सामने वाले की बात पूरी सुनना भी उतना ही आवश्यक है। निर्णय से पहले परिस्थिति को समझने और विचारों को व्यवस्थित करने की आदत से अपनी क्षमताओं का बेहतर उपयोग किया जा सकता है।"},
		{ID: "sample-education", Version: 1, Category: "education", Topic: "अध्ययन में रुचि और निरन्तरता", Demo: true, Status: "draft", Hindi: "विद्या का सम्बन्ध केवल परीक्षा के परिणाम से नहीं, समझ और अभ्यास से भी है। जिस विषय में रुचि हो, उसमें नियमित अध्ययन और अपने शब्दों में दोहराने से ज्ञान अधिक स्पष्ट होता है।\n\nअध्ययन का छोटा, नियमित क्रम बनाना उपयोगी हो सकता है। विषय कठिन लगे तो उसे छोटे भागों में बाँटें और जहाँ समझ अधूरी हो, वहाँ शिक्षक या मार्गदर्शक से सहायता लें। ग्रहविचार को अपने वास्तविक अध्ययन और प्रयासों के सन्दर्भ में पढ़ें।"},
		{ID: "sample-career", Version: 1, Category: "career", Topic: "कर्मक्षेत्र में दिशा और जिम्मेदारी", Demo: true, Status: "draft", Hindi: "कर्मक्षेत्र का विचार करते समय योग्यता, काम की प्रकृति और जिम्मेदारी निभाने की शैली को साथ देखना उचित है। किसी अवसर का मूल्य केवल उसके आकर्षण से नहीं, अपनी तैयारी और वर्तमान परिस्थिति से भी समझा जाता है।\n\nकाम का स्पष्ट क्रम, समय पर उत्तर देना और किए गए कार्य का लेखा रखना व्यावहारिक रूप से सहायक हो सकता है। परिवर्तन पर विचार करते समय उपलब्ध अवसर, आवश्यक कौशल और अपनी जिम्मेदारियों को साथ रखें। यह नमूना परिच्छेद लेखन की शैली दिखाता है; इसे व्यक्तिगत करियर-फलादेश न मानें।", Mantra: "यह निर्देश-क्षेत्र का नमूना है। वास्तविक रिपोर्ट में यहाँ सुरक्षित स्वीकृत मंत्र और उसके निर्देश मूल शब्दों में दिए जाएँगे।", Gemstone: "यह रत्न-निर्देश के प्रस्तुतीकरण का नमूना है। इस नमूने में कोई रत्न निर्धारित नहीं किया गया है।"},
		{ID: "sample-relationships", Version: 1, Category: "relationships", Topic: "आपसी समझ और संवाद", Demo: true, Status: "draft", Hindi: "सम्बन्धों में संवाद का अर्थ केवल अपनी बात कहना नहीं, दूसरे की परिस्थिति समझना भी है। अपेक्षाएँ स्पष्ट हों और मतभेद के समय शब्द संयत रहें, तो बातचीत अधिक उपयोगी हो सकती है।\n\nविवाह के विषय में किसी एक योग या दोष से पूरा निर्णय नहीं किया जाता। दोनों व्यक्तियों की कुण्डलियों, स्वभाव और वास्तविक परिस्थितियों का विचार आवश्यक है। यहाँ दिया गया नमूना विषय की भाषा और क्रम समझाने के लिए है।"},
	}
	o := &Order{ID: "design-preview", Product: "pdf", Title: "नमूना जन्मपत्रिका", Amount: 9900, Status: "paid", Snapshot: r, ReportID: r.ID, Demo: true}
	p := ChromePDF{FontPath: a.Config.FontPath, ChromePath: os.Getenv("CHROME_PATH"), Origin: a.Config.Origin}
	dir := os.Getenv("PDF_BOOK_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	html, err := p.documentLanguage(o, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mishra4you-kundli-preview-hi.html"), html, 0600); err != nil {
		t.Fatal(err)
	}
	data, err := p.RenderLanguage(context.Background(), o, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mishra4you-kundli-preview-hi.pdf"), data, 0600); err != nil {
		t.Fatal(err)
	}
}

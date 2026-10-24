package platform

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestHindiPDFRenderer checks the real Hindi PDF renderer when its integration-test flag is enabled.
func TestHindiPDFRenderer(t *testing.T) {
	if os.Getenv("RUN_PDF_TEST") != "1" {
		t.Skip("set RUN_PDF_TEST=1 with local Chrome installed for real rendering")
	}
	a := testApp(t)
	r, e := calculate(a.Config, testBirth(), seed().Rules, a.Now())
	if e != nil {
		t.Fatal(e)
	}
	o := &Order{ID: "o_rendering_fixture", Product: "pdf", Title: "व्यक्तिगत उपाय PDF", Amount: 9900, Status: "paid", Snapshot: r, ReportID: r.ID, Demo: true, TaxMode: "unconfigured"}
	renderer := ChromePDF{FontPath: "../../assets/NotoSansDevanagari-Regular.ttf", ChromePath: os.Getenv("CHROME_PATH"), Origin: a.Config.Origin}
	// Optional self-contained HTML is a local layout-debug artifact, never a customer database write.
	if output := os.Getenv("PDF_TEST_HTML"); output != "" {
		html, err := renderer.document(o)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(output, html, 0600); err != nil {
			t.Fatal(err)
		}
	}
	b, e := renderer.Render(context.Background(), o)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(string(b), "%PDF-") || !strings.Contains(string(b), "/Type0") || !strings.Contains(string(b), "/FontFile2") {
		t.Fatal("PDF does not contain embedded TrueType fonts")
	}
	output := os.Getenv("PDF_TEST_OUTPUT")
	if output == "" {
		output = filepath.Join(t.TempDir(), "hindi-report.pdf")
	}
	if e = os.WriteFile(output, b, 0600); e != nil {
		t.Fatal(e)
	}
	t.Log("Hindi PDF generated with embedded Unicode fonts; visual verification is a separate check")
}

// TestHindiPDFLongRemedyRenderer exercises multi-page remedy prose with explicit QA text, never a real prescription.
func TestHindiPDFLongRemedyRenderer(t *testing.T) {
	if os.Getenv("RUN_PDF_TEST") != "1" {
		t.Skip("set RUN_PDF_TEST=1 with local Chrome installed")
	}
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Rules = []Rule{{ID: "qa-multipage-remedy", Version: 1, Demo: true, Status: "draft", Category: "general", Topic: "QA - लंबे विवरण का लेआउट परीक्षण", Hindi: "QA-BEGIN\n" + strings.Repeat("यह केवल लेआउट परीक्षण का पाठ है। यह व्यक्तिगत फलादेश या गुरुजी का स्वीकृत निर्देश नहीं है।\n\n", 48) + "QA-READING-END", Mantra: "QA-MANTRA-BEGIN\n" + strings.Repeat("मंत्र क्षेत्र का परीक्षण: इस नमूने में कोई मंत्र, जप संख्या या अवधि निर्धारित नहीं की गई है।\n\n", 24) + "QA-MANTRA-END", Gemstone: "QA-GEMSTONE-BEGIN\nयह रत्न क्षेत्र का लेआउट नमूना है। इसमें कोई रत्न निर्धारित नहीं किया गया है।\nQA-GEMSTONE-END"}}
	o := &Order{ID: "qa-long-remedy", Amount: 9900, Snapshot: r, ReportID: r.ID, Demo: true}
	renderer := ChromePDF{FontPath: a.Config.FontPath, ChromePath: os.Getenv("CHROME_PATH"), Origin: a.Config.Origin}
	b, err := renderer.Render(context.Background(), o)
	if err != nil {
		t.Fatal(err)
	}
	output := os.Getenv("PDF_STRESS_OUTPUT")
	if output == "" {
		output = filepath.Join(t.TempDir(), "long-remedy.pdf")
	}
	if err = os.WriteFile(output, b, 0600); err != nil {
		t.Fatal(err)
	}
}

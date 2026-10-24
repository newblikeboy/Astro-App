package platform

import (
	"bytes"
	"encoding/json"
	"html/template"
	"strings"
	"testing"
	"time"
)

// TestRemediesPDFSnapshotCoverageAndSafeText checks printable parity, full remedy text and immutable snapshots.
func TestRemediesPDFSnapshotCoverageAndSafeText(t *testing.T) {
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), seed().Rules, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	approved := a.Now()
	r.Rules = []Rule{{ID: "qa-layout-only", Version: 3, Category: "career", Topic: "केवल परीक्षण विषय", Hindi: "पहला अनुच्छेद\nदूसरा अनुच्छेद <script>unsafe()</script>", Mantra: "केवल परीक्षण मंत्र क्षेत्र\nदूसरी पंक्ति", Gemstone: "केवल परीक्षण रत्न क्षेत्र\nदूसरी पंक्ति", Status: "approved", ApprovedAt: &approved, Approver: "QA approver", Demo: true}}
	before, _ := json.Marshal(r)
	zone, _ := time.LoadLocation(r.Place.Zone)
	sections := pdfSections(r, zone)
	byID := map[string]pdfSection{}
	for _, s := range sections {
		byID[s.ID] = s
		for _, b := range s.Blocks {
			for _, row := range b.Rows {
				if len(row) != len(b.Headers) {
					t.Fatalf("%s/%s: %d cells vs %d headers", s.ID, b.Title, len(row), len(b.Headers))
				}
			}
		}
	}
	for _, id := range []string{"birth-details", "charts", "analysis", "ashtakavarga", "yogas", "dasha", "yogini", "transits", "reference"} {
		if _, ok := byID[id]; !ok {
			t.Errorf("missing snapshot section %s", id)
		}
	}
	yogaBlocks := byID["yogas"].Blocks
	if len(yogaBlocks[len(yogaBlocks)-1].Rows) != len(r.Checks) {
		t.Fatal("lost core placement checks")
	}
	// D1 is on its own page. Remaining vargas, Moon and Chalit have actual charts; boundaries have a table only.
	charts := 0
	for _, b := range byID["charts"].Blocks {
		if b.Chart != "" {
			charts++
		}
	}
	if charts != len(r.Details.Vargas)-1+2 {
		t.Fatalf("lost a chart: %d", charts)
	}
	if len(byID["dasha"].Blocks) != 1+len(r.Dasha.Periods) {
		t.Fatal("lost Vimshottari parents or active five-level path")
	}
	for i, p := range r.Dasha.Periods {
		if len(byID["dasha"].Blocks[i+1].Rows) != len(p.Antardashas) {
			t.Fatal("lost Antardasha")
		}
	}
	for i, p := range r.Details.Yogini.Periods {
		if len(byID["yogini"].Blocks[i].Rows) != len(p.Antardashas) {
			t.Fatal("lost Yogini subperiod")
		}
	}
	if len(byID["ashtakavarga"].Blocks) != 1+len(r.Details.Ashtakavarga.Rows) {
		t.Fatal("lost Prastara tables")
	}
	tmpl, err := template.New("report").Funcs(pdfTemplateFunctions(zone)).Parse(pdfTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err = tmpl.Execute(&out, map[string]any{"Order": &Order{ID: "qa-order", Snapshot: r, Demo: true}, "Report": r, "Chart": pdfChart(r), "Sections": sections, "Links": pdfLinks("https://example.invalid", r.ID)})
	if err != nil {
		t.Fatal(err)
	}
	html := out.String()
	for _, expected := range []string{"पहला अनुच्छेद\nदूसरा अनुच्छेद &lt;script&gt;", r.Rules[0].Mantra, r.Rules[0].Gemstone, "QA approver", pdfDate(approved, zone), "page-frame", "white-space:pre-line", "<title>", "https://example.invalid/account/reports/"} {
		if !strings.Contains(html, expected) {
			t.Errorf("missing text or presentation contract: %s", expected)
		}
	}
	if strings.Contains(html, "<script>unsafe()") {
		t.Fatal("untrusted interpretation escaped its text context")
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("render preparation changed a purchased snapshot")
	}
}

// TestRemediesPDFOlderSnapshotAndLinks ensures older reports remain printable and unsafe origins cannot become links.
func TestRemediesPDFOlderSnapshotAndLinks(t *testing.T) {
	a := testApp(t)
	r, err := calculateCore(a.Config, testBirth(), seed().Rules, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	sections := pdfSections(r, time.UTC)
	if !strings.Contains(sections[0].Intro, "पुराने सुरक्षित संस्करण") {
		t.Fatal("older snapshot silently implies expanded details exist")
	}
	for _, origin := range []string{"", "javascript:alert(1)", "//example.com"} {
		if pdfLinks(origin, "report") != nil {
			t.Fatal("unsafe or missing origin became a link")
		}
	}
	if pdfSign(0) != "—" || pdfSign(13) != "—" || pdfText("Sun") != "सूर्य" || pdfText("Ashlesha") != "आश्लेषा" {
		t.Fatal("incorrect Hindi chart labels")
	}
}

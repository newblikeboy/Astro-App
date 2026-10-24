package platform

import (
	"astroapp/backend/internal/swisseph"
	"bytes"
	"html/template"
	"math"
	"strings"
	"testing"
	"time"
)

// TestPDFMoneyPreservesExactPaise checks PDF money formatting without losing integer-paise precision.
func TestPDFMoneyPreservesExactPaise(t *testing.T) {
	for _, tc := range []struct {
		paise int64
		want  string
	}{
		{0, "₹0.00"}, {1, "₹0.01"}, {-1, "₹-0.01"},
		{12345, "₹123.45"}, {-12345, "₹-123.45"},
		// The former float64/100 formatter displayed this as ...09.91.
		{9007199254740990, "₹90071992547409.90"},
		{math.MaxInt64, "₹92233720368547758.07"},
		{math.MinInt64, "₹-92233720368547758.08"},
	} {
		if got := pdfMoney(tc.paise); got != tc.want {
			t.Errorf("%d paise -> %s want %s", tc.paise, got, tc.want)
		}
	}
}

// TestPDFTimeDisplayOmitsSecondsWithoutChangingInstant checks minute-only PDF clocks while preserving the underlying instant.
func TestPDFTimeDisplayOmitsSecondsWithoutChangingInstant(t *testing.T) {
	value := "1995-08-15T05:00:47.123Z"
	if got := pdfMinuteInstant(value, "Asia/Kolkata"); got != "15 Aug 1995 10:30 IST" {
		t.Fatal(got)
	}
	if got := pdfMinuteInstant(value, "UTC"); got != "15 Aug 1995 05:00 UTC" {
		t.Fatal(got)
	}
	if got := pdfClockMinutes("10:30:47"); got != "10:30" {
		t.Fatal(got)
	}
	for _, value := range []string{"bad", ""} {
		if pdfMinuteInstant(value, "UTC") != "—" {
			t.Fatal("invalid timestamp exposed")
		}
	}
}

// TestPDFTemplatePreservesSignBoundaryClassification checks that PDF formatting cannot reclassify a planet at a sign boundary.
func TestPDFTemplatePreservesSignBoundaryClassification(t *testing.T) {
	// Render the real HTML template without launching Chrome or touching a
	// Store. The in-sign decimal legitimately rounds to 30 at four places,
	// but its existing DMS snapshot retains the correct preceding sign.
	position := swisseph.ZodiacPosition{Rashi: "Mesh", RashiNumber: 1, Degree: 29.99999, DMS: "29°59′59″"}
	r := &Report{Chart: &swisseph.Kundali{
		Lagna:   position,
		Planets: []swisseph.Graha{{Name: "Sun", ZodiacPosition: position, House: 1}},
	}}
	o := &Order{Amount: 9007199254740990, Snapshot: r}
	tmpl, err := template.New("report").Funcs(pdfTemplateFunctions(time.UTC)).Parse(pdfTemplate)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err = tmpl.Execute(&output, map[string]any{"Order": o, "Report": r, "Chart": template.HTML("")}); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); strings.Contains(got, "30.0000") || strings.Count(got, position.DMS) != 2 || !strings.Contains(got, "₹90071992547409.90") {
		t.Fatal("PDF altered DMS/sign classification or lost an exact paisa")
	}
}

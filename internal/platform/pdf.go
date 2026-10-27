package platform

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"html/template"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

type PDFRenderer interface {
	Render(context.Context, *Order) ([]byte, error)
}
type ChromePDF struct{ FontPath, ChromePath, Origin string }

// Format paise without a floating-point conversion. Splitting before taking
// absolute values also supports MinInt64, whose magnitude does not fit int64.
func pdfMoney(v int64) string {
	whole, fraction := v/100, v%100
	sign := ""
	if v < 0 {
		sign = "-"
		whole, fraction = -whole, -fraction
	}
	return fmt.Sprintf("₹%s%d.%02d", sign, whole, fraction)
}

// pdfTemplateFunctions registers exact-money, date, chart and text helpers used by the paid PDF template.
func pdfTemplateFunctions(zone *time.Location) template.FuncMap {
	return (pdfLocale{"hi"}).templateFunctions(zone)
}

// Rendering helpers never alter the stored birth instant or calculation data.
func pdfClockMinutes(value string) string {
	if len(value) >= 5 {
		return value[:5]
	}
	return value
}

// pdfBirthDate displays a stored Gregorian date without applying a timezone or shifting its civil day.
func pdfBirthDate(value string) string {
	at, err := time.Parse("2006-01-02", value)
	if err != nil {
		return value
	}
	return at.Format("02-01-2006")
}

// pdfMinuteInstant formats a report instant to minute precision without changing the stored timestamp.
func pdfMinuteInstant(value, zone string) string {
	at, err := time.Parse(time.RFC3339Nano, value)
	loc, zoneErr := time.LoadLocation(zone)
	if err != nil || zoneErr != nil {
		return "—"
	}
	return at.In(loc).Format("02 Jan 2006 15:04 MST")
}

// document assembles self-contained Hindi HTML from the purchased snapshot, with no live content lookup.
func (p ChromePDF) document(o *Order) ([]byte, error) {
	return p.documentLanguage(o, "hi")
}

// documentLanguage assembles a localized document solely from the immutable purchase snapshot.
func (p ChromePDF) documentLanguage(o *Order, language string) ([]byte, error) {
	if language != "en" && language != "hi" {
		return nil, fmt.Errorf("English or Hindi PDF language required")
	}
	l := pdfLocale{language}
	if o.Snapshot == nil || o.Snapshot.Chart == nil {
		return nil, fmt.Errorf("calculation snapshot missing")
	}
	font, e := os.ReadFile(p.FontPath)
	if e != nil {
		return nil, fmt.Errorf("embedded Devanagari font unavailable")
	}
	bold, e := os.ReadFile(filepath.Join(filepath.Dir(p.FontPath), "NotoSansDevanagari-Bold.ttf"))
	if e != nil {
		return nil, fmt.Errorf("embedded bold Devanagari font unavailable")
	}
	reportZone, zoneErr := time.LoadLocation(o.Snapshot.Place.Zone)
	if zoneErr != nil {
		return nil, fmt.Errorf("snapshot timezone unavailable")
	}
	t, e := template.New("report").Funcs(l.templateFunctions(reportZone)).Parse(pdfTemplate)
	if e != nil {
		return nil, e
	}
	var html bytes.Buffer
	sections := l.sections(o.Snapshot, reportZone)
	e = t.Execute(&html, map[string]any{"Language": language, "Order": o, "Report": o.Snapshot, "Font": base64.StdEncoding.EncodeToString(font), "Bold": base64.StdEncoding.EncodeToString(bold), "Chart": l.chart(o.Snapshot), "Sections": sections, "Book": l.book(o.Snapshot, reportZone, sections), "Links": pdfLinks(p.Origin, o.ReportID)})
	if e != nil {
		return nil, e
	}
	return html.Bytes(), nil
}

// Render renders an immutable purchased report snapshot as a Hindi-capable PDF file.
func (p ChromePDF) Render(ctx context.Context, o *Order) ([]byte, error) {
	return p.RenderLanguage(ctx, o, "hi")
}

// RenderLanguage prints the selected English/Hindi view without mutating snapshots or existing files.
func (p ChromePDF) RenderLanguage(ctx context.Context, o *Order, language string) ([]byte, error) {
	html, e := p.documentLanguage(o, language)
	if e != nil {
		return nil, e
	}
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.NoFirstRun, chromedp.NoDefaultBrowserCheck, chromedp.Flag("disable-background-networking", true), chromedp.Flag("disable-dev-shm-usage", true))
	if p.ChromePath != "" {
		opts = append(opts, chromedp.ExecPath(p.ChromePath))
	}
	allocator, cancel := chromedp.NewExecAllocator(ctx, opts...)
	defer cancel()
	browser, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()
	browser, timeout := context.WithTimeout(browser, 75*time.Second)
	defer timeout()
	var result []byte
	e = chromedp.Run(browser, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
		frame, e := page.GetFrameTree().Do(ctx)
		if e != nil {
			return e
		}
		return page.SetDocumentContent(frame.Frame.ID, string(html)).Do(ctx)
	}))
	// Wait for the font-ready promise and a settled layout. Immediately after replacing
	// about:blank, fonts.status can briefly read loaded before the embedded font is requested.
	if e == nil {
		e = chromedp.Run(browser, chromedp.Evaluate(`document.fonts.ready.then(() => { document.body.getBoundingClientRect(); return true; })`, nil, func(p *runtime.EvaluateParams) *runtime.EvaluateParams { return p.WithAwaitPromise(true) }))
		if e != nil {
			return nil, fmt.Errorf("PDF font preparation: %w", e)
		}
		e = chromedp.Run(browser, chromedp.ActionFunc(func(ctx context.Context) error {
			var e error
			result, _, e = page.PrintToPDF().WithPrintBackground(true).WithPreferCSSPageSize(true).WithGenerateDocumentOutline(true).WithGenerateTaggedPDF(true).
				WithDisplayHeaderFooter(true).WithHeaderTemplate(`<div style="width:100%;margin:0 20mm;padding-bottom:5px;border-bottom:1px solid #e5d9cb;font-family:Arial,sans-serif;font-size:8px;color:#8e7566;display:flex;justify-content:space-between"><span>MISHRA 4 YOU</span><span>JANMA KUNDLI</span></div>`).
				WithFooterTemplate(`<div style="width:100%;margin:0 17mm 3mm;font-family:Arial,sans-serif;font-size:8px;color:#875132;display:flex;justify-content:space-between"><span>MISHRA 4 YOU · PERSONAL KUNDLI</span><span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`).Do(ctx)
			return e
		}))
		if e != nil {
			return nil, fmt.Errorf("PDF printing: %w", e)
		}
	}
	return result, e
}

// pdfLinks uses only the configured application origin and owned-report route; no contacts or domains are invented.
func pdfLinks(origin, reportID string) map[string]string {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "https" && u.Scheme != "http") {
		return nil
	}
	base := u.Scheme + "://" + u.Host
	return map[string]string{"Report": base + "/account/reports/" + url.PathEscape(reportID), "Consultation": base + "/account/reports/" + url.PathEscape(reportID), "Account": base + "/account"}
}

// pdfCategory keeps the approved topic grouping consistent with the website's life-topic navigation.
func pdfCategory(value string) string {
	if label, ok := map[string]string{"": "सामान्य मार्गदर्शन", "general": "सामान्य मार्गदर्शन", "personality": "व्यक्तित्व", "relationships": "संबंध", "wellbeing": "स्वास्थ्य और कुशलता", "dasha": "दशा", "career": "करियर", "finance": "धन और वित्त", "marriage": "विवाह", "health": "स्वास्थ्य", "education": "शिक्षा", "family": "परिवार", "spiritual": "आध्यात्मिक मार्गदर्शन", "debt": "ऋण"}[value]; ok {
		return label
	}
	return pdfText(value)
}

//go:embed templates/remedies.html
var pdfTemplate string

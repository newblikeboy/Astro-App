package platform

import (
	"astroapp/backend/internal/swisseph"
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

//go:embed data/matching-en.json
var matchingLocaleJSON []byte

var matchingHindi = func() map[string]string {
	var data map[string]string
	if err := json.Unmarshal(matchingLocaleJSON, &data); err != nil {
		panic("invalid bundled matching locale")
	}
	out := map[string]string{}
	for hi, en := range data {
		out[en] = hi
	}
	return out
}()

// matchingText selects a matching-report label in the requested supported language.
func matchingText(language, value string, args ...any) string {
	if language == "hi" {
		if hi, ok := matchingHindi[value]; ok {
			value = hi
		}
	}
	for i, v := range args {
		value = strings.ReplaceAll(value, fmt.Sprintf("{v%d}", i), fmt.Sprint(v))
	}
	return value
}

var matchingEnglishSigns = []string{"Aries", "Taurus", "Gemini", "Cancer", "Leo", "Virgo", "Libra", "Scorpio", "Sagittarius", "Capricorn", "Aquarius", "Pisces"}
var matchingFactorWhy = map[string]string{
	"varna":   "The selected element ordering gives {v0} of {v1} points for {v2} and {v3}.",
	"vashya":  "The {v2} / {v3} pairing receives {v0} of {v1} in the selected Vashya table.",
	"tara":    "From the groom's star: count {v2}, Tara {v4}. From the bride's star: count {v3}, Tara {v5}. Inclusive counts cycle through 1 to 9; classes 3, 5 and 7 receive 0, and each other direction receives 1.5.",
	"yoni":    "The traditional {v2} / {v3} animal-group pairing receives {v0} of {v1}.",
	"maitri":  "The natural-friendship table gives {v0} of {v1} for {v2} and {v3}. Temporary planetary friendships are not used.",
	"gana":    "Bride {v2} and groom {v3} receive {v0} of {v1} in this directional Gana table. Reversing the roles can change this factor.",
	"bhakoot": "The Moon signs form a {v2}/{v3} relationship. This base method gives 0 for 2/12, 5/9 or 6/8; otherwise 7.",
	"nadi":    "The birth-star groups are {v2} and {v3}. Same Nadi receives 0; different Nadi receives 8. No Nadi cancellation bonus is added.",
}

// matchingFactorValue formats an Ashtakoota factor value for the requested PDF language.
func matchingFactorValue(language, code, value string) string {
	if code == "varna" {
		value = map[string]string{"Brahmin": "Water", "Kshatriya": "Fire", "Vaishya": "Earth", "Shudra": "Air"}[value]
	}
	if code == "tara" {
		if i := strings.Index(value, " ("); i >= 0 {
			return matchingText(language, value[:i]) + value[i:]
		}
	}
	return matchingText(language, value)
}

// matchingExplanation selects the translated factual explanation of a matching factor.
func matchingExplanation(language string, f swisseph.MatchingFactor) string {
	var b, g any = matchingFactorValue(language, f.Code, f.BrideValue), matchingFactorValue(language, f.Code, f.GroomValue)
	if f.Code == "tara" {
		b, g = f.Evidence["bride_count"], f.Evidence["groom_count"]
	}
	if f.Code == "bhakoot" {
		b, g = f.Evidence["bride_to_groom"], f.Evidence["groom_to_bride"]
	}
	return matchingText(language, matchingFactorWhy[f.Code], f.Score, f.Max, b, g, f.Evidence["bride_class"], f.Evidence["groom_class"])
}

var matchingMangalRuleText = map[string]string{
	"mars_own_or_exalted": "Mars in its own or exalted sign",
	"mars_debilitated":    "Mars in Cancer (selected cancellation tradition)",
	"mars_retrograde":     "Retrograde Mars", "mars_combust": "Mars within 17 degrees of the Sun",
	"mars_conjunct_jupiter": "Mars and Jupiter in the same sign", "mars_conjunct_moon": "Mars and Moon in the same sign", "mars_conjunct_rahu": "Mars and Rahu in the same sign",
	"jupiter_aspects_seventh_mars": "Jupiter aspects seventh-house Mars", "sign_house_exception": "Published sign-and-house exception",
	"other_saturn_or_rahu_same_house": "Partner Saturn or Rahu in the same house number", "other_saturn_in_mangal_house": "Partner Saturn in a specified Mangal house", "other_malefic_in_upachaya": "Partner Mars, Saturn or Rahu in house 3, 6 or 11",
}
var matchingMangalStatusText = map[string]string{"no_active_flags": "No remaining flags under the selected rules", "similar_active_levels": "Similar Mangal levels", "different_active_levels": "Different Mangal levels need review", "pair_exception_found": "A traditional pair exception was found"}
var matchingBandText = map[string]string{"below_18": "Below 18 base points", "18_to_24": "18 to below 24 base points", "24_to_32": "24 to below 32 base points", "32_to_36": "32 to 36 base points"}

type MatchingPDFRenderer interface {
	RenderMatching(context.Context, *MatchingResponse, string) ([]byte, error)
}

// matchingPDF recalculates matching inputs and renders a bounded, stateless PDF download.
func (a *App) matchingPDF(w http.ResponseWriter, r *http.Request) {
	var in struct {
		MatchingRequest
		Language string `json:"language"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Language != "en" && in.Language != "hi" {
		fail(w, 400, fmt.Errorf("English or Hindi report language required."))
		return
	}
	if a.matchingPDFSlots == nil || a.MatchingPDF == nil {
		fail(w, 503, fmt.Errorf("The comparison PDF could not be prepared. Please try again."))
		return
	}
	select {
	case a.matchingPDFSlots <- struct{}{}:
		defer func() { <-a.matchingPDFSlots }()
	default:
		fail(w, 429, fmt.Errorf("Matching PDF is busy. Please try again shortly."))
		return
	}
	out, failure := a.calculateMatching(r.Context(), in.MatchingRequest)
	if r.Context().Err() != nil {
		return
	}
	if failure != nil {
		matchingError(w, failure.status, failure.person, fmt.Errorf("%s", failure.message))
		return
	}
	data, err := a.MatchingPDF.RenderMatching(r.Context(), out, in.Language)
	if r.Context().Err() != nil {
		return
	}
	if err != nil || len(data) < 5 || !bytes.HasPrefix(data, []byte("%PDF-")) {
		fail(w, 503, fmt.Errorf("The comparison PDF could not be prepared. Please try again."))
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="mishra4you-kundli-matching-%s.pdf"`, in.Language))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

// matchingChartSVG draws the matching report's chart as SVG for PDF rendering.
func matchingChartSVG(person MatchingParticipant, mode, language string) template.HTML {
	chart := *person.Chart
	chart.Planets = append([]swisseph.Graha{}, person.Chart.Planets...)
	if mode == "d9" {
		chart.Lagna.RashiNumber = person.Navamsa.LagnaSign
		chart.Planets = nil
		for _, p := range person.Navamsa.Planets {
			chart.Planets = append(chart.Planets, swisseph.Graha{Name: p.Name, House: p.House})
		}
	} else if mode == "moon" {
		chart.Lagna.RashiNumber = person.Moon.RashiNumber
		for i := range chart.Planets {
			chart.Planets[i].House = (chart.Planets[i].RashiNumber-person.Moon.RashiNumber+12)%12 + 1
		}
	}
	// Labels are application-owned constant glyphs. User names never enter SVG.
	short := map[string]string{"Sun": "Su", "Moon": "Mo", "Mars": "Ma", "Mercury": "Me", "Jupiter": "Ju", "Venus": "Ve", "Saturn": "Sa", "Rahu": "Ra", "Ketu": "Ke"}
	if language == "hi" {
		short = map[string]string{"Sun": "सू", "Moon": "चं", "Mars": "मं", "Mercury": "बु", "Jupiter": "गु", "Venus": "शु", "Saturn": "श", "Rahu": "रा", "Ketu": "के"}
	}
	centers := [][2]int{{150, 82}, {77, 30}, {28, 82}, {75, 152}, {28, 225}, {77, 275}, {150, 229}, {223, 275}, {272, 225}, {225, 152}, {272, 82}, {223, 30}}
	var s strings.Builder
	s.WriteString(`<svg class="chart" viewBox="-4 -4 308 308"><rect width="300" height="300" fill="#fffcf6" stroke="#9a684b"/><path d="M0 0L300 300M300 0L0 300M150 0L300 150L150 300L0 150Z" fill="none" stroke="#9a684b"/><g font-family="Noto" text-anchor="middle" fill="#57213d">`)
	for i, xy := range centers {
		names := []string{}
		for _, p := range chart.Planets {
			if p.House == i+1 {
				names = append(names, short[p.Name])
			}
		}
		fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="11">%d</text>`, xy[0], xy[1]-10, (chart.Lagna.RashiNumber+i-1)%12+1)
		// At most three labels per line, so conjunctions cannot spill across houses.
		for start := 0; start < len(names); start += 3 {
			end := min(start+3, len(names))
			fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="12">%s</text>`, xy[0], xy[1]+5+(start/3)*14, strings.Join(names[start:end], " · "))
		}
	}
	s.WriteString(`</g></svg>`)
	return template.HTML(s.String())
}

// RenderMatching renders the complete matching report through the configured headless browser.
func (p ChromePDF) RenderMatching(ctx context.Context, report *MatchingResponse, language string) ([]byte, error) {
	if report == nil || report.Bride.Chart == nil || report.Groom.Chart == nil || report.Bride.Navamsa == nil || report.Groom.Navamsa == nil {
		return nil, fmt.Errorf("matching chart missing")
	}
	regular, err := os.ReadFile(filepath.Join(filepath.Dir(p.FontPath), "NotoSansDevanagari.ttf"))
	if err != nil {
		return nil, fmt.Errorf("report font unavailable")
	}
	bold, err := os.ReadFile(filepath.Join(filepath.Dir(p.FontPath), "NotoSansDevanagari-Bold.ttf"))
	if err != nil {
		return nil, fmt.Errorf("report bold font unavailable")
	}
	// text looks up a localized label while constructing matching PDF content.
	text := func(s string, args ...any) string { return matchingText(language, s, args...) }
	funcs := template.FuncMap{
		"clock":   pdfClockMinutes,
		"instant": pdfMinuteInstant,
		"t":       text, "fv": func(f swisseph.MatchingFactor, value string) string {
			return matchingFactorValue(language, f.Code, value)
		},
		"why": func(f swisseph.MatchingFactor) string { return matchingExplanation(language, f) },
		"sign": func(n int) string {
			if n < 1 || n > 12 {
				return ""
			}
			return text(matchingEnglishSigns[n-1])
		},
		"reference": func(v string) string {
			return text(map[string]string{"lagna": "Lagna", "moon": "Moon", "venus": "Venus", "all": "All references", "bride": "Bride", "groom": "Groom"}[v])
		},
		"rule": func(v string) string { return text(matchingMangalRuleText[v]) },
		"level": func(v string) string {
			return text(map[string]string{"none": "No flagged references", "low": "Low Mangal level", "high": "High Mangal level"}[v])
		},
		"chart": func(p MatchingParticipant, mode string) template.HTML { return matchingChartSVG(p, mode, language) },
		"band":  func(v string) string { return text(matchingBandText[v]) }, "mstatus": func(v string) string { return text(matchingMangalStatusText[v]) },
		"fixed": func(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) },
	}
	tpl, err := template.New("matching").Funcs(funcs).Parse(matchingPDFTemplate)
	if err != nil {
		return nil, err
	}
	people := []struct {
		Role   string
		Person MatchingParticipant
		Mangal swisseph.MangalAssessment
	}{{"Bride", report.Bride, report.Mangal.Bride}, {"Groom", report.Groom, report.Mangal.Groom}}
	var html bytes.Buffer
	if err = tpl.Execute(&html, map[string]any{"Language": language, "Report": report, "People": people, "Font": base64.StdEncoding.EncodeToString(regular), "Bold": base64.StdEncoding.EncodeToString(bold), "Timestamp": report.CalculatedAt.UTC().Format("02 Jan 2006 15:04 UTC")}); err != nil {
		return nil, err
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
	err = chromedp.Run(browser, chromedp.Navigate("about:blank"), chromedp.ActionFunc(func(ctx context.Context) error {
		frame, e := page.GetFrameTree().Do(ctx)
		if e != nil {
			return e
		}
		return page.SetDocumentContent(frame.Frame.ID, html.String()).Do(ctx)
	}))
	if err == nil {
		err = chromedp.Run(browser, chromedp.Poll(`document.fonts.status === 'loaded'`, nil), chromedp.ActionFunc(func(ctx context.Context) error {
			var e error
			result, _, e = page.PrintToPDF().WithPrintBackground(true).WithPreferCSSPageSize(true).WithDisplayHeaderFooter(true).WithHeaderTemplate(`<div></div>`).WithFooterTemplate(`<div style="font-size:8px;color:#796c74;width:100%;padding:0 15mm;display:flex;justify-content:space-between"><span>Mishra 4 You</span><span><span class="pageNumber"></span> / <span class="totalPages"></span></span></div>`).Do(ctx)
			return e
		}))
	}
	return result, err
}

const matchingPDFTemplate = `<!doctype html><html lang="{{.Language}}"><head><meta charset="utf-8"><style>
@font-face{font-family:Noto;src:url(data:font/ttf;base64,{{.Font}});font-weight:100 900}@font-face{font-family:Noto;src:url(data:font/ttf;base64,{{.Bold}});font-weight:700}
@page{size:A4;margin:14mm 15mm}*{box-sizing:border-box}body{font:10.5px/1.6 Noto,sans-serif;color:#402c37;margin:0}header{display:flex;justify-content:space-between;align-items:center;border-bottom:3px solid #df9a4d;padding-bottom:12px}.brand{font-size:22px;font-weight:700;color:#57213d}.brand small{font-size:10px;display:block}h1{font-size:26px;margin:20px 0 5px;color:#57213d}h2{font-size:17px;color:#57213d;border-bottom:1px solid #e8dce1;padding-bottom:6px;margin:16px 0 8px}h3{font-size:13px;margin:9px 0;color:#57213d}p{margin:7px 0}.muted{font-size:9px;color:#73636e}.score{border:1px solid #dbc9d2;background:#faf1f5;padding:12px;border-radius:9px;break-inside:avoid}.score strong{font-size:32px;color:#57213d}.score span{font-size:18px}.people{display:grid;grid-template-columns:1fr 1fr;gap:20px}.person{min-width:0;overflow-wrap:anywhere}dl{margin:8px 0}dt{font-size:9px;color:#73636e}dd{margin:0 0 3px;font-size:10px}.ref{font-size:9px;overflow-wrap:anywhere}table{width:100%;border-collapse:collapse;font-size:10px;break-inside:avoid}th,td{text-align:left;border-bottom:1px solid #e8dce1;padding:6px;vertical-align:top}th{color:#57213d;background:#f8f1f4;font-weight:700}tr{break-inside:avoid}.factor{break-inside:avoid;border-bottom:1px solid #e8dce1;padding:8px 0}.factor h3{margin:5px 0}.factor p{font-size:10px}.note{padding:10px 13px;border-left:3px solid #df9a4d;background:#fff8ed;font-size:10px;break-inside:avoid}.newpage{break-before:page}.chart-pair{display:grid;grid-template-columns:1fr 1fr;gap:30px;break-inside:avoid}.chart{display:block;width:220px;height:220px;max-width:100%;margin:7px auto}.chart-pair h3{text-align:center}.mangal-person{break-inside:avoid}.rules{font-size:10px;padding-left:18px}.method{font-size:9px;border-top:1px solid #e8dce1;padding-top:11px;margin-top:18px;break-inside:avoid}footer{font-size:9px;color:#73636e;margin-top:15px}.context{overflow-wrap:anywhere}
</style></head><body><header><div class="brand">Mishra 4 You<small>PT. RAJA MISHRA</small></div><div>{{t "Kundli matching report"}}</div></header>
<h1>{{t "Comparison summary"}}</h1><div class="score"><strong>{{.Report.Total}}</strong><span> / 36</span><p>{{t "Base Ashtakoota"}} · {{band .Report.Summary.Band}}</p><p>{{mstatus .Report.Mangal.Status}}</p></div>
<p class="muted">{{t "A score band is a traditional reference, not a marriage verdict. Read the individual factors and Mangal findings together."}}</p>
<h2>{{t "Verified birth inputs"}}</h2><div class="people">{{range .People}}<article class="person"><h3>{{t .Role}} · {{.Person.Birth.Name}}</h3><dl><dt>{{t "Birth date"}} / {{t "Birth time"}}</dt><dd>{{.Person.Birth.Date}} · {{clock .Person.Birth.Time}}</dd><dt>{{t "Birthplace"}}</dt><dd>{{if eq $.Language "hi"}}{{.Person.Place.Hindi}}{{else}}{{.Person.Place.Name}}{{end}}</dd><dt>{{t "IANA time zone"}}</dt><dd>{{.Person.Place.Zone}}</dd><dt>{{t "Latitude / longitude"}}</dt><dd>{{.Person.Place.Lat}} / {{.Person.Place.Lon}}</dd><dt>{{t "Resolved local birth time"}}</dt><dd>{{instant .Person.Chart.BirthTime .Person.Place.Zone}}</dd><dt>{{t "Resolved UTC instant"}}</dt><dd>{{instant .Person.Chart.UTC "UTC"}}</dd><dt>{{t "Lagna"}} / {{t "Moon sign"}}</dt><dd>{{sign .Person.Lagna.RashiNumber}} / {{sign .Person.Moon.RashiNumber}}</dd><dt>{{t "Birth star"}} / {{t "Pada"}}</dt><dd>{{t .Person.Moon.Nakshatra}} / {{.Person.Moon.Pada}}</dd>{{if .Person.Dasha}}<dt>{{t "Birth Dasha balance"}}</dt><dd>{{t .Person.Dasha.BirthLord}} · {{fixed .Person.Dasha.BalanceYears}} {{t "years"}}</dd>{{end}}</dl></article>{{end}}</div>
<h2>{{t "Guna Milan"}}</h2><table><thead><tr><th>{{t "Reference"}}</th><th>{{t "Bride"}}</th><th>{{t "Groom"}}</th><th>{{t "Points"}}</th></tr></thead><tbody>{{range .Report.Factors}}<tr><td>{{t .Name}}</td><td>{{fv . .BrideValue}}</td><td>{{fv . .GroomValue}}</td><td>{{.Score}} / {{.Max}}</td></tr>{{end}}</tbody></table>
<h2 class="newpage">{{t "Why these points"}}</h2>{{range .Report.Factors}}<article class="factor"><h3>{{t .Name}} · {{.Score}} / {{.Max}}</h3><p>{{why .}}</p></article>{{end}}
<section class="method"><h3>{{t "Method reference"}}</h3><p>{{t "Lahiri sidereal zodiac"}} · Swiss {{.Report.Bride.Chart.SwissVersion}}<br>{{.Report.Method}}<br>{{.Report.Mangal.Method}}</p><p>{{t "No cancellation points are added. The Mangal method is separate from the base score."}}</p><p>{{t "The Cow–Tiger Yoni pairing is scored 0 in both directions in this version."}}</p><p>{{t "AIFAS 2021: selected six-house, three-reference method; whole-sign conjunctions and aspects."}}</p><p>{{t "Calculated placements and traditional rules; no personalised predictions are generated or approved by this report."}}</p><p>{{t "Calculation time"}}: {{.Timestamp}}</p><p class="ref">AIFAS: https://www.futuresamachar.com/download/horoscope-matching-325.pdf<br>Mars combustion: https://www.futuresamachar.com/download/astrological-assessment-of-timing-of-childbirth-487.pdf</p></section>
<h2 class="newpage">{{t "Mangal analysis"}}</h2><p class="note">{{t "The six-house convention checks 1, 2, 4, 7, 8 and 12 from Lagna, Moon and Venus. One flagged reference is low coverage; two or three are high."}}</p>
{{range .People}}<article class="mangal-person"><h3>{{t .Role}} · {{.Person.Birth.Name}}</h3><p>{{t "Raw reference level"}}: {{level .Mangal.BaseLevel}} ({{.Mangal.BaseCount}}/3)<br>{{t "After selected exceptions"}}: {{level .Mangal.Level}} ({{.Mangal.ActiveCount}}/3)</p><table><thead><tr><th>{{t "Reference"}}</th><th>{{t "House"}}</th><th>{{t "Result"}}</th></tr></thead><tbody>{{range .Mangal.References}}<tr><td>{{reference .Reference}}</td><td>{{.House}}</td><td>{{if .Cancelled}}{{t "Exception applies"}}{{else if .Flagged}}{{t "Base placement"}}{{else}}{{t "Outside the checked houses"}}{{end}}</td></tr>{{end}}</tbody></table><ul class="rules">{{range .Mangal.Rules}}{{if .Met}}<li>{{rule .Code}} · {{reference .Reference}} · {{t "Source PDF page {v0}" .SourcePage}}</li>{{end}}{{end}}</ul>{{if .Mangal.ReviewConditions}}<p class="muted">{{t "Benefic strength requires review"}}</p>{{end}}</article>{{end}}
<h3>{{t "Pair-level exceptions"}}</h3><p>{{mstatus .Report.Mangal.Status}}</p><ul class="rules">{{range .Report.Mangal.Rules}}{{if .Met}}<li>{{reference .Reference}}: {{rule .Code}}</li>{{end}}{{end}}</ul><p class="muted">{{t "Full planetary strength and context-dependent exceptions still require a professional reading. A possible benefic-strength condition is not treated as a confirmed cancellation."}}</p>
<h2 class="newpage">{{t "Birth chart comparison"}}</h2>
<div class="chart-pair">{{range .People}}<div><h3>{{t .Role}} · {{t "D1 birth chart"}}</h3>{{chart .Person "d1"}}</div>{{end}}</div><div class="chart-pair">{{range .People}}<div><h3>{{t .Role}} · {{t "Navamsha D9"}}</h3>{{chart .Person "d9"}}</div>{{end}}</div><div class="chart-pair">{{range .People}}<div><h3>{{t .Role}} · {{t "Moon chart"}}</h3>{{chart .Person "moon"}}</div>{{end}}</div>
<p class="muted">{{t "These are calculated chart placements. A D9 strength or complete marriage assessment is not inferred from the chart view."}}</p>
<footer>© Mishra 4 You · {{t "Kundli matching report"}}</footer></body></html>`

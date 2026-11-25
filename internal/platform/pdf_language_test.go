package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	htmlstd "html"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestRemedyPDFLanguageVocabularyCoverage prevents authored template and calculation labels from leaking untranslated into English PDFs.
func TestRemedyPDFLanguageVocabularyCoverage(t *testing.T) {
	locale := pdfLocale{"en"}
	devanagari := regexp.MustCompile(`[\x{0900}-\x{097f}]`)
	for _, file := range []string{"pdf_sections.go", "pdf_natal_yogas.go", "pdf_numerology.go", "pdf_chart.go", "pdf_book.go"} {
		tree, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(tree, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "text" {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			value, _ := strconv.Unquote(literal.Value)
			if devanagari.MatchString(locale.text(value)) {
				t.Errorf("untranslated %s: %s", file, value)
			}
			return true
		})
	}
	for _, match := range regexp.MustCompile(`\{\{t ("[^"]*")\}\}`).FindAllStringSubmatch(pdfTemplate, -1) {
		value, err := strconv.Unquote(match[1])
		if err != nil {
			t.Fatal(err)
		}
		if devanagari.MatchString(locale.text(value)) {
			t.Errorf("untranslated template label: %s", value)
		}
	}
}

// TestRemedyPDFLanguagesPreservePurchasedText verifies original Hindi in both interfaces, excludes arbitrary prose and preserves stored snapshots.
func TestRemedyPDFLanguagesPreservePurchasedText(t *testing.T) {
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Rules = []Rule{{Hindi: "EXCLUDED-HINDI-RULE", English: "EXCLUDED-ENGLISH-RULE", Mantra: "EXCLUDED-MANTRA", Gemstone: "EXCLUDED-GEMSTONE"}}
	order := &Order{ID: "saved-purchase", Snapshot: r, ReportID: r.ID, Amount: 9900}
	before, _ := json.Marshal(order)
	renderer := ChromePDF{FontPath: a.Config.FontPath, Origin: a.Config.Origin}
	for _, language := range []string{"en", "hi"} {
		markup, err := renderer.documentLanguage(order, language)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(markup, []byte(`lang="`+language+`"`)) || bytes.Contains(markup, []byte("EXCLUDED-")) {
			t.Fatal("incorrect source-only presentation", language)
		}
		for _, entry := range r.SourceReadings {
			if !strings.Contains(htmlstd.UnescapeString(string(markup)), entry.Text) {
				t.Fatal("missing original Hindi", entry.Title)
			}
		}
	}
	after, _ := json.Marshal(order)
	if !bytes.Equal(before, after) {
		t.Fatal("rendering changed purchase")
	}
	if _, err := renderer.documentLanguage(order, "fr"); err == nil {
		t.Fatal("unsupported language")
	}
	r.Details = nil
	if _, err := renderer.documentLanguage(order, "en"); err != nil {
		t.Fatal("old calculation snapshot unavailable", err)
	}
}

type languagePDFFixture struct {
	fakePDF
	languages    []string
	seenSnapshot *Report
	failLanguage bool
	afterRender  func()
}

// RenderLanguage records only the requested immutable purchase and simulates a localized PDF or renderer failure.
func (f *languagePDFFixture) RenderLanguage(_ context.Context, order *Order, language string) ([]byte, error) {
	f.languages = append(f.languages, language)
	f.seenSnapshot = order.Snapshot
	if f.afterRender != nil {
		f.afterRender()
	}
	if f.failLanguage {
		return nil, fmt.Errorf("synthetic renderer failure")
	}
	return []byte("%PDF-1.7\nlocalized-" + language), nil
}

// TestRemedyPDFLanguageDownloadOwnershipAndRecovery verifies authorization, legacy file parity, paid entitlements, bounded rendering and retry without repurchase.
func TestRemedyPDFLanguageDownloadOwnershipAndRecovery(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	putUser(t, a, "bob")
	r := saveTestReport(t, a, "alice")
	o, err := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "pdf-language-purchase"})
	if err != nil {
		t.Fatal(err)
	}
	capture(t, a, o)
	f := &languagePDFFixture{}
	a.PDF = f
	drain(t, a, 1)
	alice, _ := authorize(t, a, "alice", "customer")
	bob, _ := authorize(t, a, "bob", "customer")
	path := "/api/orders/" + o.ID + "/pdf"
	if w := request(t, a, "GET", path+"?language=en", nil, nil, ""); w.Code != 401 {
		t.Fatal("unauthenticated download", w.Code)
	}
	if w := request(t, a, "GET", path+"?language=en", nil, bob, ""); w.Code != 404 {
		t.Fatal("other customer download", w.Code)
	}
	if w := request(t, a, "GET", path+"?language=fr", nil, alice, ""); w.Code != 400 {
		t.Fatal("invalid language accepted", w.Code)
	}
	original, err := os.ReadFile(filepath.Join(a.Config.DataDir, "pdf", o.ID+".pdf"))
	if err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "?language=hi", "?language=en"} {
		w := request(t, a, "GET", path+suffix, nil, alice, "")
		if w.Code != 200 || w.Header().Get("Content-Type") != "application/pdf" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("download %s: %d %s", suffix, w.Code, w.Body.String())
		}
		if suffix != "" {
			language := strings.TrimPrefix(suffix, "?language=")
			if !strings.Contains(w.Header().Get("Content-Disposition"), "remedies-"+language+".pdf") || !strings.Contains(w.Body.String(), "localized-"+language) {
				t.Fatal("wrong localized download")
			}
		} else if !bytes.Equal(w.Body.Bytes(), original) {
			t.Fatal("Hindi original changed")
		}
	}
	if len(f.languages) != 2 || f.languages[0] != "hi" || f.languages[1] != "en" || f.seenSnapshot.ID != r.ID {
		t.Fatal("wrong language or live report used")
	}
	// A pre-policy cached file must not expose authored prose through the legacy URL.
	if err := a.Store.Update(context.Background(), func(s *State) error { s.Orders[o.ID].Snapshot.SourceReadings = nil; return nil }); err != nil {
		t.Fatal(err)
	}
	if w := request(t, a, "GET", path, nil, alice, ""); w.Code != 200 || !strings.Contains(w.Body.String(), "localized-hi") {
		t.Fatal("legacy URL served its old authored file")
	}
	a.remedyPDFSlots <- struct{}{}
	busy := request(t, a, "GET", path+"?language=en", nil, alice, "")
	<-a.remedyPDFSlots
	if busy.Code != 429 {
		t.Fatal("unbounded concurrent render")
	}
	f.failLanguage = true
	if w := request(t, a, "GET", path+"?language=en", nil, alice, ""); w.Code != 503 {
		t.Fatal("renderer failure lost", w.Code)
	}
	f.failLanguage = false
	if w := request(t, a, "GET", path+"?language=en", nil, alice, ""); w.Code != 200 {
		t.Fatal("retry required repurchase", w.Code)
	}
	f.afterRender = func() {
		if err := a.Store.Update(context.Background(), func(s *State) error { s.Orders[o.ID].Status = "refunded"; return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if w := request(t, a, "GET", path+"?language=en", nil, alice, ""); w.Code != 404 {
		t.Fatal("revoked entitlement leaked PDF", w.Code)
	}
	unchanged, err := os.ReadFile(filepath.Join(a.Config.DataDir, "pdf", o.ID+".pdf"))
	if err != nil || !bytes.Equal(unchanged, original) {
		t.Fatal("English download overwrote original file")
	}
}

// TestRemedyPDFBilingualRenderer prints both real language variants for opt-in font and visual QA without any live customer data.
func TestRemedyPDFBilingualRenderer(t *testing.T) {
	if os.Getenv("RUN_PDF_TEST") != "1" {
		t.Skip("set RUN_PDF_TEST=1 with local Chrome installed")
	}
	a := testApp(t)
	r, err := calculate(a.Config, testBirth(), nil, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	r.Rules = []Rule{{ID: "language-qa", Topic: "Language QA", Hindi: "यह केवल भाषा और लेआउट परीक्षण का पाठ है।", English: "This is only a language and layout test.", Mantra: "मूल मंत्र क्षेत्र का परीक्षण", Gemstone: "मूल निर्देश क्षेत्र का परीक्षण"}}
	o := &Order{ID: "language-qa", Amount: 9900, Snapshot: r, ReportID: r.ID, Demo: true}
	p := ChromePDF{FontPath: a.Config.FontPath, ChromePath: os.Getenv("CHROME_PATH"), Origin: a.Config.Origin}
	dir := os.Getenv("PDF_LANGUAGE_TEST_OUTPUT")
	if dir == "" {
		dir = t.TempDir()
	}
	for _, language := range []string{"en", "hi"} {
		data, err := p.RenderLanguage(context.Background(), o, language)
		if err != nil || !bytes.HasPrefix(data, []byte("%PDF-")) || !bytes.Contains(data, []byte("/FontFile2")) {
			t.Fatal("invalid localized PDF", language, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "remedy-language-"+language+".pdf"), data, 0600); err != nil {
			t.Fatal(err)
		}
		html, err := p.documentLanguage(o, language)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "remedy-language-"+language+".html"), html, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

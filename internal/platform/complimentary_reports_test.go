package platform

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestComplimentaryReports checks free fulfilment, retries, downloads and isolation from paid products.
func TestComplimentaryReports(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "owner")
	putUser(t, a, "other")
	r := saveTestReport(t, a, "owner")
	otherReport := saveTestReport(t, a, "other")
	a.Config.Production = true
	a.Config.PaymentMode = "razorpay"
	a.Pay = nil // Any accidental provider call fails this test.
	a.PDF = &languagePDFFixture{}
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Users["owner"].Email = "newblikeboy@gmail.com"
		s.Users["other"].Email = "someone@example.com"
		s.Settings.LicenceConfirmed = true
		s.Settings.Content.Approved = true
		s.Settings.PDFEnabled = false
		s.Settings.TaxMode = "unconfigured"
		// A previous unpaid checkout must not stop the free report grant.
		s.Orders["old"] = &Order{ID: "old", UserID: "owner", Product: "pdf", ReportID: r.ID, RequestKey: "free-report-key", Status: "pending", Amount: 9900, ProviderID: "old-provider"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := authorize(t, a, "owner", "customer")
	otherCookie, _ := authorize(t, a, "other", "customer")
	var account struct {
		FreeReports bool     `json:"free_reports"`
		PDFBlockers []string `json:"pdf_blockers"`
	}
	me := request(t, a, "GET", "/api/me", nil, cookie, "")
	if err := json.Unmarshal(me.Body.Bytes(), &account); err != nil || !account.FreeReports || len(account.PDFBlockers) != 0 {
		t.Fatalf("free report availability: %s, %v", me.Body.String(), err)
	}
	if w := request(t, a, "POST", "/api/orders", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "free-report-key"}, cookie, ""); w.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d", w.Code)
	}
	var order Order
	w := request(t, a, "POST", "/api/orders", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "free-report-key"}, cookie, csrf)
	if err := json.Unmarshal(w.Body.Bytes(), &order); err != nil || w.Code != 200 || !order.Complimentary || order.Status != "paid" || order.Amount != 0 || order.Tax != 0 || order.Demo || order.ProviderID != "" || order.PaidAt == nil {
		t.Fatalf("complimentary checkout: %d %s, %v", w.Code, w.Body.String(), err)
	}
	for _, key := range []string{"free-report-key", "different-free-key"} {
		replayed, err := a.Checkout(context.Background(), "owner", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: key})
		if err != nil || replayed.ID != order.ID {
			t.Fatalf("free report duplicated: %+v, %v", replayed, err)
		}
	}
	if len(a.Store.state.Jobs) != 1 || a.Store.state.Orders["old"].Status != "pending" {
		t.Fatal("duplicate job or prior payment changed")
	}
	drain(t, a, 2)
	for _, language := range []string{"hi", "en"} {
		path := "/api/orders/" + order.ID + "/pdf?language=" + language
		if pdf := request(t, a, "GET", path, nil, cookie, ""); pdf.Code != 200 || !strings.HasPrefix(pdf.Body.String(), "%PDF-") {
			t.Fatalf("free PDF download: %d %s", pdf.Code, pdf.Body.String())
		}
		if pdf := request(t, a, "GET", path, nil, otherCookie, ""); pdf.Code != 404 {
			t.Fatalf("another user downloaded free PDF: %d", pdf.Code)
		}
	}
	if _, err := a.Checkout(context.Background(), "owner", CheckoutRequest{Product: "pdf", ReportID: otherReport.ID, Key: "foreign-report-key"}); err == nil {
		t.Fatal("another customer's report granted")
	}
	for _, in := range []CheckoutRequest{
		{Product: "pdf", ReportID: otherReport.ID, Key: "normal-paid-key"},
		{Product: "consultation", Key: "consultation-key"},
	} {
		user := "other"
		if in.Product == "consultation" {
			user = "owner"
		}
		if _, err := a.Checkout(context.Background(), user, in); err == nil || !strings.Contains(err.Error(), "Payment provider") {
			t.Fatalf("paid product bypassed payment blockers: %v", err)
		}
	}
	if f := finances(a.Store.state, a.Now().Format("2006-01"), false); f.Gross != 0 || f.PDFSales != 0 || f.Tax != 0 {
		t.Fatalf("free reports counted as revenue: %+v", f)
	}
}

// TestComplimentaryReportRequirements preserves licence and publication gates for every new report.
func TestComplimentaryReportRequirements(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "owner")
	r := saveTestReport(t, a, "owner")
	a.Config.Production = true
	a.Store.state.Users["owner"].Email = "newblikeboy@gmail.com"
	for _, licence := range []bool{false, true} {
		a.Store.state.Settings.LicenceConfirmed = licence
		a.Store.state.Settings.Content.Approved = false
		if _, err := a.Checkout(context.Background(), "owner", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "blocked-free-key"}); err == nil {
			t.Fatal("unpublished or unlicensed report granted")
		}
	}
	if freeReports(&User{Email: "newblikeboy+alias@gmail.com"}) || freeReports(&User{Email: "newblikeboy@gmail.com", Deleted: true}) || freeReports(nil) {
		t.Fatal("unintended account granted free reports")
	}
}

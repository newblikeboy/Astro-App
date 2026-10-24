package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// testApp creates an isolated development application with a controllable test clock.
func testApp(t *testing.T) *App {
	t.Helper()
	s, e := OpenStore(context.Background(), "", "")
	if e != nil {
		t.Fatal(e)
	}
	a := NewApp(s, Config{Origin: "http://localhost:5173", DataDir: t.TempDir(), EpheDir: "../../ephe", FontPath: "../../assets/NotoSansDevanagari-Regular.ttf", PaymentMode: "simulation", ZoomMode: "simulation", OTPMode: "simulation", AdminPassword: "test-admin"})
	now := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	return a
}

// putUser inserts a verified test customer into the isolated store.
func putUser(t *testing.T, a *App, user string) {
	t.Helper()
	if e := a.Store.Update(context.Background(), func(s *State) error {
		s.Users[user] = &User{ID: user, Phone: "+919876543210", Created: a.Now()}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
}

// request dispatches a JSON HTTP test request with optional session and CSRF credentials.
func request(t *testing.T, a *App, method, path string, body any, cookie *http.Cookie, csrf string) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if csrf != "" {
		req.Header.Set("X-CSRF-Token", csrf)
	}
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)
	return w
}

// authorize creates the requested customer/admin test session and returns its cookie and CSRF token.
func authorize(t *testing.T, a *App, user, role string) (*http.Cookie, string) {
	t.Helper()
	token := id("session")
	csrf := id("csrf")
	_ = a.Store.Update(context.Background(), func(s *State) error {
		s.Sessions[hash(token)] = &Session{UserID: user, Role: role, CSRF: csrf, Expires: a.Now().Add(time.Hour)}
		return nil
	})
	name := "m4y_session"
	if role == "admin" {
		name = "m4y_admin"
	}
	return &http.Cookie{Name: name, Value: token}, csrf
}

// testBirth returns the fixed Delhi birth input shared by service regression tests.
func testBirth() Birth {
	return Birth{Name: "परीक्षण व्यक्ति", Date: "1995-08-15", Time: "10:30", PlaceID: "delhi"}
}

// saveTestReport calculates and stores a report owned by the requested test customer.
func saveTestReport(t *testing.T, a *App, user string) *Report {
	t.Helper()
	r, e := calculate(a.Config, testBirth(), seed().Rules, a.Now())
	if e != nil {
		t.Fatal(e)
	}
	r.UserID = user
	_ = a.Store.Update(context.Background(), func(s *State) error { s.Reports[r.ID] = r; return nil })
	return r
}

// TestHistoricalTimezoneAndDST checks historical birth-time resolution and daylight-saving transitions.
func TestHistoricalTimezoneAndDST(t *testing.T) {
	for _, tc := range []struct {
		date, clock, zone, want string
		off                     *int
		reject                  bool
	}{
		{"1995-08-15", "10:30", "Asia/Kolkata", "1995-08-15T05:00:00Z", nil, false},
		{"1985-01-01", "00:00", "Asia/Kathmandu", "1984-12-31T18:30:00Z", nil, false},
		{"1986-01-01", "00:20", "Asia/Kathmandu", "1985-12-31T18:35:00Z", nil, false},
		{"1986-01-01", "00:05", "Asia/Kathmandu", "", nil, true},
		{"2024-03-10", "02:30", "America/New_York", "", nil, true},
		{"2024-11-03", "01:30", "America/New_York", "", nil, true},
		{"2024-11-03", "01:30", "America/New_York", "2024-11-03T05:30:00Z", ptr(-240), false},
		{"2024-11-03", "01:30", "America/New_York", "2024-11-03T06:30:00Z", ptr(-300), false},
		{"2024-11-03", "01:30", "America/New_York", "", ptr(330), true},
	} {
		got, e := ResolveWall(tc.date, tc.clock, tc.zone, tc.off)
		if tc.reject {
			if e == nil {
				t.Errorf("accepted invalid/ambiguous time: %+v", tc)
			}
			continue
		}
		if e != nil || got.UTC().Format(time.RFC3339) != tc.want {
			t.Errorf("%+v got %v, %v", tc, got, e)
		}
	}
}

// ptr creates an explicit offset pointer for timezone ambiguity tests.
func ptr(n int) *int { return &n }

// TestUnknownTimeAndApprovedVersionSelection checks unknown-time restrictions and approved interpretation version selection.
func TestUnknownTimeAndApprovedVersionSelection(t *testing.T) {
	a := testApp(t)
	b := testBirth()
	b.Unknown = true
	r, e := calculate(a.Config, b, seed().Rules, a.Now())
	if e != nil || r.Chart != nil || len(r.Rules) != 0 || r.Birth.Time != "" {
		t.Fatalf("unknown-time fabricated output: %v %v", r, e)
	}
	b.Unknown = false
	r, e = calculate(a.Config, b, seed().Rules, a.Now())
	if e != nil {
		t.Fatal(e)
	}
	now := a.Now()
	rules := map[string]Rule{"one": {ID: "approved", Version: 1, Priority: 100, Hindi: "old", Status: "approved", Approver: "Guru Ji", ApprovedAt: &now}, "two": {ID: "approved", Version: 2, Priority: 1, Hindi: "new", Status: "approved", Approver: "Guru Ji", ApprovedAt: &now}, "draft": {ID: "draft", Version: 1, Hindi: "must not appear", Status: "draft"}}
	got := matchRules(r, rules, false)
	if len(got) != 1 || got[0].Hindi != "new" {
		t.Fatalf("wrong approved version: %+v", got)
	}
	rules["retire"] = Rule{ID: "approved", Version: 3, Status: "retired"}
	if len(matchRules(r, rules, false)) != 0 {
		t.Fatal("retired rule still visible")
	}
}

// TestOwnershipCSRFAndSecureSession checks resource ownership, CSRF protection and secure session handling.
func TestOwnershipCSRFAndSecureSession(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	putUser(t, a, "bob")
	r := saveTestReport(t, a, "alice")
	cookie, csrf := authorize(t, a, "bob", "customer")
	if w := request(t, a, "GET", "/api/reports/"+r.ID, nil, cookie, ""); w.Code != 404 {
		t.Fatalf("cross-owner access: %d", w.Code)
	}
	if w := request(t, a, "POST", "/api/reports", testBirth(), cookie, ""); w.Code != 403 {
		t.Fatalf("CSRF bypass: %d", w.Code)
	}
	if w := request(t, a, "POST", "/api/reports", testBirth(), cookie, csrf); w.Code != 200 {
		t.Fatalf("authorized save %d %s", w.Code, w.Body.String())
	}
	if w := request(t, a, "GET", "/api/admin", nil, cookie, ""); w.Code != 401 {
		t.Fatal("customer entered admin")
	}
}

// TestOTPAttemptsExpiryOneTimeAndCookies checks OTP attempt limits, expiry, one-time use and session cookies.
func TestOTPAttemptsExpiryOneTimeAndCookies(t *testing.T) {
	a := testApp(t)
	w := request(t, a, "POST", "/api/auth/request", map[string]string{"phone": "9876543210"}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var challenge struct {
		ID   string `json:"challenge"`
		Code string `json:"demo_code"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &challenge)
	for i := 0; i < 5; i++ {
		w = request(t, a, "POST", "/api/auth/verify", map[string]string{"challenge": challenge.ID, "code": "wrong"}, nil, "")
		if w.Code != 401 {
			t.Fatal("wrong OTP accepted")
		}
	}
	w = request(t, a, "POST", "/api/auth/verify", map[string]string{"challenge": challenge.ID, "code": challenge.Code}, nil, "")
	if w.Code != 401 {
		t.Fatal("retry limit not persisted")
	}
	w = request(t, a, "POST", "/api/auth/request", map[string]string{"phone": "9876543211"}, nil, "")
	_ = json.Unmarshal(w.Body.Bytes(), &challenge)
	w = request(t, a, "POST", "/api/auth/verify", map[string]string{"challenge": challenge.ID, "code": challenge.Code}, nil, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("insecure session cookie")
	}
	w = request(t, a, "POST", "/api/auth/verify", map[string]string{"challenge": challenge.ID, "code": challenge.Code}, nil, "")
	if w.Code != 401 {
		t.Fatal("OTP replay accepted")
	}
	w = request(t, a, "POST", "/api/auth/request", map[string]string{"phone": "9876543212"}, nil, "")
	_ = json.Unmarshal(w.Body.Bytes(), &challenge)
	now := a.Now().Add(6 * time.Minute)
	a.Now = func() time.Time { return now }
	w = request(t, a, "POST", "/api/auth/verify", map[string]string{"challenge": challenge.ID, "code": challenge.Code}, nil, "")
	if w.Code != 401 {
		t.Fatal("expired OTP accepted")
	}
}

// TestServerPricesAndProductionPDFContentBlocker checks authoritative prices and required approved content before production PDF purchase.
func TestServerPricesAndProductionPDFContentBlocker(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	r := saveTestReport(t, a, "alice")
	cookie, csrf := authorize(t, a, "alice", "customer")
	w := request(t, a, "POST", "/api/orders", map[string]any{"product": "pdf", "report_id": r.ID, "idempotency_key": "test-key-1", "amount": 1}, cookie, csrf)
	if w.Code != 400 {
		t.Fatal("browser amount accepted")
	}
	o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "test-key-2"})
	if e != nil || o.Amount != 9900 {
		t.Fatalf("wrong price: %+v %v", o, e)
	}
	a.Config.Production = true
	_ = a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.PDFEnabled = true
		s.Settings.LicenceConfirmed = true
		s.Settings.Content.Approved = true
		s.Settings.TaxMode = "none"
		return nil
	})
	_, e = a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "test-key-3"})
	if e == nil {
		t.Fatal("sold unapproved demo remedies in production")
	}
}

// TestSignatureWebhookReplayAndPaymentState checks payment signatures, webhook replay handling and payment state transitions.
func TestSignatureWebhookReplayAndPaymentState(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	r := saveTestReport(t, a, "alice")
	o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "test-key-replay"})
	if e != nil {
		t.Fatal(e)
	}
	a.Config.PaymentMode = "razorpay"
	a.Config.WebhookSecret = "webhook-test-secret"
	body := map[string]any{"event": "payment.captured", "payload": map[string]any{"payment": map[string]any{"entity": Payment{ID: "pay_fixture", OrderID: o.ProviderID, Status: "captured", Amount: 9900, Currency: "INR"}}}}
	raw, _ := json.Marshal(body)
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/api/payments/webhook", bytes.NewReader(raw))
		req.Header.Set("X-Razorpay-Event-Id", "event-fixture")
		req.Header.Set("X-Razorpay-Signature", signature(a.Config.WebhookSecret, string(raw)))
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	_ = a.Store.View(context.Background(), func(s *State) error {
		if s.Orders[o.ID].Status != "paid" || len(s.Jobs) != 1 {
			t.Fatalf("duplicate fulfilment or wrong state %+v", s.Jobs)
		}
		return nil
	})
	bad := httptest.NewRequest("POST", "/api/payments/webhook", bytes.NewReader(raw))
	bad.Header.Set("X-Razorpay-Signature", "bad")
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, bad)
	if w.Code != 400 {
		t.Fatal("invalid HMAC accepted")
	}
	_ = a.Store.Update(context.Background(), func(s *State) error {
		p := Payment{ID: "pay_fixture", OrderID: o.ProviderID, Status: "failed", Amount: 9900, Currency: "INR"}
		if e := applyPayment(s, s.Orders[o.ID], p, a.Now()); e != nil {
			return e
		}
		if s.Orders[o.ID].Status != "paid" {
			t.Fatal("late failed event regressed capture")
		}
		p.Status = "captured"
		p.Amount = 1
		if e := applyPayment(s, s.Orders[o.ID], p, a.Now()); e == nil {
			t.Fatal("amount mismatch accepted")
		}
		return nil
	})
}

// TestConcurrentHoldsAndLatePaymentResolution checks competing booking holds and resolution of payments received after expiry.
func TestConcurrentHoldsAndLatePaymentResolution(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	var slot string
	_ = a.Store.View(context.Background(), func(s *State) error { slot = slots(s, a.Now(), false)[0].Format(time.RFC3339); return nil })
	var successes atomic.Int32
	var winner *Order
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "consultation", Slot: slot, Key: fmt.Sprintf("parallel-key-%d", i)})
			if e == nil {
				successes.Add(1)
				mu.Lock()
				winner = o
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("double-booked: %d", successes.Load())
	}
	later := a.Now().Add(11 * time.Minute)
	a.Now = func() time.Time { return later }
	replacement, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "consultation", Slot: slot, Key: "replacement-key"})
	if e != nil {
		t.Fatal(e)
	}
	_ = a.Store.Update(context.Background(), func(s *State) error {
		o := s.Orders[winner.ID]
		p := Payment{ID: "late_payment", OrderID: o.ProviderID, Status: "captured", Amount: o.Amount, Currency: "INR"}
		if e := applyPayment(s, o, p, a.Now()); e != nil {
			return e
		}
		if s.Bookings[o.BookingID].Status != "needs_resolution" {
			t.Fatal("late payment stole another hold")
		}
		if s.Bookings[replacement.BookingID].Status != "held" {
			t.Fatal("replacement hold lost")
		}
		return nil
	})
}

type fakePDF struct {
	calls atomic.Int32
	fail  bool
}

// Render counts PDF rendering attempts and optionally simulates a temporary rendering failure.
func (f *fakePDF) Render(context.Context, *Order) ([]byte, error) {
	f.calls.Add(1)
	if f.fail {
		return nil, fmt.Errorf("temporary renderer failure")
	}
	return []byte("%PDF-1.7\nfixture"), nil
}

type flakyZoom struct {
	calls atomic.Int32
	fail  bool
}

// Create counts meeting attempts and simulates a recoverable Zoom failure when requested.
func (f *flakyZoom) Create(_ context.Context, b *Booking, _ int) (Meeting, error) {
	f.calls.Add(1)
	if f.fail {
		return Meeting{}, fmt.Errorf("temporary Zoom failure")
	}
	return Meeting{"meeting-id", "https://example.invalid/attendee"}, nil
}

// Delete completes meeting cleanup in tests without contacting an external provider.
func (f *flakyZoom) Delete(context.Context, string) error { return nil }

// drain processes a bounded number of queued jobs and fails the test on an unexpected worker error.
func drain(t *testing.T, a *App, n int) {
	t.Helper()
	for i := 0; i < n; i++ {
		if e := a.WorkOnce(context.Background()); e != nil {
			t.Fatal(e)
		}
	}
}

// capture applies a matching captured-payment fixture to the current stored order.
func capture(t *testing.T, a *App, o *Order) {
	t.Helper()
	if e := a.Store.Update(context.Background(), func(s *State) error {
		current := s.Orders[o.ID]
		return applyPayment(s, current, Payment{ID: "payment_" + o.ID, OrderID: current.ProviderID, Status: "captured", Amount: current.Amount, Currency: "INR"}, a.Now())
	}); e != nil {
		t.Fatal(e)
	}
}

// TestPDFRetryIdempotencyAndImmutableSnapshot checks safe PDF retries while preserving the purchased report snapshot.
func TestPDFRetryIdempotencyAndImmutableSnapshot(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	r := saveTestReport(t, a, "alice")
	o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: r.ID, Key: "pdf-fulfilment-key"})
	if e != nil {
		t.Fatal(e)
	}
	capture(t, a, o)
	pdf := &fakePDF{fail: true}
	a.PDF = pdf
	drain(t, a, 1)
	_ = a.Store.Update(context.Background(), func(s *State) error {
		j := s.Jobs["pdf:"+o.ID+":1"]
		if j.Status != "retry" {
			t.Fatal("failure lost entitlement")
		}
		j.Next = a.Now()
		s.Rules["demo-reflection:1"] = Rule{ID: "replacement", Hindi: "changed"}
		return nil
	})
	pdf.fail = false
	drain(t, a, 5)
	if pdf.calls.Load() != 2 {
		t.Fatalf("duplicate PDF generated %d times", pdf.calls.Load())
	}
	if _, e = os.Stat(filepath.Join(a.Config.DataDir, "pdf", o.ID+".pdf")); e != nil {
		t.Fatal(e)
	}
	_ = a.Store.View(context.Background(), func(s *State) error {
		if s.Orders[o.ID].Snapshot.Rules[0].ID != "demo-reflection" {
			t.Fatal("purchase snapshot mutated")
		}
		return nil
	})
}

// TestZoomFailureRetainsPaidBookingAndRetries checks that a failed Zoom request retains the paid booking for retry.
func TestZoomFailureRetainsPaidBookingAndRetries(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "alice")
	var slot string
	_ = a.Store.View(context.Background(), func(s *State) error { slot = slots(s, a.Now(), false)[0].Format(time.RFC3339); return nil })
	o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "consultation", Slot: slot, Key: "zoom-retry-key"})
	if e != nil {
		t.Fatal(e)
	}
	capture(t, a, o)
	z := &flakyZoom{fail: true}
	a.Zoom = z
	drain(t, a, 1)
	_ = a.Store.Update(context.Background(), func(s *State) error {
		b := s.Bookings[o.BookingID]
		if b.Status != "meeting_pending" || b.JoinURL != "" || s.Orders[o.ID].Status != "paid" {
			t.Fatal("failed Zoom misrepresented as ready")
		}
		s.Jobs["zoom:"+b.ID+":1"].Next = a.Now()
		return nil
	})
	z.fail = false
	drain(t, a, 4)
	_ = a.Store.View(context.Background(), func(s *State) error {
		if s.Bookings[o.BookingID].Status != "confirmed" {
			t.Fatal("retry did not confirm")
		}
		return nil
	})
}

// TestFinanceRefundsAcrossMonthsAndNoDoubleExpenses checks cross-month refunds and prevents duplicate expense deductions.
func TestFinanceRefundsAcrossMonthsAndNoDoubleExpenses(t *testing.T) {
	s := seed()
	paid := time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC)
	s.Orders["sale"] = &Order{ID: "sale", UserID: "u", Product: "pdf", Amount: 11800, Tax: 1800, Fees: 236, Currency: "INR", ProviderID: "provider", PaymentID: "payment", Status: "paid", PaidAt: &paid}
	o := s.Orders["sale"]
	for _, r := range []struct {
		amount int64
		month  time.Month
	}{{5900, 9}, {11800, 10}} {
		p := Payment{ID: "payment", OrderID: "provider", Status: "refunded", Amount: 11800, Currency: "INR", Refunded: r.amount, Fee: 236}
		if e := applyPayment(s, o, p, time.Date(2026, r.month, 20, 0, 0, 0, 0, time.UTC)); e != nil {
			t.Fatal(e)
		}
	}
	s.Expenses["approved"] = &Expense{Date: "2026-09-18", Amount: 1000, Status: "approved"}
	s.Expenses["pending"] = &Expense{Date: "2026-09-18", Amount: 9999, Status: "pending"}
	s.Reconciliations["2026-09"] = Reconciliation{Reserve: 500}
	sept := finances(s, "2026-09", false)
	if sept.Gross != 11800 || sept.Refunds != 5900 || sept.Fees != 236 || sept.Tax != 900 || sept.Expenses != 1000 || sept.Distributable != 3264 || sept.PartnerA != 1632 {
		t.Fatalf("wrong September: %+v", sept)
	}
	oct := finances(s, "2026-10", false)
	if oct.Refunds != 5900 || oct.Tax != -900 || oct.Gross != 0 {
		t.Fatalf("wrong refund cash month %+v", oct)
	}
	before := len(o.Refunds)
	p := Payment{ID: "payment", OrderID: "provider", Status: "refunded", Amount: 11800, Currency: "INR", Refunded: 11800, Fee: 236}
	_ = applyPayment(s, o, p, time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC))
	if len(o.Refunds) != before {
		t.Fatal("duplicate refund deducted twice")
	}
}

// TestLocalStoreRollbackAndBSONRoundTrip checks local transaction rollback and BSON serialization round trips.
func TestLocalStoreRollbackAndBSONRoundTrip(t *testing.T) {
	s, e := OpenStore(context.Background(), "", filepath.Join(t.TempDir(), "state.json"))
	if e != nil {
		t.Fatal(e)
	}
	_ = s.Update(context.Background(), func(v *State) error { v.Users["rollback"] = &User{ID: "rollback"}; return fmt.Errorf("abort") })
	_ = s.View(context.Background(), func(v *State) error {
		if v.Users["rollback"] != nil {
			t.Fatal("transaction rollback failed")
		}
		return nil
	})
	state := seed()
	state.Users["test"] = &User{ID: "test", Phone: "+919999999999"}
	state.Orders["test"] = &Order{ID: "test", Amount: 350100}
	records, e := encodeState(state)
	if e != nil {
		t.Fatal(e)
	}
	if records["orders:test"].Data.Lookup("value").Document().Lookup("amount").Int32() != 350100 {
		t.Fatal("paise changed during BSON conversion")
	}
	if strings.Contains(records["users:test"].Data.String(), "350100") {
		t.Fatal("records mixed")
	}
}

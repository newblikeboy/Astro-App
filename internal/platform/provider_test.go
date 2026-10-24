package platform

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// TestRazorpayHTTPAdapterVerificationAndRefundKey checks provider verification and stable refund idempotency keys through a test HTTP adapter.
func TestRazorpayHTTPAdapterVerificationAndRefundKey(t *testing.T) {
	const secret = "fixture-secret"
	var refundKey string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "rzp_test_fixture" || pass != secret {
			t.Error("provider Basic authentication missing")
			w.WriteHeader(401)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/orders":
			var in struct {
				Amount   int64  `json:"amount"`
				Currency string `json:"currency"`
				Receipt  string `json:"receipt"`
				Partial  bool   `json:"partial_payment"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			if in.Amount != 9900 || in.Currency != "INR" || in.Partial || len(in.Receipt) > 40 {
				t.Error("invalid server-priced provider order")
			}
			fmt.Fprint(w, `{"id":"order_fixture"}`)
		case "/payments/pay_fixture":
			fmt.Fprint(w, `{"id":"pay_fixture","order_id":"order_fixture","status":"captured","amount":9900,"currency":"INR","fee":236}`)
		case "/payments/pay_fixture/refund":
			key := r.Header.Get("X-Refund-Idempotency")
			if !regexp.MustCompile(`^[a-zA-Z0-9_-]{10,40}$`).MatchString(key) {
				t.Error("provider refund key does not meet API contract")
			}
			if refundKey != "" && refundKey != key {
				t.Error("refund retry key changed")
			}
			refundKey = key
			fmt.Fprint(w, `{"id":"rfnd_fixture"}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	a := testApp(t)
	putUser(t, a, "alice")
	report := saveTestReport(t, a, "alice")
	a.Config.PaymentMode = "razorpay"
	a.Config.KeyID = "rzp_test_fixture"
	a.Config.KeySecret = secret
	a.Pay = &Razorpay{KeyID: a.Config.KeyID, Secret: secret, Client: server.Client(), Base: server.URL}
	o, e := a.Checkout(context.Background(), "alice", CheckoutRequest{Product: "pdf", ReportID: report.ID, Key: "provider-fixture-key"})
	if e != nil {
		t.Fatal(e)
	}
	if !o.Demo {
		t.Fatal("Razorpay test order counted as real sales")
	}
	if _, e = a.VerifyPayment(context.Background(), o.ID, "pay_fixture", "invalid", "alice"); e == nil {
		t.Fatal("invalid callback signature accepted")
	}
	captured, e := a.VerifyPayment(context.Background(), o.ID, "pay_fixture", signature(secret, o.ProviderID+"|pay_fixture"), "alice")
	if e != nil || captured.Status != "paid" {
		t.Fatalf("callback verification failed: %v", e)
	}
	for i := 0; i < 2; i++ {
		if _, e = a.Pay.Refund(context.Background(), captured, "refund:"+o.ID+":1"); e != nil {
			t.Fatal(e)
		}
	}
}

// TestProductionRefusesSimulationAndInvalidEnvironment checks that production configuration rejects simulated providers and invalid settings.
func TestProductionRefusesSimulationAndInvalidEnvironment(t *testing.T) {
	t.Setenv("APP_ENV", "production")
	t.Setenv("MONGODB_URI", "mongodb://fixture.invalid")
	t.Setenv("APP_ORIGIN", "https://fixture.invalid")
	t.Setenv("ADMIN_PASSWORD", "long-unique-password-for-test")
	t.Setenv("ADMIN_TOTP_SECRET", "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ")
	t.Setenv("OTP_MODE", "simulation")
	t.Setenv("PAYMENT_MODE", "simulation")
	t.Setenv("ZOOM_MODE", "simulation")
	if _, e := LoadConfig(); e == nil {
		t.Fatal("production simulations accepted")
	}
	t.Setenv("APP_ENV", "produciton")
	if _, e := LoadConfig(); e == nil {
		t.Fatal("misspelled production mode silently enabled development")
	}
	if !totp("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "287082", time.Unix(59, 0)) {
		t.Fatal("RFC6238 TOTP fixture failed")
	}
}

// TestExplicitEnvironmentFileIsolation checks that an explicitly selected environment file remains isolated from other configuration files.
func TestExplicitEnvironmentFileIsolation(t *testing.T) {
	file := filepath.Join(t.TempDir(), "fixture.env")
	if e := os.WriteFile(file, []byte("MONGODB_URI=mongodb://fixture.invalid\n"), 0600); e != nil {
		t.Fatal(e)
	}
	t.Setenv("APP_ENV", "development")
	t.Setenv("MONGODB_URI", "")
	t.Setenv("ENV_FILE", "none")
	t.Setenv("OTP_MODE", "simulation")
	t.Setenv("PAYMENT_MODE", "simulation")
	t.Setenv("ZOOM_MODE", "simulation")
	c, e := LoadConfig()
	if e != nil || c.MongoURI != "" {
		t.Fatal("disabled environment file did not preserve isolated configuration")
	}
	// A separately selected fixture file is readable without touching root .env.
	t.Setenv("ENV_FILE", file)
	if _, e = LoadConfig(); e != nil {
		t.Fatal("explicit environment file could not be loaded")
	}
}

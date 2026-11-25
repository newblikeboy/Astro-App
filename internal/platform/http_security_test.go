package platform

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPublishedProviderTemplatesCannotEnableSales rejects every example payment secret before creating reservations.
func TestPublishedProviderTemplatesCannotEnableSales(t *testing.T) {
	valid := Config{Production: true, PaymentMode: "razorpay", KeyID: "rzp_test_fixture", KeySecret: "private-fixture", WebhookSecret: "private-fixture"}
	settings := Settings{LicenceConfirmed: true, PDFEnabled: true, TaxMode: "none"}
	settings.Content.Approved = true
	for _, field := range []string{"key", "secret", "webhook"} {
		c := valid
		switch field {
		case "key":
			c.KeyID = "rzp_test_REPLACE_ME"
		case "secret":
			c.KeySecret = "REPLACE_ME"
		case "webhook":
			c.WebhookSecret = "REPLACE_ME"
		}
		if paymentConfigured(c) || len(blockers(settings, c, "pdf")) == 0 {
			t.Fatal("public template credential permitted checkout:", field)
		}
	}
	if !paymentConfigured(valid) || len(blockers(settings, valid, "pdf")) != 0 {
		t.Fatal("configured production payment was blocked")
	}
	a := testApp(t)
	putUser(t, a, "buyer")
	a.Config = valid
	a.Config.WebhookSecret = "REPLACE_ME"
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.Shop = ShopSettings{Enabled: true, TaxMode: "none", DeliveryPolicy: "Delivery fixture policy", ReturnPolicy: "Returns fixture policy"}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	_, err := a.ShopCheckout(context.Background(), "buyer", ShopCheckoutRequest{Key: "fixture-request", Items: []ShopItemRequest{{ProductID: "not-a-real-product", Quantity: 1}}, Address: DeliveryAddress{Name: "Test Buyer", Phone: "9876543210", Line1: "Fixture street 10", City: "Jaipur", State: "Rajasthan", PIN: "302001", Country: "IN"}})
	if err == nil || !strings.Contains(err.Error(), "not available yet") || len(a.Store.state.Orders) != 0 {
		t.Fatal("shop did not block before creating a reservation", err)
	}
}

// TestClientIPProxyTrust checks explicit trust, proxy-header spoofing, malformed addresses and IPv4-mapped normalization.
func TestClientIPProxyTrust(t *testing.T) {
	for _, tc := range []struct {
		name, peer, real, forwarded, want string
		trust                             bool
	}{
		{"direct spoof", "192.0.2.1:1234", "198.51.100.1", "", "192.0.2.1", true},
		{"opt-in required", "127.0.0.1:1234", "198.51.100.1", "", "127.0.0.1", false},
		{"trusted loopback", "127.0.0.1:1234", "198.51.100.1", "", "198.51.100.1", true},
		{"IPv6 proxy", "[::1]:1234", "2001:db8::2", "", "2001:db8::2", true},
		{"mapped address", "[::ffff:127.0.0.1]:1234", "::ffff:198.51.100.1", "", "198.51.100.1", true},
		{"multiple values rejected", "127.0.0.1:1234", "198.51.100.1, 198.51.100.2", "", "127.0.0.1", true},
		{"forwarded-for ignored", "127.0.0.1:1234", "", "198.51.100.1", "127.0.0.1", true},
		{"zone rejected", "127.0.0.1:1234", "fe80::1%fake", "", "127.0.0.1", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &App{Config: Config{TrustLoopbackProxy: tc.trust}}
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = tc.peer
			r.Header.Set("X-Real-IP", tc.real)
			r.Header.Set("X-Forwarded-For", tc.forwarded)
			if got := a.clientIP(r); got != tc.want {
				t.Fatalf("client IP = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestProxyRateLimitIsolation proves one proxied visitor cannot exhaust another visitor's allowance or rotate spoofed direct headers.
func TestProxyRateLimitIsolation(t *testing.T) {
	a := testApp(t)
	a.Config.TrustLoopbackProxy = true
	handler := a.Handler()
	for i := 0; i < 21; i++ {
		r := httptest.NewRequest("POST", "/api/auth/does-not-exist", nil)
		r.RemoteAddr = "127.0.0.1:1234"
		r.Header.Set("X-Real-IP", "198.51.100.1")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if i == 20 && w.Code != 429 {
			t.Fatal("visitor rate limit was not enforced")
		}
	}
	r := httptest.NewRequest("POST", "/api/auth/does-not-exist", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Real-IP", "198.51.100.2")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatal("one visitor blocked a different proxied visitor")
	}
	for i := 0; i < 21; i++ {
		r := httptest.NewRequest("POST", "/api/auth/does-not-exist", nil)
		r.RemoteAddr = "192.0.2.10:1234"
		r.Header.Set("X-Real-IP", "198.51.100."+strings.Repeat("1", i%3+1))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if i == 20 && w.Code != 429 {
			t.Fatal("direct client bypassed the limiter using a spoofed proxy header")
		}
	}
}

// TestPrivateStaticFilesAreNotServed creates actual sensitive files under the public directory and verifies HTTP denial.
func TestPrivateStaticFilesAreNotServed(t *testing.T) {
	t.Chdir(t.TempDir())
	paths := []string{"/.env", "/.git/HEAD", "/assets/app.env", "/assets/config.env.production", "/private.key", "/backup.sql", "/development-state.json"}
	for _, name := range append(append([]string{}, paths...), "/index.html", "/assets/main.js", "/.well-known/security.txt") {
		file := filepath.Join("web", "dist", filepath.FromSlash(strings.TrimPrefix(name, "/")))
		if err := os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte("security-fixture-content"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a := testApp(t)
	for _, name := range append(paths, "/%2eenv", "/%5c.env") {
		w := request(t, a, "GET", name, nil, nil, "")
		if w.Code != 404 || strings.Contains(w.Body.String(), "security-fixture-content") {
			t.Fatalf("private static path exposed: %s (%d)", name, w.Code)
		}
	}
	for _, name := range []string{"/login", "/assets/main.js", "/.well-known/security.txt"} {
		if w := request(t, a, "GET", name, nil, nil, ""); w.Code != 200 {
			t.Fatalf("valid public route blocked: %s", name)
		}
	}
	if w := request(t, a, "POST", "/login", nil, nil, ""); w.Code != 405 {
		t.Fatal("static fallback accepted a state-changing method")
	}
}

// TestSecurityProtectedRouteMatrix denies anonymous access before reading or mutating private customer and administrator records.
func TestSecurityProtectedRouteMatrix(t *testing.T) {
	a := testApp(t)
	a.Config.Production = true
	for _, tc := range []struct{ method, path string }{
		{"GET", "/api/me"}, {"GET", "/api/account/export"}, {"GET", "/api/reports/private"},
		{"POST", "/api/reports"}, {"DELETE", "/api/reports/private"}, {"DELETE", "/api/account"},
		{"PUT", "/api/privacy"}, {"PUT", "/api/auth/credentials"}, {"GET", "/api/kundli/match/profiles"},
		{"POST", "/api/orders"}, {"POST", "/api/orders/private/verify"}, {"GET", "/api/orders/private/pdf"},
		{"POST", "/api/bookings/private"}, {"POST", "/api/shop/checkout"}, {"POST", "/api/shop/orders/private/cancel"},
		{"GET", "/api/admin"}, {"GET", "/api/admin/shop"}, {"GET", "/api/admin/video-links"},
		{"GET", "/api/admin/finance.csv"}, {"GET", "/api/admin/receipts/private"},
		{"PUT", "/api/admin/settings"}, {"POST", "/api/admin/refund"}, {"POST", "/api/admin/receipts"},
	} {
		if w := request(t, a, tc.method, tc.path, map[string]any{}, nil, ""); w.Code != 401 {
			t.Fatalf("anonymous access: %s %s returned %d", tc.method, tc.path, w.Code)
		}
	}
	if w := request(t, a, "POST", "/api/orders/private/simulate", map[string]bool{"success": true}, nil, ""); w.Code != 404 {
		t.Fatal("production exposed simulated payment confirmation")
	}
}

// TestCrossOriginActionsStayDenied proves a valid session and CSRF token cannot bypass the site's origin boundary.
func TestCrossOriginActionsStayDenied(t *testing.T) {
	a := testApp(t)
	a.Config.Production = true
	a.Config.Origin = "https://mishra4you.com"
	putUser(t, a, "private-owner")
	cookie, csrf := authorize(t, a, "private-owner", "customer")
	r := httptest.NewRequest("PUT", "/api/privacy", bytes.NewBufferString(`{"marketing":true,"analytics":false}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "https://attacker.example")
	r.Header.Set("X-CSRF-Token", csrf)
	r.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusForbidden || a.Store.state.Users["private-owner"].Marketing {
		t.Fatal("cross-origin action changed customer data")
	}
}

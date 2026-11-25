package platform

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

// TestAdminEmailPasswordLogin requires both credentials in production and preserves administrator session protection.
func TestAdminEmailPasswordLogin(t *testing.T) {
	a := testApp(t)
	a.Config.Production = true
	a.Config.Origin = "https://fixture.invalid"
	a.Config.AdminEmail = "owner@example.test"
	a.Config.AdminPassword = "Demo!789"
	for _, body := range []map[string]string{
		{"password": "Demo!789"},
		{"email": "other@example.test", "password": "Demo!789"},
		{"email": "owner@example.test", "password": "wrong"},
		{"email": "owner@example.test"},
	} {
		w := request(t, a, "POST", "/api/admin/login", body, nil, "")
		if w.Code != http.StatusUnauthorized || len(w.Result().Cookies()) != 0 || w.Body.String() != "{\"error\":\"invalid admin credentials\"}\n" {
			t.Fatalf("invalid credentials granted access: %d %s", w.Code, w.Body.String())
		}
	}
	w := request(t, a, "POST", "/api/admin/login", map[string]string{
		"email": " OWNER@EXAMPLE.TEST ", "password": "Demo!789",
	}, nil, "")
	if w.Code != http.StatusOK {
		t.Fatalf("email/password login failed: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != "m4y_admin" || !cookies[0].Secure || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatal("administrator cookie protections changed")
	}
	if !strings.Contains(w.Body.String(), "\"csrf\"") {
		t.Fatal("administrator session lacks CSRF token")
	}
	if e := a.Store.View(context.Background(), func(s *State) error {
		for _, session := range s.Sessions {
			if session.Role != "admin" {
				t.Fatal("login created a customer session")
			}
		}
		return nil
	}); e != nil {
		t.Fatal(e)
	}
	if got := request(t, a, "GET", "/api/admin", nil, cookies[0], ""); got.Code != http.StatusOK {
		t.Fatalf("administrator session was not accepted: %d", got.Code)
	}
	if got := request(t, a, "GET", "/api/me", nil, cookies[0], ""); got.Code != http.StatusUnauthorized {
		t.Fatal("administrator session granted customer access")
	}
	if got := request(t, a, "PUT", "/api/admin/settings", seed().Settings, cookies[0], ""); got.Code != http.StatusForbidden {
		t.Fatal("administrator mutation accepted without CSRF")
	}
}

// TestProductionAdminConfiguration accepts email/password without TOTP and rejects unusable production credentials.
func TestProductionAdminConfiguration(t *testing.T) {
	for key, value := range map[string]string{
		"ENV_FILE": "none", "APP_ENV": "production", "APP_ORIGIN": "https://fixture.invalid",
		"MONGODB_URI": "mongodb://fixture.invalid", "ADMIN_EMAIL": " Owner@Example.Test ", "ADMIN_PASSWORD": "Demo!789",
		"PAYMENT_MODE": "razorpay", "RAZORPAY_KEY_ID": "rzp_test_fixture", "RAZORPAY_KEY_SECRET": "fixture", "RAZORPAY_WEBHOOK_SECRET": "fixture",
		"OTP_MODE": "simulation", "OTP_GATEWAY_URL": "", "OTP_GATEWAY_TOKEN": "",
		"ZOOM_MODE": "zoom", "ZOOM_ACCOUNT_ID": "fixture", "ZOOM_CLIENT_ID": "fixture", "ZOOM_CLIENT_SECRET": "fixture",
		"NOTIFICATION_URL": "https://fixture.invalid/notifications", "NOTIFICATION_TOKEN": "fixture",
		"ALLOW_LIVE_PAYMENTS": "false", "ADMIN_TOTP_SECRET": "",
	} {
		t.Setenv(key, value)
	}
	c, err := LoadConfig()
	if err != nil || c.AdminEmail != "owner@example.test" {
		t.Fatalf("valid email/password configuration rejected: %v", err)
	}
	for _, email := range []string{"", "not-an-email", "Name <owner@example.test>", "local-admin@example.test"} {
		t.Setenv("ADMIN_EMAIL", email)
		if _, err := LoadConfig(); err == nil {
			t.Fatal("unconfigured or invalid production admin email accepted")
		}
	}
	t.Setenv("ADMIN_EMAIL", "owner@example.test")
	for _, password := range []string{"", "short", "local-admin-change-me", "REPLACE_WITH_PRIVATE_PASSWORD_AT_LEAST_8_CHARACTERS"} {
		t.Setenv("ADMIN_PASSWORD", password)
		if _, err := LoadConfig(); err == nil {
			t.Fatal("unusable production admin password accepted")
		}
	}
}

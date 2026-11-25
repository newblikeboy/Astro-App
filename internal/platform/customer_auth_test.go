package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// customerSignup registers a fixture through the public API and returns its cookie, ID and CSRF token.
func customerSignup(t *testing.T, a *App, email, password string) (*http.Cookie, string, string) {
	t.Helper()
	w := request(t, a, "POST", "/api/auth/register", map[string]string{"email": email, "password": password}, nil, "")
	if w.Code != 200 {
		t.Fatalf("signup: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		ID   string `json:"user_id"`
		CSRF string `json:"csrf"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || out.ID == "" || out.CSRF == "" {
		t.Fatal("incomplete signup session")
	}
	return cookies[0], out.ID, out.CSRF
}

// TestCustomerEmailPasswordLogin verifies canonical emails, exact passwords, private hashes and existing session protection.
func TestCustomerEmailPasswordLogin(t *testing.T) {
	a := testApp(t)
	a.Config.Production = true
	password := "  Secret!123  "
	cookie, userID, _ := customerSignup(t, a, " Reader@Example.Test ", password)
	if cookie.Name != "m4y_session" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("insecure customer cookie")
	}
	credential := a.Store.state.PasswordCredentials[userID]
	if a.Store.state.Users[userID].Email != "reader@example.test" || credential.Hash == password || bcrypt.CompareHashAndPassword([]byte(credential.Hash), []byte(password)) != nil {
		t.Fatal("email/password storage is incorrect")
	}
	if _, err := bcrypt.Cost(customerDummyHash); err != nil {
		t.Fatal("invalid unknown-account timing hash")
	}
	records, err := encodeState(a.Store.state)
	if err != nil {
		t.Fatal(err)
	}
	if records["password_credentials:"+userID].Data.Lookup("value").Document().Lookup("password_hash").StringValue() != credential.Hash || strings.Contains(records["users:"+userID].Data.String(), credential.Hash) {
		t.Fatal("BSON records failed to separate the credential hash from the public user")
	}
	for _, path := range []string{"/api/me", "/api/account/export"} {
		w := request(t, a, "GET", path, nil, cookie, "")
		if w.Code != 200 || strings.Contains(w.Body.String(), credential.Hash) || strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), password) {
			t.Fatalf("profile/export exposes credentials: %s", path)
		}
	}
	if w := request(t, a, "PUT", "/api/privacy", map[string]bool{"marketing": true}, cookie, ""); w.Code != 403 {
		t.Fatal("password session bypassed CSRF")
	}
	if w := request(t, a, "GET", "/api/admin", nil, cookie, ""); w.Code != 401 {
		t.Fatal("customer entered admin")
	}
	for _, body := range []map[string]string{
		{"email": "reader@example.test", "password": "wrong-password"},
		{"email": "absent@example.test", "password": password},
		{"email": "reader@example.test", "password": strings.TrimSpace(password)},
		{"email": "reader@example.test", "password": strings.Repeat("a", 73)},
	} {
		w := request(t, a, "POST", "/api/auth/login", body, nil, "")
		if w.Code != 401 || len(w.Result().Cookies()) != 0 || !strings.Contains(w.Body.String(), "ईमेल या पासवर्ड सही नहीं है।") {
			t.Fatal("invalid login accepted or account existence disclosed")
		}
	}
	w := request(t, a, "POST", "/api/auth/login", map[string]string{"email": " READER@EXAMPLE.TEST ", "password": password}, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if a.Store.state.Sessions[hash(cookie.Value)] != nil {
		t.Fatal("login did not rotate existing session")
	}
	if w := request(t, a, "POST", "/api/logout", nil, w.Result().Cookies()[0], ""); w.Code != 403 {
		t.Fatal("logout bypassed CSRF")
	}
}

// TestCustomerSignupValidationAndUniqueness rejects weak/malformed credentials and serializes concurrent same-email registrations.
func TestCustomerSignupValidationAndUniqueness(t *testing.T) {
	a := testApp(t)
	for _, body := range []map[string]string{
		{"email": "not-email", "password": "Secret!123"},
		{"email": "Name <reader@example.test>", "password": "Secret!123"},
		{"email": "reader@example.test", "password": "short"},
		{"email": "reader@example.test", "password": strings.Repeat("a", 73)},
		{"email": "reader@example.test", "password": strings.Repeat("क", 25)},
	} {
		if w := request(t, a, "POST", "/api/auth/register", body, nil, ""); w.Code != 422 {
			t.Fatalf("invalid signup: %d %s", w.Code, w.Body.String())
		}
	}
	var group sync.WaitGroup
	codes := make(chan int, 2)
	for _, email := range []string{"Reader@Example.Test", "reader@example.test"} {
		group.Add(1)
		go func(email string) {
			defer group.Done()
			codes <- request(t, a, "POST", "/api/auth/register", map[string]string{"email": email, "password": "Secret!123"}, nil, "").Code
		}(email)
	}
	group.Wait()
	close(codes)
	counts := map[int]int{}
	for code := range codes {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 || len(a.Store.state.Users) != 1 || len(a.Store.state.PasswordCredentials) != 1 {
		t.Fatalf("duplicate accounts: %v", counts)
	}
	for _, path := range []string{"/api/auth/request", "/api/auth/verify"} {
		w := request(t, a, "POST", path, map[string]string{"phone": "9876543210"}, nil, "")
		if w.Code != 404 || len(w.Result().Cookies()) != 0 {
			t.Fatal("retired OTP endpoint still authenticates")
		}
	}
}

// TestLegacyCustomerCredentials preserves ownership and requires the existing customer session and CSRF to link credentials once.
func TestLegacyCustomerCredentials(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "legacy")
	cookie, csrf := authorize(t, a, "legacy", "customer")
	a.Store.state.Reports["saved"] = &Report{ID: "saved", UserID: "legacy"}
	a.Store.state.Orders["paid"] = &Order{ID: "paid", UserID: "legacy", Status: "paid"}
	body := map[string]string{"email": "legacy@example.test", "password": "Legacy!123"}
	if w := request(t, a, "PUT", "/api/auth/credentials", body, nil, ""); w.Code != 401 {
		t.Fatal("anonymous legacy account claim")
	}
	if w := request(t, a, "PUT", "/api/auth/credentials", body, cookie, ""); w.Code != 403 {
		t.Fatal("legacy link bypassed CSRF")
	}
	if w := request(t, a, "POST", "/api/auth/register", map[string]string{"email": "claim@example.test", "password": "Secret!123", "phone": "+919876543210"}, nil, ""); w.Code != 400 {
		t.Fatal("signup accepted legacy phone ownership claim")
	}
	if w := request(t, a, "PUT", "/api/auth/credentials", body, cookie, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if len(a.Store.state.Users) != 1 || a.Store.state.Users["legacy"].Email != body["email"] || a.Store.state.Reports["saved"].UserID != "legacy" || a.Store.state.Orders["paid"].UserID != "legacy" {
		t.Fatal("legacy ownership changed")
	}
	body["password"] = "Replacement!123"
	if w := request(t, a, "PUT", "/api/auth/credentials", body, cookie, csrf); w.Code != 409 {
		t.Fatal("existing password overwritten without verification")
	}
	w := request(t, a, "POST", "/api/auth/login", map[string]string{"email": "legacy@example.test", "password": "Legacy!123"}, nil, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user_id":"legacy"`) {
		t.Fatal("legacy login lost ownership")
	}
}

// TestCustomerCredentialsPersistenceDeletionAndLimits verifies saved hashes survive reopen and deleted accounts cannot log in.
func TestCustomerCredentialsPersistenceDeletionAndLimits(t *testing.T) {
	a := testApp(t)
	path := filepath.Join(t.TempDir(), "store.json")
	store, err := OpenStore(context.Background(), "", path)
	if err != nil {
		t.Fatal(err)
	}
	a.Store = store
	cookie, userID, csrf := customerSignup(t, a, "persist@example.test", "Secret!123")
	reopened, err := OpenStore(context.Background(), "", path)
	if err != nil {
		t.Fatal(err)
	}
	a.Store = reopened
	if w := request(t, a, "POST", "/api/auth/login", map[string]string{"email": "persist@example.test", "password": "Secret!123"}, nil, ""); w.Code != 200 {
		t.Fatal("persisted password unusable")
	}
	if w := request(t, a, "DELETE", "/api/account", nil, cookie, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if a.Store.state.Users[userID].Email != "" || a.Store.state.PasswordCredentials[userID] != nil {
		t.Fatal("deletion retained credentials")
	}
	if w := request(t, a, "POST", "/api/auth/login", map[string]string{"email": "persist@example.test", "password": "Secret!123"}, nil, ""); w.Code != 401 {
		t.Fatal("deleted account authenticated")
	}
	for range 15 {
		a.limits.allow("customer-auth:"+hash("blocked@example.test"), 15, 15*time.Minute)
	}
	if w := request(t, a, "POST", "/api/auth/login", map[string]string{"email": " BLOCKED@EXAMPLE.TEST ", "password": "Secret!123"}, nil, ""); w.Code != 429 {
		t.Fatal("email guess limit bypassed")
	}
}

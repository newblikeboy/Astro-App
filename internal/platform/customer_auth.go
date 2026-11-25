package platform

import (
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// A valid cost-10 hash makes unknown-account login perform the same password work.
var customerDummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

type customerCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// normalizeCustomerEmail accepts a bare email address and gives account lookup one canonical spelling.
func normalizeCustomerEmail(value string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(value))
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || !strings.Contains(email, "@") {
		return "", fmt.Errorf("कृपया सही ईमेल पता लिखें।")
	}
	return email, nil
}

// customerPassword validates new passwords without trimming or silently truncating bcrypt's byte limit.
func customerPassword(value string) error {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) < 8 {
		return fmt.Errorf("पासवर्ड कम से कम 8 अक्षर का होना चाहिए।")
	}
	if len(value) > 72 {
		return fmt.Errorf("पासवर्ड बहुत लंबा है। कृपया छोटा पासवर्ड रखें।")
	}
	return nil
}

// customerAuthAllowed limits guesses for the same normalized email even when requests use different IPs.
func (a *App) customerAuthAllowed(w http.ResponseWriter, email string) bool {
	if !a.limits.allow("customer-auth:"+hash(email), 15, 15*time.Minute) {
		fail(w, 429, fmt.Errorf("बहुत अधिक प्रयास; 15 मिनट बाद आएँ"))
		return false
	}
	return true
}

// registerCustomer creates a distinct email/password account; it never claims a legacy account by phone number.
func (a *App) registerCustomer(w http.ResponseWriter, r *http.Request) {
	var in customerCredentials
	if !decode(w, r, &in) {
		return
	}
	email, err := normalizeCustomerEmail(in.Email)
	if err == nil {
		err = customerPassword(in.Password)
	}
	if err != nil {
		fail(w, 422, err)
		return
	}
	if !a.customerAuthAllowed(w, email) {
		return
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 503, fmt.Errorf("खाता अभी नहीं बन सका। फिर कोशिश करें।"))
		return
	}
	userID := id("u")
	duplicate := false
	err = a.Store.Update(r.Context(), func(s *State) error {
		for _, u := range s.Users {
			if !u.Deleted && u.Email == email {
				duplicate = true
				return fmt.Errorf("email already registered")
			}
		}
		s.Users[userID] = &User{ID: userID, Email: email, Created: a.Now()}
		s.PasswordCredentials[userID] = &PasswordCredential{Hash: string(digest)}
		audit(s, userID, "auth.registered", userID)
		return nil
	})
	if duplicate {
		fail(w, 409, fmt.Errorf("इस ईमेल से खाता पहले से है। कृपया लॉग इन करें।"))
		return
	}
	if err != nil {
		fail(w, 503, fmt.Errorf("खाता अभी नहीं बन सका। फिर कोशिश करें।"))
		return
	}
	if err = a.setSession(w, r, userID, "customer"); err != nil {
		fail(w, 503, fmt.Errorf("session could not be created"))
	}
}

// loginCustomer checks the stored password hash and creates the existing protected customer session.
func (a *App) loginCustomer(w http.ResponseWriter, r *http.Request) {
	var in customerCredentials
	if !decode(w, r, &in) {
		return
	}
	email, err := normalizeCustomerEmail(in.Email)
	if err != nil {
		fail(w, 401, fmt.Errorf("ईमेल या पासवर्ड सही नहीं है।"))
		return
	}
	if !a.customerAuthAllowed(w, email) {
		return
	}
	userID := ""
	digest := customerDummyHash
	err = a.Store.ViewKinds(r.Context(), []string{"users", "password_credentials"}, func(s *State) error {
		for _, u := range s.Users {
			if !u.Deleted && u.Email == email {
				if credential := s.PasswordCredentials[u.ID]; credential != nil {
					userID, digest = u.ID, []byte(credential.Hash)
				}
				break
			}
		}
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("login storage unavailable"))
		return
	}
	passwordErr := bcrypt.CompareHashAndPassword(digest, []byte(in.Password))
	if userID == "" || passwordErr != nil || len(in.Password) > 72 {
		fail(w, 401, fmt.Errorf("ईमेल या पासवर्ड सही नहीं है।"))
		return
	}
	if err = a.setSession(w, r, userID, "customer"); err != nil {
		fail(w, 503, fmt.Errorf("session could not be created"))
	}
}

// setCustomerCredentials links a signed-in legacy account once, preserving its ID and owned records.
func (a *App) setCustomerCredentials(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in customerCredentials
	if !decode(w, r, &in) {
		return
	}
	email, err := normalizeCustomerEmail(in.Email)
	if err == nil {
		err = customerPassword(in.Password)
	}
	if err != nil {
		fail(w, 422, err)
		return
	}
	if !a.customerAuthAllowed(w, email) {
		return
	}
	digest, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		fail(w, 503, fmt.Errorf("खाता अभी नहीं बदल सका। फिर कोशिश करें।"))
		return
	}
	conflict := false
	err = a.Store.Update(r.Context(), func(s *State) error {
		u := s.Users[sess.UserID]
		if u == nil || u.Deleted {
			return fmt.Errorf("account unavailable")
		}
		if s.PasswordCredentials[u.ID] != nil {
			conflict = true
			return fmt.Errorf("इस खाते के लिए ईमेल और पासवर्ड पहले से सेट हैं।")
		}
		for _, other := range s.Users {
			if !other.Deleted && other.ID != u.ID && other.Email == email {
				conflict = true
				return fmt.Errorf("इस ईमेल से खाता पहले से है। कृपया दूसरा ईमेल लिखें।")
			}
		}
		u.Email = email
		s.PasswordCredentials[u.ID] = &PasswordCredential{Hash: string(digest)}
		audit(s, u.ID, "auth.credentials-set", u.ID)
		return nil
	})
	if conflict {
		fail(w, 409, err)
		return
	}
	if err != nil {
		fail(w, 503, fmt.Errorf("खाता अभी नहीं बदल सका। फिर कोशिश करें।"))
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

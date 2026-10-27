package platform

import (
	"context"
	"crypto/rand"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var videoID = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

type bucket struct {
	Count int
	Until time.Time
}
type limiter struct {
	mu      sync.Mutex
	entries map[string]bucket
}

// newLimiter creates an empty, process-local request-rate limiter.
func newLimiter() *limiter { return &limiter{entries: map[string]bucket{}} }

// allow counts a request in its keyed time window and reports whether the limit permits it.
func (l *limiter) allow(key string, n int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if len(l.entries) > 10000 {
		for k, v := range l.entries {
			if now.After(v.Until) {
				delete(l.entries, k)
			}
		}
	}
	v := l.entries[key]
	if now.After(v.Until) {
		v = bucket{Until: now.Add(window)}
	}
	v.Count++
	l.entries[key] = v
	return v.Count <= n
}

// jsonOut writes a value as a UTF-8 JSON response.
func jsonOut(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(v)
}

// fail writes an HTTP status and the shared JSON error envelope.
func fail(w http.ResponseWriter, status int, e error) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	jsonOut(w, map[string]string{"error": e.Error()})
}

// decode accepts a bounded JSON request body and rejects unsupported content types or malformed input.
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fail(w, 415, fmt.Errorf("JSON required"))
		return false
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		fail(w, 400, fmt.Errorf("invalid request fields"))
		return false
	}
	if d.Decode(&struct{}{}) != io.EOF {
		fail(w, 400, fmt.Errorf("one JSON object required"))
		return false
	}
	return true
}

// session resolves the separate customer or administrator cookie and rejects expired or unavailable accounts.
func (a *App) session(r *http.Request, admin bool) (*Session, error) {
	name := "m4y_session"
	if admin {
		name = "m4y_admin"
	}
	cookie, e := r.Cookie(name)
	if e != nil {
		return nil, fmt.Errorf("कृपया लॉग इन करें")
	}
	var session *Session
	e = a.Store.ViewKinds(r.Context(), []string{"sessions", "users"}, func(s *State) error {
		v := s.Sessions[hash(cookie.Value)]
		if v == nil || !v.Expires.After(a.Now()) || admin && v.Role != "admin" || !admin && v.Role != "customer" {
			return fmt.Errorf("session expired")
		}
		if !admin {
			u := s.Users[v.UserID]
			if u == nil || u.Deleted {
				return fmt.Errorf("account unavailable")
			}
		}
		session = v
		return nil
	})
	return session, e
}

// guard requires the requested session role and checks CSRF protection for state-changing requests.
func (a *App) guard(w http.ResponseWriter, r *http.Request, admin bool) (*Session, bool) {
	s, e := a.session(r, admin)
	if e != nil {
		fail(w, 401, e)
		return nil, false
	}
	if r.Method != "GET" && r.Method != "HEAD" && !same(r.Header.Get("X-CSRF-Token"), s.CSRF) {
		fail(w, 403, fmt.Errorf("security token expired; refresh and retry"))
		return nil, false
	}
	return s, true
}

// setSession persists a hashed session token and issues the appropriate customer or administrator cookie.
func (a *App) setSession(w http.ResponseWriter, r *http.Request, user, role string) error {
	token := id("session")
	expires := a.Now().Add(7 * 24 * time.Hour)
	name := "m4y_session"
	if role == "admin" {
		expires = a.Now().Add(time.Hour)
		name = "m4y_admin"
	}
	sess := &Session{user, role, id("csrf"), expires}
	e := a.Store.Update(r.Context(), func(s *State) error {
		if old, e := r.Cookie(name); e == nil {
			delete(s.Sessions, hash(old.Value))
		}
		s.Sessions[hash(token)] = sess
		return nil
	})
	if e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: token, Path: "/", HttpOnly: true, Secure: a.Config.Production, SameSite: http.SameSiteLaxMode, Expires: expires})
	jsonOut(w, map[string]any{"user_id": user, "csrf": sess.CSRF, "role": role})
	return nil
}

// Handler registers API routes and wraps them with origin, request-limit and security-header handling.
func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if e := a.Store.Ping(ctx); e != nil {
			fail(w, 503, fmt.Errorf("storage unavailable"))
			return
		}
		jsonOut(w, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /api/config", a.publicConfig)
	mux.HandleFunc("GET /api/video-links/{slug}", a.publicVideoLink)
	mux.HandleFunc("GET /api/admin/video-links", a.adminVideoLinks)
	mux.HandleFunc("PUT /api/admin/video-links/{slug}", a.saveVideoLink)
	mux.HandleFunc("GET /api/panchang", a.dailyPanchang)
	mux.HandleFunc("GET /api/panchang/calendar", a.panchangCalendar)
	mux.HandleFunc("GET /api/panchang/vivah", a.vivahMuhurat)
	mux.HandleFunc("GET /api/sky/{tool}", a.skyTools)
	mux.HandleFunc("GET /api/places", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if len([]rune(q)) > 100 {
			fail(w, 400, fmt.Errorf("Location search must be 100 characters or fewer"))
			return
		}
		jsonOut(w, searchPlaces(q))
	})
	mux.HandleFunc("GET /api/places/{id}", func(w http.ResponseWriter, r *http.Request) {
		p, ok := findPlace(r.PathValue("id"))
		if !ok {
			fail(w, 404, fmt.Errorf("Location not found"))
			return
		}
		jsonOut(w, p)
	})
	mux.HandleFunc("POST /api/kundli", a.guestReport)
	mux.HandleFunc("POST /api/kundli/dasha", a.dashaBranch)
	mux.HandleFunc("POST /api/kundli/saturn", a.kundliSaturn)
	mux.HandleFunc("POST /api/kundli/match", a.matching)
	mux.HandleFunc("POST /api/kundli/match/pdf", a.matchingPDF)
	mux.HandleFunc("GET /api/kundli/match/profiles", a.matchingProfiles)
	mux.HandleFunc("POST /api/auth/request", a.requestOTP)
	mux.HandleFunc("POST /api/auth/verify", a.verifyOTP)
	mux.HandleFunc("GET /api/me", a.me)
	mux.HandleFunc("POST /api/logout", a.logout)
	mux.HandleFunc("POST /api/reports", a.saveReport)
	mux.HandleFunc("GET /api/reports/{id}", a.getReport)
	mux.HandleFunc("DELETE /api/reports/{id}", a.deleteReport)
	mux.HandleFunc("PUT /api/privacy", a.privacy)
	mux.HandleFunc("DELETE /api/account", a.deleteAccount)
	mux.HandleFunc("GET /api/account/export", a.exportAccount)
	mux.HandleFunc("GET /api/slots", a.getSlots)
	mux.HandleFunc("GET /api/shop", a.shopCatalog)
	mux.HandleFunc("GET /api/shop/images/{name}", a.shopImage)
	mux.HandleFunc("POST /api/shop/checkout", a.shopCheckoutHTTP)
	mux.HandleFunc("POST /api/shop/orders/{id}/cancel", a.shopCancelHTTP)
	mux.HandleFunc("GET /api/admin/shop", a.adminShop)
	mux.HandleFunc("PUT /api/admin/shop/settings", a.adminShopSettings)
	mux.HandleFunc("POST /api/admin/shop/products", a.adminGemstone)
	mux.HandleFunc("POST /api/admin/shop/images", a.adminShopImage)
	mux.HandleFunc("POST /api/admin/shop/orders/{id}/fulfilment", a.adminShopFulfilment)
	mux.HandleFunc("POST /api/orders", a.checkoutHTTP)
	mux.HandleFunc("POST /api/orders/{id}/verify", a.verifyHTTP)
	mux.HandleFunc("POST /api/orders/{id}/simulate", a.simulateHTTP)
	mux.HandleFunc("GET /api/orders/{id}/pdf", a.downloadPDF)
	mux.HandleFunc("POST /api/bookings/{id}", a.changeBookingHTTP)
	mux.HandleFunc("POST /api/payments/webhook", a.webhook)
	mux.HandleFunc("POST /api/events", a.event)
	mux.HandleFunc("POST /api/admin/login", a.adminLogin)
	mux.HandleFunc("GET /api/admin", a.adminOverview)
	mux.HandleFunc("PUT /api/admin/settings", a.adminSettings)
	mux.HandleFunc("POST /api/admin/rules", a.adminRule)
	mux.HandleFunc("POST /api/admin/jobs/retry", a.adminRetry)
	mux.HandleFunc("POST /api/admin/reconcile", a.adminReconcile)
	mux.HandleFunc("POST /api/admin/refund", a.adminRefund)
	mux.HandleFunc("POST /api/admin/expenses", a.adminExpense)
	mux.HandleFunc("POST /api/admin/expenses/{id}/approve", a.approveExpense)
	mux.HandleFunc("POST /api/admin/receipts", a.uploadReceipt)
	mux.HandleFunc("GET /api/admin/receipts/{id}", a.getReceipt)
	mux.HandleFunc("POST /api/admin/distributions", a.adminDistribution)
	mux.HandleFunc("PUT /api/admin/month", a.adminMonth)
	mux.HandleFunc("GET /api/admin/finance.csv", a.financeCSV)
	mux.HandleFunc("POST /api/admin/bookings/{id}", a.adminBooking)
	mux.HandleFunc("GET /fonts/noto.ttf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "font/ttf")
		http.ServeFile(w, r, filepath.Join(filepath.Dir(a.Config.FontPath), "NotoSansDevanagari.ttf"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			fail(w, 404, fmt.Errorf("endpoint not found"))
			return
		}
		p := filepath.Join("web", "dist", filepath.Clean("/"+r.URL.Path))
		if stat, e := os.Stat(p); e == nil && !stat.IsDir() {
			http.ServeFile(w, r, p)
			return
		}
		http.ServeFile(w, r, "web/dist/index.html")
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://checkout.razorpay.com; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: https://i.ytimg.com; connect-src 'self' https://api.razorpay.com; frame-src https://www.youtube-nocookie.com https://api.razorpay.com https://checkout.razorpay.com; object-src 'none'; base-uri 'self'; frame-ancestors 'none'")
		if a.Config.Production {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") {
			w.Header().Set("Cache-Control", "no-store")
			w.Header().Set("X-Robots-Tag", "noindex, nofollow")
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.URL.Path != "/api/payments/webhook" {
			if origin := r.Header.Get("Origin"); origin != "" && !a.allowedOrigin(origin) {
				fail(w, 403, fmt.Errorf("origin not allowed"))
				return
			}
		}
		ip, _, e := net.SplitHostPort(r.RemoteAddr)
		if e != nil {
			ip = r.RemoteAddr
		}
		if !a.limits.allow("all:"+ip, 600, time.Minute) {
			fail(w, 429, fmt.Errorf("too many requests"))
			return
		}
		if r.URL.Path == "/api/kundli/dasha" && !a.limits.allow("dasha:"+ip, 60, time.Minute) {
			fail(w, 429, fmt.Errorf("Too many Dasha requests; please try again shortly."))
			return
		}
		if r.URL.Path == "/api/shop/checkout" && !a.limits.allow("shop-write:"+ip, 20, time.Minute) {
			fail(w, 429, fmt.Errorf("Please wait before submitting another gemstone request."))
			return
		}
		if r.URL.Path == "/api/kundli/saturn" && !a.limits.allow("saturn:"+ip, 8, time.Minute) {
			fail(w, 429, fmt.Errorf("Please wait before calculating another Saturn timeline."))
			return
		}
		if r.URL.Path == "/api/panchang/vivah" && !a.limits.allow("vivah:"+ip, 12, time.Minute) {
			fail(w, 429, fmt.Errorf("Please wait before calculating another marriage month."))
			return
		}
		if (r.URL.Path == "/api/kundli" || r.URL.Path == "/api/kundli/match" || r.URL.Path == "/api/kundli/match/pdf") && !a.limits.allow("calc:"+ip, 20, time.Minute) {
			fail(w, 429, fmt.Errorf("कृपया थोड़ी देर बाद प्रयास करें"))
			return
		}
		if (r.URL.Path == "/api/kundli/match" || r.URL.Path == "/api/kundli/match/pdf") && !a.limits.allow("match:"+ip, 10, time.Minute) {
			fail(w, 429, fmt.Errorf("कृपया थोड़ी देर बाद प्रयास करें"))
			return
		}
		if r.URL.Path == "/api/kundli/match/pdf" && !a.limits.allow("matching-pdf:"+ip, 3, time.Minute) {
			fail(w, 429, fmt.Errorf("Please wait before downloading another comparison."))
			return
		}
		if (strings.HasPrefix(r.URL.Path, "/api/panchang") || strings.HasPrefix(r.URL.Path, "/api/sky/")) && !a.limits.allow("daily:"+ip, 60, time.Minute) {
			fail(w, 429, fmt.Errorf("कृपया थोड़ी देर बाद प्रयास करें"))
			return
		}
		if strings.HasPrefix(r.URL.Path, "/api/auth/") || r.URL.Path == "/api/admin/login" {
			if !a.limits.allow("auth:"+ip, 20, 15*time.Minute) {
				fail(w, 429, fmt.Errorf("बहुत अधिक प्रयास; 15 मिनट बाद आएँ"))
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

// allowedOrigin permits the configured site origin and the explicit development loopback origins.
func (a *App) allowedOrigin(origin string) bool {
	if origin == a.Config.Origin {
		return true
	}
	if a.Config.Production {
		return false
	}
	u, e := url.Parse(origin)
	if e != nil || u.Scheme != "http" {
		return false
	}
	host := u.Hostname()
	return (host == "localhost" || host == "127.0.0.1" || host == "::1") && (u.Port() == "5173" || u.Port() == "8080")
}

// publicConfig reads only settings for public content and checkout availability;
// its bounded snapshot excludes unrelated reports, orders and analytics records.
func (a *App) publicConfig(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	err := a.Store.ViewKinds(ctx, []string{"settings"}, func(s *State) error {
		jsonOut(w, map[string]any{"simulation": a.Config.PaymentMode == "simulation", "development": !a.Config.Production, "otp_simulation": a.Config.OTPMode == "simulation", "settings": s.Settings, "video_catalog": FeaturedVideoCatalog(), "pdf_blockers": blockers(s.Settings, a.Config, "pdf"), "booking_blockers": blockers(s.Settings, a.Config, "consultation"), "duration": duration(s.Settings, a.Config.Production), "duration_demo": s.Settings.Duration == 0, "payment_key": a.Config.KeyID, "storage": map[bool]string{true: "MongoDB Atlas", false: "local development file"}[a.Store.client != nil]})
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Website settings are temporarily unavailable. Please try again."))
	}
}

// guestReport calculates a guest Kundli after input and licence checks without saving a customer record.
func (a *App) guestReport(w http.ResponseWriter, r *http.Request) {
	var in Birth
	if !decode(w, r, &in) {
		return
	}
	var rules map[string]Rule
	if e := a.Store.View(r.Context(), func(s *State) error {
		if a.Config.Production && !s.Settings.LicenceConfirmed {
			return fmt.Errorf("licence not confirmed")
		}
		rules = s.Rules
		return nil
	}); e != nil {
		fail(w, 503, fmt.Errorf("storage unavailable"))
		return
	}
	out, e := calculate(a.Config, in, rules, a.Now())
	if e != nil {
		fail(w, 422, e)
		return
	}
	jsonOut(w, out)
}

// requestOTP creates a rate-limited phone verification challenge and sends it through the configured gateway.
func (a *App) requestOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Phone string `json:"phone"`
	}
	if !decode(w, r, &in) {
		return
	}
	phone, e := normalizePhone(in.Phone)
	if e != nil {
		fail(w, 422, e)
		return
	}
	n, e := rand.Int(rand.Reader, big.NewInt(1000000))
	if e != nil {
		fail(w, 503, fmt.Errorf("secure random unavailable"))
		return
	}
	code := fmt.Sprintf("%06d", n.Int64())
	key := id("otp")
	digest, e := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
	if e != nil {
		fail(w, 503, fmt.Errorf("OTP unavailable"))
		return
	}
	e = a.Store.Update(r.Context(), func(s *State) error {
		count := 0
		for _, o := range s.OTPs {
			if o.Phone == phone && o.Expires.After(a.Now().Add(-10*time.Minute)) {
				count++
			}
		}
		if count >= 5 {
			return fmt.Errorf("इस नंबर के लिए अधिक OTP माँगे गए हैं; 15 मिनट बाद आएँ")
		}
		s.OTPs[key] = &OTP{Phone: phone, Hash: string(digest), Expires: a.Now().Add(5 * time.Minute)}
		return nil
	})
	if e != nil {
		fail(w, 429, e)
		return
	}
	if e = a.OTP.Send(r.Context(), phone, code); e != nil {
		_ = a.Store.Update(r.Context(), func(s *State) error { delete(s.OTPs, key); return nil })
		fail(w, 503, e)
		return
	}
	out := map[string]any{"challenge": key, "expires_in": 300}
	if !a.Config.Production && a.Config.OTPMode == "simulation" {
		out["demo_code"] = code
	}
	jsonOut(w, out)
}

// verifyOTP consumes a valid, unexpired OTP challenge and creates a customer session.
func (a *App) verifyOTP(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if !decode(w, r, &in) {
		return
	}
	var user string
	var denial error
	e := a.Store.Update(r.Context(), func(s *State) error {
		o := s.OTPs[in.Challenge]
		if o == nil || o.Used || !o.Expires.After(a.Now()) || o.Attempts >= 5 {
			denial = fmt.Errorf("OTP समाप्त या पहले उपयोग हुआ है")
			return nil
		}
		o.Attempts++
		if bcrypt.CompareHashAndPassword([]byte(o.Hash), []byte(in.Code)) != nil {
			denial = fmt.Errorf("OTP सही नहीं है")
			return nil
		}
		o.Used = true
		for _, u := range s.Users {
			if u.Phone == o.Phone && !u.Deleted {
				user = u.ID
				break
			}
		}
		if user == "" {
			u := &User{ID: id("u"), Phone: o.Phone, Created: a.Now()}
			s.Users[u.ID] = u
			user = u.ID
		}
		audit(s, user, "auth.verified", user)
		return nil
	})
	if e != nil {
		fail(w, 503, fmt.Errorf("login storage unavailable"))
		return
	}
	if denial != nil {
		fail(w, 401, denial)
		return
	}
	if e = a.setSession(w, r, user, "customer"); e != nil {
		fail(w, 503, fmt.Errorf("session could not be created"))
	}
}

// me returns the signed-in customer's profile, owned reports, orders and bookings.
func (a *App) me(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	_ = a.Store.View(r.Context(), func(s *State) error {
		reports := []*Report{}
		orders := []Order{}
		bookings := []*Booking{}
		jobs := []Job{}
		for _, v := range s.Reports {
			if v.UserID == sess.UserID {
				reports = append(reports, v)
			}
		}
		for _, v := range s.Orders {
			if v.UserID == sess.UserID {
				copy := *v
				copy.Snapshot = nil
				copy.RequestKey = ""
				orders = append(orders, copy)
				for _, j := range s.Jobs {
					if j.Ref == v.ID && j.Kind == "pdf" {
						jobs = append(jobs, *j)
					}
				}
			}
		}
		for _, v := range s.Bookings {
			if v.UserID == sess.UserID {
				bookings = append(bookings, v)
			}
		}
		sort.Slice(reports, func(i, j int) bool { return reports[i].Created.After(reports[j].Created) })
		sort.Slice(orders, func(i, j int) bool { return orders[i].Created.After(orders[j].Created) })
		jsonOut(w, map[string]any{"user": s.Users[sess.UserID], "csrf": sess.CSRF, "reports": reports, "orders": orders, "bookings": bookings, "pdf_jobs": jobs})
		return nil
	})
}

// logout revokes the selected session and expires its matching cookie.
func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	admin := r.URL.Query().Get("admin") == "true"
	sess, ok := a.guard(w, r, admin)
	if !ok {
		return
	}
	name := "m4y_session"
	if admin {
		name = "m4y_admin"
	}
	cookie, _ := r.Cookie(name)
	e := a.Store.Update(r.Context(), func(s *State) error {
		delete(s.Sessions, hash(cookie.Value))
		audit(s, sess.UserID, "auth.logout", "")
		return nil
	})
	if e != nil {
		fail(w, 503, e)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", HttpOnly: true, Secure: a.Config.Production, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	jsonOut(w, map[string]bool{"ok": true})
}

// saveReport recalculates submitted birth details and saves the report under the signed-in customer.
func (a *App) saveReport(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var b Birth
	if !decode(w, r, &b) {
		return
	}
	var rules map[string]Rule
	_ = a.Store.View(r.Context(), func(s *State) error { rules = s.Rules; return nil })
	out, e := calculate(a.Config, b, rules, a.Now())
	if e != nil {
		fail(w, 422, e)
		return
	}
	out.UserID = sess.UserID
	e = a.Store.Update(r.Context(), func(s *State) error {
		count := 0
		for _, r := range s.Reports {
			if r.UserID == sess.UserID {
				count++
			}
		}
		if count >= 50 {
			return fmt.Errorf("50 saved reports limit; remove an older report")
		}
		s.Reports[out.ID] = out
		audit(s, sess.UserID, "report.saved", out.ID)
		return nil
	})
	if e != nil {
		fail(w, 422, e)
		return
	}
	jsonOut(w, out)
}

// getReport returns a saved report only to its owning customer.
func (a *App) getReport(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	e := a.Store.View(r.Context(), func(s *State) error {
		v := s.Reports[r.PathValue("id")]
		if v == nil || v.UserID != sess.UserID {
			return fmt.Errorf("report not found")
		}
		jsonOut(w, v)
		return nil
	})
	if e != nil {
		fail(w, 404, e)
	}
}

// deleteReport removes an owned saved report after customer authentication and CSRF checks.
func (a *App) deleteReport(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		v := s.Reports[r.PathValue("id")]
		if v == nil || v.UserID != sess.UserID {
			return fmt.Errorf("report not found")
		}
		delete(s.Reports, v.ID)
		audit(s, sess.UserID, "report.deleted", v.ID)
		return nil
	})
	if e != nil {
		fail(w, 404, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// privacy updates the customer's optional consent preferences.
func (a *App) privacy(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in struct {
		Marketing bool `json:"marketing"`
		Analytics bool `json:"analytics"`
	}
	if !decode(w, r, &in) {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		s.Users[sess.UserID].Marketing = in.Marketing
		s.Users[sess.UserID].Analytics = in.Analytics
		audit(s, sess.UserID, "privacy.marketing-consent-updated", strconv.FormatBool(in.Marketing))
		return nil
	})
	if e != nil {
		fail(w, 503, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// deleteAccount checks outstanding business obligations, removes customer data and schedules file cleanup.
func (a *App) deleteAccount(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		for _, b := range s.Bookings {
			if b.UserID == sess.UserID && (b.Status == "confirmed" || b.Status == "meeting_pending" || b.Status == "needs_resolution" || b.Status == "cancel_pending") {
				return fmt.Errorf("पहले सक्रिय परामर्श रद्द करें और सहायता से निपटारा पूरा करें")
			}
		}
		for _, o := range s.Orders {
			if o.UserID == sess.UserID && o.Gemstone != nil && o.PaidAt != nil && o.Status != "refunded" && o.Gemstone.Fulfilment != "delivered" {
				return fmt.Errorf("Please complete or resolve your gemstone delivery before deleting your account.")
			}
			if o.UserID == sess.UserID && o.Status == "pending" {
				return fmt.Errorf("पहले लंबित भुगतान का निपटारा करें")
			}
		}
		u := s.Users[sess.UserID]
		for k, v := range s.OTPs {
			if v.Phone == u.Phone {
				delete(s.OTPs, k)
			}
		}
		u.Phone = ""
		u.Marketing = false
		u.Deleted = true
		for k, v := range s.Reports {
			if v.UserID == u.ID {
				delete(s.Reports, k)
			}
		}
		for _, o := range s.Orders {
			if o.UserID == u.ID {
				if o.Gemstone != nil {
					o.Gemstone.Address = DeliveryAddress{}
				}
				o.Snapshot = nil
				enqueue(s, "privacy_file", o.ID, 1, "")
			}
		}
		for k, v := range s.Sessions {
			if v.UserID == u.ID {
				delete(s.Sessions, k)
			}
		}
		audit(s, u.ID, "privacy.deletion", u.ID)
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]string{"status": "deleted", "note": "Private profiles removed; minimum accounting records retained pending legally approved retention policy."})
}

// exportAccount returns the authenticated customer's account data through the existing ownership-filtered view.
func (a *App) exportAccount(w http.ResponseWriter, r *http.Request) { a.me(w, r) }

// getSlots returns currently available consultation times and their scheduling timezone.
func (a *App) getSlots(w http.ResponseWriter, r *http.Request) {
	_ = a.Store.View(r.Context(), func(s *State) error {
		jsonOut(w, map[string]any{"slots": slots(s, a.Now(), a.Config.Production), "duration": duration(s.Settings, a.Config.Production), "simulation": !a.Config.Production, "zone": s.Settings.Zone, "change_hours": s.Settings.ChangeHours})
		return nil
	})
}

// checkoutHTTP authenticates the customer and delegates PDF/consultation order creation to Checkout.
func (a *App) checkoutHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in CheckoutRequest
	if !decode(w, r, &in) {
		return
	}
	out, e := a.Checkout(r.Context(), sess.UserID, in)
	if e != nil {
		fail(w, 409, e)
		return
	}
	copy := *out
	copy.Snapshot = nil
	jsonOut(w, copy)
}

// verifyHTTP authenticates the customer and verifies a checkout payment against the provider.
func (a *App) verifyHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in struct {
		PaymentID string `json:"payment_id"`
		Signature string `json:"signature"`
	}
	if !decode(w, r, &in) {
		return
	}
	if a.Config.PaymentMode != "razorpay" {
		fail(w, 409, fmt.Errorf("Razorpay is not configured"))
		return
	}
	out, e := a.VerifyPayment(r.Context(), r.PathValue("id"), in.PaymentID, in.Signature, sess.UserID)
	if e != nil {
		fail(w, 400, e)
		return
	}
	copy := *out
	copy.Snapshot = nil
	jsonOut(w, copy)
}

// simulateHTTP confirms a development payment explicitly; the route is unavailable in production.
func (a *App) simulateHTTP(w http.ResponseWriter, r *http.Request) {
	if a.Config.Production || a.Config.PaymentMode != "simulation" {
		fail(w, 404, fmt.Errorf("not found"))
		return
	}
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in struct {
		Success bool `json:"success"`
	}
	if !decode(w, r, &in) {
		return
	}
	var out *Order
	e := a.Store.Update(r.Context(), func(s *State) error {
		o := s.Orders[r.PathValue("id")]
		if o == nil || o.UserID != sess.UserID || !o.Demo {
			return fmt.Errorf("order not found")
		}
		status := "captured"
		if !in.Success {
			status = "failed"
		}
		p := Payment{ID: "sim_pay_" + o.ID, OrderID: o.ProviderID, Status: status, Amount: o.Amount, Currency: "INR"}
		if e := applyPayment(s, o, p, a.Now()); e != nil {
			return e
		}
		out = o
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	copy := *out
	copy.Snapshot = nil
	jsonOut(w, copy)
}

// downloadPDF serves the original file for legacy links, or the selected-language presentation of the paid snapshot.
func (a *App) downloadPDF(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	selectedLanguage := r.URL.Query().Get("language")
	language := selectedLanguage
	if language == "" {
		language = "hi" // Preserve the original download URL and existing Hindi file.
	}
	if language != "en" && language != "hi" {
		fail(w, 400, fmt.Errorf("Choose English or Hindi for the remedy PDF."))
		return
	}
	var purchased *Order
	e := a.Store.View(r.Context(), func(s *State) error {
		o := s.Orders[r.PathValue("id")]
		if o == nil || o.UserID != sess.UserID || o.Status != "paid" || o.Product != "pdf" {
			return fmt.Errorf("PDF not found")
		}
		j := s.Jobs["pdf:"+o.ID+":1"]
		if j == nil || j.Status != "done" {
			return fmt.Errorf("PDF तैयार हो रहा है; बाद में पुनः प्रयास करें")
		}
		purchased = o
		return nil
	})
	if e != nil {
		fail(w, 404, e)
		return
	}
	if selectedLanguage != "" {
		a.downloadRemedyPDF(w, r, purchased, sess.UserID, language)
		return
	}
	p := filepath.Join(a.Config.DataDir, "pdf", r.PathValue("id")+".pdf")
	if _, e = os.Stat(p); e != nil {
		fail(w, 503, fmt.Errorf("PDF storage unavailable; admin can regenerate without repurchase"))
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", `attachment; filename="Mishra4You-report.pdf"`)
	http.ServeFile(w, r, p)
}

// changeBookingHTTP validates a customer booking-change request before applying the scheduling rules.
func (a *App) changeBookingHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in struct {
		Action string `json:"action"`
		Slot   string `json:"slot"`
	}
	if !decode(w, r, &in) {
		return
	}
	if e := a.ChangeBooking(r.Context(), sess.UserID, r.PathValue("id"), in.Action, in.Slot, false); e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// webhook verifies Razorpay webhook authenticity and applies payment updates with replay protection.
func (a *App) webhook(w http.ResponseWriter, r *http.Request) {
	if a.Config.PaymentMode != "razorpay" {
		fail(w, 404, fmt.Errorf("not configured"))
		return
	}
	body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if e != nil || !verifySignature(a.Config.WebhookSecret, string(body), r.Header.Get("X-Razorpay-Signature")) {
		fail(w, 400, fmt.Errorf("invalid webhook signature"))
		return
	}
	eventID := r.Header.Get("X-Razorpay-Event-Id")
	if eventID == "" || len(eventID) > 200 {
		fail(w, 400, fmt.Errorf("event ID required"))
		return
	}
	var in struct {
		Event   string `json:"event"`
		Payload struct {
			Payment struct {
				Entity Payment `json:"entity"`
			} `json:"payment"`
			Refund struct {
				Entity struct {
					PaymentID string `json:"payment_id"`
				} `json:"entity"`
			} `json:"refund"`
		} `json:"payload"`
	}
	if json.Unmarshal(body, &in) != nil {
		fail(w, 400, fmt.Errorf("invalid event"))
		return
	}
	p := in.Payload.Payment.Entity
	if in.Payload.Refund.Entity.PaymentID != "" {
		p, e = a.Pay.Fetch(r.Context(), in.Payload.Refund.Entity.PaymentID)
		if e != nil {
			fail(w, 503, e)
			return
		}
	}
	e = a.Store.Update(r.Context(), func(s *State) error {
		if s.Webhooks[eventID] {
			return nil
		}
		if p.OrderID != "" {
			found := false
			for _, o := range s.Orders {
				if o.ProviderID == p.OrderID {
					found = true
					if e := applyPayment(s, o, p, a.Now()); e != nil {
						return e
					}
				}
			}
			if !found {
				return fmt.Errorf("provider order not registered yet; retry")
			}
		}
		s.Webhooks[eventID] = true
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

var eventNames = map[string]bool{"landing_visit": true, "form_started": true, "kundli_generated": true, "offer_viewed": true, "checkout_started": true}
var attribution = regexp.MustCompile(`^[a-zA-Z0-9_.-]{0,80}$`)

// event records an allowlisted analytics event only when the request includes consent.
func (a *App) event(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Consent  bool   `json:"consent"`
		Name     string `json:"name"`
		Product  string `json:"product"`
		Source   string `json:"source"`
		Medium   string `json:"medium"`
		Campaign string `json:"campaign"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !in.Consent || !eventNames[in.Name] || in.Product != "" && in.Product != "pdf" && in.Product != "consultation" || !attribution.MatchString(in.Source) || !attribution.MatchString(in.Medium) || !attribution.MatchString(in.Campaign) {
		fail(w, 400, fmt.Errorf("unsupported analytics event or missing consent"))
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		s.Events = append(s.Events, Event{in.Name, in.Product, in.Source, in.Medium, in.Campaign, a.Now()})
		if len(s.Events) > 100000 {
			s.Events = s.Events[len(s.Events)-100000:]
		}
		return nil
	})
	if e != nil {
		fail(w, 503, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// adminLogin checks administrator credentials and TOTP before creating a separate administrator session.
func (a *App) adminLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Password string `json:"password"`
		TOTP     string `json:"totp"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !same(in.Password, a.Config.AdminPassword) || a.Config.Production && !totp(a.Config.AdminTOTP, in.TOTP, a.Now()) {
		fail(w, 401, fmt.Errorf("invalid admin credentials"))
		return
	}
	if a.Config.Production {
		key := "admin-totp:" + hash(in.TOTP)
		var reused bool
		e := a.Store.Update(r.Context(), func(s *State) error {
			reused = s.OTPs[key] != nil && s.OTPs[key].Expires.After(a.Now())
			if !reused {
				s.OTPs[key] = &OTP{Used: true, Expires: a.Now().Add(90 * time.Second)}
			}
			return nil
		})
		if e != nil || reused {
			fail(w, 401, fmt.Errorf("TOTP already used; wait for a new code"))
			return
		}
	}
	if e := a.setSession(w, r, "admin", "admin"); e != nil {
		fail(w, 503, e)
	}
}

// adminOverview returns administrator operational data and finance totals for the selected business month.
func (a *App) adminOverview(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	_ = a.Store.View(r.Context(), func(s *State) error {
		month := r.URL.Query().Get("month")
		if month == "" {
			loc, err := time.LoadLocation(s.Settings.Zone)
			if err != nil || s.Settings.Zone == "" {
				fail(w, 422, fmt.Errorf("configured business timezone is invalid; select an IANA timezone in settings"))
				return nil
			}
			month = a.Now().In(loc).Format("2006-01")
		}
		demo := !a.Config.Production && r.URL.Query().Get("demo") == "true"
		users := []map[string]any{}
		for _, u := range s.Users {
			phone := "deleted"
			if len(u.Phone) > 4 {
				phone = "••••••" + u.Phone[len(u.Phone)-4:]
			}
			users = append(users, map[string]any{"id": u.ID, "phone": phone, "created": u.Created, "marketing": u.Marketing, "deleted": u.Deleted})
		}
		orders := []Order{}
		for _, o := range s.Orders {
			copy := *o
			copy.Snapshot = nil
			orders = append(orders, copy)
		}
		profiles := []map[string]any{}
		for _, v := range s.Reports {
			profiles = append(profiles, map[string]any{"id": v.ID, "user_id": v.UserID, "created": v.Created, "place": v.Place.Name})
		}
		jsonOut(w, map[string]any{"csrf": sess.CSRF, "settings": s.Settings, "rules": s.Rules, "users": users, "profiles": profiles, "orders": orders, "bookings": s.Bookings, "jobs": s.Jobs, "expenses": s.Expenses, "distributions": s.Distributions, "month": s.Reconciliations[month], "finance": finances(s, month, demo), "audit": s.Audits, "analytics": s.Events, "pdf_blockers": blockers(s.Settings, a.Config, "pdf"), "booking_blockers": blockers(s.Settings, a.Config, "consultation")})
		return nil
	})
}

// adminSettings validates and saves administrator settings with an audit entry.
func (a *App) adminSettings(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Settings
	if !decode(w, r, &in) {
		return
	}
	if e := validateSettings(in); e != nil {
		fail(w, 422, e)
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error { s.Settings = in; audit(s, sess.UserID, "settings.updated", ""); return nil })
	if e != nil {
		fail(w, 503, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// adminRule creates or updates versioned interpretation rules while preserving approved versions.
func (a *App) adminRule(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Rule
	if !decode(w, r, &in) {
		return
	}
	if len(in.ID) < 3 || len(in.ID) > 80 || in.Version < 1 || len(in.Hindi) < 10 || in.Sign < 0 || in.Sign > 12 || in.Status != "draft" && in.Status != "approved" && in.Status != "retired" {
		fail(w, 422, fmt.Errorf("invalid rule; ID, version and interpretation required"))
		return
	}
	if in.Demo && in.Status == "approved" {
		fail(w, 422, fmt.Errorf("demo content cannot be approved"))
		return
	}
	if in.House < 0 || in.House > 12 || in.LagnaSign < 0 || in.LagnaSign > 12 || in.Nakshatra < 0 || in.Nakshatra > 27 || (in.Planet == "" && (in.Sign != 0 || in.House != 0)) {
		fail(w, 422, fmt.Errorf("invalid chart conditions; sign/house requires a planet"))
		return
	}
	validCategory := in.Category == ""
	for _, category := range []string{"general", "personality", "education", "career", "relationships", "family", "wellbeing", "dasha"} {
		validCategory = validCategory || in.Category == category
	}
	validDasha := in.DashaLord == ""
	for _, planet := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"} {
		validDasha = validDasha || in.DashaLord == planet
	}
	if !validCategory || !validDasha {
		fail(w, 422, fmt.Errorf("unsupported Faladesh category or dasha lord"))
		return
	}
	if in.Planet != "" {
		valid := false
		for _, p := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"} {
			valid = valid || p == in.Planet
		}
		if !valid {
			fail(w, 422, fmt.Errorf("unsupported planet"))
			return
		}
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		key := fmt.Sprintf("%s:%d", in.ID, in.Version)
		if old, exists := s.Rules[key]; exists && old.Status == "approved" {
			return fmt.Errorf("approved version immutable; create a new version")
		}
		if in.Status == "approved" {
			if in.Approver == "" {
				return fmt.Errorf("named Guru Ji approval required")
			}
			t := a.Now()
			in.ApprovedAt = &t
		} else {
			in.ApprovedAt = nil
		}
		s.Rules[key] = in
		audit(s, sess.UserID, "content.rule-version-saved", key)
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, in)
}

// adminRetry requeues a failed job while preventing unsafe duplicate provider operations.
func (a *App) adminRetry(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if !decode(w, r, &in) {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		j := s.Jobs[in.ID]
		if j == nil {
			return fmt.Errorf("job not found")
		}
		if j.Status == "running" && j.Lease.After(a.Now()) {
			return fmt.Errorf("job still running")
		}
		if j.Kind == "refund" && j.Status == "done" {
			return fmt.Errorf("refund already submitted; reconcile payment")
		}
		j.Status = "pending"
		j.Next = a.Now()
		j.Error = ""
		audit(s, sess.UserID, "job.retry-requested", j.ID)
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// adminReconcile runs provider reconciliation after checking administrator access.
func (a *App) adminReconcile(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	a.Reconcile(r.Context())
	jsonOut(w, map[string]bool{"ok": true})
}

// adminRefund records a refund reason and queues a refund for an eligible fully paid order.
func (a *App) adminRefund(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in struct {
		ID     string `json:"id"`
		Reason string `json:"reason"`
	}
	if !decode(w, r, &in) {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		o := s.Orders[in.ID]
		if o == nil || o.Status != "paid" || o.Refunded != 0 {
			return fmt.Errorf("fully paid, unrefunded order required")
		}
		if len(in.Reason) < 5 {
			return fmt.Errorf("refund reason required")
		}
		o.RefundReason = in.Reason
		enqueue(s, "refund", o.ID, 1, "")
		audit(s, sess.UserID, "refund.approved", o.ID)
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]string{"status": "queued"})
}

// adminExpense records a validated expense and prevents the same receipt from being deducted twice.
func (a *App) adminExpense(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Expense
	if !decode(w, r, &in) {
		return
	}
	_, dateErr := time.Parse("2006-01-02", in.Date)
	if dateErr != nil || in.Amount <= 0 || in.Amount > 100000000 || in.Category == "" || in.Description == "" || in.Receipt == "" {
		fail(w, 422, fmt.Errorf("date, positive paise amount, category, description and uploaded receipt required"))
		return
	}
	if in.Category == "payment_fee" || in.Category == "tax" || in.Category == "refund" || in.Category == "partner_contribution" {
		fail(w, 422, fmt.Errorf("fees, taxes, refunds and contributions are not operating expenses"))
		return
	}
	in.ID = id("expense")
	in.Status = "pending"
	in.Approver = ""
	e := a.Store.Update(r.Context(), func(s *State) error {
		if _, e := os.Stat(filepath.Join(a.Config.DataDir, "receipts", filepath.Base(in.Receipt))); e != nil {
			return fmt.Errorf("receipt not found")
		}
		for _, existing := range s.Expenses {
			if existing.Receipt == in.Receipt {
				return fmt.Errorf("this receipt is already recorded; do not deduct an expense twice")
			}
		}
		s.Expenses[in.ID] = &in
		audit(s, sess.UserID, "expense.created", in.ID)
		return nil
	})
	if e != nil {
		fail(w, 422, e)
		return
	}
	jsonOut(w, in)
}

// approveExpense records an administrator's expense approval and updates the affected accounting state.
func (a *App) approveExpense(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	e := a.Store.Update(r.Context(), func(s *State) error {
		v := s.Expenses[r.PathValue("id")]
		if v == nil {
			return fmt.Errorf("expense not found")
		}
		v.Status = "approved"
		v.Approver = sess.UserID
		if r, ok := s.Reconciliations[v.Date[:7]]; ok {
			r.Closed = false
			s.Reconciliations[v.Date[:7]] = r
		}
		audit(s, sess.UserID, "expense.approved", v.ID)
		return nil
	})
	if e != nil {
		fail(w, 404, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// uploadReceipt validates an administrator receipt upload and stores its bounded PDF/image file.
func (a *App) uploadReceipt(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	body, e := io.ReadAll(http.MaxBytesReader(w, r.Body, 5<<20))
	if e != nil {
		fail(w, 413, fmt.Errorf("receipt limit 5MB"))
		return
	}
	kind := http.DetectContentType(body)
	ext := ""
	switch kind {
	case "application/pdf":
		ext = ".pdf"
	case "image/png":
		ext = ".png"
	case "image/jpeg":
		ext = ".jpg"
	default:
		fail(w, 415, fmt.Errorf("PDF, PNG or JPEG receipt required"))
		return
	}
	name := "receipt_" + hash(string(body)) + ext
	dir := filepath.Join(a.Config.DataDir, "receipts")
	if os.MkdirAll(dir, 0700) != nil || os.WriteFile(filepath.Join(dir, name), body, 0600) != nil {
		fail(w, 503, fmt.Errorf("receipt storage unavailable"))
		return
	}
	jsonOut(w, map[string]string{"receipt": name})
}

// getReceipt serves a stored receipt only after administrator authentication.
func (a *App) getReceipt(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	name := r.PathValue("id")
	if name != filepath.Base(name) || !strings.HasPrefix(name, "receipt_") {
		fail(w, 404, fmt.Errorf("not found"))
		return
	}
	w.Header().Set("Content-Disposition", "attachment")
	http.ServeFile(w, r, filepath.Join(a.Config.DataDir, "receipts", name))
}

// adminDistribution records a partner distribution while rejecting a reused payment reference.
func (a *App) adminDistribution(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Distribution
	if !decode(w, r, &in) {
		return
	}
	if _, e := time.Parse("2006-01", in.Month); e != nil || in.Amount <= 0 || in.Amount > maxExactMoney || in.Partner != "A" && in.Partner != "B" || len(in.Reference) < 3 {
		fail(w, 422, fmt.Errorf("month, partner A/B, positive amount and transfer reference required"))
		return
	}
	in.ID = id("distribution")
	e := a.Store.Update(r.Context(), func(s *State) error {
		for _, d := range s.Distributions {
			if d.Reference == in.Reference {
				return fmt.Errorf("distribution reference already recorded")
			}
		}
		s.Distributions[in.ID] = &in
		audit(s, sess.UserID, "distribution.recorded", in.ID)
		return nil
	})
	if e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, in)
}

// adminMonth opens or closes an accounting month and records the administrator's action.
func (a *App) adminMonth(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Reconciliation
	if !decode(w, r, &in) {
		return
	}
	if _, e := time.Parse("2006-01", in.Month); e != nil || in.Reserve < 0 || in.Reserve > maxExactMoney || in.Closed && len(in.StatementReference) < 3 {
		fail(w, 422, fmt.Errorf("valid month, non-negative reserve and statement reference required"))
		return
	}
	in.By = sess.UserID
	e := a.Store.Update(r.Context(), func(s *State) error {
		s.Reconciliations[in.Month] = in
		audit(s, sess.UserID, "month.reconciled", in.Month)
		return nil
	})
	if e != nil {
		fail(w, 503, e)
		return
	}
	jsonOut(w, in)
}

// financeCSV exports the selected month's administrator finance data as CSV.
func (a *App) financeCSV(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	month := r.URL.Query().Get("month")
	if _, e := time.Parse("2006-01", month); e != nil {
		fail(w, 422, fmt.Errorf("valid month required"))
		return
	}
	_ = a.Store.View(r.Context(), func(s *State) error {
		f := finances(s, month, !a.Config.Production && r.URL.Query().Get("demo") == "true")
		if f.CalculationError != "" {
			fail(w, 422, fmt.Errorf("%s", f.CalculationError))
			return nil
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="monthly-finance.csv"`)
		c := csv.NewWriter(w)
		_ = c.Write([]string{"month", "metric", "paise"})
		for _, row := range []struct {
			k string
			v int64
		}{{"pdf_sales", f.PDFSales}, {"consultation_sales", f.ConsultationSales}, {"gemstone_sales", f.GemstoneSales}, {"gross", f.Gross}, {"refunds", f.Refunds}, {"payment_fees", f.Fees}, {"tax", f.Tax}, {"approved_expenses", f.Expenses}, {"reserves", f.Reserves}, {"estimated_distributable", f.Distributable}, {"partner_a_50_percent", f.PartnerA}, {"partner_b_50_percent", f.PartnerB}, {"actual_distributions", f.Distributed}, {"remaining", f.Undistributed}} {
			_ = c.Write([]string{month, row.k, strconv.FormatInt(row.v, 10)})
		}
		c.Flush()
		return nil
	})
}

// adminBooking applies an administrator booking action through the shared scheduling service.
func (a *App) adminBooking(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Action string `json:"action"`
		Slot   string `json:"slot"`
	}
	if !decode(w, r, &in) {
		return
	}
	if e := a.ChangeBooking(r.Context(), sess.UserID, r.PathValue("id"), in.Action, in.Slot, true); e != nil {
		fail(w, 409, e)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// Migrate is exposed for the command-line migration operation.
func Migrate(ctx context.Context, url string, database ...string) error {
	s, e := OpenStore(ctx, url, "", database...)
	if e == nil {
		s.Close()
	}
	return e
}

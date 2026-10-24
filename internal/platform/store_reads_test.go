package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestViewKindsIsolationAndValidation checks independent partial snapshots, fresh
// values after updates, omitted private records and rejected/cancelled selections.
func TestViewKindsIsolationAndValidation(t *testing.T) {
	a := testApp(t)
	a.Store.state.Users["private"] = &User{ID: "private", Phone: "private fixture"}
	a.Store.state.Gemstones = map[string]*Gemstone{"ruby": {ID: "ruby", Stock: 7}}
	if err := a.Store.ViewKinds(context.Background(), []string{"settings", "gemstones"}, func(s *State) error {
		if len(s.Users) != 0 || len(s.Reports) != 0 || s.Gemstones["ruby"].Stock != 7 {
			t.Fatal("partial selection leaked or omitted records")
		}
		s.Gemstones["ruby"].Stock = 99
		s.Settings.Availability[0].From = "00:00"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if a.Store.state.Gemstones["ruby"].Stock != 7 || a.Store.state.Settings.Availability[0].From == "00:00" {
		t.Fatal("snapshot mutation changed stored records")
	}
	if err := a.Store.Update(context.Background(), func(s *State) error { s.Settings.Duration = 47; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.Store.ViewKinds(context.Background(), []string{"settings"}, func(s *State) error {
		if s.Settings.Duration != 47 {
			t.Fatal("partial read returned stale settings")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	for _, kinds := range [][]string{nil, {"unknown"}, {"settings", "unknown"}} {
		if err := a.Store.ViewKinds(context.Background(), kinds, func(*State) error { t.Fatal("invalid selection reached callback"); return nil }); err == nil {
			t.Fatal("invalid selection accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Store.ViewKinds(ctx, []string{"settings"}, func(*State) error { t.Fatal("cancelled read reached callback"); return nil }); err == nil {
		t.Fatal("cancelled selection accepted")
	}
}

// TestPublicReadsSkipUnrelatedReports uses a deliberately unserialisable private
// report to prove config, shop, health and session reads never clone that entity.
func TestPublicReadsSkipUnrelatedReports(t *testing.T) {
	a := testApp(t)
	putUser(t, a, "fixture")
	cookie, _ := authorize(t, a, "fixture", "customer")
	a.Store.state.Reports["unrelated"] = &Report{Chart: &swisseph.Kundali{JulianDayUT1: math.NaN()}}
	a.Store.state.Orders["unrelated"] = &Order{Snapshot: a.Store.state.Reports["unrelated"]}
	for _, path := range []string{"/api/config", "/api/shop", "/api/health"} {
		if response := request(t, a, "GET", path, nil, nil, ""); response.Code != 200 || response.Body.Len() == 0 {
			t.Fatalf("%s decoded unrelated report: %d %s", path, response.Code, response.Body.String())
		}
	}
	req := httptest.NewRequest("GET", "/api/me", nil)
	req.AddCookie(cookie)
	if _, err := a.session(req, false); err != nil {
		t.Fatalf("session lookup decoded unrelated report: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, path := range []string{"/api/config", "/api/health"} {
		req := httptest.NewRequest(http.MethodGet, path, nil).WithContext(ctx)
		response := httptest.NewRecorder()
		a.Handler().ServeHTTP(response, req)
		if response.Code != 503 {
			t.Fatalf("%s returned empty success on cancelled storage read: %d", path, response.Code)
		}
	}
}

// TestOrderSummaryPreservesSnapshots checks that read-time report omission keeps
// hold quantities available and never changes or replaces an immutable snapshot.
func TestOrderSummaryPreservesSnapshots(t *testing.T) {
	a := testApp(t)
	report := &Report{Birth: Birth{Name: "Original snapshot"}}
	a.Store.state.Orders["o"] = &Order{ID: "o", Status: "pending", Snapshot: report, Gemstone: &GemstoneSale{
		HoldUntil: a.Now().Add(time.Hour), Items: []ShopLine{{ProductID: "ruby", Quantity: 2}},
	}}
	if err := a.Store.ViewWithoutOrderSnapshots(context.Background(), []string{"orders"}, func(s *State) error {
		order := s.Orders["o"]
		if order.Snapshot != nil || heldShopQuantity(s, "ruby", "", a.Now()) != 2 {
			t.Fatal("summary omitted hold data or retained the report payload")
		}
		order.Gemstone.Items[0].Quantity = 99
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if a.Store.state.Orders["o"].Snapshot != report || report.Birth.Name != "Original snapshot" || a.Store.state.Orders["o"].Gemstone.Items[0].Quantity != 2 {
		t.Fatal("summary modified a stored order or immutable report")
	}
	if err := a.Store.ViewKinds(context.Background(), []string{"orders"}, func(s *State) error {
		if s.Orders["o"].Snapshot == nil || s.Orders["o"].Snapshot.Birth.Name != "Original snapshot" {
			t.Fatal("full order read lost its snapshot")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// TestIdleWorkerAvoidsPersistence checks that an idle tick performs no write,
// while an expired session still triggers the existing atomic cleanup path.
func TestIdleWorkerAvoidsPersistence(t *testing.T) {
	a := testApp(t)
	a.Store.path = filepath.Join(t.TempDir(), "state.json")
	if err := a.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(a.Store.path); !os.IsNotExist(err) {
		t.Fatal("idle tick persisted an unchanged state")
	}
	a.Store.state.Sessions["expired"] = &Session{Expires: a.Now().Add(-time.Second)}
	if err := a.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if a.Store.state.Sessions["expired"] != nil {
		t.Fatal("preflight skipped expired-session cleanup")
	}
	if _, err := os.Stat(a.Store.path); err != nil {
		t.Fatal("cleanup was not persisted")
	}
}

// TestWorkerPreflightBoundaries checks due-job leases, cleanup grace periods and
// paid/reserved stock distinctions before the worker takes its write transaction.
func TestWorkerPreflightBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name    string
		prepare func(*State)
		want    bool
	}{
		{"pending future", func(s *State) { s.Jobs["j"] = &Job{Status: "pending", Next: now.Add(time.Second)} }, false},
		{"retry due", func(s *State) { s.Jobs["j"] = &Job{Status: "retry", Next: now} }, true},
		{"lease active", func(s *State) { s.Jobs["j"] = &Job{Status: "running", Lease: now.Add(time.Second)} }, false},
		{"lease expired", func(s *State) { s.Jobs["j"] = &Job{Status: "running", Lease: now.Add(-time.Second)} }, true},
		{"job done", func(s *State) { s.Jobs["j"] = &Job{Status: "done"} }, false},
		{"OTP grace boundary", func(s *State) { s.OTPs["o"] = &OTP{Expires: now.Add(-time.Hour)} }, false},
		{"OTP grace expired", func(s *State) { s.OTPs["o"] = &OTP{Expires: now.Add(-time.Hour - time.Second)} }, true},
		{"booking hold boundary", func(s *State) { s.Bookings["b"] = &Booking{Status: "held", HoldUntil: now} }, true},
		{"unpaid stock hold", func(s *State) { s.Orders["o"] = &Order{Status: "pending", Gemstone: &GemstoneSale{HoldUntil: now}} }, true},
		{"paid stock hold", func(s *State) {
			s.Orders["o"] = &Order{Status: "paid", PaidAt: &now, Gemstone: &GemstoneSale{HoldUntil: now}}
		}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := seed()
			tc.prepare(s)
			if got := workerNeedsUpdate(s, now); got != tc.want {
				t.Fatalf("needed=%v, want %v", got, tc.want)
			}
		})
	}
}

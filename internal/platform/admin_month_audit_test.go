package platform

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestAdminDefaultMonthUsesConfiguredBusinessTimezone checks the default finance month in the configured business timezone.
func TestAdminDefaultMonthUsesConfiguredBusinessTimezone(t *testing.T) {
	// testApp uses an isolated in-memory store and simulated providers.
	a := testApp(t)
	now := time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC)
	a.Now = func() time.Time { return now }
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.Zone = "Asia/Kolkata"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	cookie, csrf := authorize(t, a, "admin-month-audit", "admin")
	for _, tc := range []struct {
		path, month string
	}{
		{"/api/admin", "2026-10"},
		{"/api/admin?month=2026-09", "2026-09"},
	} {
		w := request(t, a, "GET", tc.path, nil, cookie, csrf)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body)
		}
		var got struct {
			Finance Finance `json:"finance"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got.Finance.Month != tc.month {
			t.Fatalf("%s selected %s want %s", tc.path, got.Finance.Month, tc.month)
		}
	}
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.Zone = "Invalid/Business_Zone"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	w := request(t, a, "GET", "/api/admin", nil, cookie, csrf)
	if w.Code != 422 || !strings.Contains(w.Body.String(), "business timezone") {
		t.Fatalf("invalid business zone silently defaulted: %d %s", w.Code, w.Body)
	}
}

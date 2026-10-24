package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"
)

// TestReceiptDeduplicationAndApprovalDoNotDoubleDeduct checks that duplicate receipts and repeated approvals cannot deduct expenses twice.
func TestReceiptDeduplicationAndApprovalDoNotDoubleDeduct(t *testing.T) {
	a := testApp(t)
	cookie, csrf := authorize(t, a, "admin", "admin")
	receipt := ""
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("POST", "/api/admin/receipts", bytes.NewBufferString("%PDF-1.4\nreceipt fixture"))
		req.AddCookie(cookie)
		req.Header.Set("X-CSRF-Token", csrf)
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, req)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
		var response map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &response)
		if receipt != "" && receipt != response["receipt"] {
			t.Fatal("same uploaded receipt received different deduplication ID")
		}
		receipt = response["receipt"]
	}
	expense := Expense{Date: "2026-10-18", Category: "infrastructure", Description: "Approved test receipt", Amount: 170000, Receipt: receipt}
	w := request(t, a, "POST", "/api/admin/expenses", expense, cookie, csrf)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var saved Expense
	_ = json.Unmarshal(w.Body.Bytes(), &saved)
	if w = request(t, a, "POST", "/api/admin/expenses", expense, cookie, csrf); w.Code != 422 {
		t.Fatalf("duplicate expense accepted: %d", w.Code)
	}
	for i := 0; i < 2; i++ {
		if w = request(t, a, "POST", "/api/admin/expenses/"+saved.ID+"/approve", map[string]string{}, cookie, csrf); w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	_ = a.Store.View(context.Background(), func(s *State) error {
		if got := finances(s, "2026-10", false).Expenses; got != 170000 {
			t.Fatalf("approved receipt deducted more than once: %d", got)
		}
		return nil
	})
}

// TestEqualPartnerAllocationRetainsOddPaise checks that partner allocations preserve an odd paise rather than losing it to rounding.
func TestEqualPartnerAllocationRetainsOddPaise(t *testing.T) {
	s := seed()
	now := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	s.Orders["odd"] = &Order{Product: "pdf", PaidAt: &now, Amount: 9901}
	f := finances(s, "2026-10", false)
	if f.PartnerA != 4950 || f.PartnerB != 4950 || f.RoundingRemainder != 1 {
		t.Fatalf("unequal allocation or lost remainder: %+v", f)
	}
}

package platform

import (
	"context"
	"math"
	"testing"
	"time"
)

// TestOrderTaxRoundingAudit checks tax rounding at integer-paise boundaries.
func TestOrderTaxRoundingAudit(t *testing.T) {
	// Hand-computed rational tax splits, paise and basis points. A gross9903p
	// inclusive at18% contains1510.627...p tax, rounded half-up to1511p.
	for _, c := range []struct {
		mode                   string
		base, rate, total, tax int64
	}{
		{"inclusive", 9903, 1800, 9903, 1511},
		{"inclusive", 11800, 1800, 11800, 1800},
		{"inclusive", 3, 2000, 3, 1}, // exactly half a paisa
		{"exclusive", 3, 5000, 5, 2},
		{"exclusive", 9903, 1800, 11686, 1783},
		{"none", 9903, 0, 9903, 0},
		{"unconfigured", 9903, 0, 9903, 0},
	} {
		total, tax, err := orderAmounts(c.base, c.rate, c.mode)
		if err != nil || total != c.total || tax != c.tax {
			t.Fatalf("%+v: total=%d tax=%d err=%v", c, total, tax, err)
		}
	}
	if _, _, err := orderAmounts(math.MaxInt64, 5000, "exclusive"); err == nil {
		t.Fatal("unsafe base accepted")
	}
}

// TestFinanceArithmeticAudit checks finance totals against independently computed arithmetic.
func TestFinanceArithmeticAudit(t *testing.T) {
	s := seed()
	s.Expenses["cost"] = &Expense{Date: "2026-10-18", Status: "approved", Amount: 100}
	s.Reconciliations["2026-10"] = Reconciliation{Reserve: math.MaxInt64}
	f := finances(s, "2026-10", false)
	if f.CalculationError == "" || f.Distributable != 0 {
		t.Fatalf("reserve overflow produced money: %+v", f)
	}
	s.Reconciliations["2026-10"] = Reconciliation{Reserve: maxExactMoney}
	f = finances(s, "2026-10", false)
	if f.CalculationError != "" || f.Distributable != 0 {
		t.Fatalf("large exact deduction not calculated safely: %+v", f)
	}
	paid := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	s.Orders["a"] = &Order{Product: "pdf", PaidAt: &paid, Amount: maxExactMoney}
	s.Orders["b"] = &Order{Product: "pdf", PaidAt: &paid, Amount: 1}
	if f = finances(s, "2026-10", false); f.CalculationError == "" {
		t.Fatal("inexact browser monetary total returned")
	}
}

// TestFinancialUnsafeInputsAudit rejects unsafe financial inputs before totals are calculated.
func TestFinancialUnsafeInputsAudit(t *testing.T) {
	a := testApp(t)
	cookie, csrf := authorize(t, a, "admin", "admin")
	for _, c := range []struct {
		path   string
		method string
		body   any
	}{
		{"/api/admin/month", "PUT", Reconciliation{Month: "2026-10", Reserve: math.MaxInt64}},
		{"/api/admin/distributions", "POST", Distribution{Month: "2026-10", Amount: math.MaxInt64, Partner: "A", Reference: "audit-transfer"}},
	} {
		if w := request(t, a, c.method, c.path, c.body, cookie, csrf); w.Code != 422 {
			t.Fatalf("unsafe input %s: %d", c.path, w.Code)
		}
	}
	if err := a.Store.Update(context.Background(), func(s *State) error {
		s.Reconciliations["2026-10"] = Reconciliation{Reserve: math.MaxInt64}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	w := request(t, a, "GET", "/api/admin/finance.csv?month=2026-10", nil, cookie, "")
	if w.Code != 422 {
		t.Fatalf("unsafe CSV exported: %d", w.Code)
	}
}

// TestHistoricalRefundExactArithmeticAudit checks exact refund arithmetic for historical orders.
func TestHistoricalRefundExactArithmeticAudit(t *testing.T) {
	s := seed()
	paid := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)
	o := &Order{ID: "historical", ProviderID: "provider", PaymentID: "payment", Currency: "INR", Amount: 9007199254740990, Tax: 6004799503160660, PaidAt: &paid, Status: "paid"}
	s.Orders[o.ID] = o
	p := Payment{ID: "payment", OrderID: "provider", Currency: "INR", Status: "refunded", Amount: o.Amount, Refunded: 4503599627370495}
	if err := applyPayment(s, o, p, paid); err != nil {
		t.Fatal(err)
	}
	if len(o.Refunds) != 1 || o.Refunds[0].Tax != 3002399751580330 {
		t.Fatalf("overflowed tax allocation: %+v", o.Refunds)
	}
	p.Fee = maxExactMoney + 1
	if err := applyPayment(s, o, p, paid); err == nil {
		t.Fatal("inexact provider fee accepted")
	}
}

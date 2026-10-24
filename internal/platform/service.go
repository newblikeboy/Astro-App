package platform

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"net/http"
	"sort"
	"strings"
	"time"
)

type App struct {
	Store            *Store
	Config           Config
	Pay              PaymentProvider
	Zoom             MeetingProvider
	OTP              OTPProvider
	Notify           NotificationProvider
	PDF              PDFRenderer
	MatchingPDF      MatchingPDFRenderer
	matchingPDFSlots chan struct{}
	Now              func() time.Time
	limits           *limiter
	daily            dailyCache
	sky              skyCache
}

// NewApp connects storage to the selected payment, meeting, notification and PDF adapters.
func NewApp(s *Store, c Config) *App {
	client := &http.Client{Timeout: 20 * time.Second}
	a := &App{Store: s, Config: c, Now: func() time.Time { return time.Now().UTC() }, limits: newLimiter()}
	if c.PaymentMode == "razorpay" {
		a.Pay = &Razorpay{KeyID: c.KeyID, Secret: c.KeySecret, Client: client}
	} else {
		a.Pay = SimPayment{}
	}
	if c.ZoomMode == "zoom" {
		a.Zoom = &Zoom{c.ZoomAccount, c.ZoomClient, c.ZoomSecret, c.ZoomUser, client}
	} else {
		a.Zoom = SimMeeting{}
	}
	a.OTP = OTPGateway{Gateway{c.OTPURL, c.OTPToken, client, c.OTPMode == "simulation"}}
	a.Notify = NotificationGateway{Gateway{c.NotificationURL, c.NotificationToken, client, !c.Production && c.NotificationURL == ""}}
	a.PDF = ChromePDF{FontPath: c.FontPath, ChromePath: c.ChromePath, Origin: c.Origin}
	a.MatchingPDF = ChromePDF{FontPath: c.FontPath, ChromePath: c.ChromePath, Origin: c.Origin}
	a.matchingPDFSlots = make(chan struct{}, 1)
	return a
}

// duration returns the configured consultation length, allowing the documented development fallback.
func duration(s Settings, production bool) int {
	if s.Duration == 0 && !production {
		return 30
	}
	return s.Duration
}

// blockers lists unresolved production requirements that prevent the requested paid product from being sold.
func blockers(s Settings, c Config, product string) []string {
	out := []string{}
	if !c.Production {
		return out
	}
	if !s.LicenceConfirmed {
		out = append(out, "Swiss Ephemeris licence decision required")
	}
	if !s.Content.Approved {
		out = append(out, "Homepage and legal policies require approval")
	}
	if s.TaxMode == "unconfigured" {
		out = append(out, "Tax treatment must be configured")
	}
	if product == "pdf" && !s.PDFEnabled {
		out = append(out, "PDF sales disabled")
	}
	if product == "consultation" {
		if !s.BookingsEnabled {
			out = append(out, "Live bookings disabled")
		}
		if s.Duration <= 0 {
			out = append(out, "Consultation duration required")
		}
	}
	return out
}

// available checks buffered overlap against active bookings, optionally excluding the booking being moved.
func available(s *State, start, end, now time.Time, exclude string) bool {
	for _, b := range s.Bookings {
		if b.ID == exclude {
			continue
		}
		active := b.Status == "held" && now.Before(b.HoldUntil) || b.Status == "meeting_pending" || b.Status == "confirmed" || b.Status == "cancel_pending"
		if active && start.Before(b.End.Add(time.Duration(s.Settings.Buffer)*time.Minute)) && end.Add(time.Duration(s.Settings.Buffer)*time.Minute).After(b.Start) {
			return false
		}
	}
	return true
}

// slots expands business-local availability into sorted, unique UTC slots while honoring lead time and holds.
func slots(s *State, now time.Time, production bool) []time.Time {
	out := []time.Time{}
	settings := s.Settings
	minutes := duration(settings, production)
	if minutes <= 0 {
		return out
	}
	loc, e := time.LoadLocation(settings.Zone)
	if e != nil {
		return out
	}
	base := now.In(loc)
	// Iterate civil date labels independently of the current local clock. A
	// midnight DST gap can normalize AddDate(base) into the preceding day.
	// ResolveWall below still resolves each window in the configured zone.
	civilBase := time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, time.UTC)
	lead := now.Add(time.Duration(settings.LeadHours) * time.Hour)
	seenSlots := map[time.Time]bool{}
	blocked := map[string]bool{}
	for _, d := range settings.Blocked {
		blocked[d] = true
	}
	for day := 0; day <= settings.HorizonDays; day++ {
		d := civilBase.AddDate(0, 0, day)
		if blocked[d.Format("2006-01-02")] {
			continue
		}
		for _, w := range settings.Availability {
			if int(d.Weekday()) != w.Day {
				continue
			}
			from, e1 := ResolveWall(d.Format("2006-01-02"), w.From, settings.Zone, nil)
			to, e2 := ResolveWall(d.Format("2006-01-02"), w.To, settings.Zone, nil)
			if e1 != nil || e2 != nil {
				continue
			}
			for t := from; !t.Add(time.Duration(minutes) * time.Minute).After(to); t = t.Add(time.Duration(minutes+settings.Buffer) * time.Minute) {
				end := t.Add(time.Duration(minutes) * time.Minute)
				if t.After(lead) && available(s, t, end, now, "") {
					utc := t.UTC()
					if !seenSlots[utc] {
						seenSlots[utc] = true
						out = append(out, utc)
					}
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

type CheckoutRequest struct {
	Product  string `json:"product"`
	ReportID string `json:"report_id"`
	Slot     string `json:"slot"`
	Key      string `json:"idempotency_key"`
}

// Checkout creates an idempotent PDF/consultation order using server prices and transactional slot reservations.
func (a *App) Checkout(ctx context.Context, user string, in CheckoutRequest) (*Order, error) {
	if in.Product != "pdf" && in.Product != "consultation" {
		return nil, fmt.Errorf("invalid product")
	}
	if len(in.Key) < 8 || len(in.Key) > 100 {
		return nil, fmt.Errorf("idempotency key required")
	}
	var order *Order
	e := a.Store.Update(ctx, func(s *State) error {
		for _, o := range s.Orders {
			if o.UserID == user && o.RequestKey == in.Key {
				order = o
				return nil
			}
		}
		if u := s.Users[user]; u == nil || u.Deleted {
			return fmt.Errorf("verified account required")
		}
		if bs := blockers(s.Settings, a.Config, in.Product); len(bs) > 0 {
			return fmt.Errorf("checkout blocked: %s", bs[0])
		}
		o := &Order{ID: id("o"), UserID: user, Product: in.Product, Currency: "INR", Status: "creating", Created: a.Now(), Demo: a.Config.PaymentMode == "simulation" || strings.HasPrefix(a.Config.KeyID, "rzp_test_"), RequestKey: in.Key, TaxMode: s.Settings.TaxMode, TaxBPS: s.Settings.TaxBPS}
		if in.Product == "pdf" {
			r := s.Reports[in.ReportID]
			if r == nil || r.UserID != user {
				return fmt.Errorf("पहले अपनी कुंडली बनाएँ और सहेजें")
			}
			if r.Chart == nil {
				return fmt.Errorf("व्यक्तिगत उपाय PDF के लिए सही जन्म समय आवश्यक है")
			}
			for _, existing := range s.Orders {
				if a.Config.Production && existing.Demo {
					continue
				}
				if existing.UserID == user && existing.ReportID == r.ID && existing.Product == "pdf" && existing.Status != "refunded" {
					order = existing
					return nil
				}
			}
			snap := *r
			snap.Rules = matchRules(r, s.Rules, !a.Config.Production)
			if a.Config.Production {
				interpretation, remedy := false, false
				for _, rule := range snap.Rules {
					if !rule.Demo && rule.Status == "approved" {
						interpretation = interpretation || rule.Hindi != ""
						remedy = remedy || rule.Mantra != "" || rule.Gemstone != ""
					}
				}
				if !interpretation || !remedy {
					return fmt.Errorf("इस कुंडली के लिए स्वीकृत व्याख्या और उपाय अधूरे हैं; PDF बिक्री बंद है")
				}
			}
			o.Snapshot = &snap
			o.ReportID = r.ID
			o.Amount = s.Settings.PDFPrice
			o.Title = "व्यक्तिगत उपाय PDF"
		} else {
			t, e := time.Parse(time.RFC3339, in.Slot)
			if e != nil {
				return fmt.Errorf("उपलब्ध समय चुनें")
			}
			found := false
			for _, slot := range slots(s, a.Now(), a.Config.Production) {
				if slot.Equal(t) {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("यह समय अब उपलब्ध नहीं है; दूसरा समय चुनें")
			}
			b := &Booking{ID: id("b"), UserID: user, OrderID: o.ID, Start: t.UTC(), End: t.Add(time.Duration(duration(s.Settings, a.Config.Production)) * time.Minute).UTC(), Status: "held", HoldUntil: a.Now().Add(10 * time.Minute), Version: 1, Demo: o.Demo}
			s.Bookings[b.ID] = b
			o.BookingID = b.ID
			o.Amount = s.Settings.ConsultationPrice
			o.Title = "गुरुजी से व्यक्तिगत परामर्श"
		}
		var amountErr error
		o.Amount, o.Tax, amountErr = orderAmounts(o.Amount, s.Settings.TaxBPS, s.Settings.TaxMode)
		if amountErr != nil {
			return amountErr
		}
		s.Orders[o.ID] = o
		order = o
		audit(s, user, "checkout.created", o.ID)
		return nil
	})
	if e != nil {
		return nil, e
	}
	if order.Status != "creating" {
		return order, nil
	}
	// Reserve before calling the provider; failed creation can be retried without a new hold.
	providerID, e := a.Pay.Create(ctx, order)
	if e != nil {
		return nil, e
	}
	e = a.Store.Update(ctx, func(s *State) error {
		o := s.Orders[order.ID]
		if o == nil {
			return fmt.Errorf("order removed")
		}
		if o.ProviderID == "" {
			o.ProviderID = providerID
			o.Status = "pending"
		}
		order = o
		return nil
	})
	return order, e
}

// applyPayment validates provider ownership/totals and applies captures or refunds without double-processing them.
func applyPayment(s *State, o *Order, p Payment, now time.Time) error {
	if o.Amount <= 0 || o.Amount > maxExactMoney || o.Tax < 0 || o.Tax > o.Amount || o.Refunded < 0 || o.Refunded > o.Amount || p.OrderID != o.ProviderID || p.Amount != o.Amount || p.Currency != "INR" {
		return fmt.Errorf("payment/order mismatch")
	}
	if p.Refunded < 0 || p.Refunded > p.Amount || p.Fee < 0 || p.Fee > maxExactMoney {
		return fmt.Errorf("invalid payment totals")
	}
	for _, other := range s.Orders {
		if other.ID != o.ID && other.PaymentID == p.ID && p.ID != "" {
			return fmt.Errorf("payment already assigned")
		}
	}
	if p.Status == "failed" {
		if o.Status == "pending" {
			o.Status = "failed"
		}
		return nil
	}
	if p.Status != "captured" && p.Status != "refunded" {
		return nil
	}
	if o.PaymentID != "" && o.PaymentID != p.ID {
		return fmt.Errorf("another captured payment requires manual review")
	}
	o.PaymentID = p.ID
	if p.Fee > o.Fees {
		o.Fees = p.Fee
		if o.PaidAt != nil {
			reopenMonth(s, *o.PaidAt)
		}
	}
	firstCapture := o.PaidAt == nil
	if firstCapture {
		t := now
		o.PaidAt = &t
		reopenMonth(s, t)
	}
	if p.Refunded > o.Refunded {
		reopenMonth(s, now)
		delta := p.Refunded - o.Refunded
		// Cumulative floor allocation keeps split refunds exact and refunds all
		// tax on a full refund, without overflowing historical large amounts.
		allocatedTax := func(refunded int64) int64 {
			n := new(big.Int).Mul(big.NewInt(o.Tax), big.NewInt(refunded))
			return n.Quo(n, big.NewInt(o.Amount)).Int64()
		}
		taxDelta := allocatedTax(p.Refunded) - allocatedTax(o.Refunded)
		o.Refunds = append(o.Refunds, RefundRecord{At: now, Amount: delta, Tax: taxDelta})
		o.Refunded = p.Refunded
		t := now
		o.RefundAt = &t
		audit(s, "payment-provider", "payment.refund-recorded", o.ID)
	}
	if o.Refunded == o.Amount {
		if o.Status == "refunded" {
			return nil
		}
		o.Status = "refunded"
		refundGemstone(s, o)
		if b := s.Bookings[o.BookingID]; b != nil {
			b.Status = "cancel_pending"
			b.JoinURL = ""
			b.Version++
			enqueue(s, "cancel_zoom", b.ID, b.Version, "")
		}
		return nil
	}
	if !firstCapture {
		return nil
	}
	o.Status = "paid"
	o.PaidAt = &now
	if o.Product == "pdf" {
		enqueue(s, "pdf", o.ID, 1, "")
	} else if o.Product == "gemstone" {
		captureGemstone(s, o, now)
	} else {
		b := s.Bookings[o.BookingID]
		if b == nil {
			return fmt.Errorf("booking missing")
		}
		if b.Start.After(now) && available(s, b.Start, b.End, now, b.ID) {
			b.Status = "meeting_pending"
			enqueue(s, "zoom", b.ID, b.Version, "")
		} else {
			b.Status = "needs_resolution"
			audit(s, "system", "booking.late-payment-resolution", b.ID)
		}
	}
	audit(s, "payment-provider", "payment.verified", o.ID)
	if u := s.Users[o.UserID]; u != nil && u.Analytics {
		s.Events = append(s.Events, Event{Name: "verified_payment", Product: o.Product, At: now})
	}
	return nil
}

// VerifyPayment checks customer ownership and signature, fetches the provider payment and applies its state.
func (a *App) VerifyPayment(ctx context.Context, orderID, paymentID, sig, user string) (*Order, error) {
	var order *Order
	e := a.Store.View(ctx, func(s *State) error {
		o := s.Orders[orderID]
		if o == nil || o.UserID != user {
			return fmt.Errorf("order not found")
		}
		order = o
		return nil
	})
	if e != nil {
		return nil, e
	}
	if !verifySignature(a.Config.KeySecret, order.ProviderID+"|"+paymentID, sig) {
		return nil, fmt.Errorf("invalid payment signature")
	}
	p, e := a.Pay.Fetch(ctx, paymentID)
	if e != nil {
		return nil, e
	}
	e = a.Store.Update(ctx, func(s *State) error {
		o := s.Orders[orderID]
		if e := applyPayment(s, o, p, a.Now()); e != nil {
			return e
		}
		order = o
		return nil
	})
	return order, e
}

// ChangeBooking cancels, reschedules or updates an eligible booking while preserving hold and provider-job rules.
func (a *App) ChangeBooking(ctx context.Context, user, id, action, slot string, admin bool) error {
	return a.Store.Update(ctx, func(s *State) error {
		b := s.Bookings[id]
		if b == nil || !admin && b.UserID != user {
			return fmt.Errorf("booking not found")
		}
		o := s.Orders[b.OrderID]
		if o == nil || o.Status != "paid" {
			return fmt.Errorf("paid booking required")
		}
		if !admin && b.Start.Before(a.Now().Add(time.Duration(s.Settings.ChangeHours)*time.Hour)) && b.Status != "needs_resolution" {
			return fmt.Errorf("बदलाव की समय-सीमा बीत गई है; सहायता से संपर्क करें")
		}
		if action == "cancel" {
			if b.Status == "cancelled" || b.Status == "cancel_pending" {
				return nil
			}
			b.Status = "cancel_pending"
			b.JoinURL = ""
			b.Version++
			enqueue(s, "cancel_zoom", b.ID, b.Version, "")
			audit(s, user, "booking.cancellation-requested", b.ID)
			return nil
		}
		if action != "reschedule" {
			return fmt.Errorf("invalid booking action")
		}
		if b.Status != "confirmed" && b.Status != "meeting_pending" && b.Status != "needs_resolution" {
			return fmt.Errorf("booking cannot be rescheduled")
		}
		t, e := time.Parse(time.RFC3339, slot)
		if e != nil {
			return fmt.Errorf("invalid slot")
		}
		found := false
		for _, v := range slots(s, a.Now(), a.Config.Production) {
			if v.Equal(t) {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("slot unavailable")
		}
		if m := s.Meetings[b.ID]; m.ID != "" {
			enqueue(s, "delete_meeting", b.ID, b.Version, m.ID)
			delete(s.Meetings, b.ID)
		}
		b.Start = t
		b.End = t.Add(time.Duration(duration(s.Settings, a.Config.Production)) * time.Minute)
		b.Version++
		b.Status = "meeting_pending"
		b.JoinURL = ""
		enqueue(s, "zoom", b.ID, b.Version, "")
		audit(s, user, "booking.rescheduled", b.ID)
		return nil
	})
}

type Finance struct {
	GemstoneSales     int64  `json:"gemstone_sales"`
	CalculationError  string `json:"calculation_error,omitempty"`
	Month             string `json:"month"`
	PDFSales          int64  `json:"pdf_sales"`
	ConsultationSales int64  `json:"consultation_sales"`
	Gross             int64  `json:"gross"`
	Refunds           int64  `json:"refunds"`
	Fees              int64  `json:"fees"`
	Tax               int64  `json:"tax"`
	Expenses          int64  `json:"expenses"`
	Reserves          int64  `json:"reserves"`
	Distributable     int64  `json:"distributable"`
	PartnerA          int64  `json:"partner_a"`
	PartnerB          int64  `json:"partner_b"`
	Distributed       int64  `json:"distributed"`
	Undistributed     int64  `json:"undistributed"`
	TaxConfigured     bool   `json:"tax_configured"`
	Unreconciled      bool   `json:"unreconciled"`
	RoundingRemainder int64  `json:"rounding_remainder"`
}

// Currency uses integer paise. Both tax modes round half-up to one paisa;
// inclusive tax previously truncated, which understated some amounts by 1p.
func orderAmounts(base, rate int64, mode string) (total, tax int64, err error) {
	if base <= 0 || base > 100000000 || rate < 0 || rate > 5000 {
		return 0, 0, fmt.Errorf("invalid order amounts")
	}
	total = base
	switch mode {
	case "inclusive":
		denominator := int64(10000) + rate
		tax = (base*rate + denominator/2) / denominator
	case "exclusive":
		tax = (base*rate + 5000) / 10000
		total += tax
	case "none", "unconfigured":
	default:
		return 0, 0, fmt.Errorf("invalid tax mode")
	}
	return total, tax, nil
}

// JSON/browser monetary amounts must also remain exact JavaScript integers.
const maxExactMoney int64 = 1<<53 - 1

// finances aggregates the month's orders, taxes, refunds, expenses and distributions using exact integer arithmetic.
func finances(s *State, month string, demo bool) Finance {
	f := Finance{Month: month, TaxConfigured: s.Settings.TaxMode != "unconfigured", Unreconciled: true}
	totals := map[string]*big.Int{}
	// add adds paise to an arbitrary-precision ledger bucket before supported-range validation.
	add := func(key string, amount int64) {
		if totals[key] == nil {
			totals[key] = new(big.Int)
		}
		totals[key].Add(totals[key], big.NewInt(amount))
	}
	// amount reads a ledger bucket, returning an exact zero when no entries exist.
	amount := func(key string) *big.Int {
		if totals[key] == nil {
			return new(big.Int)
		}
		return totals[key]
	}
	loc, e := time.LoadLocation(s.Settings.Zone)
	if e != nil {
		loc = time.UTC
	}
	for _, o := range s.Orders {
		if o.Demo != demo || o.PaidAt == nil {
			continue
		}
		if o.PaidAt.In(loc).Format("2006-01") == month {
			if o.Product == "pdf" {
				add("pdf", o.Amount)
			} else if o.Product == "gemstone" {
				add("gemstone", o.Amount)
			} else {
				add("consultation", o.Amount)
			}
			add("gross", o.Amount)
			add("fees", o.Fees)
			add("tax", o.Tax)
		}
		for _, refund := range o.Refunds {
			if refund.At.In(loc).Format("2006-01") == month {
				add("refunds", refund.Amount)
				// Avoid negating MinInt64 before the checked accumulation.
				if totals["tax"] == nil {
					totals["tax"] = new(big.Int)
				}
				totals["tax"].Sub(totals["tax"], big.NewInt(refund.Tax))
			}
		}
	}
	for _, e := range s.Expenses {
		if e.Status == "approved" && len(e.Date) >= 7 && e.Date[:7] == month {
			add("expenses", e.Amount)
		}
	}
	if r, ok := s.Reconciliations[month]; ok {
		add("reserves", r.Reserve)
		f.Unreconciled = !r.Closed
	}
	net := new(big.Int).Set(amount("gross"))
	for _, key := range []string{"refunds", "fees", "tax", "expenses", "reserves"} {
		net.Sub(net, amount(key))
	}
	if net.Sign() < 0 {
		net.SetInt64(0)
	}
	totals["distributable"] = net
	for _, d := range s.Distributions {
		if d.Month == month {
			add("distributed", d.Amount)
		}
	}
	totals["undistributed"] = new(big.Int).Sub(net, amount("distributed"))
	for key, dest := range map[string]*int64{
		"pdf": &f.PDFSales, "consultation": &f.ConsultationSales,
		"gemstone": &f.GemstoneSales,
		"gross":    &f.Gross, "fees": &f.Fees, "tax": &f.Tax,
		"refunds": &f.Refunds, "expenses": &f.Expenses, "reserves": &f.Reserves,
		"distributable": &f.Distributable, "distributed": &f.Distributed, "undistributed": &f.Undistributed,
	} {
		value := amount(key)
		if !value.IsInt64() || value.Cmp(big.NewInt(maxExactMoney)) > 0 || value.Cmp(big.NewInt(-maxExactMoney)) < 0 {
			return Finance{Month: month, TaxConfigured: f.TaxConfigured, Unreconciled: true, CalculationError: "Financial totals exceed the supported exact range. Review the ledger."}
		}
		*dest = value.Int64()
	}
	f.PartnerA = f.Distributable / 2
	f.PartnerB = f.PartnerA
	f.RoundingRemainder = f.Distributable % 2
	return f
}

// reopenMonth marks the affected business-local accounting month open after a later financial change.
func reopenMonth(s *State, at time.Time) {
	loc, e := time.LoadLocation(s.Settings.Zone)
	if e != nil {
		loc = time.UTC
	}
	month := at.In(loc).Format("2006-01")
	if r, ok := s.Reconciliations[month]; ok && r.Closed {
		r.Closed = false
		s.Reconciliations[month] = r
	}
}

// validateSettings rejects unsupported money, scheduling, timezone, tax and shop configuration values.
func validateSettings(s Settings) error {
	if err := validateShopSettings(s.Shop); err != nil {
		return err
	}
	if s.PDFPrice <= 0 || s.ConsultationPrice <= 0 || s.PDFPrice > 100000000 || s.ConsultationPrice > 100000000 {
		return fmt.Errorf("price out of range")
	}
	if s.Duration < 0 || s.Duration > 180 || s.Buffer < 0 || s.Buffer > 120 || s.LeadHours < 0 || s.LeadHours > 720 || s.HorizonDays < 1 || s.HorizonDays > 180 || s.ChangeHours < 0 || s.ChangeHours > 720 {
		return fmt.Errorf("invalid scheduling configuration")
	}
	if s.BookingsEnabled && s.Duration < 5 {
		return fmt.Errorf("duration must be decided before enabling live bookings")
	}
	if _, e := time.LoadLocation(s.Zone); e != nil {
		return fmt.Errorf("invalid scheduling timezone")
	}
	if s.TaxMode != "unconfigured" && s.TaxMode != "inclusive" && s.TaxMode != "exclusive" && s.TaxMode != "none" {
		return fmt.Errorf("invalid tax mode")
	}
	if s.TaxBPS < 0 || s.TaxBPS > 5000 || s.TaxMode == "none" && s.TaxBPS != 0 {
		return fmt.Errorf("invalid tax rate")
	}
	for _, w := range s.Availability {
		f, e := time.Parse("15:04", w.From)
		t, e2 := time.Parse("15:04", w.To)
		if e != nil || e2 != nil || !t.After(f) || w.Day < 0 || w.Day > 6 {
			return fmt.Errorf("invalid availability window")
		}
	}
	for _, d := range s.Blocked {
		if _, e := time.Parse("2006-01-02", d); e != nil {
			return fmt.Errorf("invalid blocked date")
		}
	}
	if len(s.Content.Videos) > 10 {
		return fmt.Errorf("maximum ten videos")
	}
	seenVideos := map[string]bool{}
	for _, v := range s.Content.Videos {
		if !videoID.MatchString(v) {
			return fmt.Errorf("YouTube video ID must contain 11 letters, digits, - or _")
		}
		if seenVideos[v] {
			return fmt.Errorf("duplicate YouTube video ID")
		}
		seenVideos[v] = true
	}
	if math.IsNaN(float64(s.TaxBPS)) {
		return fmt.Errorf("invalid tax")
	}
	return nil
}

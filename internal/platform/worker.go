package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RunWorker runs scheduled job processing and payment reconciliation until the server context is cancelled.
func (a *App) RunWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	reconcile := time.NewTicker(2 * time.Minute)
	defer reconcile.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if e := a.WorkOnce(ctx); e != nil { /* error status persisted; do not log PII */
			}
		case <-reconcile.C:
			a.Reconcile(ctx)
		}
	}
}

// WorkOnce checks a small read snapshot before atomically cleaning expired state
// and claiming a due job, then performs its operation and records completion/retry.
func (a *App) WorkOnce(ctx context.Context) error {
	var job *Job
	now := a.Now()
	needed := false
	if err := a.Store.ViewWithoutOrderSnapshots(ctx, []string{"orders", "otps", "sessions", "bookings", "jobs"}, func(s *State) error {
		needed = workerNeedsUpdate(s, now)
		return nil
	}); err != nil {
		return err
	}
	if !needed {
		return nil
	}
	// The preflight is an optimisation only. Cleanup and job claiming below
	// recheck current records inside the existing atomic write transaction.
	e := a.Store.Update(ctx, func(s *State) error {
		// Remove expired authentication state and release abandoned holds.
		for _, o := range s.Orders {
			if o.Gemstone != nil && o.PaidAt == nil && !now.Before(o.Gemstone.HoldUntil) && (o.Status == "creating" || o.Status == "pending" || o.Status == "failed" || o.Status == "creation_failed") {
				o.Status, o.Gemstone.Fulfilment = "expired", "payment_expired"
			}
		}
		for k, v := range s.OTPs {
			if now.After(v.Expires.Add(time.Hour)) {
				delete(s.OTPs, k)
			}
		}
		for k, v := range s.Sessions {
			if now.After(v.Expires) {
				delete(s.Sessions, k)
			}
		}
		for _, b := range s.Bookings {
			if b.Status == "held" && !now.Before(b.HoldUntil) {
				b.Status = "expired"
			}
		}
		for _, j := range s.Jobs {
			if (j.Status == "pending" || j.Status == "retry" || j.Status == "running" && now.After(j.Lease)) && !j.Next.After(now) {
				j.Status = "running"
				j.Attempts++
				j.Lease = now.Add(3 * time.Minute)
				copy := *j
				job = &copy
				break
			}
		}
		return nil
	})
	if e != nil || job == nil {
		return e
	}
	var order *Order
	var booking *Booking
	var meeting Meeting
	phone := ""
	skip := false
	e = a.Store.View(ctx, func(s *State) error {
		order = s.Orders[job.Ref]
		booking = s.Bookings[job.Ref]
		meeting = s.Meetings[job.Ref]
		if job.Kind == "pdf" {
			if order == nil || order.Status != "paid" {
				skip = true
			}
		} else if job.Kind == "zoom" || job.Kind == "cancel_zoom" {
			if booking == nil || booking.Version != job.Version {
				skip = true
			}
		}
		if job.Kind == "notification" {
			if booking != nil {
				if booking.Version != job.Version || booking.Status != "confirmed" && job.Payload != "booking_cancelled" {
					skip = true
				}
				if u := s.Users[booking.UserID]; u != nil && !u.Deleted {
					phone = u.Phone
				}
			} else if order != nil {
				if u := s.Users[order.UserID]; u != nil && !u.Deleted {
					phone = u.Phone
				}
			}
			if phone == "" {
				skip = true
			}
		}
		return nil
	})
	if e != nil {
		return e
	}
	var result Meeting
	if !skip {
		switch job.Kind {
		case "pdf":
			var b []byte
			b, e = a.PDF.Render(ctx, order)
			if e == nil {
				dir := filepath.Join(a.Config.DataDir, "pdf")
				e = os.MkdirAll(dir, 0700)
				if e == nil {
					target := filepath.Join(dir, order.ID+".pdf")
					tmp := target + ".tmp"
					e = os.WriteFile(tmp, b, 0600)
					if e == nil {
						e = os.Rename(tmp, target)
					}
				}
			}
		case "zoom":
			result, e = a.Zoom.Create(ctx, booking, int(booking.End.Sub(booking.Start).Minutes()))
		case "cancel_zoom":
			if meeting.ID != "" {
				e = a.Zoom.Delete(ctx, meeting.ID)
			}
		case "delete_meeting":
			e = a.Zoom.Delete(ctx, job.Payload)
		case "privacy_file":
			e = os.Remove(filepath.Join(a.Config.DataDir, "pdf", job.Ref+".pdf"))
			if os.IsNotExist(e) {
				e = nil
			}
		case "notification":
			e = a.Notify.Send(ctx, phone, job.Payload, job.ID)
		case "refund":
			if order == nil || order.PaidAt == nil {
				skip = true
			} else {
				_, e = a.Pay.Refund(ctx, order, job.ID)
			}
		default:
			e = fmt.Errorf("unknown job type")
		}
	}
	workErr := e
	return a.Store.Update(ctx, func(s *State) error {
		j := s.Jobs[job.ID]
		if j == nil || !j.Lease.Equal(job.Lease) {
			return nil
		}
		if workErr != nil {
			j.Status = "retry"
			j.Error = fmt.Sprintf("%s provider/renderer operation failed; inspect configuration or retry", j.Kind)
			j.Next = a.Now().Add(time.Duration(1<<min(j.Attempts, 10)) * time.Second)
			audit(s, "worker", "job.retry", j.ID)
			return nil
		}
		j.Status = "done"
		j.Error = ""
		if skip {
			return nil
		}
		switch job.Kind {
		case "zoom":
			b := s.Bookings[j.Ref]
			if b == nil || b.Version != j.Version || b.Status != "meeting_pending" {
				if result.ID != "" {
					enqueue(s, "delete_meeting", j.Ref, j.Version, result.ID)
				}
				break
			}
			s.Meetings[b.ID] = result
			b.JoinURL = result.JoinURL
			b.Status = "confirmed"
			enqueue(s, "notification", b.ID, b.Version, "booking_confirmed")
			reminderID := fmt.Sprintf("notification:%s:%d", b.ID, b.Version+100000)
			if b.Start.Add(-time.Hour).After(a.Now()) {
				s.Jobs[reminderID] = &Job{ID: reminderID, Kind: "notification", Ref: b.ID, Version: b.Version, Status: "pending", Next: b.Start.Add(-time.Hour), Payload: "booking_reminder"}
			}
			audit(s, "worker", "booking.confirmed", b.ID)
			if u := s.Users[b.UserID]; u != nil && u.Analytics {
				s.Events = append(s.Events, Event{Name: "consultation_confirmed", Product: "consultation", At: a.Now()})
			}
		case "cancel_zoom":
			b := s.Bookings[j.Ref]
			if b != nil && b.Version == j.Version {
				b.Status = "cancelled"
				b.JoinURL = ""
				delete(s.Meetings, b.ID)
				enqueue(s, "notification", b.ID, b.Version, "booking_cancelled")
				audit(s, "worker", "booking.cancelled", b.ID)
			}
		case "pdf":
			enqueue(s, "notification", j.Ref, 1, "pdf_ready")
			audit(s, "worker", "pdf.delivered", j.Ref)
			if o := s.Orders[j.Ref]; o != nil {
				if u := s.Users[o.UserID]; u != nil && u.Analytics {
					s.Events = append(s.Events, Event{Name: "pdf_delivered", Product: "pdf", At: a.Now()})
				}
			}
		case "refund":
			if o := s.Orders[j.Ref]; o != nil {
				if o.Demo {
					p := Payment{ID: o.PaymentID, OrderID: o.ProviderID, Status: "refunded", Amount: o.Amount, Currency: o.Currency, Refunded: o.Amount, Fee: o.Fees}
					if e := applyPayment(s, o, p, a.Now()); e != nil {
						return e
					}
				} else {
					audit(s, "worker", "refund.submitted-awaiting-provider", o.ID)
				}
			}
		}
		return nil
	})
}

// workerNeedsUpdate detects due jobs and expired authentication/reservations in a
// small read snapshot, avoiding a full-state write transaction on every idle tick.
// A concurrent change missed here is picked up by the next two-second worker tick.
func workerNeedsUpdate(s *State, now time.Time) bool {
	for _, o := range s.Orders {
		if o.Gemstone != nil && o.PaidAt == nil && !now.Before(o.Gemstone.HoldUntil) && (o.Status == "creating" || o.Status == "pending" || o.Status == "failed" || o.Status == "creation_failed") {
			return true
		}
	}
	for _, o := range s.OTPs {
		if now.After(o.Expires.Add(time.Hour)) {
			return true
		}
	}
	for _, session := range s.Sessions {
		if now.After(session.Expires) {
			return true
		}
	}
	for _, booking := range s.Bookings {
		if booking.Status == "held" && !now.Before(booking.HoldUntil) {
			return true
		}
	}
	for _, job := range s.Jobs {
		if (job.Status == "pending" || job.Status == "retry" || job.Status == "running" && now.After(job.Lease)) && !job.Next.After(now) {
			return true
		}
	}
	return false
}

// Reconcile fetches outstanding Razorpay payment states and applies confirmed changes transactionally.
func (a *App) Reconcile(ctx context.Context) {
	if a.Config.PaymentMode != "razorpay" {
		return
	}
	orders := []*Order{}
	_ = a.Store.View(ctx, func(s *State) error {
		for _, o := range s.Orders {
			if !o.Demo && o.ProviderID != "" && o.Status != "refunded" && o.Created.After(a.Now().AddDate(0, -3, 0)) {
				orders = append(orders, o)
			}
		}
		return nil
	})
	for _, o := range orders {
		if ctx.Err() != nil {
			return
		}
		payments, e := a.Pay.Payments(ctx, o.ProviderID)
		if e != nil {
			continue
		}
		for _, p := range payments {
			_ = a.Store.Update(ctx, func(s *State) error {
				current := s.Orders[o.ID]
				if current == nil {
					return nil
				}
				return applyPayment(s, current, p, a.Now())
			})
		}
	}
}

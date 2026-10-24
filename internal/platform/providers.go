package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Payment struct {
	ID       string `json:"id"`
	OrderID  string `json:"order_id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Refunded int64  `json:"amount_refunded"`
	Fee      int64  `json:"fee"`
}
type PaymentProvider interface {
	Create(context.Context, *Order) (string, error)
	Fetch(context.Context, string) (Payment, error)
	Payments(context.Context, string) ([]Payment, error)
	Refund(context.Context, *Order, string) (string, error)
}
type Razorpay struct {
	KeyID, Secret string
	Client        *http.Client
	Base          string
}

// call sends an authenticated, bounded JSON request to the configured Razorpay API endpoint.
func (r *Razorpay) call(ctx context.Context, method, path string, body any, out any, headers map[string]string) error {
	var b []byte
	if body != nil {
		b, _ = json.Marshal(body)
	}
	base := r.Base
	if base == "" {
		base = "https://api.razorpay.com/v1"
	}
	req, e := http.NewRequestWithContext(ctx, method, base+path, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.SetBasicAuth(r.KeyID, r.Secret)
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, e := r.Client.Do(req)
	if e != nil {
		return fmt.Errorf("payment provider unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("payment provider HTTP %d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
	}
	return nil
}

// Create creates a provider order for the server's INR amount and internal receipt identifier.
func (r *Razorpay) Create(ctx context.Context, o *Order) (string, error) {
	var out struct {
		ID string `json:"id"`
	}
	e := r.call(ctx, "POST", "/orders", map[string]any{"amount": o.Amount, "currency": "INR", "receipt": o.ID, "partial_payment": false}, &out, nil)
	if e == nil && out.ID == "" {
		e = fmt.Errorf("empty provider order")
	}
	return out.ID, e
}

// Fetch retrieves one provider payment for server-side verification.
func (r *Razorpay) Fetch(ctx context.Context, p string) (Payment, error) {
	var out Payment
	e := r.call(ctx, "GET", "/payments/"+url.PathEscape(p), nil, &out, nil)
	return out, e
}

// Payments lists provider payments associated with a provider order.
func (r *Razorpay) Payments(ctx context.Context, o string) ([]Payment, error) {
	var out struct {
		Items []Payment `json:"items"`
	}
	e := r.call(ctx, "GET", "/orders/"+url.PathEscape(o)+"/payments", nil, &out, nil)
	return out.Items, e
}

// Refund requests a full refund with a stable provider-safe idempotency key.
func (r *Razorpay) Refund(ctx context.Context, o *Order, key string) (string, error) {
	// Razorpay keys allow only letters, digits, hyphens and underscores, max receipt 40.
	key = "m4y_" + hash(key)[:32]
	var out struct {
		ID string `json:"id"`
	}
	e := r.call(ctx, "POST", "/payments/"+url.PathEscape(o.PaymentID)+"/refund", map[string]any{"amount": o.Amount, "receipt": key, "speed": "normal"}, &out, map[string]string{"X-Refund-Idempotency": key})
	return out.ID, e
}

type SimPayment struct{}

// Create returns a deterministic development order ID without contacting a payment provider.
func (SimPayment) Create(_ context.Context, o *Order) (string, error) { return "sim_" + o.ID, nil }

// Fetch rejects implicit simulation confirmation so a local payment must be confirmed explicitly.
func (SimPayment) Fetch(context.Context, string) (Payment, error) {
	return Payment{}, fmt.Errorf("simulation requires explicit local confirmation")
}

// Payments returns no remote payments for the development payment adapter.
func (SimPayment) Payments(context.Context, string) ([]Payment, error) { return []Payment{}, nil }

// Refund returns a stable simulated refund ID without moving real money.
func (SimPayment) Refund(_ context.Context, _ *Order, key string) (string, error) {
	return "sim_refund_" + key, nil
}

type MeetingProvider interface {
	Create(context.Context, *Booking, int) (Meeting, error)
	Delete(context.Context, string) error
}
type Zoom struct {
	Account, ClientID, Secret, User string
	Client                          *http.Client
}

// token obtains a server-to-server OAuth access token for the configured Zoom account.
func (z *Zoom) token(ctx context.Context) (string, error) {
	u := "https://zoom.us/oauth/token?grant_type=account_credentials&account_id=" + url.QueryEscape(z.Account)
	req, _ := http.NewRequestWithContext(ctx, "POST", u, nil)
	req.SetBasicAuth(z.ClientID, z.Secret)
	res, e := z.Client.Do(req)
	if e != nil {
		return "", fmt.Errorf("Zoom authentication unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return "", fmt.Errorf("Zoom authentication HTTP %d", res.StatusCode)
	}
	var out struct {
		Token string `json:"access_token"`
	}
	e = json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out)
	if out.Token == "" && e == nil {
		e = fmt.Errorf("empty Zoom token")
	}
	return out.Token, e
}

// call sends an authenticated Zoom API request and treats an already-deleted meeting as deleted.
func (z *Zoom) call(ctx context.Context, method, path string, body any, out any) error {
	token, e := z.token(ctx)
	if e != nil {
		return e
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, method, "https://api.zoom.us/v2"+path, bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	res, e := z.Client.Do(req)
	if e != nil {
		return fmt.Errorf("Zoom unavailable")
	}
	defer res.Body.Close()
	if method == "DELETE" && res.StatusCode == 404 {
		return nil
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Zoom HTTP %d", res.StatusCode)
	}
	if out != nil {
		return json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(out)
	}
	return nil
}

// Create recovers or creates the booking's versioned Zoom meeting and retains only the attendee join URL.
func (z *Zoom) Create(ctx context.Context, b *Booking, duration int) (Meeting, error) {
	// Recover a meeting created before a worker crashed. Never request or persist start_url.
	var list struct {
		Meetings []struct {
			ID      json.Number `json:"id"`
			Agenda  string      `json:"agenda"`
			JoinURL string      `json:"join_url"`
		} `json:"meetings"`
		Next string `json:"next_page_token"`
	}
	marker := fmt.Sprintf("m4y:%s:%d", b.ID, b.Version)
	page := ""
	for {
		e := z.call(ctx, "GET", "/users/"+url.PathEscape(z.User)+"/meetings?type=scheduled&page_size=300&next_page_token="+url.QueryEscape(page), nil, &list)
		if e != nil {
			return Meeting{}, e
		}
		for _, m := range list.Meetings {
			if m.Agenda == marker {
				if m.JoinURL != "" {
					return Meeting{m.ID.String(), m.JoinURL}, nil
				}
				var out struct {
					ID   json.Number `json:"id"`
					Join string      `json:"join_url"`
				}
				if e = z.call(ctx, "GET", "/meetings/"+m.ID.String(), nil, &out); e != nil {
					return Meeting{}, e
				}
				return Meeting{out.ID.String(), out.Join}, nil
			}
		}
		if list.Next == "" {
			break
		}
		page = list.Next
	}
	var out struct {
		ID   json.Number `json:"id"`
		Join string      `json:"join_url"`
	}
	e := z.call(ctx, "POST", "/users/"+url.PathEscape(z.User)+"/meetings", map[string]any{"topic": "Mishra 4 You · व्यक्तिगत परामर्श", "agenda": marker, "type": 2, "start_time": b.Start.UTC().Format("2006-01-02T15:04:05Z"), "duration": duration, "timezone": "UTC", "settings": map[string]any{"waiting_room": true, "join_before_host": false, "approval_type": 2}}, &out)
	if e == nil && (out.ID == "" || !strings.HasPrefix(out.Join, "https://")) {
		e = fmt.Errorf("Zoom did not return a valid attendee link")
	}
	return Meeting{out.ID.String(), out.Join}, e
}

// Delete deletes the specified remote Zoom meeting through the shared adapter.
func (z *Zoom) Delete(ctx context.Context, id string) error {
	return z.call(ctx, "DELETE", "/meetings/"+url.PathEscape(id), nil, nil)
}

type SimMeeting struct{}

// Create returns a development meeting reference without creating a remote meeting.
func (SimMeeting) Create(_ context.Context, b *Booking, _ int) (Meeting, error) {
	return Meeting{"sim_" + b.ID, ""}, nil
}

// Delete completes a simulated meeting deletion without contacting Zoom.
func (SimMeeting) Delete(context.Context, string) error { return nil }

type OTPProvider interface {
	Send(context.Context, string, string) error
}
type NotificationProvider interface {
	Send(context.Context, string, string, string) error
}
type Gateway struct {
	URL, Token string
	Client     *http.Client
	Simulation bool
}

// post posts a JSON message to the configured authenticated gateway, or skips delivery in simulation.
func (g Gateway) post(ctx context.Context, body any) error {
	if g.Simulation {
		return nil
	}
	b, _ := json.Marshal(body)
	req, e := http.NewRequestWithContext(ctx, "POST", g.URL, bytes.NewReader(b))
	if e != nil {
		return e
	}
	req.Header.Set("Authorization", "Bearer "+g.Token)
	req.Header.Set("Content-Type", "application/json")
	res, e := g.Client.Do(req)
	if e != nil {
		return fmt.Errorf("notification gateway unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("notification gateway HTTP %d", res.StatusCode)
	}
	return nil
}

type OTPGateway struct{ Gateway }

// Send sends a phone authentication code with its purpose and expiry metadata.
func (g OTPGateway) Send(ctx context.Context, phone, code string) error {
	return g.post(ctx, map[string]string{"phone": phone, "code": code, "purpose": "authentication", "expires_in": "300"})
}

type NotificationGateway struct{ Gateway }

// Send sends a transactional notification template with an idempotency key.
func (g NotificationGateway) Send(ctx context.Context, phone, kind, key string) error {
	return g.post(ctx, map[string]string{"phone": phone, "template": kind, "idempotency_key": key})
}

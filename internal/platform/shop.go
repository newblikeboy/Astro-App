package platform

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

type ShopSettings struct {
	Enabled        bool   `json:"enabled"`
	ShippingFee    int64  `json:"shipping_fee"`
	TaxMode        string `json:"tax_mode"`
	TaxBPS         int64  `json:"tax_bps"`
	DeliveryPolicy string `json:"delivery_policy"`
	ReturnPolicy   string `json:"return_policy"`
}

type Gemstone struct {
	ID               string                  `json:"id"`
	Slug             string                  `json:"slug"`
	Name             string                  `json:"name"`
	HindiName        string                  `json:"hindi_name"`
	Kind             string                  `json:"kind"`
	Description      string                  `json:"description"`
	HindiDescription string                  `json:"hindi_description"`
	Price            int64                   `json:"price"`
	Weight           int64                   `json:"weight_milli_carat"`
	Stock            int                     `json:"stock"`
	Available        int                     `json:"available"`
	Origin           string                  `json:"origin"`
	Material         string                  `json:"material"`
	Treatment        string                  `json:"treatment"`
	Shape            string                  `json:"shape"`
	Dimensions       string                  `json:"dimensions"`
	Image            string                  `json:"image"`
	CertificateLab   string                  `json:"certificate_lab"`
	CertificateNo    string                  `json:"certificate_no"`
	CertificateURL   string                  `json:"certificate_url"`
	Status           string                  `json:"status"`
	Version          int                     `json:"version"`
	Updated          time.Time               `json:"updated"`
	Preview          bool                    `json:"preview"`
	Sample           bool                    `json:"sample"`
	Details          map[string]GemstoneText `json:"details,omitempty"`
	Reference        *GemstoneReference      `json:"reference,omitempty"`
}

type GemstoneText struct {
	EN string `json:"en"`
	HI string `json:"hi"`
}

type GemstoneReference struct {
	CheckedAt string `json:"checked_at"`
	Rate      int64  `json:"rate_paise_per_carat"`
	Source    string `json:"source"`
}

type ShopItemRequest struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}
type ShopLine struct {
	ProductID string   `json:"product_id"`
	Quantity  int      `json:"quantity"`
	Product   Gemstone `json:"product"`
}
type DeliveryAddress struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Line1   string `json:"line1"`
	Line2   string `json:"line2"`
	City    string `json:"city"`
	State   string `json:"state"`
	PIN     string `json:"pin"`
	Country string `json:"country"`
}
type GemstoneSale struct {
	Items           []ShopLine      `json:"items"`
	Address         DeliveryAddress `json:"address"`
	Subtotal        int64           `json:"subtotal"`
	Shipping        int64           `json:"shipping"`
	DeliveryPolicy  string          `json:"delivery_policy"`
	ReturnPolicy    string          `json:"return_policy"`
	HoldUntil       time.Time       `json:"hold_until"`
	Fulfilment      string          `json:"fulfilment"`
	InventoryTaken  bool            `json:"inventory_taken"`
	Restocked       bool            `json:"restocked"`
	Courier         string          `json:"courier"`
	TrackingNumber  string          `json:"tracking_number"`
	TrackingURL     string          `json:"tracking_url"`
	ShippedAt       *time.Time      `json:"shipped_at,omitempty"`
	DeliveredAt     *time.Time      `json:"delivered_at,omitempty"`
	ShipmentVersion int             `json:"shipment_version"`
}

var shopSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
var indiaPIN = regexp.MustCompile(`^[1-9][0-9]{5}$`)
var indiaPhone = regexp.MustCompile(`^(?:\+91)?[6-9][0-9]{9}$`)

// shopReady reports whether gemstone checkout is enabled and its merchant settings are valid.
func shopReady(s ShopSettings) bool {
	return s.Enabled && validateShopSettings(s) == nil
}

// validateShopSettings validates gemstone shipping, tax and policy settings before checkout can be enabled.
func validateShopSettings(s ShopSettings) error {
	if s.ShippingFee < 0 || s.ShippingFee > 10000000 || len(s.DeliveryPolicy) > 6000 || len(s.ReturnPolicy) > 6000 {
		return fmt.Errorf("Invalid shipping fee or policy length.")
	}
	if s.TaxMode != "" && s.TaxMode != "unconfigured" && s.TaxMode != "inclusive" && s.TaxMode != "exclusive" && s.TaxMode != "none" || s.TaxBPS < 0 || s.TaxBPS > 5000 || s.TaxMode == "none" && s.TaxBPS != 0 {
		return fmt.Errorf("Invalid gemstone tax configuration.")
	}
	if s.Enabled && (s.TaxMode == "" || s.TaxMode == "unconfigured" || len(strings.TrimSpace(s.DeliveryPolicy)) < 10 || len(strings.TrimSpace(s.ReturnPolicy)) < 10) {
		return fmt.Errorf("Set gemstone tax treatment, delivery and return policies before enabling checkout.")
	}
	return nil
}

// validHTTPS accepts a bounded HTTPS URL with a hostname and no embedded credentials.
func validHTTPS(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil && len(raw) <= 1000
}

// validateGemstone validates catalogue identity, prices, integer weight, stock, disclosure and certificate metadata.
func validateGemstone(p Gemstone) error {
	allowedDetails := map[string]bool{"group": true, "planet": true, "mineral": true, "hardness": true, "color": true, "tradition": true, "wearing": true, "care": true, "buying": true}
	for key, text := range p.Details {
		if !allowedDetails[key] || len(text.EN) > 3000 || len(text.HI) > 6000 {
			return fmt.Errorf("Enter valid gemstone guide fields within their length limits.")
		}
	}
	if p.Reference != nil && (!validHTTPS(p.Reference.Source) || p.Reference.Rate <= 0 || p.Reference.Rate > 10000000000) {
		return fmt.Errorf("Enter a valid price reference source and rate.")
	}
	if p.Reference != nil {
		if _, err := time.Parse("2006-01-02", p.Reference.CheckedAt); err != nil {
			return fmt.Errorf("Enter a valid research date in YYYY-MM-DD format.")
		}
	}
	if !shopSlug.MatchString(p.Slug) || len(p.Slug) > 100 || len(strings.TrimSpace(p.Name)) < 2 || len(p.Name) > 150 || len(p.HindiName) > 250 || len(p.Description) > 6000 || len(p.HindiDescription) > 10000 {
		return fmt.Errorf("Enter a name, a unique lowercase slug and valid description lengths.")
	}
	if p.Kind != "precious" && p.Kind != "semi-precious" && p.Kind != "spiritual" {
		return fmt.Errorf("Choose a gemstone category.")
	}
	if p.Status != "draft" && p.Status != "published" && p.Status != "archived" {
		return fmt.Errorf("Invalid product status.")
	}
	if p.Price < 0 || p.Price > 10000000000 || p.Weight < 0 || p.Weight > 100000000 || p.Stock < 0 || p.Stock > 10000 {
		return fmt.Errorf("Product price, weight or stock is out of range.")
	}
	if p.Material != "natural" && p.Material != "lab-grown" && p.Material != "not-specified" {
		return fmt.Errorf("Specify the material disclosure.")
	}
	for _, field := range []string{p.Origin, p.Treatment, p.Shape, p.Dimensions, p.CertificateLab, p.CertificateNo} {
		if len(field) > 200 {
			return fmt.Errorf("Product detail is too long.")
		}
	}
	if p.Image != "" && (!strings.HasPrefix(p.Image, "/api/shop/images/") || !shopImageName.MatchString(strings.TrimPrefix(p.Image, "/api/shop/images/"))) {
		return fmt.Errorf("Upload a product photo using the photo control.")
	}
	if (p.CertificateLab == "") != (p.CertificateNo == "") || p.CertificateURL != "" && (!validHTTPS(p.CertificateURL) || p.CertificateLab == "") {
		return fmt.Errorf("A certificate needs its laboratory and report number; the optional link must use HTTPS.")
	}
	if p.Status == "published" && (p.Price <= 0 || p.Weight <= 0 || len(strings.TrimSpace(p.Description)) < 10 || p.Treatment == "") {
		return fmt.Errorf("Published stones need a price per piece, carat weight, description and treatment disclosure.")
	}
	return nil
}

// heldShopQuantity counts unexpired, unpaid reservations for a product while optionally excluding one order.
func heldShopQuantity(s *State, productID, exclude string, now time.Time) int {
	count := 0
	for _, o := range s.Orders {
		if o.ID == exclude || o.Gemstone == nil || o.PaidAt != nil || !now.Before(o.Gemstone.HoldUntil) || o.Status == "cancelled" || o.Status == "expired" {
			continue
		}
		for _, line := range o.Gemstone.Items {
			if line.ProductID == productID {
				count += line.Quantity
			}
		}
	}
	return count
}

// shopAvailable subtracts active reservations from a product's stock without returning a negative quantity.
func shopAvailable(s *State, p *Gemstone, exclude string, now time.Time) int {
	return max(0, p.Stock-heldShopQuantity(s, p.ID, exclude, now))
}

// publicGemstone copies a product with the currently available quantity for public catalogue responses.
func publicGemstone(s *State, p *Gemstone, now time.Time) Gemstone {
	out := *p
	out.Available = shopAvailable(s, p, "", now)
	out.Stock = out.Available
	return out
}

// orderedGemstones returns name-sorted catalogue entries, filtering unpublished products from public results.
func orderedGemstones(s *State, now time.Time, public bool) []Gemstone {
	out := []Gemstone{}
	for _, p := range s.Gemstones {
		if !public || p.Status == "published" {
			copy := publicGemstone(s, p, now)
			if !public {
				copy.Stock = p.Stock
			}
			out = append(out, copy)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// previewGemstones builds clearly marked, unpriced development previews that cannot be checked out.
func previewGemstones() []Gemstone {
	out := []Gemstone{}
	for i, names := range [][3]string{{"Ruby", "Manik", "precious"}, {"Yellow sapphire", "Pukhraj", "precious"}, {"Emerald", "Panna", "precious"}, {"Blue sapphire", "Neelam", "precious"}, {"Pearl", "Moti", "precious"}, {"Amethyst", "Katela", "semi-precious"}} {
		out = append(out, Gemstone{ID: fmt.Sprintf("preview-%d", i), Slug: strings.ReplaceAll(strings.ToLower(names[0]), " ", "-"), Name: names[0], HindiName: names[1], Kind: names[2], Material: "not-specified", Status: "preview", Preview: true})
	}
	return out
}

type ShopCheckoutRequest struct {
	Items   []ShopItemRequest `json:"items"`
	Address DeliveryAddress   `json:"address"`
	Key     string            `json:"idempotency_key"`
}

// validateDelivery checks the required Indian delivery address, mobile number and six-digit PIN.
func validateDelivery(v DeliveryAddress) error {
	if len(strings.TrimSpace(v.Name)) < 2 || len(v.Name) > 120 || !indiaPhone.MatchString(v.Phone) || len(strings.TrimSpace(v.Line1)) < 5 || len(v.Line1) > 250 || len(v.Line2) > 250 || len(strings.TrimSpace(v.City)) < 2 || len(v.City) > 100 || len(strings.TrimSpace(v.State)) < 2 || len(v.State) > 100 || !indiaPIN.MatchString(v.PIN) || v.Country != "IN" {
		return fmt.Errorf("Enter a complete Indian delivery address, valid mobile number and six-digit PIN code.")
	}
	return nil
}

// ShopCheckout snapshots server-priced cart items and reserves stock transactionally before creating payment.
func (a *App) ShopCheckout(ctx context.Context, user string, in ShopCheckoutRequest) (*Order, error) {
	if len(in.Key) < 8 || len(in.Key) > 100 || len(in.Items) < 1 || len(in.Items) > 10 {
		return nil, fmt.Errorf("Choose 1-10 products and provide an order request key.")
	}
	if err := validateDelivery(in.Address); err != nil {
		return nil, err
	}
	var order *Order
	err := a.Store.Update(ctx, func(s *State) error {
		if u := s.Users[user]; u == nil || u.Deleted {
			return fmt.Errorf("A verified account is required.")
		}
		for _, o := range s.Orders {
			if o.UserID == user && o.RequestKey == in.Key {
				if o.Product != "gemstone" {
					return fmt.Errorf("Use a new order request key.")
				}
				if o.PaidAt == nil && !a.Now().Before(o.Gemstone.HoldUntil) {
					return fmt.Errorf("This reservation expired. Start a new checkout.")
				}
				if o.Status == "creating" || o.Status == "creation_failed" || o.Status == "cancelled" {
					return fmt.Errorf("Payment setup is incomplete. Check your orders or cancel this reservation before starting again.")
				}
				order = o
				return nil
			}
		}
		settings := s.Settings.Shop
		if !shopReady(settings) {
			return fmt.Errorf("Online gemstone checkout is not available yet.")
		}
		items := []ShopLine{}
		seen := map[string]bool{}
		var subtotal int64
		for _, item := range in.Items {
			p := s.Gemstones[item.ProductID]
			if p != nil && p.Sample {
				return fmt.Errorf("Sample listings are for cart estimates. The seller must confirm stock and prices before checkout.")
			}
			if seen[item.ProductID] || item.Quantity < 1 || item.Quantity > 10 || p == nil || p.Status != "published" || p.Preview {
				return fmt.Errorf("A selected product or quantity is unavailable.")
			}
			seen[item.ProductID] = true
			if shopAvailable(s, p, "", a.Now()) < item.Quantity {
				return fmt.Errorf("%s no longer has enough stock. Please update your bag.", p.Name)
			}
			if err := validateGemstone(*p); err != nil {
				return err
			}
			subtotal += p.Price * int64(item.Quantity)
			items = append(items, ShopLine{ProductID: p.ID, Quantity: item.Quantity, Product: *p})
		}
		amount, tax, err := orderAmounts(subtotal+settings.ShippingFee, settings.TaxBPS, settings.TaxMode)
		if err != nil {
			return err
		}
		order = &Order{ID: id("o"), UserID: user, Product: "gemstone", Title: "Gemstone order", Amount: amount, Tax: tax, TaxMode: settings.TaxMode, TaxBPS: settings.TaxBPS, Currency: "INR", Status: "creating", Created: a.Now(), Demo: a.Config.PaymentMode == "simulation" || strings.HasPrefix(a.Config.KeyID, "rzp_test_"), RequestKey: in.Key,
			Gemstone: &GemstoneSale{Items: items, Address: in.Address, Subtotal: subtotal, Shipping: settings.ShippingFee, DeliveryPolicy: settings.DeliveryPolicy, ReturnPolicy: settings.ReturnPolicy, HoldUntil: a.Now().Add(15 * time.Minute), Fulfilment: "awaiting_payment", ShipmentVersion: 1}}
		s.Orders[order.ID] = order
		audit(s, user, "shop.checkout-created", order.ID)
		return nil
	})
	if err != nil || order.Status != "creating" {
		return order, err
	}
	// Persist the stock reservation before calling the existing payment provider.
	provider, err := a.Pay.Create(ctx, order)
	if err != nil {
		return nil, err
	}
	err = a.Store.Update(ctx, func(s *State) error {
		o := s.Orders[order.ID]
		if o == nil {
			return fmt.Errorf("Order unavailable.")
		}
		if o.ProviderID == "" {
			o.ProviderID, o.Status = provider, "pending"
		}
		order = o
		return nil
	})
	return order, err
}

// Captured money must be retained even if an expired reservation lost its stock.
// In that case fulfilment requires resolution instead of overselling a stone.
func captureGemstone(s *State, o *Order, now time.Time) {
	g := o.Gemstone
	if g == nil {
		return
	}
	for _, line := range g.Items {
		p := s.Gemstones[line.ProductID]
		if p == nil || shopAvailable(s, p, o.ID, now) < line.Quantity {
			g.Fulfilment = "needs_resolution"
			audit(s, "payment-provider", "shop.stock-resolution-required", o.ID)
			return
		}
	}
	for _, line := range g.Items {
		p := s.Gemstones[line.ProductID]
		p.Stock -= line.Quantity
		p.Version++
		p.Updated = now
	}
	g.InventoryTaken, g.Fulfilment = true, "processing"
}

// refundGemstone restores eligible gemstone stock once and flags dispatched returns for administrator review.
func refundGemstone(s *State, o *Order) {
	g := o.Gemstone
	if g == nil {
		return
	}
	if g.ShippedAt != nil {
		g.Fulfilment = "return_review"
		return
	}
	if g.InventoryTaken && !g.Restocked {
		for _, line := range g.Items {
			if p := s.Gemstones[line.ProductID]; p != nil {
				p.Stock += line.Quantity
				p.Version++
			}
		}
		g.Restocked = true
	}
	g.Fulfilment = "refunded"
}

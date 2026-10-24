package platform

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var shopImageName = regexp.MustCompile(`^gem_[A-Za-z0-9_-]+\.(?:jpg|png)$`)

// shopCatalog serves published products and public merchant settings, with development previews when appropriate.
func (a *App) shopCatalog(w http.ResponseWriter, r *http.Request) {
	err := a.Store.ViewWithoutOrderSnapshots(r.Context(), []string{"settings", "gemstones", "orders"}, func(s *State) error {
		products := orderedGemstones(s, a.Now(), true)
		preview := len(products) == 0 && !a.Config.Production
		if preview {
			products = previewGemstones()
		}
		jsonOut(w, map[string]any{"products": products, "settings": s.Settings.Shop, "checkout_available": shopReady(s.Settings.Shop), "preview": preview})
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("The gemstone collection is temporarily unavailable."))
	}
}

// shopCheckoutHTTP authenticates a cart checkout request before invoking the gemstone checkout service.
func (a *App) shopCheckoutHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	var in ShopCheckoutRequest
	if !decode(w, r, &in) {
		return
	}
	out, err := a.ShopCheckout(r.Context(), sess.UserID, in)
	if err != nil {
		fail(w, 409, err)
		return
	}
	jsonOut(w, out)
}

// shopCancelHTTP cancels an owned unpaid gemstone reservation and releases its held quantities.
func (a *App) shopCancelHTTP(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	err := a.Store.Update(r.Context(), func(s *State) error {
		o := s.Orders[r.PathValue("id")]
		if o == nil || o.UserID != sess.UserID || o.Gemstone == nil || o.PaidAt != nil {
			return fmt.Errorf("Only your unpaid gemstone reservations can be cancelled here.")
		}
		o.Status, o.Gemstone.Fulfilment = "cancelled", "cancelled"
		audit(s, sess.UserID, "shop.reservation-cancelled", o.ID)
		return nil
	})
	if err != nil {
		fail(w, 409, err)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// adminShop returns the administrator catalogue and gemstone order-management data.
func (a *App) adminShop(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	err := a.Store.View(r.Context(), func(s *State) error {
		orders := []*Order{}
		for _, o := range s.Orders {
			if o.Product == "gemstone" {
				orders = append(orders, o)
			}
		}
		sort.Slice(orders, func(i, j int) bool { return orders[i].Created.After(orders[j].Created) })
		jsonOut(w, map[string]any{"products": orderedGemstones(s, a.Now(), false), "settings": s.Settings.Shop, "orders": orders})
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Gemstone management is temporarily unavailable."))
	}
}

// adminShopSettings validates and saves gemstone merchant settings with administrator audit history.
func (a *App) adminShopSettings(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in ShopSettings
	if !decode(w, r, &in) {
		return
	}
	if err := validateShopSettings(in); err != nil {
		fail(w, 422, err)
		return
	}
	err := a.Store.Update(r.Context(), func(s *State) error {
		s.Settings.Shop = in
		audit(s, sess.UserID, "shop.settings-saved", "shop")
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Shop settings could not be saved."))
		return
	}
	jsonOut(w, in)
}

// adminGemstone creates or updates an administrator catalogue product with version and validation checks.
func (a *App) adminGemstone(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in Gemstone
	if !decode(w, r, &in) {
		return
	}
	in.Name, in.Slug = strings.TrimSpace(in.Name), strings.TrimSpace(in.Slug)
	if err := validateGemstone(in); err != nil {
		fail(w, 422, err)
		return
	}
	err := a.Store.Update(r.Context(), func(s *State) error {
		for _, p := range s.Gemstones {
			if p.Slug == in.Slug && p.ID != in.ID {
				return fmt.Errorf("This product slug is already used.")
			}
		}
		if in.ID == "" {
			in.ID = id("stone")
			in.Version = 1
		} else {
			old := s.Gemstones[in.ID]
			if old == nil || old.Version != in.Version {
				return fmt.Errorf("This stone changed. Reload its current stock before saving.")
			}
			in.Version++
		}
		if in.Stock < heldShopQuantity(s, in.ID, "", a.Now()) {
			return fmt.Errorf("Stock cannot be lower than active checkout reservations.")
		}
		in.Available, in.Preview, in.Updated = 0, false, a.Now()
		p := in
		s.Gemstones[p.ID] = &p
		audit(s, sess.UserID, "shop.product-saved", p.ID)
		return nil
	})
	if err != nil {
		fail(w, 409, err)
		return
	}
	jsonOut(w, in)
}

// adminShopFulfilment updates paid gemstone order fulfilment and validates shipment tracking details.
func (a *App) adminShopFulfilment(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, true)
	if !ok {
		return
	}
	var in struct {
		Status  string `json:"status"`
		Courier string `json:"courier"`
		Number  string `json:"tracking_number"`
		URL     string `json:"tracking_url"`
		Version int    `json:"version"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Courier) > 120 || len(in.Number) > 150 || in.URL != "" && !validHTTPS(in.URL) {
		fail(w, 422, fmt.Errorf("Enter valid shipment details and an HTTPS tracking link."))
		return
	}
	err := a.Store.Update(r.Context(), func(s *State) error {
		o := s.Orders[r.PathValue("id")]
		if o == nil || o.Gemstone == nil || o.PaidAt == nil || o.Status != "paid" || o.Refunded != 0 {
			return fmt.Errorf("A paid, unrefunded gemstone order is required.")
		}
		g := o.Gemstone
		if g.ShipmentVersion != in.Version {
			return fmt.Errorf("The shipment changed. Reload before updating it.")
		}
		if in.Status == "processing" && g.Fulfilment == "needs_resolution" {
			captureGemstone(s, o, a.Now())
			if g.Fulfilment != "processing" {
				return fmt.Errorf("Restock the reserved products before resolving this order.")
			}
		} else if in.Status == "shipped" && g.Fulfilment == "processing" {
			if strings.TrimSpace(in.Courier) == "" || strings.TrimSpace(in.Number) == "" {
				return fmt.Errorf("Courier and tracking number are required before dispatch.")
			}
			now := a.Now()
			g.ShippedAt = &now
			g.Courier, g.TrackingNumber, g.TrackingURL = in.Courier, in.Number, in.URL
			g.Fulfilment = "shipped"
		} else if in.Status == "delivered" && g.Fulfilment == "shipped" {
			now := a.Now()
			g.DeliveredAt = &now
			g.Fulfilment = "delivered"
		} else {
			return fmt.Errorf("Invalid fulfilment transition.")
		}
		g.ShipmentVersion++
		audit(s, sess.UserID, "shop.fulfilment-updated", o.ID)
		return nil
	})
	if err != nil {
		fail(w, 409, err)
		return
	}
	jsonOut(w, map[string]bool{"ok": true})
}

// adminShopImage validates and stores an administrator-uploaded gemstone image.
func (a *App) adminShopImage(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.guard(w, r, true); !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		fail(w, 422, fmt.Errorf("Upload a JPEG or PNG photo smaller than 5 MB."))
		return
	}
	defer r.MultipartForm.RemoveAll()
	f, _, err := r.FormFile("image")
	if err != nil {
		fail(w, 422, fmt.Errorf("Choose a product photo."))
		return
	}
	defer f.Close()
	b, err := io.ReadAll(io.LimitReader(f, (5<<20)+1))
	if err != nil || len(b) > 5<<20 {
		fail(w, 422, fmt.Errorf("Photo exceeds 5 MB."))
		return
	}
	info, format, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil || (format != "jpeg" && format != "png") || info.Width > 8000 || info.Height > 8000 || int64(info.Width)*int64(info.Height) > 32000000 {
		fail(w, 422, fmt.Errorf("Use a valid JPEG or PNG photo up to 8000 pixels per side and 32 megapixels."))
		return
	}
	ext := ".jpg"
	if format == "png" {
		ext = ".png"
	}
	name := id("gem") + ext
	dir := filepath.Join(a.Config.DataDir, "shop-images")
	if err = os.MkdirAll(dir, 0700); err == nil {
		err = os.WriteFile(filepath.Join(dir, name), b, 0600)
	}
	if err != nil {
		fail(w, 503, fmt.Errorf("The photo could not be saved."))
		return
	}
	jsonOut(w, map[string]string{"image": "/api/shop/images/" + name})
}

// shopImage serves a stored gemstone image with its recorded image content type.
func (a *App) shopImage(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if !shopImageName.MatchString(name) {
		http.NotFound(w, r)
		return
	}
	contentType := "image/jpeg"
	if strings.HasSuffix(name, ".png") {
		contentType = "image/png"
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeFile(w, r, filepath.Join(a.Config.DataDir, "shop-images", name))
}

package platform

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// shopFixture creates an isolated enabled shop with test customers, priced stock and a valid cart request.
func shopFixture(t *testing.T, stock int) (*App, ShopCheckoutRequest) {
	t.Helper()
	a := testApp(t)
	putUser(t, a, "buyer")
	putUser(t, a, "other")
	err := a.Store.Update(context.Background(), func(s *State) error {
		s.Settings.Shop = ShopSettings{Enabled: true, ShippingFee: 12500, TaxMode: "exclusive", TaxBPS: 300, DeliveryPolicy: "Delivery after dispatch confirmation.", ReturnPolicy: "Contact the seller to request a return."}
		s.Gemstones["ruby"] = &Gemstone{ID: "ruby", Slug: "ruby-425", Name: "Ruby", Description: "Synthetic QA stone record only.", Kind: "precious", Price: 123456, Weight: 4250, Stock: stock, Treatment: "Not assessed", Material: "not-specified", Status: "published", Version: 1}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return a, ShopCheckoutRequest{Key: "shop-request-001", Items: []ShopItemRequest{{ProductID: "ruby", Quantity: 1}}, Address: DeliveryAddress{Name: "QA Buyer", Phone: "9876543210", Line1: "12 Test Street", City: "Delhi", State: "Delhi", PIN: "110001", Country: "IN"}}
}

// TestShopCheckoutServerTotalsSnapshotAndCaptureReplay checks authoritative checkout totals, order snapshots and repeated payment capture.
func TestShopCheckoutServerTotalsSnapshotAndCaptureReplay(t *testing.T) {
	a, in := shopFixture(t, 2)
	o, err := a.ShopCheckout(context.Background(), "buyer", in)
	if err != nil {
		t.Fatal(err)
	}
	if o.Amount != 140035 || o.Tax != 4079 || o.Gemstone.Subtotal != 123456 || o.Gemstone.Shipping != 12500 {
		t.Fatalf("wrong exact totals: %+v", o)
	}
	again, err := a.ShopCheckout(context.Background(), "buyer", in)
	if err != nil || again.ID != o.ID {
		t.Fatal("idempotency", err)
	}
	err = a.Store.Update(context.Background(), func(s *State) error {
		s.Gemstones["ruby"].Price = 777777
		current := s.Orders[o.ID]
		p := Payment{ID: "shop-payment", OrderID: current.ProviderID, Amount: current.Amount, Currency: "INR", Status: "captured"}
		if err := applyPayment(s, current, p, a.Now()); err != nil {
			return err
		}
		if err := applyPayment(s, current, p, a.Now()); err != nil {
			return err
		}
		if s.Gemstones["ruby"].Stock != 1 || current.Gemstone.Items[0].Product.Price != 123456 || current.Gemstone.Fulfilment != "processing" || len(s.Jobs) != 0 {
			t.Fatal("stock, immutable price or accidental PDF/Zoom work")
		}
		f := finances(s, a.Now().Format("2006-01"), true)
		if f.GemstoneSales != 140035 || f.ConsultationSales != 0 || f.Gross != 140035 {
			t.Fatal("gemstone misclassified in ledger", f)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestShopLastStoneConcurrentReservationsAndExpiry checks competing reservations for the last stone and release of expired holds.
func TestShopLastStoneConcurrentReservationsAndExpiry(t *testing.T) {
	a, in := shopFixture(t, 1)
	var wg sync.WaitGroup
	var mu sync.Mutex
	successes := []*Order{}
	for i, user := range []string{"buyer", "other"} {
		wg.Add(1)
		go func(i int, user string) {
			defer wg.Done()
			req := in
			req.Key += user
			o, e := a.ShopCheckout(context.Background(), user, req)
			if e == nil {
				mu.Lock()
				successes = append(successes, o)
				mu.Unlock()
			}
		}(i, user)
	}
	wg.Wait()
	if len(successes) != 1 {
		t.Fatalf("last stone reserved %d times", len(successes))
	}
	old := successes[0]
	base := a.Now()
	a.Now = func() time.Time { return base.Add(16 * time.Minute) }
	if err := a.WorkOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	in.Key = "replacement-reservation"
	newOrder, err := a.ShopCheckout(context.Background(), "buyer", in)
	if err != nil {
		t.Fatal(err)
	}
	err = a.Store.Update(context.Background(), func(s *State) error {
		o := s.Orders[old.ID]
		if err := applyPayment(s, o, Payment{ID: "late-shop-payment", OrderID: o.ProviderID, Amount: o.Amount, Currency: "INR", Status: "captured"}, a.Now()); err != nil {
			return err
		}
		if o.Status != "paid" || o.Gemstone.Fulfilment != "needs_resolution" || o.Gemstone.InventoryTaken || s.Gemstones["ruby"].Stock != 1 || s.Orders[newOrder.ID].Status != "pending" {
			t.Fatal("late capture stole the replacement reservation")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestShopRefundStockOnceAndDispatchedReturnReview checks one-time refund restocking and review requirements after dispatch.
func TestShopRefundStockOnceAndDispatchedReturnReview(t *testing.T) {
	for _, shipped := range []bool{false, true} {
		a, in := shopFixture(t, 1)
		o, err := a.ShopCheckout(context.Background(), "buyer", in)
		if err != nil {
			t.Fatal(err)
		}
		err = a.Store.Update(context.Background(), func(s *State) error {
			o := s.Orders[o.ID]
			p := Payment{ID: "refundable", OrderID: o.ProviderID, Amount: o.Amount, Currency: "INR", Status: "captured"}
			if err := applyPayment(s, o, p, a.Now()); err != nil {
				return err
			}
			if shipped {
				now := a.Now()
				o.Gemstone.ShippedAt = &now
				o.Gemstone.Fulfilment = "shipped"
			}
			p.Refunded, p.Status = o.Amount, "refunded"
			if err := applyPayment(s, o, p, a.Now()); err != nil {
				return err
			}
			if err := applyPayment(s, o, p, a.Now()); err != nil {
				return err
			}
			want := 1
			if shipped {
				want = 0
			}
			if s.Gemstones["ruby"].Stock != want {
				t.Fatal("refund restocked more than once or restocked a dispatched stone")
			}
			if shipped && o.Gemstone.Fulfilment != "return_review" {
				t.Fatal("dispatched refund needs inspection")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestShopAdminPermissionsVersionsAndCatalog checks catalogue administrator permissions and version conflicts.
func TestShopAdminPermissionsVersionsAndCatalog(t *testing.T) {
	a, _ := shopFixture(t, 1)
	user, csrf := authorize(t, a, "buyer", "customer")
	admin, adminCSRF := authorize(t, a, "admin", "admin")
	if w := request(t, a, "GET", "/api/admin/shop", nil, user, csrf); w.Code != 401 && w.Code != 403 {
		t.Fatal("customer reached shop admin", w.Code)
	}
	if w := request(t, a, "POST", "/api/admin/shop/products", Gemstone{}, admin, ""); w.Code != 403 {
		t.Fatal("missing CSRF accepted", w.Code)
	}
	var p Gemstone
	_ = a.Store.View(context.Background(), func(s *State) error { p = *s.Gemstones["ruby"]; return nil })
	p.Name = "Ruby edited"
	if w := request(t, a, "POST", "/api/admin/shop/products", p, admin, adminCSRF); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(t, a, "POST", "/api/admin/shop/products", p, admin, adminCSRF); w.Code != 409 {
		t.Fatal("stale stock overwrite accepted")
	}
	if w := request(t, a, "GET", "/api/shop", nil, nil, ""); w.Code != 200 || strings.Contains(w.Body.String(), "9876543210") {
		t.Fatal("catalog failed or disclosed a phone")
	}
	if validateShopSettings(ShopSettings{Enabled: true}) == nil {
		t.Fatal("checkout enabled without policies/tax")
	}
	if validateGemstone(Gemstone{Slug: "bad", Name: "Bad", Kind: "precious", Status: "published", Material: "natural", Price: 1, Weight: 1, Description: "Product description", Treatment: "None", CertificateURL: "javascript:alert(1)"}) == nil {
		t.Fatal("invalid certificate URL")
	}
}

// TestShopShipmentRequiresPaidOrderAndTracking rejects shipment updates without confirmed payment or required tracking details.
func TestShopShipmentRequiresPaidOrderAndTracking(t *testing.T) {
	a, in := shopFixture(t, 1)
	o, err := a.ShopCheckout(context.Background(), "buyer", in)
	if err != nil {
		t.Fatal(err)
	}
	admin, csrf := authorize(t, a, "admin", "admin")
	path := "/api/admin/shop/orders/" + o.ID + "/fulfilment"
	body := map[string]any{"status": "shipped", "version": 1, "courier": "QA Courier", "tracking_number": "QA123", "tracking_url": "https://example.com/track/QA123"}
	if w := request(t, a, "POST", path, body, admin, csrf); w.Code != 409 {
		t.Fatal("unpaid shipment allowed")
	}
	capture(t, a, o)
	if w := request(t, a, "POST", path, body, admin, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w := request(t, a, "POST", path, body, admin, csrf); w.Code != 409 {
		t.Fatal("stale shipment accepted")
	}
	body = map[string]any{"status": "delivered", "version": 2}
	if w := request(t, a, "POST", path, body, admin, csrf); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
}

// TestShopPhotoUploadAndMime checks product-photo uploads and accepted content types.
func TestShopPhotoUploadAndMime(t *testing.T) {
	a := testApp(t)
	cookie, csrf := authorize(t, a, "admin", "admin")
	var data bytes.Buffer
	form := multipart.NewWriter(&data)
	part, _ := form.CreateFormFile("image", "actual.png")
	_ = png.Encode(part, image.NewRGBA(image.Rect(0, 0, 16, 16)))
	_ = form.Close()
	req := httptest.NewRequest("POST", "/api/admin/shop/images", &data)
	req.Header.Set("Content-Type", form.FormDataContentType())
	req.Header.Set("X-CSRF-Token", csrf)
	req.AddCookie(cookie)
	w := httptest.NewRecorder()
	a.Handler().ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	imageResponse := request(t, a, http.MethodGet, out["image"], nil, nil, "")
	if imageResponse.Code != 200 || imageResponse.Header().Get("Content-Type") != "image/png" {
		t.Fatal("image MIME incorrect", imageResponse.Header())
	}
}

// TestShopRequestValidationAndCartTampering rejects malformed shop requests and manipulated cart inputs.
func TestShopRequestValidationAndCartTampering(t *testing.T) {
	a, in := shopFixture(t, 2)
	in.Items = append(in.Items, in.Items[0])
	if _, err := a.ShopCheckout(context.Background(), "buyer", in); err == nil {
		t.Fatal("duplicate cart line accepted")
	}
	in.Items = in.Items[:1]
	in.Items[0].Quantity = -1
	if _, err := a.ShopCheckout(context.Background(), "buyer", in); err == nil {
		t.Fatal("negative quantity accepted")
	}
	in.Items[0].Quantity = 1
	in.Address.PIN = "000000"
	if _, err := a.ShopCheckout(context.Background(), "buyer", in); err == nil {
		t.Fatal("invalid delivery accepted")
	}
	if len(seed().Gemstones) != 0 {
		t.Fatal("fake merchandise seeded as real stock")
	}
	if _, err := encodeState(a.Store.state); err != nil {
		t.Fatal("new shop entities cannot persist", err)
	}
}

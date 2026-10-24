package platform

import (
	"context"
	"testing"
)

// TestStarterCatalogueSeedPreservesEdits checks complete bilingual guides, valid prices and repeat-safe seeding.
func TestStarterCatalogueSeedPreservesEdits(t *testing.T) {
	products, err := StarterGemstones()
	if err != nil || len(products) != 16 {
		t.Fatalf("starter products: %d, %v", len(products), err)
	}
	for _, p := range products {
		if p.Price*1000 != p.Reference.Rate*p.Weight {
			t.Fatalf("incorrect reference total: %s", p.Slug)
		}
		for _, key := range []string{"planet", "mineral", "hardness", "color", "tradition", "wearing", "care", "buying", "group"} {
			if p.Details[key].EN == "" || p.Details[key].HI == "" {
				t.Fatalf("missing bilingual %s/%s", p.Slug, key)
			}
		}
	}
	a := testApp(t)
	count, err := SeedGemstoneCatalog(context.Background(), a.Store)
	if err != nil || count != 16 {
		t.Fatalf("seed count: %d, %v", count, err)
	}
	err = a.Store.Update(context.Background(), func(s *State) error {
		p := s.Gemstones["catalog-ruby"]
		p.Price = 987600
		p.Description = "Merchant edited this description."
		p.Stock = 2
		p.Sample = false
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	count, err = SeedGemstoneCatalog(context.Background(), a.Store)
	if err != nil || count != 0 {
		t.Fatalf("repeat seed: %d, %v", count, err)
	}
	err = a.Store.View(context.Background(), func(s *State) error {
		p := s.Gemstones["catalog-ruby"]
		if p.Price != 987600 || p.Stock != 2 || p.Sample || p.Description != "Merchant edited this description." {
			t.Fatal("seed overwrote merchant edits")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TestSampleListingsCannotCheckout checks the server refuses sample inventory even when merchant checkout is enabled.
func TestSampleListingsCannotCheckout(t *testing.T) {
	a, request := shopFixture(t, 10)
	err := a.Store.Update(context.Background(), func(s *State) error { s.Gemstones["ruby"].Sample = true; return nil })
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.ShopCheckout(context.Background(), "buyer", request); err == nil {
		t.Fatal("sample listing accepted for payment")
	}
	_ = a.Store.View(context.Background(), func(s *State) error {
		if len(s.Orders) != 0 {
			t.Fatal("sample created an order")
		}
		return nil
	})
}

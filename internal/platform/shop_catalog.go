package platform

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"
)

//go:embed data/gemstone-catalog.json
var starterGemstoneData []byte

// StarterGemstones decodes the researched sample catalogue without modifying storage or merchant settings.
func StarterGemstones() ([]Gemstone, error) {
	var products []Gemstone
	if err := json.Unmarshal(starterGemstoneData, &products); err != nil {
		return nil, err
	}
	for _, p := range products {
		if !p.Sample || p.Preview || p.Status != "published" {
			return nil, fmt.Errorf("invalid sample catalogue flags")
		}
		if err := validateGemstone(p); err != nil {
			return nil, err
		}
	}
	return products, nil
}

// SeedGemstoneCatalog adds missing sample slugs while preserving merchant edits, stock, settings and orders.
func SeedGemstoneCatalog(ctx context.Context, store *Store) (int, error) {
	products, err := StarterGemstones()
	if err != nil {
		return 0, err
	}
	added := 0
	err = store.Update(ctx, func(s *State) error {
		added = 0
		if s.Gemstones == nil {
			s.Gemstones = map[string]*Gemstone{}
		}
		existing := map[string]bool{}
		for _, p := range s.Gemstones {
			existing[p.Slug] = true
		}
		for _, product := range products {
			if existing[product.Slug] || s.Gemstones[product.ID] != nil {
				continue
			}
			product.Updated = time.Now().UTC()
			p := product
			s.Gemstones[p.ID] = &p
			added++
		}
		if added > 0 {
			audit(s, "local-operator", "shop.sample-catalog-added", "starter-catalog-2026-10-20")
		}
		return nil
	})
	return added, err
}

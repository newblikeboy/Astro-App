package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type dailyResult struct {
	swisseph.Panchang
	Place Place `json:"place"`
}

type dailyCacheEntry struct {
	result   dailyResult
	expires  time.Time
	sequence uint64
}

// Per-App, deterministic, bounded cache. The lock also coalesces concurrent
// misses; Swiss already serializes all calculations across the process.
type dailyCache struct {
	mu       sync.Mutex
	entries  map[string]dailyCacheEntry
	sequence uint64
}

// dailyPanchang validates an India place/date request and serves its bounded, cached native daily calculation.
func (a *App) dailyPanchang(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for key, values := range q {
		if (key != "date" && key != "place_id" && key != "sunrise") || len(values) != 1 {
			fail(w, 400, fmt.Errorf("invalid Panchang query"))
			return
		}
	}
	place, ok := findPlace(q.Get("place_id"))
	if !ok || place.Zone != "Asia/Kolkata" {
		fail(w, 400, fmt.Errorf("choose a location from the suggestions"))
		return
	}
	date := q.Get("date")
	parsed, err := time.Parse("2006-01-02", date)
	if err != nil || parsed.Format("2006-01-02") != date || date < swisseph.MinDate || date > swisseph.MaxDate {
		fail(w, 400, fmt.Errorf("date must be YYYY-MM-DD between %s and %s", swisseph.MinDate, swisseph.MaxDate))
		return
	}
	convention := q.Get("sunrise")
	if convention == "" {
		convention = "upper-limb"
	}
	if convention != "upper-limb" && convention != "hindu" {
		fail(w, 400, fmt.Errorf("unsupported sunrise convention"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	key := place.ID + "/" + date + "/" + convention
	c := &a.daily
	c.mu.Lock()
	if ctx.Err() != nil {
		c.mu.Unlock()
		fail(w, 503, fmt.Errorf("Panchang is busy; please retry"))
		return
	}
	now := a.Now()
	c.sequence++
	if entry, ok := c.entries[key]; ok && now.Before(entry.expires) {
		entry.sequence = c.sequence
		c.entries[key] = entry
		c.mu.Unlock()
		w.Header().Set("Cache-Control", "public, max-age=300")
		jsonOut(w, entry.result)
		return
	}
	p, err := swisseph.CalculatePanchang(ctx, a.Config.EpheDir, date, place.Zone, convention, place.Lat, place.Lon)
	if err != nil {
		c.mu.Unlock()
		fail(w, 503, fmt.Errorf("Panchang calculation unavailable; please retry"))
		return
	}
	if c.entries == nil {
		c.entries = map[string]dailyCacheEntry{}
	}
	if len(c.entries) >= 128 {
		oldestKey := ""
		oldest := ^uint64(0)
		for k, e := range c.entries {
			if e.sequence < oldest {
				oldest, oldestKey = e.sequence, k
			}
		}
		delete(c.entries, oldestKey)
	}
	result := dailyResult{p, place}
	c.entries[key] = dailyCacheEntry{result, now.Add(6 * time.Hour), c.sequence}
	c.mu.Unlock()
	w.Header().Set("Cache-Control", "public, max-age=300")
	jsonOut(w, result)
}

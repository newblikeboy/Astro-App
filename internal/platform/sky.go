package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type skyCacheEntry struct {
	data     []byte
	expires  time.Time
	sequence uint64
}
type skyCache struct {
	mu       sync.Mutex
	entries  map[string]skyCacheEntry
	sequence uint64
}

// Bounds both the total number and the size of cached JSON responses. Misses
// coalesce without holding a cache lock while writing to a network client.
func (a *App) cachedSky(ctx context.Context, key string, calculate func() (any, error)) ([]byte, error) {
	c := &a.sky
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := a.Now()
	c.sequence++
	if e, ok := c.entries[key]; ok && now.Before(e.expires) {
		e.sequence = c.sequence
		c.entries[key] = e
		return e.data, nil
	}
	value, err := calculate()
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(b) > 512*1024 {
		return nil, fmt.Errorf("sky response exceeds size limit")
	}
	if c.entries == nil {
		c.entries = map[string]skyCacheEntry{}
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
	c.entries[key] = skyCacheEntry{b, now.Add(6 * time.Hour), c.sequence}
	return b, nil
}

// skyTools validates and serves cached native planet-position, annual-transit, combustion and eclipse requests.
func (a *App) skyTools(w http.ResponseWriter, r *http.Request) {
	tool := r.PathValue("tool")
	if tool != "positions" && tool != "transits" && tool != "combustion" && tool != "eclipses" {
		fail(w, 404, fmt.Errorf("sky tool not found"))
		return
	}
	allowed := map[string]bool{"place_id": true, "year": tool != "positions", "date": tool == "positions", "time": tool == "positions"}
	q := r.URL.Query()
	for k, v := range q {
		if !allowed[k] || len(v) != 1 {
			fail(w, 400, fmt.Errorf("invalid sky query"))
			return
		}
	}
	place, ok := findPlace(q.Get("place_id"))
	if !ok || place.Zone != "Asia/Kolkata" {
		fail(w, 400, fmt.Errorf("choose a location from the suggestions"))
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()
	key := ""
	var compute func() (any, error)
	if tool == "positions" {
		date, clock := q.Get("date"), q.Get("time")
		if len(date) != 10 || len(clock) != 5 || date < swisseph.MinDate || date > swisseph.MaxDate {
			fail(w, 400, fmt.Errorf("invalid sky date or time"))
			return
		}
		at, err := ResolveWall(date, clock, place.Zone, nil)
		if err != nil || at.In(mustIndiaZone()).Format("2006-01-02 15:04") != date+" "+clock {
			fail(w, 400, fmt.Errorf("invalid sky date or time"))
			return
		}
		key = tool + "/" + date + "/" + clock
		// compute calculates the selected date/time's geocentric planet positions on a cache miss.
		compute = func() (any, error) { return swisseph.CalculatePlanetPositions(ctx, a.Config.EpheDir, at) }
	} else {
		year, err := strconv.Atoi(q.Get("year"))
		if err != nil || strconv.Itoa(year) != q.Get("year") || year < swisseph.MinYear || year > swisseph.MaxYear {
			fail(w, 400, fmt.Errorf("year must be between %d and %d", swisseph.MinYear, swisseph.MaxYear))
			return
		}
		key = tool + "/" + strconv.Itoa(year)
		if tool == "transits" {
			// compute calculates the selected civil year's sign ingresses and motion stations on a cache miss.
			compute = func() (any, error) { return swisseph.CalculatePlanetTransitions(ctx, a.Config.EpheDir, year) }
		} else if tool == "combustion" {
			// compute finds annual combustion entry/exit boundaries using the shared Kundli angular limits.
			compute = func() (any, error) { return swisseph.CalculatePlanetCombustions(ctx, a.Config.EpheDir, year) }
		} else {
			key += "/" + place.ID
			// compute calculates the selected year's global and selected-place eclipse circumstances on a cache miss.
			compute = func() (any, error) {
				return swisseph.CalculateEclipses(ctx, a.Config.EpheDir, year, place.Lat, place.Lon)
			}
		}
	}
	b, err := a.cachedSky(ctx, key, compute)
	if err != nil {
		fail(w, 503, fmt.Errorf("Sky calculations unavailable. Please try again."))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(b)
}

// mustIndiaZone loads the fixed India timezone used by the public sky tools.
func mustIndiaZone() *time.Location { loc, _ := time.LoadLocation("Asia/Kolkata"); return loc }

package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"net/http"
	"time"
)

// A stateless branch query: birth inputs are recalculated, not stored. No
// browser-supplied longitude, period boundary or lord is accepted as authority.
func (a *App) dashaBranch(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Birth Birth  `json:"birth"`
		Path  []int  `json:"path"`
		AsOf  string `json:"as_of"`
	}
	if !decode(w, r, &in) {
		return
	}
	if len(in.Path) > 4 {
		fail(w, 422, fmt.Errorf("Choose a Dasha level from Mahadasha through Prana."))
		return
	}
	for i, index := range in.Path {
		limit := 9
		if i == 0 {
			limit = 18
		}
		if index < 0 || index >= limit {
			fail(w, 422, fmt.Errorf("Invalid Dasha selection."))
			return
		}
	}
	if in.Birth.Unknown || in.Birth.Time == "" {
		fail(w, 422, fmt.Errorf("A known birth time is required for Dasha calculations."))
		return
	}
	if a.Config.Production {
		if err := a.Store.View(r.Context(), func(s *State) error {
			if !s.Settings.LicenceConfirmed {
				return fmt.Errorf("licence not confirmed")
			}
			return nil
		}); err != nil {
			fail(w, 503, fmt.Errorf("Dasha calculations are unavailable right now."))
			return
		}
	}
	if r.Context().Err() != nil {
		return
	}
	now := a.Now()
	asOf := now
	if in.AsOf != "" {
		parsed, err := time.Parse(time.RFC3339Nano, in.AsOf)
		if err != nil || parsed.After(now) || parsed.Year() < 1940 || parsed.Year() > 2099 {
			fail(w, 422, fmt.Errorf("Invalid Dasha reference time."))
			return
		}
		asOf = parsed
	}
	report, err := calculateCore(a.Config, in.Birth, nil, now)
	if err != nil {
		fail(w, 422, err)
		return
	}
	if r.Context().Err() != nil {
		return
	}
	birth, err := ResolveWall(in.Birth.Date, in.Birth.Time, report.Place.Zone, in.Birth.Offset)
	if err != nil || asOf.Before(birth) {
		fail(w, 422, fmt.Errorf("Dasha reference time cannot precede birth."))
		return
	}
	var dasha swisseph.Vimshottari
	for _, p := range report.Chart.Planets {
		if p.Name == "Moon" {
			dasha = swisseph.CalculateVimshottari(p.Longitude, birth, asOf)
			break
		}
	}
	parent, periods, err := swisseph.DashaBranch(dasha, in.Path)
	if err != nil {
		fail(w, 422, fmt.Errorf("Invalid Dasha selection."))
		return
	}
	path := append([]int{}, in.Path...)
	ancestors := []swisseph.DashaStep{}
	for depth, index := range path {
		_, siblings, e := swisseph.DashaBranch(dasha, path[:depth])
		if e != nil || index >= len(siblings) {
			fail(w, 422, fmt.Errorf("Invalid Dasha selection."))
			return
		}
		p := siblings[index]
		ancestors = append(ancestors, swisseph.DashaStep{Level: depth + 1, Path: append([]int{}, path[:depth+1]...), Lord: p.Lord, Start: p.Start, End: p.End})
	}
	jsonOut(w, map[string]any{"method": swisseph.VimshottariMethod, "level": len(path) + 1, "path": path, "parent": parent, "ancestors": ancestors, "periods": periods, "current_path": dasha.CurrentPath, "birth_lord": dasha.BirthLord, "balance_years": dasha.BalanceYears, "as_of": asOf.UTC(), "timezone": report.Place.Zone, "year_days": swisseph.DashaYearDays})
}

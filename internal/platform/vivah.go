package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"fmt"
	"net/http"
	"time"
)

// vivahMuhurat validates a trusted India location and one supported civil month,
// then serves a bounded cached native marriage shortlist without creating state.
func (a *App) vivahMuhurat(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for key, values := range q {
		if (key != "month" && key != "place_id" && key != "sunrise") || len(values) != 1 {
			fail(w, 400, fmt.Errorf("invalid marriage query"))
			return
		}
	}
	place, ok := findPlace(q.Get("place_id"))
	if !ok || place.Zone != "Asia/Kolkata" {
		fail(w, 400, fmt.Errorf("choose a location from the suggestions"))
		return
	}
	month := q.Get("month")
	parsed, err := time.Parse("2006-01", month)
	if err != nil || parsed.Format("2006-01") != month || month < swisseph.MinDate[:7] || month > swisseph.MaxDate[:7] {
		fail(w, 400, fmt.Errorf("choose a month between 1940-01 and 2099-12"))
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
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	key := "vivah-v2/" + place.ID + "/" + month + "/" + convention
	body, err := a.cachedSky(ctx, key, func() (any, error) {
		result, err := swisseph.CalculateVivahMonth(ctx, a.Config.EpheDir, month, place.Zone, convention, place.Lat, place.Lon)
		if err != nil {
			return nil, err
		}
		return struct {
			swisseph.VivahMonth
			Place Place `json:"place"`
		}{result, place}, nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Marriage calculations unavailable. Please try again."))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(body)
}

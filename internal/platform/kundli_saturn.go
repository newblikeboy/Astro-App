package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"net/http"
)

// kundliSaturn validates an on-demand Sade Sati/Dhaiya request and calculates its native transit timeline.
func (a *App) kundliSaturn(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Birth     Birth `json:"birth"`
		StartYear int   `json:"start_year"`
		EndYear   int   `json:"end_year"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Birth.Unknown || in.Birth.Time == "" || in.StartYear < 1940 || in.EndYear > 2100 || in.EndYear <= in.StartYear || in.EndYear-in.StartYear > 30 {
		fail(w, 422, fmt.Errorf("Choose a known birth time and a 1-30 year window within 1940-2100."))
		return
	}
	if a.Config.Production {
		if err := a.Store.View(r.Context(), func(s *State) error {
			if !s.Settings.LicenceConfirmed {
				return fmt.Errorf("licence not confirmed")
			}
			return nil
		}); err != nil {
			fail(w, 503, fmt.Errorf("Kundli calculations are unavailable right now."))
			return
		}
	}
	report, err := calculateCore(a.Config, in.Birth, nil, a.Now())
	if err != nil {
		fail(w, 422, err)
		return
	}
	moon := 0
	for _, p := range report.Chart.Planets {
		if p.Name == "Moon" {
			moon = p.RashiNumber
		}
	}
	out, err := swisseph.CalculateSaturnPeriods(r.Context(), a.Config.EpheDir, report.Place.Zone, moon, in.StartYear, in.EndYear)
	if err != nil {
		fail(w, 422, fmt.Errorf("Saturn periods could not be calculated for this window."))
		return
	}
	jsonOut(w, out)
}

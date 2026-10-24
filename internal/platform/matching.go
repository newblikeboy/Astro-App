package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

type MatchingRequest struct {
	Bride Birth `json:"bride"`
	Groom Birth `json:"groom"`
}

type MatchingMarsChecks struct {
	FromLagna    int  `json:"from_lagna"`
	FromMoon     int  `json:"from_moon"`
	LagnaFlagged bool `json:"lagna_flagged"`
	MoonFlagged  bool `json:"moon_flagged"`
}

type MatchingParticipant struct {
	Birth      Birth                   `json:"birth"`
	Place      Place                   `json:"place"`
	Moon       swisseph.ZodiacPosition `json:"moon"`
	Lagna      swisseph.ZodiacPosition `json:"lagna"`
	MarsChecks MatchingMarsChecks      `json:"mars_checks"`
	Chart      *swisseph.Kundali       `json:"chart"`
	Navamsa    *swisseph.Navamsa       `json:"navamsa"`
	Dasha      *swisseph.Vimshottari   `json:"dasha"`
}

type MatchingSummary struct {
	Band        string   `json:"band"`
	ReviewCodes []string `json:"review_codes"`
}

type MatchingResponse struct {
	swisseph.Matching
	CalculatedAt time.Time           `json:"calculated_at"`
	Bride        MatchingParticipant `json:"bride"`
	Groom        MatchingParticipant `json:"groom"`
	Settings     map[string]string   `json:"settings"`
	Mangal       swisseph.MangalPair `json:"mangal"`
	Summary      MatchingSummary     `json:"summary"`
}

// matchingParticipant validates one matching participant and resolves their actual birth instant and native chart.
func matchingParticipant(report *Report) (MatchingParticipant, error) {
	if report == nil || report.Chart == nil {
		return MatchingParticipant{}, fmt.Errorf("natal chart unavailable")
	}
	p := MatchingParticipant{Birth: report.Birth, Place: report.Place, Lagna: report.Chart.Lagna, Chart: report.Chart, Navamsa: report.Navamsa, Dasha: report.Dasha}
	moonFound, marsFound := false, false
	// validPosition checks that a supplied chart position contains a finite normalized longitude.
	validPosition := func(position swisseph.ZodiacPosition) bool {
		return !math.IsNaN(position.Longitude) && !math.IsInf(position.Longitude, 0) &&
			position.Longitude >= 0 && position.Longitude < 360 &&
			position.RashiNumber == int(position.Longitude/30)+1
	}
	for _, planet := range report.Chart.Planets {
		if planet.Name == "Moon" {
			if moonFound || !validPosition(planet.ZodiacPosition) {
				return MatchingParticipant{}, fmt.Errorf("invalid natal Moon")
			}
			p.Moon, moonFound = planet.ZodiacPosition, true
		}
		if planet.Name == "Mars" {
			if marsFound || !validPosition(planet.ZodiacPosition) {
				return MatchingParticipant{}, fmt.Errorf("invalid natal Mars")
			}
			marsFound = true
		}
	}
	if !moonFound || !marsFound || !validPosition(p.Lagna) {
		return MatchingParticipant{}, fmt.Errorf("incomplete natal chart")
	}
	for _, check := range placementChecks(*report.Chart) {
		switch check.Code {
		case "mangal-lagna":
			p.MarsChecks.FromLagna, p.MarsChecks.LagnaFlagged = check.House, check.Matches
		case "mangal-moon":
			p.MarsChecks.FromMoon, p.MarsChecks.MoonFlagged = check.House, check.Matches
		}
	}
	if p.MarsChecks.FromLagna < 1 || p.MarsChecks.FromLagna > 12 || p.MarsChecks.FromMoon < 1 || p.MarsChecks.FromMoon > 12 {
		return MatchingParticipant{}, fmt.Errorf("incomplete Mangal geometry")
	}
	return p, nil
}

// matchingError maps a matching calculation failure to the public HTTP error response.
func matchingError(w http.ResponseWriter, status int, person string, err error) {
	w.WriteHeader(status)
	value := map[string]string{"error": err.Error()}
	if person != "" {
		value["person"] = person
	}
	jsonOut(w, value)
}

// matching computes a stateless two-person result. It does not save profiles,
// reports, analytics or partner details, and does not invoke paid providers.
func (a *App) matching(w http.ResponseWriter, r *http.Request) {
	var in MatchingRequest
	if !decode(w, r, &in) {
		return
	}
	out, failure := a.calculateMatching(r.Context(), in)
	if r.Context().Err() != nil {
		return
	}
	if failure != nil {
		matchingError(w, failure.status, failure.person, fmt.Errorf("%s", failure.message))
		return
	}
	jsonOut(w, out)
}

type matchingFailure struct {
	status          int
	person, message string
}

// Shared by screen and PDF: every result is recomputed from trusted BirthDTOs.
// Client-supplied scores, charts, exception claims or report snapshots are never accepted.
func (a *App) calculateMatching(ctx context.Context, in MatchingRequest) (*MatchingResponse, *matchingFailure) {
	for _, participant := range []struct {
		role  string
		birth Birth
	}{{"bride", in.Bride}, {"groom", in.Groom}} {
		if participant.birth.Unknown || strings.TrimSpace(participant.birth.Time) == "" {
			return nil, &matchingFailure{422, participant.role, "An accurate birth time is required for Kundli matching."}
		}
	}
	if a.Config.Production {
		if err := a.Store.View(ctx, func(s *State) error {
			if !s.Settings.LicenceConfirmed {
				return fmt.Errorf("licence not confirmed")
			}
			return nil
		}); err != nil {
			return nil, &matchingFailure{503, "", "Matching calculations are unavailable right now."}
		}
	}
	now := a.Now()
	reports := make([]*Report, 0, 2)
	for _, participant := range []struct {
		role  string
		birth Birth
	}{{"bride", in.Bride}, {"groom", in.Groom}} {
		if ctx.Err() != nil {
			return nil, &matchingFailure{408, "", "Matching request canceled."}
		}
		report, err := calculateCore(a.Config, participant.birth, nil, now)
		if err != nil {
			return nil, &matchingFailure{422, participant.role, err.Error()}
		}
		reports = append(reports, report)
	}
	if ctx.Err() != nil {
		return nil, &matchingFailure{408, "", "Matching request canceled."}
	}
	bride, err := matchingParticipant(reports[0])
	if err != nil {
		return nil, &matchingFailure{503, "", "Matching calculations are unavailable right now. Please try again."}
	}
	groom, err := matchingParticipant(reports[1])
	if err != nil {
		return nil, &matchingFailure{503, "", "Matching calculations are unavailable right now. Please try again."}
	}
	score, err := swisseph.CalculateMatching(*reports[0].Chart, *reports[1].Chart)
	if err != nil {
		return nil, &matchingFailure{503, "", "Matching calculations are unavailable right now. Please try again."}
	}
	mangal, err := swisseph.CalculateMangalPair(*bride.Chart, *groom.Chart)
	if err != nil {
		return nil, &matchingFailure{503, "", "Matching calculations are unavailable right now. Please try again."}
	}
	band := "32_to_36"
	if score.Total < 18 {
		band = "below_18"
	} else if score.Total < 24 {
		band = "18_to_24"
	} else if score.Total < 32 {
		band = "24_to_32"
	}
	review := append([]string{}, score.Cautions...)
	if mangal.Status == "different_active_levels" || mangal.Status == "pair_exception_found" {
		review = append(review, "mangal_pair_review")
	}
	return &MatchingResponse{
		Matching: score, CalculatedAt: now.UTC(), Bride: bride, Groom: groom,
		Mangal: mangal, Summary: MatchingSummary{Band: band, ReviewCodes: review},
		Settings: map[string]string{
			"calculation_engine": reports[0].Engine,
			"swiss_version":      reports[0].Chart.SwissVersion,
			"zodiac":             "Lahiri sidereal; geocentric apparent",
			"houses":             "Whole sign; Mangal from Lagna, Moon and Venus",
			"bride_timezone":     bride.Place.Zone,
			"groom_timezone":     groom.Place.Zone,
			"score_basis":        "AIFAS 2021 North Indian base Ashtakoota; no cancellation bonuses",
			"mangal_method":      swisseph.MangalMethod,
			"mangal_level":       "Reference coverage: 0 none, 1 low, 2 or 3 high; raw and selected exceptions shown separately",
			"clock":              "UTC via Swiss; before 1972 Swiss uses its documented UT1 convention",
		},
	}, nil
}

type MatchingProfile struct {
	ID      string    `json:"id"`
	Birth   Birth     `json:"birth"`
	Place   Place     `json:"place"`
	Created time.Time `json:"created"`
}

// Explicit, authenticated opt-in only. Returns a customer's eligible saved
// birth inputs, never partner records, orders, phones or stored report readings.
func (a *App) matchingProfiles(w http.ResponseWriter, r *http.Request) {
	sess, ok := a.guard(w, r, false)
	if !ok {
		return
	}
	profiles := []MatchingProfile{}
	err := a.Store.View(r.Context(), func(s *State) error {
		for _, report := range s.Reports {
			if report.UserID != sess.UserID || report.Chart == nil || report.Birth.Unknown || report.Birth.Date < "1940-01-01" || report.Birth.Time == "" {
				continue
			}
			place, exists := findPlace(report.Birth.PlaceID)
			if !exists {
				continue
			}
			profiles = append(profiles, MatchingProfile{report.ID, report.Birth, place, report.Created})
		}
		return nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Saved Kundlis are unavailable right now."))
		return
	}
	sort.Slice(profiles, func(i, j int) bool {
		if profiles[i].Created.Equal(profiles[j].Created) {
			return profiles[i].ID < profiles[j].ID
		}
		return profiles[i].Created.After(profiles[j].Created)
	})
	if len(profiles) > 50 {
		profiles = profiles[:50]
	}
	jsonOut(w, map[string]any{"profiles": profiles})
}

package platform

import (
	"astroapp/backend/internal/swisseph"
	"context"
	"fmt"
	"net/http"
	"time"
)

type calendarDay struct {
	Date      string                    `json:"date"`
	Sunrise   time.Time                 `json:"sunrise"`
	Sunset    time.Time                 `json:"sunset"`
	Tithi     []swisseph.PanchangPeriod `json:"tithi"`
	Nakshatra []swisseph.PanchangPeriod `json:"nakshatra"`
	Calendar  swisseph.PanchangCalendar `json:"calendar"`
	Moon      swisseph.ZodiacPosition   `json:"moon"`
}

// panchangCalendar validates a weekly/monthly request and assembles native daily summaries under cache limits.
func (a *App) panchangCalendar(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	for key, values := range q {
		if (key != "date" && key != "place_id" && key != "sunrise" && key != "view") || len(values) != 1 {
			fail(w, 400, fmt.Errorf("invalid calendar query"))
			return
		}
	}
	place, ok := findPlace(q.Get("place_id"))
	if !ok || place.Zone != "Asia/Kolkata" {
		fail(w, 400, fmt.Errorf("choose a location from the suggestions"))
		return
	}
	date := q.Get("date")
	at, err := time.Parse("2006-01-02", date)
	if err != nil || at.Format("2006-01-02") != date || date < swisseph.MinDate || date > swisseph.MaxDate {
		fail(w, 400, fmt.Errorf("unsupported calendar date"))
		return
	}
	view := q.Get("view")
	if view != "week" && view != "month" {
		fail(w, 400, fmt.Errorf("choose week or month"))
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
	start, end := at.AddDate(0, 0, -int(at.Weekday())), at.AddDate(0, 0, 7-int(at.Weekday()))
	if view == "month" {
		start = time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC)
		end = start.AddDate(0, 1, 0)
	}
	// Clip partial edge weeks to the shared supported civil-date interval.
	// The month containing MinDate still includes its first supported day.
	minimum, _ := time.Parse("2006-01-02", swisseph.MinDate)
	maximum, _ := time.Parse("2006-01-02", swisseph.MaxDate)
	maximumExclusive := maximum.AddDate(0, 0, 1)
	if start.Before(minimum) {
		start = minimum
	}
	if end.After(maximumExclusive) {
		end = maximumExclusive
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	key := "panchang-calendar/" + place.ID + "/" + start.Format("2006-01-02") + "/" + view + "/" + convention
	body, err := a.cachedSky(ctx, key, func() (any, error) {
		days := []calendarDay{}
		// Serial bounded work: one request never fans out 31 competing Swiss calls.
		for day := start; day.Before(end); day = day.AddDate(0, 0, 1) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			p, e := swisseph.CalculatePanchang(ctx, a.Config.EpheDir, day.Format("2006-01-02"), place.Zone, convention, place.Lat, place.Lon)
			if e != nil {
				return nil, e
			}
			days = append(days, calendarDay{p.Date, p.Sunrise, p.Sunset, p.Tithi, p.Nakshatra, p.Details.Calendar, p.Details.Moon})
		}
		return struct {
			Place Place         `json:"place"`
			View  string        `json:"view"`
			Days  []calendarDay `json:"days"`
		}{place, view, days}, nil
	})
	if err != nil {
		fail(w, 503, fmt.Errorf("Panchang calendar unavailable; please retry"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_, _ = w.Write(body)
}

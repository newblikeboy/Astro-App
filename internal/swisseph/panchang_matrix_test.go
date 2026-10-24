package swisseph

import (
	"context"
	"testing"
	"time"
)

// TestCriticalAuditPanchangMatrix checks Panchang invariants across a matrix of dates and locations.
func TestCriticalAuditPanchangMatrix(t *testing.T) {
	locations := []struct {
		name     string
		lat, lon float64
	}{
		{"Delhi", 28.6139, 77.209}, {"Srinagar", 34.0837, 74.7973},
		{"Itanagar", 27.1, 93.62}, {"Chennai", 13.0827, 80.2707},
		{"Mumbai", 19.076, 72.8777}, {"PortBlair", 11.67, 92.75},
		{"Kavaratti", 10.57, 72.64}, {"Nagpur", 21.1466, 79.0889},
	}
	dates := []string{"1972-01-02", "2000-02-29", "2023-07-18", "2024-01-01", "2024-04-08", "2024-04-09", "2026-03-19", "2026-06-21", "2026-10-18", "2026-12-21", "2038-01-19", "2099-12-30"}
	for _, location := range locations {
		for _, date := range dates {
			for _, method := range []string{"upper-limb", "hindu"} {
				t.Run(location.name+"/"+date+"/"+method, func(t *testing.T) {
					p, err := CalculatePanchang(context.Background(), "../../ephe", date, "Asia/Kolkata", method, location.lat, location.lon)
					if err != nil {
						t.Fatal(err)
					}
					if !p.Sunset.After(p.Sunrise) || !p.NextSunrise.After(p.Sunset) || p.NextSunrise.Sub(p.Sunrise) < 23*time.Hour || p.NextSunrise.Sub(p.Sunrise) > 25*time.Hour {
						t.Fatal("invalid sunrise day")
					}
					for kind, rows := range [][]PanchangPeriod{p.Tithi, p.Nakshatra, p.Yoga, p.Karana} {
						count := []int{30, 27, 27, 60}[kind]
						if len(rows) == 0 || !rows[0].Start.Equal(p.Sunrise) || rows[len(rows)-1].End.Before(p.NextSunrise) {
							t.Fatalf("incomplete limb %d", kind)
						}
						for i, r := range rows {
							if r.Number < 1 || r.Number > count || r.Name == "" || !r.End.After(r.Start) || (i > 0 && (!r.Start.Equal(rows[i-1].End) || r.Number != rows[i-1].Number%count+1)) {
								t.Fatalf("invalid limb %d row%d", kind, i)
							}
						}
					}
					for index, rows := range [][]PanchangPeriod{p.ChoghadiyaDay, p.ChoghadiyaNight, p.HoraDay, p.HoraNight} {
						start, end, want := p.Sunrise, p.Sunset, 8
						if index%2 == 1 {
							start, end = p.Sunset, p.NextSunrise
						}
						if index >= 2 {
							want = 12
						}
						if len(rows) != want || !rows[0].Start.Equal(start) || !rows[len(rows)-1].End.Equal(end) {
							t.Fatal("period partition incomplete")
						}
						for i, r := range rows {
							if !r.End.After(r.Start) || (i > 0 && !r.Start.Equal(rows[i-1].End)) {
								t.Fatal("period overlap/gap")
							}
						}
					}
					for _, event := range []*time.Time{p.Details.Moonrise, p.Details.Moonset} {
						if event != nil && (event.Before(p.Sunrise) || !event.Before(p.NextSunrise)) {
							t.Fatal("Moon event belongs to a different sunrise day")
						}
					}
					for _, z := range []ZodiacPosition{p.Details.Sun, p.Details.Moon} {
						if !finite(z.Longitude) || z.Longitude < 0 || z.Longitude >= 360 || z.RashiNumber < 1 || z.RashiNumber > 12 || z.NakshatraNumber < 1 || z.NakshatraNumber > 27 || z.Pada < 1 || z.Pada > 4 {
							t.Fatal("invalid zodiac classification")
						}
					}
				})
			}
		}
	}
}

package swisseph

import (
	"context"
	"encoding/json"
	"math"
	"os"
	"testing"
	"time"
)

// TestPanchangIndependentJPLPhases compares lunar phase boundaries with independent NASA/JPL reference instants.
func TestPanchangIndependentJPLPhases(t *testing.T) {
	var fixture struct {
		Tolerance float64 `json:"tolerance_seconds"`
		Cases     []struct {
			Date                string
			Latitude, Longitude float64
			Tithi, Karana       []struct {
				Number int
				End    time.Time
			}
		}
	}
	raw, err := os.ReadFile("testdata/jpl-panchang-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Cases) < 7 || fixture.Tolerance <= 0 || fixture.Tolerance > 2 {
		t.Fatal("missing independent JPL audit cases or invalid tolerance")
	}
	for _, c := range fixture.Cases {
		t.Run(c.Date, func(t *testing.T) {
			p, err := CalculatePanchang(context.Background(), "../../ephe", c.Date, "Asia/Kolkata", "upper-limb", c.Latitude, c.Longitude)
			if err != nil {
				t.Fatal(err)
			}
			for kind, rows := range [][]PanchangPeriod{p.Tithi, p.Karana} {
				reference := c.Tithi
				if kind == 1 {
					reference = c.Karana
				}
				for _, got := range rows {
					found := false
					for _, want := range reference {
						if want.Number == got.Number && math.Abs(got.End.Sub(want.End).Seconds()) <= fixture.Tolerance {
							found = true
							t.Logf("limb=%d number=%d JPL residual=%.3fs", kind, got.Number, got.End.Sub(want.End).Seconds())
							break
						}
					}
					if !found {
						t.Errorf("limb=%d number=%d end=%s outside independent JPL tolerance", kind, got.Number, got.End)
					}
				}
			}
		})
	}
}

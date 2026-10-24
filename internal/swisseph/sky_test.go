package swisseph

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"testing"
	"time"
)

// TestSkyPlanetOfficialPositions compares planet positions with official Swiss reference coordinates.
func TestSkyPlanetOfficialPositions(t *testing.T) {
	b, err := os.ReadFile("testdata/swetest-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var refs []struct {
		UTC     string
		Planets map[string]float64
	}
	if err = json.Unmarshal(b, &refs); err != nil {
		t.Fatal(err)
	}
	for _, r := range refs {
		at, _ := time.Parse(time.RFC3339, r.UTC)
		p, err := CalculatePlanetPositions(context.Background(), "../../ephe", at)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Planets) != 9 {
			t.Fatal("missing planets")
		}
		for _, v := range p.Planets {
			if want, ok := r.Planets[v.Name]; ok && math.Abs(v.Longitude-want) > 1e-6 {
				t.Errorf("%s %s got %.9f want %.9f", r.UTC, v.Name, v.Longitude, want)
			}
		}
		if math.Abs(normalizeDegrees(p.Planets[8].Longitude-p.Planets[7].Longitude)-180) > 1e-9 {
			t.Fatal("nodes not opposite")
		}
	}
}

// TestSkyAnnualTransitions checks annual ingress and station results and their ordering.
func TestSkyAnnualTransitions(t *testing.T) {
	p, err := CalculatePlanetTransitions(context.Background(), "../../ephe", 2026)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	stations := 0
	retroIngress := 0
	err = withSky(context.Background(), "../../ephe", "Asia/Kolkata", func(s *skySession) error {
		for i, e := range p.Events {
			if e.At.Year() != 2026 || i > 0 && e.At.Before(p.Events[i-1].At) {
				t.Fatal("event outside year or out of order")
			}
			id := -1
			for k, name := range skyNames {
				if e.Planet == name {
					id = k
				}
			}
			if id < 0 {
				t.Fatal("unknown planet")
			}
			jd, _ := s.jd(e.At)
			before, err := s.position(jd-2.0/86400, id)
			if err != nil {
				return err
			}
			after, err := s.position(jd+2.0/86400, id)
			if err != nil {
				return err
			}
			if e.Kind == "Sign ingress" {
				counts[e.Planet]++
				if before.Rashi == after.Rashi || before.Rashi != e.FromSign || after.Rashi != e.ToSign {
					t.Errorf("bad ingress %+v before=%s after=%s", e, before.Rashi, after.Rashi)
				}
				if e.Retrograde {
					retroIngress++
				}
			} else {
				stations++
				if before.Retrograde == after.Retrograde || after.Retrograde != e.Retrograde {
					t.Errorf("bad station %+v before=%g after=%g", e, before.Speed, after.Speed)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if counts["Sun"] != 12 || counts["Moon"] < 150 || stations < 10 || retroIngress == 0 {
		t.Fatalf("missing annual events: counts=%v stations=%d retro ingress=%d", counts, stations, retroIngress)
	}
	// Solar ingresses progress one sign; retrograde re-entries move backwards.
	for _, e := range p.Events {
		if e.Kind != "Sign ingress" {
			continue
		}
		from, to := -1, -1
		for i, name := range rashiNames {
			if name == e.FromSign {
				from = i
			}
			if name == e.ToSign {
				to = i
			}
		}
		delta := (to - from + 12) % 12
		want := 1
		if e.Retrograde {
			want = 11
		}
		if delta != want {
			t.Fatalf("invalid adjacent crossing %+v", e)
		}
	}
}

// TestSkyEclipseCLIReferences compares native eclipse results with official Swiss command-line fixtures.
func TestSkyEclipseCLIReferences(t *testing.T) {
	var refs struct {
		Global []struct {
			Kind, Type string
			Maximum    time.Time
		}
		Local []struct {
			Latitude, Longitude  float64
			Type                 string
			Maximum, First, Last time.Time
		}
	}
	b, err := os.ReadFile("testdata/sky-reference.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(b, &refs); err != nil {
		t.Fatal(err)
	}
	p, err := CalculateEclipses(context.Background(), "../../ephe", 2026, 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Events) != len(refs.Global) {
		t.Fatalf("got %d events want %d", len(p.Events), len(refs.Global))
	}
	for i, e := range p.Events {
		want := refs.Global[i]
		if e.Kind != want.Kind || e.Type != want.Type || math.Abs(e.Maximum.Sub(want.Maximum).Seconds()) > 1 {
			t.Errorf("global eclipse got %+v want %+v", e, want)
		}
		if e.Kind == "Solar" && e.Visible {
			t.Fatal("2026 solar eclipse incorrectly visible in Delhi")
		}
		if e.Maximum.Month() == time.March {
			if !e.Visible || e.Local == nil || e.Local.Type != "Partial" {
				t.Fatalf("must distinguish global total from locally partial at moonrise: %+v", e)
			}
		}
		if e.Local != nil {
			if !e.Local.VisibleEnd.After(e.Local.VisibleStart) {
				t.Fatal("invalid visibility duration")
			}
			for _, c := range e.Local.Contacts {
				if c.At.IsZero() {
					t.Fatal("zero contact emitted")
				}
			}
		}
	}
	for _, r := range refs.Local {
		p, err := CalculateEclipses(context.Background(), "../../ephe", 2020, r.Latitude, r.Longitude)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, e := range p.Events {
			if e.Kind != "Solar" || e.Maximum.Month() != time.June {
				continue
			}
			found = true
			if !e.Visible || e.Local == nil || e.Type != "Annular" || e.Local.Type != r.Type {
				t.Fatalf("wrong local eclipse: %+v", e)
			}
			if math.Abs(e.Local.Maximum.Sub(r.Maximum).Seconds()) > 1 || math.Abs(e.Local.VisibleStart.Sub(r.First).Seconds()) > 1 || math.Abs(e.Local.VisibleEnd.Sub(r.Last).Seconds()) > 1 {
				t.Errorf("local times disagree with CLI %+v", e.Local)
			}
		}
		if !found {
			t.Fatal("missing June 2020 eclipse")
		}
	}
}

// TestSkyPenumbralAndLimits checks penumbral eclipse handling and supported search limits.
func TestSkyPenumbralAndLimits(t *testing.T) {
	p, err := CalculateEclipses(context.Background(), "../../ephe", 2027, 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range p.Events {
		if e.Kind == "Lunar" {
			n++
			if e.Type != "Penumbral" {
				t.Fatal("wrong 2027 lunar eclipse type")
			}
		}
	}
	if n != 3 {
		t.Fatalf("missing penumbral eclipse: %d", n)
	}
	for _, year := range []int{1972, 2099} {
		p, err := CalculateEclipses(context.Background(), "../../ephe", year, 28.6139, 77.209)
		if err != nil {
			t.Fatal(err)
		}
		if len(p.Events) < 2 {
			t.Fatal("incomplete edge year")
		}
		for _, e := range p.Events {
			if e.Maximum.Year() != year {
				t.Fatal("edge year overflow")
			}
		}
	}
	for _, year := range []int{1939, 2100} {
		if _, err := CalculatePlanetTransitions(context.Background(), "../../ephe", year); err == nil {
			t.Fatal("invalid year accepted")
		}
		if _, err := CalculateEclipses(context.Background(), "../../ephe", year, 28, 77); err == nil {
			t.Fatal("invalid eclipse year accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := CalculatePlanetTransitions(ctx, "../../ephe", 2026); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, err := CalculateEclipses(context.Background(), t.TempDir(), 2026, 28, 77); err == nil {
		t.Fatal("allowed fallback with missing data")
	}
}

// TestSkyTotalSolarObscurationAndReanchoredMaximum checks total-solar obscuration and local maximum re-anchoring.
func TestSkyTotalSolarObscurationAndReanchoredMaximum(t *testing.T) {
	p, err := CalculateEclipses(context.Background(), "../../ephe", 2024, 32.7767, -96.7970)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range p.Events {
		if e.Kind != "Solar" || e.Maximum.Month() != time.April {
			continue
		}
		found = true
		if !e.Visible || e.Local == nil || e.Local.Type != "Total" || e.Local.Obscuration == nil {
			t.Fatalf("Dallas must see totality: %+v", e)
		}
		if *e.Local.Obscuration != 1 {
			t.Errorf("the physical fraction of the solar disc hidden during totality must be 1, got %g", *e.Local.Obscuration)
		}
		// Official swetest 2.10.03, -b8.4.2024 -ut0 -solecl -local
		// -geopos-96.797,32.7767,0. CLI emits UT1, within 1 second of UTC.
		for _, check := range []struct {
			got  time.Time
			want string
		}{
			{e.Local.Maximum, "2024-04-08T18:42:39Z"},
			{e.Local.VisibleStart, "2024-04-08T17:23:18.3Z"},
			{e.Local.VisibleEnd, "2024-04-08T20:02:41.7Z"},
		} {
			want, _ := time.Parse(time.RFC3339Nano, check.want)
			if math.Abs(check.got.Sub(want).Seconds()) > 1 {
				t.Errorf("Dallas local eclipse time got %s want %s", check.got, want)
			}
		}
	}
	if !found {
		t.Fatal("missing April 2024 solar eclipse")
	}
	// In Delhi the March 2026 lunar eclipse is already past its geometric
	// maximum when the Moon rises. Swiss moves its returned maximum to the
	// observable moonrise, but its MAX_VISIBLE flag refers to the original
	// (hidden) maximum. Metadata must describe the returned time consistently.
	p, err = CalculateEclipses(context.Background(), "../../ephe", 2026, 28.6139, 77.209)
	if err != nil {
		t.Fatal(err)
	}
	found = false
	for _, e := range p.Events {
		if e.Kind != "Lunar" || e.Maximum.Month() != time.March {
			continue
		}
		found = true
		if e.Local == nil || !e.Local.Maximum.Equal(e.Local.VisibleStart) {
			t.Fatalf("expected a moonrise-limited maximum: %+v", e)
		}
		if !e.Local.MaximumAboveHorizon {
			t.Error("returned visible maximum at moonrise must be marked above the apparent horizon")
		}
	}
	if !found {
		t.Fatal("missing March 2026 lunar eclipse")
	}
}

// TestSkyNASAIndependentEclipseCatalogs compares eclipse classification and timing with independent NASA catalogues.
func TestSkyNASAIndependentEclipseCatalogs(t *testing.T) {
	// NASA/GSFC Five Millennium Catalogs, retrieved 19 October 2026:
	// https://eclipse.gsfc.nasa.gov/SEcat5/SE2001-2100.html
	// https://eclipse.gsfc.nasa.gov/LEcat5/LE2001-2100.html
	// Catalog maximums are TD, not UTC. Subtract each catalog's published
	// Delta-T to obtain UT1. The 30-second allowance includes the catalog's
	// older Delta-T predictions, rounding and different ephemeris models;
	// it is a cross-engine reference, not a one-second accuracy claim.
	refs := []struct {
		kind, typ, td string
		deltaT        int
	}{
		{"Solar", "Hybrid", "2023-04-20T04:17:56Z", 73},
		{"Solar", "Annular", "2023-10-14T18:00:41Z", 74},
		{"Lunar", "Penumbral", "2023-05-05T17:24:05Z", 73},
		{"Lunar", "Partial", "2023-10-28T20:15:18Z", 74},
		{"Solar", "Total", "2024-04-08T18:18:29Z", 74},
		{"Solar", "Annular", "2024-10-02T18:46:13Z", 74},
		{"Lunar", "Penumbral", "2024-03-25T07:13:59Z", 74},
		{"Lunar", "Partial", "2024-09-18T02:45:25Z", 74},
		{"Solar", "Partial", "2025-03-29T10:48:36Z", 75},
		{"Solar", "Partial", "2025-09-21T19:43:04Z", 75},
		{"Lunar", "Total", "2025-03-14T06:59:56Z", 75},
		{"Lunar", "Total", "2025-09-07T18:12:58Z", 75},
		{"Solar", "Annular", "2026-02-17T12:13:06Z", 75},
		{"Solar", "Total", "2026-08-12T17:47:06Z", 75},
		{"Lunar", "Total", "2026-03-03T11:34:52Z", 75},
		{"Lunar", "Partial", "2026-08-28T04:14:04Z", 75},
		{"Solar", "Annular", "2027-02-06T16:00:48Z", 76},
		{"Solar", "Total", "2027-08-02T10:07:50Z", 76},
		{"Lunar", "Penumbral", "2027-02-20T23:14:06Z", 76},
		{"Lunar", "Penumbral", "2027-07-18T16:04:09Z", 76},
		{"Lunar", "Penumbral", "2027-08-17T07:14:59Z", 76},
	}
	for year := 2023; year <= 2027; year++ {
		p, err := CalculateEclipses(context.Background(), "../../ephe", year, 28.6139, 77.209)
		if err != nil {
			t.Fatal(err)
		}
		matched := 0
		for _, ref := range refs {
			td, err := time.Parse(time.RFC3339, ref.td)
			if err != nil {
				t.Fatal(err)
			}
			if td.Year() != year {
				continue
			}
			want := td.Add(-time.Duration(ref.deltaT) * time.Second)
			found := false
			for _, e := range p.Events {
				if e.Kind != ref.kind || e.Maximum.UTC().Format("2006-01-02") != want.Format("2006-01-02") {
					continue
				}
				found = true
				matched++
				if e.Type != ref.typ || math.Abs(e.Maximum.Sub(want).Seconds()) > 30 {
					t.Errorf("NASA reference %s %s: got %s %s, want %s within 30 seconds", ref.kind, ref.td, e.Type, e.Maximum, ref.typ)
				}
			}
			if !found {
				t.Errorf("NASA eclipse missing: %s %s", ref.kind, ref.td)
			}
		}
		if len(p.Events) != matched {
			t.Errorf("%d: %d events, %d NASA references", year, len(p.Events), matched)
		}
	}
}

// TestSkyPositionsAndSpeedsAtRangeEdges checks planet positions and speeds near the supported date endpoints.
func TestSkyPositionsAndSpeedsAtRangeEdges(t *testing.T) {
	// Official swetest 2.10.03 with -utc, -sid1, -p0123456m, -fPls.
	// Longitude and speed output is rounded to seven decimal places.
	for _, ref := range []struct {
		utc   string
		lon   [8]float64
		speed [8]float64
	}{
		{"1972-01-02T00:00:00Z", [8]float64{257.2117214, 91.8433426, 340.5569642, 234.6576455, 239.1013160, 288.5546191, 36.9016061, 283.1103761}, [8]float64{1.0189516, 13.5127790, 0.6469296, 1.0299109, 0.2212438, 1.2334879, -0.0519570, -0.0529920}},
		{"2099-12-30T18:29:00Z", [8]float64{254.0962742, 114.9775496, 3.7742427, 260.7578631, 175.8185520, 293.3355251, 180.3040272, 325.7476369}, [8]float64{1.0187203, 13.9219304, 0.3999497, 1.6183345, 0.1092982, 1.2041679, 0.0598157, -0.0529919}},
	} {
		at, _ := time.Parse(time.RFC3339, ref.utc)
		p, err := CalculatePlanetPositions(context.Background(), "../../ephe", at)
		if err != nil {
			t.Fatal(err)
		}
		for i := range ref.lon {
			if math.Abs(p.Planets[i].Longitude-ref.lon[i]) > 1e-6 || math.Abs(p.Planets[i].Speed-ref.speed[i]) > 1e-6 {
				t.Errorf("%s %s differs from official CLI: %+v", ref.utc, skyNames[i], p.Planets[i])
			}
			if p.Planets[i].Retrograde != (ref.speed[i] < 0) {
				t.Errorf("%s %s incorrect retrograde state", ref.utc, skyNames[i])
			}
		}
	}
}

// TestSkyCloseIngressPairAcrossStation checks that both closely spaced ingresses around a station are retained.
func TestSkyCloseIngressPairAcrossStation(t *testing.T) {
	// Synthetic parabola with a station at zero and two genuine crossings
	// 0.5 seconds apart. A +/-1 second validation probe sees Mesh at both
	// ends and discards both roots. Search each monotonic side independently.
	get := func(jd float64) (SkyPlanet, error) {
		seconds := jd * 86400
		lon := 30 + 0.001 - 0.016*seconds*seconds
		return SkyPlanet{ZodiacPosition: zodiacPosition(lon), Speed: -0.032 * seconds * 86400, Retrograde: seconds > 0}, nil
	}
	for _, check := range []struct {
		a, b, want float64
		from, to   string
	}{
		{-5.0 / 86400, 0, -0.25 / 86400, "Mesh", "Vrishabh"},
		{0, 5.0 / 86400, 0.25 / 86400, "Vrishabh", "Mesh"},
	} {
		root, err := refineSignIngress(get, check.a, check.b)
		if err != nil || root == nil {
			t.Fatalf("valid near-station ingress discarded: %v", err)
		}
		if math.Abs(root.at-check.want)*86400 > 0.05 || root.from.Rashi != check.from || root.to.Rashi != check.to {
			t.Errorf("incorrect near-station root: %+v", root)
		}
	}
	if root, err := refineSignIngress(get, -5.0/86400, -1.0/86400); err != nil || root != nil {
		t.Errorf("invented root without a crossing: %+v %v", root, err)
	}
	errUnavailable := errors.New("position unavailable")
	if _, err := refineSignIngress(func(float64) (SkyPlanet, error) { return SkyPlanet{}, errUnavailable }, 0, 1); !errors.Is(err, errUnavailable) {
		t.Errorf("position failure hidden: %v", err)
	}
}

// TestSkyTransitionCivilYearEdgesAndTimeScales checks annual transit boundaries in local civil time and native time-scale conversion.
func TestSkyTransitionCivilYearEdgesAndTimeScales(t *testing.T) {
	for _, year := range []int{1972, 2000, 2099} {
		p, err := CalculatePlanetTransitions(context.Background(), "../../ephe", year)
		if err != nil {
			t.Fatal(err)
		}
		solar := 0
		err = withSky(context.Background(), "../../ephe", "Asia/Kolkata", func(s *skySession) error {
			for i, e := range p.Events {
				if e.At.Year() != year || i > 0 && e.At.Before(p.Events[i-1].At) {
					t.Errorf("civil-year membership/order failed: %+v", e)
				}
				index := -1
				for id, name := range skyNames {
					if name == e.Planet {
						index = id
					}
				}
				if index < 0 {
					t.Fatalf("unknown planet: %s", e.Planet)
				}
				jd, err := s.jd(e.At)
				if err != nil {
					return err
				}
				if math.Abs(s.instant(jd).Sub(e.At).Seconds()) > 0.001 {
					t.Errorf("UTC/UT1 round trip exceeds 1 millisecond: %s", e.At)
				}
				if e.Kind == "Sign ingress" {
					if e.Planet == "Sun" {
						solar++
					}
					at, err := s.position(jd, index)
					if err != nil {
						return err
					}
					boundaryDistance := math.Min(math.Mod(at.Longitude, 30), 30-math.Mod(at.Longitude, 30))
					if boundaryDistance > 0.00001 {
						t.Errorf("sign root not at a 30-degree boundary: %+v lon=%g", e, at.Longitude)
					}
				} else {
					if e.Planet == "Sun" || e.Planet == "Moon" || e.Planet == "Rahu" || e.Planet == "Ketu" {
						t.Errorf("unexpected station for %s", e.Planet)
					}
					at, err := s.position(jd, index)
					if err != nil {
						return err
					}
					if math.Abs(at.Speed) > 0.000001 {
						t.Errorf("station speed is not zero within root tolerance: %+v speed=%g", e, at.Speed)
					}
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if solar != 12 {
			t.Errorf("%d: expected 12 solar sign ingresses, got %d", year, solar)
		}
	}
	for _, coords := range [][2]float64{{math.NaN(), 77}, {28, math.Inf(1)}, {67, 77}, {28, 181}} {
		if _, err := CalculateEclipses(context.Background(), "../../ephe", 2026, coords[0], coords[1]); err == nil {
			t.Errorf("invalid coordinates accepted: %v", coords)
		}
	}
}

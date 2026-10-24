package swisseph

import (
	"math"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestZodiacBoundaries checks sign classification at zodiac boundaries.
func TestZodiacBoundaries(t *testing.T) {
	for _, tc := range []struct {
		lon              float64
		rashi, nak, pada int
	}{{0, 1, 1, 1}, {30, 2, 3, 2}, {360, 1, 1, 1}, {-1, 12, 27, 4}} {
		p := zodiacPosition(tc.lon)
		if p.RashiNumber != tc.rashi || p.NakshatraNumber != tc.nak || p.Pada != tc.pada {
			t.Fatalf("longitude %v: unexpected classification %+v", tc.lon, p)
		}
	}
}

// TestNormalizedZodiacHalfOpenRangeAndDisplay checks normalized zodiac coordinates and display at half-open sign boundaries.
func TestNormalizedZodiacHalfOpenRangeAndDisplay(t *testing.T) {
	// The negative sub-ULP case previously rounded to 360 and indexed sign 13.
	for _, lon := range []float64{-math.SmallestNonzeroFloat64, -1e-15, math.Nextafter(360, 0), -360, 360, 720, -720} {
		got := normalizeDegrees(lon)
		if got < 0 || got >= 360 || !finite(got) {
			t.Fatalf("normalization %.20g produced %.20g", lon, got)
		}
		p := zodiacPosition(lon)
		if p.RashiNumber < 1 || p.RashiNumber > 12 || p.NakshatraNumber < 1 || p.NakshatraNumber > 27 || p.Pada < 1 || p.Pada > 4 {
			t.Fatalf("invalid boundary classification: %+v", p)
		}
		if lon < 0 && lon > -1 && p.RashiNumber != 12 {
			t.Fatalf("negative angle lost its side of the zodiac boundary: %+v", p)
		}
	}
	for sign := 0; sign < 12; sign++ {
		// Truncation must never display 30 degrees in the preceding sign.
		p := zodiacPosition(float64(sign*30) + 29 + 59.0/60 + 59.9/3600)
		if p.RashiNumber != sign+1 || p.DMS != "29°59′59″" {
			t.Fatalf("sign/DMS carry: %+v", p)
		}
	}
}

// TestKundaliSupportedUTCBoundariesAndFrames checks supported UTC endpoints and the declared chart reference frames.
func TestKundaliSupportedUTCBoundariesAndFrames(t *testing.T) {
	for _, tc := range []struct {
		utc string
		ok  bool
	}{
		{"1939-12-30T23:59:59Z", false}, {"1939-12-31T00:00:00Z", true},
		{"1940-01-01T00:00:00Z", true}, {"1972-01-01T00:00:00Z", true},
		{"2099-12-31T23:59:59Z", true}, {"2100-01-01T00:00:00Z", false},
	} {
		at, err := time.Parse(time.RFC3339, tc.utc)
		if err != nil {
			t.Fatal(err)
		}
		chart, err := CalculateKundali("../../ephe", at, 28.6139, 77.209)
		if (err == nil) != tc.ok {
			t.Fatalf("supported UTC boundary %s: %v", tc.utc, err)
		}
		if !tc.ok {
			continue
		}
		for _, house := range chart.Houses {
			want := (chart.Lagna.RashiNumber+house.Number-2)%12 + 1
			if house.RashiNumber != want || house.CuspLongitude != float64((want-1)*30) {
				t.Fatalf("whole-sign house %s: %+v", tc.utc, house)
			}
		}
		for _, planet := range chart.Planets {
			want := (planet.RashiNumber-chart.Lagna.RashiNumber+12)%12 + 1
			if planet.House != want || planet.Retrograde != (planet.Speed < 0) {
				t.Fatalf("placement/speed %s: %+v", tc.utc, planet)
			}
		}
		if chart.Planets[7].Speed >= 0 || chart.Planets[8].Speed != chart.Planets[7].Speed || math.Abs(normalizeDegrees(chart.Planets[8].Longitude-chart.Planets[7].Longitude)-180) > 1e-10 {
			t.Fatalf("mean/opposite node contract %s", tc.utc)
		}
	}
}

// TestCriticalAuditAdditionalOfficialSwetestVectors compares additional native chart cases with official Swiss reference vectors.
func TestCriticalAuditAdditionalOfficialSwetestVectors(t *testing.T) {
	// Captured 2026-10-19 from the official Astrodienst swetest CLI, not
	// CalculateKundali. Example: swetest -b1.1.1972 -utc0:00 -p0123456m
	// -sid1 -eswe -edirephe -fPls -g, -head -house77.209,28.6139,W
	// Planet order below follows this API (Sun/Moon/Mars/Mercury/Jupiter/
	// Venus/Saturn/mean Rahu), rather than the CLI's numeric body order.
	for _, tc := range []struct {
		utc       string
		lat, lon  float64
		lagna     float64
		positions [8]float64
		speeds    [8]float64
	}{
		{"1972-01-01T00:00:00Z", 28.6139, 77.209, 231.6344621,
			[8]float64{256.1927890, 78.1603874, 339.9103914, 233.6556906, 238.8798351, 287.3208516, 36.9543041, 283.1633681},
			[8]float64{1.0189147, 13.8466453, .6462127, .9728592, .2217147, 1.2340483, -.0534347, -.0529920}},
		{"2099-12-31T23:59:59Z", -33.8688, 151.2093, 319.5247069,
			[8]float64{255.3492321, 132.1601052, 4.2702359, 262.7509722, 175.9511250, 294.8155736, 180.3764533, 325.6824648},
			[8]float64{1.0188607, 14.0118841, .4066100, 1.6228407, .1062862, 1.2027054, .0579607, -.0529918}},
		{"2000-02-29T12:00:00Z", 40.7128, -74.006, 326.7973800,
			[8]float64{316.3503264, 251.6807092, 349.3660490, 318.6509916, 8.7215912, 290.1010548, 18.5102694, 98.0608947},
			[8]float64{1.0040706, 11.8207825, .7518336, -1.0066134, .1891531, 1.2354889, .0814601, -.0529920}},
		{"2026-10-19T00:00:00Z", 27.7172, 85.324, 175.9495317,
			[8]float64{181.3912261, 274.9373336, 107.5886640, 205.1187751, 118.3254143, 189.5692962, 345.9505219, 302.5282389},
			[8]float64{.9929378, 11.9413139, .5411009, .5705708, .1489987, -.5539556, -.0742095, -.0529920}},
	} {
		t.Run(tc.utc, func(t *testing.T) {
			at, err := time.Parse(time.RFC3339, tc.utc)
			if err != nil {
				t.Fatal(err)
			}
			chart, err := CalculateKundali("../../ephe", at, tc.lat, tc.lon)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(chart.Lagna.Longitude-tc.lagna) > 1e-6 {
				t.Fatalf("CLI lagna got %.9f want %.9f", chart.Lagna.Longitude, tc.lagna)
			}
			for i := range tc.positions {
				p := chart.Planets[i]
				if math.Abs(p.Longitude-tc.positions[i]) > 1e-6 || math.Abs(p.Speed-tc.speeds[i]) > 1e-6 {
					t.Errorf("CLI %s got %.9f / %.9f want %.9f / %.9f", p.Name, p.Longitude, p.Speed, tc.positions[i], tc.speeds[i])
				}
			}
		})
	}
}

// TestKundaliIntegration checks the native Kundali calculation as a complete workflow.
func TestKundaliIntegration(t *testing.T) {
	epheDir := os.Getenv("SWEPH_TEST_EPHE")
	if epheDir == "" {
		epheDir = filepath.Join("..", "..", "ephe")
	}
	for _, name := range []string{"sepl_18.se1", "semo_18.se1"} {
		if _, err := os.Stat(filepath.Join(epheDir, name)); err != nil {
			t.Fatalf("integration test needs %s; set SWEPH_TEST_EPHE: %v", name, err)
		}
	}
	birth, err := time.Parse(time.RFC3339, "1995-08-15T10:30:00+05:30")
	if err != nil {
		t.Fatal(err)
	}
	a, err := CalculateKundali(epheDir, birth, 28.6139, 77.2090)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Planets) != 9 || len(a.Houses) != 12 {
		t.Fatalf("wrong counts: %+v", a)
	}
	if a.UTC != "1995-08-15T05:00:00Z" {
		t.Fatalf("wrong UTC: %s", a.UTC)
	}
	if a.Lagna.Rashi != "Kanya" || a.Lagna.Longitude < 178.45 || a.Lagna.Longitude > 178.47 {
		t.Fatalf("unexpected demo lagna: %+v", a.Lagna)
	}
	moon := a.Planets[1]
	if moon.Rashi != "Meen" || moon.Nakshatra != "Revati" || moon.Pada != 3 || moon.House != 7 {
		t.Fatalf("unexpected demo Moon: %+v", moon)
	}
	if math.Abs(normalizeDegrees(a.Planets[8].Longitude-a.Planets[7].Longitude)-180) > 1e-10 {
		t.Fatal("Rahu and Ketu must be opposite")
	}
	// Same instant expressed in UTC, at a different place: geocentric planets
	// should agree while the ascendant changes.
	b, err := CalculateKundali(epheDir, birth.UTC(), 19.0760, 72.8777)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range a.Planets {
		if math.Abs(p.Longitude-b.Planets[i].Longitude) > 1e-10 {
			t.Fatalf("geocentric longitude changed with place/timezone: %s", p.Name)
		}
	}
	if math.Abs(a.Lagna.Longitude-b.Lagna.Longitude) < 1 {
		t.Fatal("lagna did not change with location")
	}
	c, err := CalculateKundali(epheDir, birth.Add(-10*time.Hour-15*time.Minute), 28.6139, 77.2090)
	if err != nil {
		t.Fatal(err)
	}
	if c.UTC != "1995-08-14T18:45:00Z" {
		t.Fatalf("wrong UTC rollover: %s", c.UTC)
	}
	if _, err := CalculateKundali(epheDir, birth, math.NaN(), 77); err == nil {
		t.Fatal("NaN latitude accepted")
	}
	if _, err := CalculateKundali(t.TempDir(), birth, 28, 77); err == nil {
		t.Fatal("missing ephemeris accepted")
	}
	// New and previous entrypoints must share the same C-state lock.
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sun, err := SunPosition(epheDir, birth)
			if err != nil {
				t.Error(err)
				return
			}
			chart, err := CalculateKundali(epheDir, birth, 28.6139, 77.2090)
			if err != nil {
				t.Error(err)
				return
			}
			if math.Abs(sun.Longitude-chart.Planets[0].Longitude) > 1e-10 {
				t.Error("Sun results disagree")
			}
		}()
	}
	wg.Wait()
}

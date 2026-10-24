package swisseph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSwissFlagsMatch checks fallback and conflicting engine bits, including
// positive return codes that must not be mistaken for successful Swiss use.
func TestSwissFlagsMatch(t *testing.T) {
	// These numeric flag values are the bundled swephexp.h public constants.
	const jpl, swiss, moshier, speed, sidereal int32 = 1, 2, 4, 256, 65536
	requested := swiss | speed | sidereal
	for _, tc := range []struct {
		name     string
		returned int32
		want     bool
	}{
		{"Swiss", requested, true},
		{"Swiss with extra non-engine flag", requested | 2048, true},
		{"negative error", -1, false},
		{"no flags", 0, false},
		{"positive Moshier fallback", moshier | speed | sidereal, false},
		{"positive JPL result", jpl | speed | sidereal, false},
		{"Swiss and Moshier together", requested | moshier, false},
		{"Swiss and JPL together", requested | jpl, false},
		{"missing speed", swiss | sidereal, false},
		{"missing sidereal", swiss | speed, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := swissFlagsMatch(tc.returned, requested); got != tc.want {
				t.Fatalf("flags=%d: accepted=%v, want %v", tc.returned, got, tc.want)
			}
		})
	}
	if !swissFlagsMatch(swiss, swiss) {
		t.Fatal("tropical Swiss result rejected")
	}
}

// TestEphemerisDataFailuresRejectResults uses isolated file copies to check
// missing and corrupt data without changing the application's ephemeris files.
func TestEphemerisDataFailuresRejectResults(t *testing.T) {
	t.Setenv("SE_EPHE_PATH", "")
	at := time.Date(2026, 10, 18, 6, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		file string
		mode string
	}{
		{"missing planet file", "sepl_18.se1", "missing"},
		{"missing Moon file", "semo_18.se1", "missing"},
		{"corrupt planet file", "sepl_18.se1", "corrupt"},
		{"corrupt Moon file", "semo_18.se1", "corrupt"},
		{"empty planet file", "sepl_18.se1", "empty"},
		{"empty Moon file", "semo_18.se1", "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, name := range []string{"sepl_18.se1", "semo_18.se1"} {
				if name == tc.file && tc.mode == "missing" {
					continue
				}
				data, err := os.ReadFile(filepath.Join("../../ephe", name))
				if err != nil {
					t.Fatal(err)
				}
				if name == tc.file {
					if tc.mode == "corrupt" {
						data = []byte("invalid ephemeris data\n")
					} else {
						data = nil
					}
				}
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			// check rejects any API result accepted despite its required damaged data.
			check := func(name string, err error) {
				t.Helper()
				if err == nil {
					t.Fatalf("%s accepted %s", name, tc.name)
				}
				t.Logf("%s rejected: %v", name, err)
			}
			_, err := CalculateKundali(dir, at, 28.6139, 77.209)
			check("Kundli", err)
			_, err = CalculatePanchang(context.Background(), dir, "2026-10-18", "Asia/Kolkata", "upper-limb", 28.6139, 77.209)
			check("Panchang", err)
			_, err = CalculatePlanetPositions(context.Background(), dir, at)
			check("planet positions", err)
			_, err = CalculatePlanetTransitions(context.Background(), dir, 2026)
			check("transits", err)
			_, err = CalculatePlanetCombustions(context.Background(), dir, 2026)
			check("combustion", err)
			_, err = CalculateEclipses(context.Background(), dir, 2026, 28.6139, 77.209)
			check("eclipses", err)
			if tc.file == "sepl_18.se1" {
				_, err = SunPosition(dir, at)
				check("Sun CLI wrapper", err)
			}
		})
	}
	// A successful calculation after the failure cases also checks C-state cleanup.
	chart, err := CalculateKundali("../../ephe", at, 28.6139, 77.209)
	if err != nil || len(chart.Planets) != 9 {
		t.Fatalf("valid data did not recover after failures: %v", err)
	}
	t.Logf("valid control: Swiss %s, %d Grahas", chart.SwissVersion, len(chart.Planets))
}

// TestNativeMoshierFallbackRejected forces a real Swiss fallback by probing a
// date beyond the bundled file span through the internal session. Public dates
// remain restricted; the positive Moshier return must still be rejected.
func TestNativeMoshierFallbackRejected(t *testing.T) {
	t.Setenv("SE_EPHE_PATH", "")
	err := withSky(context.Background(), "../../ephe", "UTC", func(s *skySession) error {
		jd, err := s.jd(time.Date(2500, 1, 1, 12, 0, 0, 0, time.UTC))
		if err != nil {
			return err
		}
		for _, index := range []int{0, 1} {
			p, err := s.position(jd, index)
			// 65860 = Moshier (4) + speed (256) + sidereal (65536)
			// + Swiss's automatic non-nutation flag (64).
			if err == nil || !strings.Contains(err.Error(), "flags=65860") || p.Name != "" {
				t.Fatalf("expected a rejected positive Moshier result for %s, got %+v / %v", skyNames[index], p, err)
			}
			t.Logf("%s: %v; no position returned", skyNames[index], err)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

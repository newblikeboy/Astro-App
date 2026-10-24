package swisseph

import (
	"math"
	"testing"
	"time"
)

// TestNavamsaTraditionalMapping checks Navamsa placement against traditional sign mappings.
func TestNavamsaTraditionalMapping(t *testing.T) {
	// Independent traditional seed table: movable=self, fixed=ninth,
	// dual=fifth. Check every division in all twelve natal signs.
	seeds := []int{1, 10, 7, 4, 1, 10, 7, 4, 1, 10, 7, 4}
	for sign, seed := range seeds {
		for part := 0; part < 9; part++ {
			longitude := float64(sign*30) + (float64(part)+0.5)*10/3
			want := (seed-1+part)%12 + 1
			if got := navamsaSign(longitude); got != want {
				t.Fatalf("sign %d division %d: got %d want %d", sign+1, part+1, got, want)
			}
		}
	}
	for _, f := range []struct {
		lon  float64
		sign int
	}{{0, 1}, {10.0/3 - 1e-9, 1}, {10.0 / 3, 2}, {30, 10}, {60, 7}, {359.999999, 12}, {360, 1}, {-1e-6, 12}} {
		if navamsaSign(f.lon) != f.sign {
			t.Errorf("boundary %.12f: %d", f.lon, navamsaSign(f.lon))
		}
	}
	chart := Kundali{Lagna: zodiacPosition(30), Planets: []Graha{{Name: "Sun", ZodiacPosition: zodiacPosition(45)}}}
	d9 := CalculateNavamsa(chart)
	if d9.LagnaSign != 10 || d9.Planets[0].RashiNumber != 2 || d9.Planets[0].House != 5 || !d9.Planets[0].Vargottama {
		t.Fatalf("bad D9: %+v", d9)
	}
}

// TestVimshottariBirthBalanceAndAntardashas checks the birth mahadasha balance and proportional antardasha periods.
func TestVimshottariBirthBalanceAndAntardashas(t *testing.T) {
	birth := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	// Half Ashwini = half of Ketu's seven years remaining. At birth the
	// antardasha is Rahu, not Ketu: subperiods start before the birth instant.
	d := CalculateVimshottari(20.0/3, birth, birth)
	if d.BirthLord != "Ketu" || math.Abs(d.BalanceYears-3.5) > 1e-10 || d.CurrentLord != "Ketu" || d.CurrentSubLord != "Rahu" {
		t.Fatalf("bad balance: %+v", d)
	}
	if !d.Periods[0].Start.Equal(time.Date(1996, 7, 2, 3, 0, 0, 0, time.UTC)) || !d.Periods[0].End.Equal(time.Date(2003, 7, 2, 21, 0, 0, 0, time.UTC)) {
		t.Fatalf("independent Julian-year date fixture: %+v", d.Periods[0])
	}
	for i, p := range d.Periods {
		if i > 0 && !p.Start.Equal(d.Periods[i-1].End) {
			t.Fatal("gap between mahadashas")
		}
		if len(p.Antardashas) != 9 || !p.Antardashas[0].Start.Equal(p.Start) || !p.Antardashas[8].End.Equal(p.End) {
			t.Fatal("subperiods don't partition mahadasha")
		}
		for j, sub := range p.Antardashas {
			if !sub.End.After(sub.Start) || (j > 0 && !sub.Start.Equal(p.Antardashas[j-1].End)) {
				t.Fatal("subperiod gap")
			}
		}
	}
	boundary := d.Periods[0].End
	if got := CalculateVimshottari(20.0/3, birth, boundary); got.CurrentLord != "Venus" || got.CurrentSubLord != "Venus" {
		t.Fatal("period boundaries must be start-inclusive, end-exclusive")
	}
	if total := d.Periods[8].End.Sub(d.Periods[0].Start).Hours() / 24; math.Abs(total-120*365.25) > 1e-8 {
		t.Fatal("cycle is not 120 Julian years")
	}
	for star := 0; star < 27; star++ {
		got := CalculateVimshottari((float64(star)+0.25)*360/27, birth, birth)
		if got.BirthLord != dashaLords[star%9] || math.Abs(got.BalanceYears-0.75*dashaYears[star%9]) > 1e-10 {
			t.Fatalf("nakshatra %d: %+v", star+1, got)
		}
	}
}

// TestVimshottariContinuesBeyondFirstCycle checks that Vimshottari periods continue beyond the initial 120-year cycle.
func TestVimshottariContinuesBeyondFirstCycle(t *testing.T) {
	birth := time.Date(1972, 1, 1, 12, 0, 0, 0, time.UTC)
	atBirth := CalculateVimshottari(20.0/3, birth, birth)
	cycleEnd := atBirth.Periods[8].End
	boundary := CalculateVimshottari(20.0/3, birth, cycleEnd)
	if len(boundary.Periods) != 18 || boundary.CurrentLord != "Ketu" || boundary.CurrentSubLord != "Ketu" {
		t.Fatalf("second cycle boundary has no valid running lord: %+v", boundary)
	}
	afterOneCycle := CalculateVimshottari(20.0/3, birth, birth.Add(43830*24*time.Hour))
	if len(afterOneCycle.Periods) != 18 || afterOneCycle.CurrentLord != "Ketu" || afterOneCycle.CurrentSubLord != "Rahu" {
		t.Fatal("120 Julian years must repeat the natal phase")
	}
	for i, p := range afterOneCycle.Periods {
		if i < 9 && (!p.Start.Equal(atBirth.Periods[i].Start) || !p.End.Equal(atBirth.Periods[i].End)) {
			t.Fatal("adding a cycle changed the natal cycle")
		}
		if i > 0 && !p.Start.Equal(afterOneCycle.Periods[i-1].End) {
			t.Fatal("gap at mahadasha/cycle boundary")
		}
		if !p.Antardashas[0].Start.Equal(p.Start) || !p.Antardashas[8].End.Equal(p.End) {
			t.Fatal("subperiod coverage changed in the second cycle")
		}
	}
	old := CalculateVimshottari(20.0/3, birth, time.Date(2099, 12, 31, 0, 0, 0, 0, time.UTC))
	if old.CurrentLord == "" || old.CurrentSubLord == "" {
		t.Fatal("supported 1972 birth / 2099 as-of is missing its running dasha")
	}
	before := CalculateVimshottari(20.0/3, birth, birth.Add(-time.Second))
	if before.CurrentLord != "" || before.CurrentSubLord != "" || len(before.Periods) != 9 {
		t.Fatal("a pre-birth as-of must not assign a running birth dasha")
	}
	for _, lon := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		invalid := CalculateVimshottari(lon, birth, birth)
		if len(invalid.Periods) != 0 || invalid.CurrentLord != "" || navamsaSign(lon) != 0 {
			t.Fatal("invalid longitude produced a divisional sign or dasha")
		}
	}
}

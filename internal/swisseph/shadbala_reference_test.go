package swisseph

import (
	"context"
	"math"
	"testing"
	"time"
)

// TestShadbalaPublishedIndependentComponents compares the shared geometric
// components with AstroSage's published Pooja Sharma example, PDF p.108:
// https://www.astrosage.com/pdf/vedic-report-en.pdf . Full totals are not made
// equivalent: its Saptavargaja, luminary and minimum-Sun conventions differ.
func TestShadbalaPublishedIndependentComponents(t *testing.T) {
	at := time.Date(1978, 8, 23, 18, 23, 18, 0, time.UTC)
	chart, e := CalculateKundali("../../ephe", at, 28+40.0/60, 77+13.0/60)
	if e != nil {
		t.Fatal(e)
	}
	birth, chalit, e := CalculateBirthDetails(context.Background(), "../../ephe", "Asia/Kolkata", at, chart)
	if e != nil {
		t.Fatal(e)
	}
	bala, e := CalculateShadbala(context.Background(), "../../ephe", "Asia/Kolkata", at, chart, birth, chalit)
	if e != nil {
		t.Fatal(e)
	}
	uchcha := [7]float64{21.10, 54.46, 16.89, 44.43, 59.64, 1.44, 36.66}
	dig := [7]float64{2.63, 25.86, 16.62, 35.74, 43.86, 42.04, 28.15}
	oja := [7]float64{30, 0, 15, 0, 15, 30, 30}
	kendra := [7]float64{60, 15, 30, 15, 15, 30, 60}
	for i, p := range bala.Planets {
		if math.Abs(p.SthanaParts[0].Virupas-uchcha[i]) > .02 || math.Abs(p.Dig-dig[i]) > .03 || p.SthanaParts[2].Virupas != oja[i] || p.SthanaParts[3].Virupas != kendra[i] {
			t.Errorf("published %s geometric components: uchcha %.5f / %.2f; dig %.5f / %.2f; oja %.0f / %.0f; kendra %.0f / %.0f", p.Planet, p.SthanaParts[0].Virupas, uchcha[i], p.Dig, dig[i], p.SthanaParts[2].Virupas, oja[i], p.SthanaParts[3].Virupas, kendra[i])
		}
	}
	if bala.Planets[0].Required != 390 {
		t.Fatal("BPHS 27.32-33 Sun requirement must not be replaced by another provider's 300")
	}
}

package platform

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestKundliExpansionSnapshotAndCoreIsolation checks supplementary report snapshots without changing the core chart.
func TestKundliExpansionSnapshotAndCoreIsolation(t *testing.T) {
	a := testApp(t)
	birth := testBirth()
	full, e := calculate(a.Config, birth, nil, a.Now())
	if e != nil {
		t.Fatal(e)
	}
	if full.Details == nil || len(full.Details.Vargas) != 16 || len(full.Details.Birth.Limbs) != 4 || len(full.Details.Analysis.Conditions) != 9 || full.Details.Ashtakavarga.Total != 337 || len(full.Details.Yogini.Periods) < 32 {
		t.Fatal("complete Kundli details missing")
	}
	if full.Details.YogaChecks == nil || full.Details.YogaChecks.Method == "" || len(full.Details.YogaChecks.KaalSarp.Planets) != 7 || len(full.Details.YogaChecks.Rajyog.Pairs) == 0 {
		t.Fatal("new natal yoga checks missing from full calculation")
	}
	if full.Numerology == nil || full.Numerology.Mulank.Value != 6 || full.Numerology.Bhagyank.Value != 2 {
		t.Fatal("known-time Kundli lost date-based numerology")
	}
	raw, e := json.Marshal(full)
	if e != nil {
		t.Fatal(e)
	}
	var restored Report
	if e = json.Unmarshal(raw, &restored); e != nil || restored.Details == nil || restored.Details.Birth.Chakra.Nadi != full.Details.Birth.Chakra.Nadi || len(restored.Details.Chalit.Houses) != 12 {
		t.Fatal("new details lost in immutable snapshot", e)
	}
	if restored.Details.YogaChecks == nil || restored.Details.YogaChecks.KaalSarp.Status != full.Details.YogaChecks.KaalSarp.Status || len(restored.Details.YogaChecks.Rajyog.Pairs) != len(full.Details.YogaChecks.Rajyog.Pairs) {
		t.Fatal("natal yoga evidence lost in saved snapshot")
	}
	core, e := calculateCore(a.Config, birth, nil, a.Now())
	if e != nil || core.Details != nil || core.Chart == nil || core.Dasha == nil {
		t.Fatal("branch/matching core isolation", e)
	}
	unknown := birth
	unknown.Unknown = true
	u, e := calculate(a.Config, unknown, nil, a.Now())
	if e != nil || u.Details != nil {
		t.Fatal("unknown birth invented details", e)
	}
}

// TestKundliSaturnValidationAndNativeContract checks Saturn timeline validation and the native calculation contract.
func TestKundliSaturnValidationAndNativeContract(t *testing.T) {
	a := testApp(t)
	for _, body := range []string{
		`{"birth":{"unknown_time":true},"start_year":2020,"end_year":2030}`,
		`{"start_year":1939,"end_year":1941}`,
		`{"start_year":2020,"end_year":2051}`,
		`{"start_year":2020,"end_year":2020}`,
		`{"start_year":2099,"end_year":2101}`,
	} {
		req := httptest.NewRequest("POST", "/api/kundli/saturn", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		a.kundliSaturn(w, req)
		if w.Code != 422 {
			t.Fatalf("invalid Saturn body: %d %s", w.Code, w.Body.String())
		}
	}
	birth := testBirth()
	payload, _ := json.Marshal(map[string]any{"birth": birth, "start_year": 2023, "end_year": 2024})
	req := httptest.NewRequest("POST", "/api/kundli/saturn", strings.NewReader(string(payload)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	a.kundliSaturn(w, req)
	if w.Code != 200 {
		t.Fatalf("native timeline failed %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Start, End time.Time
		Periods    []struct {
			Start, End time.Time
			Kind       string
		}
	}
	if e := json.Unmarshal(w.Body.Bytes(), &out); e != nil || !out.End.After(out.Start) || len(out.Periods) == 0 {
		t.Fatal("invalid timeline", e)
	}
	for _, p := range out.Periods {
		if !p.End.After(p.Start) || p.Start.Before(out.Start) || p.End.After(out.End) || (p.Kind != "Sade Sati" && p.Kind != "Dhaiya") {
			t.Fatal("invalid period")
		}
	}
}

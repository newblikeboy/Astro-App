package platform

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

// TestNationwidePlaceSearch checks nationwide birthplace lookup, aliases and regional qualifiers.
func TestNationwidePlaceSearch(t *testing.T) {
	for _, tc := range []struct{ query, name, region string }{
		{"Gorakhpur", "Gorakhpur", "Uttar Pradesh"},
		{"गोरखपुर", "Gorakhpur", "Uttar Pradesh"},
		{"Kochi Kerala", "Kochi", "Kerala"},
		{"Guwahati", "Guwahati", "Assam"},
		{"Leh Ladakh", "Leh", "Ladakh"},
		{"Shillong", "Shillong", "Meghalaya"},
		{"Kavaratti", "Kavaratti", "Lakshadweep"},
		{"Port Blair", "Port Blair", "Andaman and Nicobar"},
		{"Srinagar Jammu", "Srinagar", "Jammu and Kashmir"},
		{"Thiruvananthapuram", "Thiruvananthapuram", "Kerala"},
		{"Allahabad", "Prayagraj", "Uttar Pradesh"},
		{"Bangalore", "Bengaluru", "Karnataka"},
	} {
		t.Run(tc.query, func(t *testing.T) {
			matches := searchPlaces(tc.query)
			if len(matches) == 0 || len(matches) > 25 {
				t.Fatalf("unexpected results: %d", len(matches))
			}
			p := matches[0]
			if !strings.Contains(p.Name, tc.name) || !strings.Contains(p.Name, tc.region) {
				t.Fatalf("first result = %s", p.Name)
			}
			resolved, ok := findPlace(p.ID)
			if !ok || resolved != p || p.Zone != "Asia/Kolkata" {
				t.Fatalf("place not trusted/resolvable: %+v", p)
			}
		})
	}
	if got := searchPlaces("zzzz-no-such-place-zzzz"); len(got) != 0 {
		t.Fatalf("unmatched search returned %d results", len(got))
	}
	// A district-qualified query distinguishes this village from the UP city.
	got := searchPlaces("Gorakhpur Fatehabad")
	if len(got) == 0 || !strings.Contains(got[0].Name, "Haryana") {
		t.Fatalf("district search = %+v", got)
	}
	if len(catalog.entries) < 500000 {
		t.Fatalf("catalog unexpectedly small: %d", len(catalog.entries))
	}
	states := map[string]bool{}
	for _, p := range catalog.entries {
		if strings.HasPrefix(p.place.ID, "geonames-") {
			parts := strings.Split(p.place.Name, ", ")
			states[parts[len(parts)-2]] = true
		}
	}
	if len(states) != 36 {
		t.Fatalf("state/UT coverage = %d", len(states))
	}
}

// TestPlacesAPIAndCalculationUseSameCatalog checks that place search and calculation resolve the same trusted catalogue records.
func TestPlacesAPIAndCalculationUseSameCatalog(t *testing.T) {
	a := testApp(t)
	w := request(t, a, "GET", "/api/places?q="+url.QueryEscape("Gorakhpur Uttar Pradesh"), nil, nil, "")
	var found []Place
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &found) != nil || len(found) == 0 {
		t.Fatalf("search failed: %s", w.Body.String())
	}
	p := found[0]
	w = request(t, a, "GET", "/api/places/"+p.ID, nil, nil, "")
	var restored Place
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &restored) != nil || restored != p {
		t.Fatal("selected place cannot be restored")
	}
	b := testBirth()
	b.PlaceID = p.ID
	r, err := calculate(a.Config, b, seed().Rules, a.Now())
	if err != nil {
		t.Fatal(err)
	}
	if r.Place != p || r.Chart == nil || r.Chart.Settings["timezone"] != "Asia/Kolkata" {
		t.Fatal("calculation did not resolve source location")
	}
	if w = request(t, a, "GET", "/api/places/made-up", nil, nil, ""); w.Code != 404 {
		t.Fatalf("unknown ID status = %d", w.Code)
	}
	if w = request(t, a, "GET", "/api/places?q="+strings.Repeat("a", 101), nil, nil, ""); w.Code != 400 {
		t.Fatalf("oversize query status = %d", w.Code)
	}
	b.PlaceID = "made-up"
	if _, err = calculate(a.Config, b, seed().Rules, a.Now()); err == nil {
		t.Fatal("unknown birthplace accepted")
	}
	for _, original := range Places {
		if got, ok := findPlace(original.ID); !ok || got != original {
			t.Fatalf("legacy location changed: %s", original.ID)
		}
	}
}

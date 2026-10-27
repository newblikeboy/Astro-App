package numerology

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

// TestChaldeanNames checks independent full-name sums, case/separators, 9 totals and unsupported spellings.
func TestChaldeanNames(t *testing.T) {
	for _, tc := range []struct {
		name         string
		total, value int
	}{
		{"Raja Babu", 16, 7}, {"raja-babu", 16, 7}, {"RAJA.BABU", 16, 7},
		{"Mary Jane", 20, 2}, {"Abf", 11, 2}, {"John Smith", 35, 8}, {"Af", 9, 9},
	} {
		r, err := CalculateName(tc.name)
		if err != nil || r.Number.Steps[0] != tc.total || r.Number.Value != tc.value || len(r.Letters) != len(r.Number.Digits) || r.Meaning.Hindi == "" || r.Meaning.English == "" {
			t.Fatalf("%s: %+v %v", tc.name, r, err)
		}
	}
	for _, name := range []string{"", "A", "...", "राजा बाबू", "Raja बाबू", "Raja 1", "José", "Raja@Babu", "ıra", "ſara", strings.Repeat("a", 101)} {
		if _, err := CalculateName(name); err == nil {
			t.Fatalf("accepted unsupported spelling %q", name)
		}
	}
}

// TestExtendedProfiles verifies all nine authored profiles, date-series bounds and planetary yantra magic sums.
func TestExtendedProfiles(t *testing.T) {
	for n := 1; n <= 9; n++ {
		date := "2000-01-0" + string(rune('0'+n))
		r, err := CalculateFull(date, "Raja Babu")
		if err != nil || r.Extended == nil {
			t.Fatal(err)
		}
		e := r.Extended
		if e.Method != ExtendedMethod || e.NameStatus != "available" || e.Naamank.Number.Value != 7 || e.Profile.Number != n || len(e.Profile.Facts) != 9 || len(e.Profile.Guidance) != 9 {
			t.Fatalf("incomplete profile %d", n)
		}
		for _, fact := range e.Profile.Facts {
			if fact.ID == "" || fact.Label.Hindi == "" || fact.Label.English == "" || fact.Value.Hindi == "" || fact.Value.English == "" {
				t.Fatal("missing bilingual fact", n)
			}
		}
		ids := map[string]bool{}
		for _, part := range e.Profile.Guidance {
			if ids[part.ID] || part.Title.Hindi == "" || part.Title.English == "" || part.Text.Hindi == "" || part.Text.English == "" {
				t.Fatal("missing or duplicate topic", n)
			}
			ids[part.ID] = true
		}
		previous := 0
		for _, day := range e.LuckyDates {
			if day <= previous || day > 31 || ((day-1)%9+1 != n && (day-1)%9+1 != r.Bhagyank.Value) {
				t.Fatal("incorrect date series", n, day)
			}
			previous = day
		}
		grid := e.Profile.Yantra
		if len(grid) != 3 {
			t.Fatal("missing yantra")
		}
		sum := grid[0][0] + grid[0][1] + grid[0][2]
		for i := 0; i < 3; i++ {
			if len(grid[i]) != 3 || grid[i][0]+grid[i][1]+grid[i][2] != sum || grid[0][i]+grid[1][i]+grid[2][i] != sum {
				t.Fatal("invalid yantra", n)
			}
		}
		if grid[0][0]+grid[1][1]+grid[2][2] != sum || grid[0][2]+grid[1][1]+grid[2][0] != sum {
			t.Fatal("invalid diagonals", n)
		}
	}
}

// TestExtendedEvidenceAndIsolation checks the supplied case, missing-script behaviour and no shared mutable catalogue data.
func TestExtendedEvidenceAndIsolation(t *testing.T) {
	r, err := CalculateFull("1995-11-29", "Raja Babu")
	if err != nil || r.Mulank.Value != 2 || r.Bhagyank.Value != 1 || r.Extended.Naamank.Number.Value != 7 {
		t.Fatal(r, err)
	}
	if !reflect.DeepEqual(r.Extended.LuckyNumbers, []int{2, 1}) || !reflect.DeepEqual(r.Extended.LuckyDates, []int{1, 2, 10, 11, 19, 20, 28, 29}) || !reflect.DeepEqual(r.Extended.Profile.Yantra, [][]int{{7, 2, 9}, {8, 6, 4}, {3, 10, 5}}) {
		t.Fatal("fixture differs", r.Extended)
	}
	before, _ := json.Marshal(r)
	r.Extended.Profile.Facts[0].Value.Hindi = "changed"
	r.Extended.Profile.Yantra[0][0] = 999
	again, _ := CalculateFull("1995-11-29", "Raja Babu")
	after, _ := json.Marshal(again)
	if string(before) != string(after) {
		t.Fatal("profile shared mutable state")
	}
	missing, err := CalculateFull("1995-11-29", "राजा बाबू")
	if err != nil || missing.Extended.NameStatus != "needs_latin_spelling" || missing.Extended.Naamank != nil || missing.Mulank.Value != 2 {
		t.Fatal("unsupported script affected date reading", err)
	}
	if _, err := CalculateFull("1995-02-30", "Raja Babu"); err == nil {
		t.Fatal("invalid date admitted")
	}
}

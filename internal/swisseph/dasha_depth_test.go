package swisseph

import (
	"math/big"
	"testing"
	"time"
)

// TestDashaFiveLevelExactPartitions checks that all five Dasha levels exactly partition their parent periods.
func TestDashaFiveLevelExactPartitions(t *testing.T) {
	// Independent integer-rational oracle for every descendant in one cycle:
	// 9 + 81 + 729 + 6561 + 59049 intervals. Includes the 20-year parent.
	d := CalculateVimshottari(0, time.Date(2000, 1, 1, 0, 0, 31, 0, time.UTC), time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC))
	counts := [5]int{}
	var walk func([]int, int)
	// walk recursively checks that each Dasha branch partitions its parent into nine contiguous subperiods.
	walk = func(path []int, level int) {
		parent, periods, err := DashaBranch(d, path)
		if err != nil || len(periods) != 9 {
			t.Fatalf("branch %v: %v", path, err)
		}
		counts[level-1] += len(periods)
		if parent != nil {
			if !periods[0].Start.Equal(parent.Start) || !periods[8].End.Equal(parent.End) || periods[0].Lord != parent.Lord {
				t.Fatalf("parent coverage/order %v", path)
			}
			if level >= 3 {
				sum := int64(0)
				for _, p := range periods {
					for k, name := range dashaLords {
						if name == p.Lord {
							sum += int64(dashaYears[k])
							break
						}
					}
					n := new(big.Int).Mul(big.NewInt(int64(parent.End.Sub(parent.Start))), big.NewInt(sum))
					n.Add(n, big.NewInt(60))
					n.Div(n, big.NewInt(120))
					if !p.End.Equal(parent.Start.Add(time.Duration(n.Int64()))) {
						t.Fatalf("rational boundary %v", path)
					}
				}
			}
		}
		for i, p := range periods {
			if !p.End.After(p.Start) || (i > 0 && !p.Start.Equal(periods[i-1].End)) {
				t.Fatalf("gap/overlap at %v/%d", path, i)
			}
			if level < 5 {
				walk(append(append([]int{}, path...), i), level+1)
			}
		}
	}
	walk(nil, 1)
	if counts != [5]int{9, 81, 729, 6561, 59049} {
		t.Fatal(counts)
	}
}

// TestDashaCurrentFiveLevelsAndBoundaries checks the active five-level Dasha sequence at interval boundaries.
func TestDashaCurrentFiveLevelsAndBoundaries(t *testing.T) {
	birth := time.Date(2000, 1, 1, 12, 0, 37, 0, time.UTC)
	d := CalculateVimshottari(20.0/3, birth, birth)
	if d.Method != VimshottariMethod || d.MaxDepth != 5 || len(d.CurrentPath) != 5 {
		t.Fatal("five-level metadata missing")
	}
	if d.CurrentPath[0].Lord != d.CurrentLord || d.CurrentPath[1].Lord != d.CurrentSubLord {
		t.Fatal("legacy current-lord mismatch")
	}
	for level, step := range d.CurrentPath {
		if step.Level != level+1 || len(step.Path) != level+1 || birth.Before(step.Start) || !birth.Before(step.End) {
			t.Fatal("invalid active path")
		}
		path := step.Path[:level]
		_, periods, e := DashaBranch(d, path)
		if e != nil {
			t.Fatal(e)
		}
		for i, p := range periods {
			at := CalculateVimshottari(20.0/3, birth, p.Start)
			if p.Start.Before(birth) {
				continue
			}
			if len(at.CurrentPath) != 5 || at.CurrentPath[level].Lord != p.Lord || !at.CurrentPath[level].Start.Equal(p.Start) {
				t.Fatalf("start inclusive at depth %d/%d", level+1, i)
			}
			before := CalculateVimshottari(20.0/3, birth, p.End.Add(-time.Nanosecond))
			if before.CurrentPath[level].Lord != p.Lord {
				t.Fatal("end minus nanosecond excluded")
			}
		}
	}
	for _, path := range [][]int{{-1}, {9}, {0, 9}, {0, 0, 0, 0, 0}} {
		if _, _, err := DashaBranch(d, path); err == nil {
			t.Fatal("invalid depth/path accepted", path)
		}
	}
	for _, p := range []DashaPeriod{{Lord: "invalid", Start: birth, End: birth.Add(time.Hour)}, {Lord: "Sun", Start: birth, End: birth}} {
		if _, e := DashaChildren(p); e == nil {
			t.Fatal("bad parent accepted")
		}
	}
}

// TestDashaHandComputedFiveLevelSeconds compares precise five-level periods with hand-computed durations.
func TestDashaHandComputedFiveLevelSeconds(t *testing.T) {
	// Ketu at the very beginning of Ashwini; consecutive own-lord periods
	// last 7 * (7/120)^(depth-1) Julian years. Derive with rational arithmetic.
	birth := time.Date(2000, 1, 1, 0, 0, 31, 0, time.UTC)
	d := CalculateVimshottari(0, birth, birth)
	for depth := 1; depth <= 5; depth++ {
		path := make([]int, depth-1)
		_, periods, err := DashaBranch(d, path)
		if err != nil {
			t.Fatal(err)
		}
		seconds := new(big.Rat).SetInt64(7 * 36525 * 864)
		for i := 1; i < depth; i++ {
			seconds.Mul(seconds, big.NewRat(7, 120))
		}
		f, _ := seconds.Float64()
		got := periods[0].End.Sub(birth).Seconds()
		if delta := got - f; delta < -0.000001 || delta > 0.000001 {
			t.Fatalf("hand fixture depth %d: %.9f vs %.9f", depth, got, f)
		}
		if !periods[0].Start.Equal(birth) {
			t.Fatal("birth seconds discarded")
		}
	}
}

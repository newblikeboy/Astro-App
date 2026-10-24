package platform

import (
	"testing"
	"time"
)

// TestSlotsRetainCivilDayAcrossMidnightDSTGap checks that a midnight DST gap cannot move booking slots to another civil day.
func TestSlotsRetainCivilDayAcrossMidnightDSTGap(t *testing.T) {
	// Santiago skips 00:00-00:59 on 6 September 2026. Carrying the current
	// 00:30 clock into AddDate can normalize back into 5 September; the valid
	// Sunday daytime window must still be offered in its new UTC-03 offset.
	now, err := time.Parse(time.RFC3339, "2026-09-05T00:30:00-04:00")
	if err != nil {
		t.Fatal(err)
	}
	s := &State{Settings: Settings{
		Zone: "America/Santiago", Duration: 30, HorizonDays: 1,
		Availability: []Window{{Day: 0, From: "10:00", To: "11:00"}},
	}}
	got := slots(s, now, true)
	want := []string{"2026-09-06T13:00:00Z", "2026-09-06T13:30:00Z"}
	if len(got) != len(want) {
		t.Fatalf("day after midnight DST gap lost availability: %v", got)
	}
	for i, at := range got {
		if at.Format(time.RFC3339) != want[i] {
			t.Fatalf("slot %d got %s want %s", i, at, want[i])
		}
	}
}

// TestSlotsDeduplicateOverlappingWindowsAndRetainOrdering checks that overlapping availability windows produce unique, ordered slots.
func TestSlotsDeduplicateOverlappingWindowsAndRetainOrdering(t *testing.T) {
	now := time.Date(2026, 10, 18, 3, 0, 0, 0, time.UTC)
	s := &State{Settings: Settings{
		Zone: "Asia/Kolkata", Duration: 30,
		Availability: []Window{
			{Day: 0, From: "10:30", To: "12:00"},
			{Day: 0, From: "10:00", To: "11:30"},
			{Day: 0, From: "10:00", To: "11:30"},
		},
	}}
	got := slots(s, now, true)
	want := []string{"2026-10-18T04:30:00Z", "2026-10-18T05:00:00Z", "2026-10-18T05:30:00Z", "2026-10-18T06:00:00Z"}
	if len(got) != len(want) {
		t.Fatalf("overlapping windows produced duplicate/missing slots: %v", got)
	}
	for i, at := range got {
		if at.Format(time.RFC3339) != want[i] {
			t.Fatalf("slot order %d got %s want %s", i, at, want[i])
		}
	}
}

// TestSlotsPreserveLeadBufferAndHalfOpenContracts checks booking lead time, buffers and half-open interval boundaries.
func TestSlotsPreserveLeadBufferAndHalfOpenContracts(t *testing.T) {
	now := time.Date(2026, 10, 18, 8, 0, 0, 0, time.UTC)
	s := &State{
		Settings: Settings{
			Zone: "UTC", Duration: 30, Buffer: 15, LeadHours: 1,
			Availability: []Window{{Day: 0, From: "09:00", To: "12:00"}},
		},
		Bookings: map[string]*Booking{
			"confirmed": {ID: "confirmed", Start: now.Add(150 * time.Minute), End: now.Add(180 * time.Minute), Status: "confirmed"},
			// A hold expiring exactly now is inactive, even at a valid slot.
			"expired": {ID: "expired", Start: now.Add(105 * time.Minute), End: now.Add(135 * time.Minute), Status: "held", HoldUntil: now},
		},
	}
	got := slots(s, now, true)
	// 09:00 equals the strict lead cutoff and is excluded. 09:45 ends
	// exactly one buffer before the booking; 11:15 starts exactly one
	// buffer after it. Touching half-open boundaries must remain available.
	want := []string{"2026-10-18T09:45:00Z", "2026-10-18T11:15:00Z"}
	if len(got) != len(want) {
		t.Fatalf("lead/buffer boundaries changed: %v", got)
	}
	for i, at := range got {
		if at.Format(time.RFC3339) != want[i] {
			t.Fatalf("boundary slot %d got %s want %s", i, at, want[i])
		}
	}
	s.Bookings["expired"].HoldUntil = now.Add(time.Second)
	if got = slots(s, now, true); len(got) != 1 || got[0].Format(time.RFC3339) != want[1] {
		t.Fatalf("an active hold was ignored: %v", got)
	}
	s.Settings.Blocked = []string{"2026-10-18"}
	if got = slots(s, now, true); len(got) != 0 {
		t.Fatalf("blocked civil date was offered: %v", got)
	}
}

package swisseph

import (
	"context"
	"errors"
	"math"
	"testing"
)

// TestMonotoneCrossBoundaries checks angular boundary search endpoints and wrap handling.
func TestMonotoneCrossBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name                           string
		initial, target, maxDays, want float64
		direction                      int
	}{
		{"forward wrap", 359.8, 0, 1, 0.2 / 12, 1},
		{"backward wrap", 0.2, 0, 1, -0.2 / 12, -1},
		{"exclusive forward", 0, 0, 31, 30, 1},
		{"exclusive backward", 0, 0, 31, -30, -1},
		{"nearby forward boundary", 0, 5e-8, 1, 5e-8 / 12, 1},
		{"nearby backward boundary", 5e-8, 0, 1, -5e-8 / 12, -1},
		{"partial final sample", 0, 1, 0.1, 1.0 / 12, 1},
		{"crossing at search limit", 0, 1.2, 0.1, 0.1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// fn models constant forward angular motion for exact-boundary and wraparound search tests.
			fn := func(jd float64) (float64, error) { return normalizeDegrees(tc.initial + 12*jd), nil }
			got, err := monotoneCross(context.Background(), 0, tc.target, tc.direction, tc.maxDays, fn)
			if err != nil || math.Abs(got-tc.want) > 0.1/86400 {
				t.Fatalf("got %.12f, %v; want %.12f within 0.1 second", got, err, tc.want)
			}
			if float64(tc.direction)*got <= 0 || float64(tc.direction)*got > tc.maxDays {
				t.Fatal("crossing outside exclusive search window")
			}
		})
	}
}

// TestMonotoneCrossRejectsReverseMotion checks that a monotone boundary search rejects reverse angular motion.
func TestMonotoneCrossRejectsReverseMotion(t *testing.T) {
	for _, direction := range []int{-1, 1} {
		// A decreasing longitude must not become an almost-360-degree jump.
		fn := func(jd float64) (float64, error) { return 150 - 0.4*jd, nil }
		if got, err := monotoneCross(context.Background(), 0, 151, direction, 1, fn); err == nil {
			t.Fatalf("accepted reverse motion: direction=%d root=%f", direction, got)
		}
	}
	// Endpoints alone appear valid; the midpoint exposes a reversal.
	fn := func(jd float64) (float64, error) {
		if jd == 0 {
			return 0, nil
		}
		if jd == 0.25 {
			return 3, nil
		}
		return 4, nil
	}
	if _, err := monotoneCross(context.Background(), 0, 2, 1, 1, fn); err == nil {
		t.Fatal("accepted non-monotonic bisection midpoint")
	}
}

// TestMonotoneCrossCancellationAndErrors checks cancellation and propagated errors during boundary search.
func TestMonotoneCrossCancellationAndErrors(t *testing.T) {
	t.Run("already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		calls := 0
		_, err := monotoneCross(ctx, 0, 1, 1, 1, func(float64) (float64, error) {
			calls++
			return 0, nil
		})
		if !errors.Is(err, context.Canceled) || calls != 0 {
			t.Fatalf("got %v after %d calls", err, calls)
		}
	})
	t.Run("cancel during bisection", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		calls := 0
		_, err := monotoneCross(ctx, 0, 1, 1, 1, func(jd float64) (float64, error) {
			calls++
			if calls == 3 { // initial value, six-hour endpoint, first midpoint
				cancel()
			}
			return 12 * jd, nil
		})
		if !errors.Is(err, context.Canceled) || calls != 3 {
			t.Fatalf("got %v after %d calls", err, calls)
		}
	})
	t.Run("callback error", func(t *testing.T) {
		want := errors.New("ephemeris unavailable")
		_, err := monotoneCross(context.Background(), 0, 1, 1, 1, func(float64) (float64, error) { return 0, want })
		if !errors.Is(err, want) {
			t.Fatalf("lost callback error: %v", err)
		}
	})
	t.Run("outside horizon", func(t *testing.T) {
		_, err := monotoneCross(context.Background(), 0, 13, 1, 1, func(jd float64) (float64, error) { return 12 * jd, nil })
		if err == nil {
			t.Fatal("accepted crossing beyond maxDays")
		}
	})
	for _, failAt := range []int{1, 2, 3} {
		t.Run(string(rune('0'+failAt))+" non-finite callback", func(t *testing.T) {
			calls := 0
			_, err := monotoneCross(context.Background(), 0, 1, 1, 1, func(jd float64) (float64, error) {
				calls++
				if calls == failAt {
					return math.NaN(), nil
				}
				return 12 * jd, nil
			})
			if err == nil || calls != failAt {
				t.Fatalf("accepted non-finite longitude: err=%v calls=%d", err, calls)
			}
		})
	}
}

// TestMonotoneCrossInvalidInputs rejects invalid angular search inputs.
func TestMonotoneCrossInvalidInputs(t *testing.T) {
	// fn returns a monotone sample angle for checking invalid search inputs.
	fn := func(jd float64) (float64, error) { return jd, nil }
	for _, tc := range []struct {
		name                string
		at, target, maxDays float64
		direction           int
	}{
		{"zero direction", 0, 1, 1, 0},
		{"other direction", 0, 1, 1, 2},
		{"zero horizon", 0, 1, 0, 1},
		{"negative horizon", 0, 1, -1, 1},
		{"infinite horizon", 0, 1, math.Inf(1), 1},
		{"NaN horizon", 0, 1, math.NaN(), 1},
		{"infinite epoch", math.Inf(1), 1, 1, 1},
		{"NaN target", 0, math.NaN(), 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := monotoneCross(context.Background(), tc.at, tc.target, tc.direction, tc.maxDays, fn); err == nil {
				t.Fatal("accepted invalid search input")
			}
		})
	}
}

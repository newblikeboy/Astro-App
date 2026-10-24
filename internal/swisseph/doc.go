// Package swisseph wraps the bundled native Swiss Ephemeris and implements the
// declared Kundli, Panchang, matching and astronomical timing calculations.
// Native C state is process-wide and must remain behind engineMu. Coordinate,
// calendar, node and Dasha conventions are documented in docs/CALCULATIONS.md;
// independent numerical references live in testdata. Do not round calculations
// to the clock precision used by the website or adjust them to force parity
// with an external site's unexplained settings.
package swisseph

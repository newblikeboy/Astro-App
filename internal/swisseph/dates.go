package swisseph

import "time"

// Civil dates accepted by public daily tools. Adjacent-date astronomical
// searches may extend outside these labels without changing the public range.
const (
	MinDate = "1940-01-01"
	MaxDate = "2099-12-30"
	MinYear = 1940
	MaxYear = 2099
)

// Native instant entrypoints allow one preceding UTC day so local midnight on
// 1 January 1940 in India, Sydney or another eastern zone remains calculable.
// Public birth/date validation applies its own civil-date range before calling
// these functions. The upper native boundary remains 31 December 2099 UTC.
func supportedCalculationInstant(at time.Time) bool {
	return !at.Before(time.Date(MinYear-1, time.December, 31, 0, 0, 0, 0, time.UTC)) &&
		at.Before(time.Date(MaxYear+1, time.January, 1, 0, 0, 0, 0, time.UTC))
}

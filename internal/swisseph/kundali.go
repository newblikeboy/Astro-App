package swisseph

/*
#include <stdlib.h>
#include "swephexp.h"
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
	"unsafe"
)

type ZodiacPosition struct {
	Longitude       float64 `json:"longitude"`
	RashiNumber     int     `json:"rashi_number"`
	Rashi           string  `json:"rashi"`
	Degree          float64 `json:"degree_in_rashi"`
	DMS             string  `json:"degree_dms"`
	NakshatraNumber int     `json:"nakshatra_number"`
	Nakshatra       string  `json:"nakshatra"`
	Pada            int     `json:"pada"`
}

type Graha struct {
	Name string `json:"name"`
	ZodiacPosition
	House      int     `json:"house"`
	Speed      float64 `json:"longitude_speed_deg_per_day"`
	Retrograde bool    `json:"retrograde"`
}

type RashiHouse struct {
	Number        int     `json:"house"`
	RashiNumber   int     `json:"rashi_number"`
	Rashi         string  `json:"rashi"`
	CuspLongitude float64 `json:"cusp_longitude"`
}

type Kundali struct {
	BirthTime    string            `json:"birth_time"`
	UTC          string            `json:"utc"`
	Latitude     float64           `json:"latitude"`
	Longitude    float64           `json:"longitude"`
	JulianDayUT1 float64           `json:"julian_day_ut1"`
	SwissVersion string            `json:"swiss_version"`
	Settings     map[string]string `json:"settings"`
	Lagna        ZodiacPosition    `json:"lagna"`
	Planets      []Graha           `json:"planets"`
	Houses       []RashiHouse      `json:"houses"`
}

var rashiNames = [...]string{"Mesh", "Vrishabh", "Mithun", "Kark", "Simha", "Kanya",
	"Tula", "Vrishchik", "Dhanu", "Makar", "Kumbh", "Meen"}
var nakshatraNames = [...]string{"Ashwini", "Bharani", "Krittika", "Rohini", "Mrigashira",
	"Ardra", "Punarvasu", "Pushya", "Ashlesha", "Magha", "Purva Phalguni",
	"Uttara Phalguni", "Hasta", "Chitra", "Swati", "Vishakha", "Anuradha",
	"Jyeshtha", "Mula", "Purva Ashadha", "Uttara Ashadha", "Shravana",
	"Dhanishtha", "Shatabhisha", "Purva Bhadrapada", "Uttara Bhadrapada", "Revati"}

// finite reports whether a floating-point calculation value is neither NaN nor infinity.
func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// normalizeDegrees wraps an angle into the half-open range [0,360) without producing the upper endpoint.
func normalizeDegrees(value float64) float64 {
	value = math.Mod(value, 360)
	if value < 0 {
		value += 360
	}
	// Adding 360 to a negative sub-ULP remainder can round to 360 itself.
	// Keep the documented half-open range: callers use it to index signs.
	if value >= 360 {
		return math.Nextafter(360, 0)
	}
	if value == 0 {
		return 0
	}
	return value
}

// zodiacPosition classifies unrounded longitude into Rashi, Nakshatra and Pada, adding rounded display degrees.
func zodiacPosition(value float64) ZodiacPosition {
	lon := normalizeDegrees(value)
	sign := int(lon / 30)
	degree := lon - float64(sign*30)
	nak := min(26, int(lon*27/360))
	pada := min(107, int(lon*108/360))%4 + 1
	// Truncate only display seconds; keep full precision for classification.
	seconds := int(math.Floor(degree * 3600))
	return ZodiacPosition{
		Longitude: lon, RashiNumber: sign + 1, Rashi: rashiNames[sign], Degree: degree,
		DMS:             fmt.Sprintf("%d°%02d′%02d″", seconds/3600, (seconds%3600)/60, seconds%60),
		NakshatraNumber: nak + 1, Nakshatra: nakshatraNames[nak], Pada: pada,
	}
}

// CalculateKundali returns a Lahiri sidereal D1 chart using Mean Rahu and
// whole-sign houses. birthTime must already represent the correct instant.
// Requires swisseph.go from the previous step (CGO flags and shared engineMu).
func CalculateKundali(epheDir string, birthTime time.Time, latitude, longitude float64) (Kundali, error) {
	var out Kundali
	if !finite(latitude) || math.Abs(latitude) >= 90 || !finite(longitude) || math.Abs(longitude) > 180 {
		return out, fmt.Errorf("invalid coordinates: latitude must be between -90 and 90, longitude between -180 and 180")
	}
	utc := birthTime.UTC()
	if !supportedCalculationInstant(utc) {
		return out, fmt.Errorf("date outside supported calculation range %d-%d", MinYear, MaxYear)
	}
	absDir, err := filepath.Abs(epheDir)
	if err != nil {
		return out, err
	}
	for _, name := range []string{"sepl_18.se1", "semo_18.se1"} {
		info, err := os.Stat(filepath.Join(absDir, name))
		if err != nil || info.IsDir() {
			return out, fmt.Errorf("missing %s in %s", name, absDir)
		}
	}

	// Same mutex as SunPosition. All library settings and calculations are
	// protected together; TLSOFF is set for this package in swisseph.go.
	engineMu.Lock()
	defer engineMu.Unlock()
	cPath := C.CString(absDir)
	defer C.free(unsafe.Pointer(cPath))
	C.swe_set_ephe_path(cPath)
	defer C.swe_close()
	C.swe_set_sid_mode(C.SE_SIDM_LAHIRI, 0, 0)

	var jd [2]C.double
	var message [256]C.char
	seconds := float64(utc.Second()) + float64(utc.Nanosecond())/1e9
	status := C.swe_utc_to_jd(C.int(utc.Year()), C.int(utc.Month()), C.int(utc.Day()),
		C.int(utc.Hour()), C.int(utc.Minute()), C.double(seconds), C.SE_GREG_CAL, &jd[0], &message[0])
	if status < 0 {
		return out, fmt.Errorf("time conversion: %s", C.GoString(&message[0]))
	}
	var version [256]C.char
	C.swe_version(&version[0])
	out = Kundali{
		BirthTime: birthTime.Format(time.RFC3339Nano), UTC: utc.Format(time.RFC3339Nano),
		Latitude: latitude, Longitude: longitude, JulianDayUT1: float64(jd[1]),
		SwissVersion: C.GoString(&version[0]),
		Settings: map[string]string{"zodiac": "Sidereal", "ayanamsa": "Lahiri",
			"rahu": "Mean Node", "houses": "Whole Sign (D1)",
			"coordinates": "Geocentric", "ephemeris": "Swiss data files"},
		Planets: make([]Graha, 0, 9), Houses: make([]RashiHouse, 0, 12),
	}
	flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SIDEREAL | C.SEFLG_SPEED)
	objects := []struct {
		name string
		id   C.int
	}{
		{"Sun", C.SE_SUN}, {"Moon", C.SE_MOON}, {"Mars", C.SE_MARS},
		{"Mercury", C.SE_MERCURY}, {"Jupiter", C.SE_JUPITER},
		{"Venus", C.SE_VENUS}, {"Saturn", C.SE_SATURN}, {"Rahu", C.SE_MEAN_NODE},
	}
	for _, body := range objects {
		var values [6]C.double
		message[0] = 0
		returned := C.swe_calc_ut(jd[1], body.id, flags, &values[0], &message[0])
		if returned < 0 {
			return Kundali{}, fmt.Errorf("%s: %s", body.name, C.GoString(&message[0]))
		}
		if !swissFlagsMatch(int32(returned), int32(flags)) {
			return Kundali{}, fmt.Errorf("%s: requested Swiss sidereal result not returned (flags=%d): %s", body.name, int32(returned), C.GoString(&message[0]))
		}
		lon, speed := float64(values[0]), float64(values[3])
		if !finite(lon) || !finite(speed) {
			return Kundali{}, fmt.Errorf("%s: invalid numerical result", body.name)
		}
		out.Planets = append(out.Planets, Graha{Name: body.name, ZodiacPosition: zodiacPosition(lon),
			Speed: speed, Retrograde: speed < 0})
	}
	rahu := out.Planets[7]
	out.Planets = append(out.Planets, Graha{Name: "Ketu", ZodiacPosition: zodiacPosition(rahu.Longitude + 180),
		Speed: rahu.Speed, Retrograde: rahu.Retrograde})

	// C cusp array is 1-based; ascmc[0] is exact lagna, not cusps[1].
	var cusps [13]C.double
	var ascmc [10]C.double
	status = C.swe_houses_ex(jd[1], C.SEFLG_SIDEREAL,
		C.double(latitude), C.double(longitude), C.int('W'), &cusps[0], &ascmc[0])
	if status < 0 || !finite(float64(ascmc[0])) {
		return Kundali{}, fmt.Errorf("unable to calculate lagna and houses for this location/time")
	}
	out.Lagna = zodiacPosition(float64(ascmc[0]))
	for i := range out.Planets {
		out.Planets[i].House = (out.Planets[i].RashiNumber-out.Lagna.RashiNumber+12)%12 + 1
	}
	for i := 1; i <= 12; i++ {
		if !finite(float64(cusps[i])) {
			return Kundali{}, fmt.Errorf("invalid house cusp %d", i)
		}
		p := zodiacPosition(float64(cusps[i]))
		out.Houses = append(out.Houses, RashiHouse{Number: i, RashiNumber: p.RashiNumber,
			Rashi: p.Rashi, CuspLongitude: p.Longitude})
	}
	return out, nil
}

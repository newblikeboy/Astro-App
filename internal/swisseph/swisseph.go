package swisseph

/*
#cgo CFLAGS: -O2 -DTLSOFF
#cgo LDFLAGS: -lm
#cgo linux LDFLAGS: -ldl
#include <stdlib.h>
#include "swephexp.h"
*/
import "C"

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"
)

// TLSOFF makes the C library's state process-wide. Every call to it must use
// this same mutex, including future planet/house functions.
var engineMu sync.Mutex

// swissFlagsMatch accepts only the requested flags with Swiss as the sole
// returned ephemeris; errors, Moshier/JPL fallback and mixed engine flags fail.
func swissFlagsMatch(returned, requested int32) bool {
	const engineMask = int32(C.SEFLG_JPLEPH | C.SEFLG_SWIEPH | C.SEFLG_MOSEPH)
	return returned >= 0 && returned&engineMask == int32(C.SEFLG_SWIEPH) && returned&requested == requested
}

type Position struct {
	Planet       string  `json:"planet"`
	UTC          string  `json:"utc"`
	SwissVersion string  `json:"swiss_version"`
	Zodiac       string  `json:"zodiac"`
	Ayanamsa     string  `json:"ayanamsa"`
	Ephemeris    string  `json:"ephemeris"`
	Longitude    float64 `json:"longitude"`
	RashiNumber  int     `json:"rashi_number"`
	Rashi        string  `json:"rashi"`
	Degree       float64 `json:"degree_in_rashi"`
	Speed        float64 `json:"longitude_speed_deg_per_day"`
	Retrograde   bool    `json:"retrograde"`
}

// SunPosition calculates the geocentric Lahiri Sun using Swiss files and rejects missing data or invalid instants.
func SunPosition(epheDir string, birthTime time.Time) (Position, error) {
	var out Position
	absDir, err := filepath.Abs(epheDir)
	if err != nil {
		return out, err
	}
	info, err := os.Stat(filepath.Join(absDir, "sepl_18.se1"))
	if err != nil || info.IsDir() {
		return out, fmt.Errorf("planet data file missing in %s", absDir)
	}
	utc := birthTime.UTC()
	if !supportedCalculationInstant(utc) {
		return out, fmt.Errorf("date outside supported calculation range %d-%d", MinYear, MaxYear)
	}

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
	status := C.swe_utc_to_jd(
		C.int(utc.Year()), C.int(utc.Month()), C.int(utc.Day()),
		C.int(utc.Hour()), C.int(utc.Minute()), C.double(seconds),
		C.SE_GREG_CAL, &jd[0], &message[0],
	)
	if status < 0 {
		return out, fmt.Errorf("time conversion: %s", C.GoString(&message[0]))
	}

	// jd[0] is TT; jd[1] is UT1, required by swe_calc_ut.
	var values [6]C.double
	flags := C.int32(C.SEFLG_SWIEPH | C.SEFLG_SIDEREAL | C.SEFLG_SPEED)
	returned := C.swe_calc_ut(jd[1], C.SE_SUN, flags, &values[0], &message[0])
	if returned < 0 {
		return out, fmt.Errorf("Sun calculation: %s", C.GoString(&message[0]))
	}
	if !swissFlagsMatch(int32(returned), int32(flags)) {
		return out, fmt.Errorf("requested Swiss result was not returned; fallback rejected (flags=%d): %s", int32(returned), C.GoString(&message[0]))
	}
	lon := float64(values[0])
	speed := float64(values[3])
	if math.IsNaN(lon) || math.IsInf(lon, 0) || math.IsNaN(speed) || math.IsInf(speed, 0) {
		return out, fmt.Errorf("invalid numerical result")
	}
	lon = math.Mod(math.Mod(lon, 360)+360, 360)
	sign := int(lon / 30)
	rashis := [...]string{"Mesh", "Vrishabh", "Mithun", "Kark", "Simha", "Kanya",
		"Tula", "Vrishchik", "Dhanu", "Makar", "Kumbh", "Meen"}
	var version [256]C.char
	C.swe_version(&version[0])
	return Position{
		Planet: "Sun", UTC: utc.Format(time.RFC3339Nano),
		SwissVersion: C.GoString(&version[0]), Zodiac: "Sidereal",
		Ayanamsa: "Lahiri", Ephemeris: "Swiss data files",
		Longitude: lon, RashiNumber: sign + 1, Rashi: rashis[sign],
		Degree: lon - float64(sign*30), Speed: speed, Retrograde: speed < 0,
	}, nil
}

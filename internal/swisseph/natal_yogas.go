package swisseph

import (
	"fmt"
	"math"
)

// NatalYogaMethod identifies the exact nodal enclosure and whole-sign lord-relationship convention.
const NatalYogaMethod = "m4y-natal-yogas-longitude-kendra-trikona-v2"

// A numerical tolerance, not an astrological conjunction orb. Boundary cases
// remain reviewable rather than silently being called present or absent.
const nodeBoundaryDegrees = 1e-7

type KaalSarpPlanet struct {
	Planet    string  `json:"planet"`
	Longitude float64 `json:"longitude"`
	FromRahu  float64 `json:"from_rahu_degrees"`
	Side      string  `json:"side"`
}

type KaalSarpAssessment struct {
	Status        string           `json:"status"`
	Type          string           `json:"type,omitempty"`
	Arc           string           `json:"arc"`
	RahuLongitude float64          `json:"rahu_longitude"`
	KetuLongitude float64          `json:"ketu_longitude"`
	RahuHouse     int              `json:"rahu_house"`
	KetuHouse     int              `json:"ketu_house"`
	Planets       []KaalSarpPlanet `json:"planets"`
}

type RajyogPair struct {
	KendraHouse  int      `json:"kendra_house"`
	TrikonaHouse int      `json:"trikona_house"`
	KendraLord   string   `json:"kendra_lord"`
	TrikonaLord  string   `json:"trikona_lord"`
	KendraSign   int      `json:"kendra_lord_sign"`
	TrikonaSign  int      `json:"trikona_lord_sign"`
	Links        []string `json:"links"`
}

type YogakarakaLord struct {
	Planet       string `json:"planet"`
	KendraHouse  int    `json:"kendra_house"`
	TrikonaHouse int    `json:"trikona_house"`
}

type RajyogAssessment struct {
	Matches     bool             `json:"matches"`
	Pairs       []RajyogPair     `json:"pairs"`
	Yogakarakas []YogakarakaLord `json:"yogakarakas"`
}

type NatalYogaChecks struct {
	Method   string             `json:"method"`
	KaalSarp KaalSarpAssessment `json:"kaal_sarp"`
	Rajyog   RajyogAssessment   `json:"rajyog"`
}

// CalculateNatalYogaChecks validates a full D1 chart and records geometric yoga
// evidence using unrounded longitudes; supplied sign/house metadata is not trusted.
// It does not calculate final strength, cancellations or predicted outcomes.
func CalculateNatalYogaChecks(chart Kundali) (NatalYogaChecks, error) {
	if !finite(chart.Lagna.Longitude) || chart.Lagna.Longitude < 0 || chart.Lagna.Longitude >= 360 {
		return NatalYogaChecks{}, fmt.Errorf("invalid natal ascendant")
	}
	positions := map[string]Graha{}
	for _, p := range chart.Planets {
		if _, exists := positions[p.Name]; exists || !finite(p.Longitude) || p.Longitude < 0 || p.Longitude >= 360 {
			return NatalYogaChecks{}, fmt.Errorf("invalid or duplicate natal planet")
		}
		p.ZodiacPosition = zodiacPosition(p.Longitude)
		positions[p.Name] = p
	}
	for _, name := range []string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn", "Rahu", "Ketu"} {
		if _, ok := positions[name]; !ok {
			return NatalYogaChecks{}, fmt.Errorf("missing natal %s", name)
		}
	}
	if math.Abs(normalizeDegrees(positions["Ketu"].Longitude-positions["Rahu"].Longitude)-180) > nodeBoundaryDegrees {
		return NatalYogaChecks{}, fmt.Errorf("natal lunar nodes must be opposite")
	}
	lagna := zodiacPosition(chart.Lagna.Longitude).RashiNumber
	return NatalYogaChecks{Method: NatalYogaMethod, KaalSarp: kaalSarpAssessment(positions, lagna), Rajyog: rajyogAssessment(positions, lagna)}, nil
}

// kaalSarpAssessment tests both open nodal semicircles. A planet on either node
// makes an otherwise enclosed chart a boundary case; split planets always break enclosure.
func kaalSarpAssessment(positions map[string]Graha, lagna int) KaalSarpAssessment {
	rahu, ketu := positions["Rahu"], positions["Ketu"]
	out := KaalSarpAssessment{Status: "absent", Arc: "none", RahuLongitude: rahu.Longitude, KetuLongitude: ketu.Longitude, RahuHouse: relativeSign(rahu.RashiNumber, lagna), KetuHouse: relativeSign(ketu.RashiNumber, lagna), Planets: []KaalSarpPlanet{}}
	first, second, boundary := 0, 0, 0
	for _, name := range matchingLords {
		p := positions[name]
		delta := normalizeDegrees(p.Longitude - rahu.Longitude)
		side := "node_boundary"
		if delta <= nodeBoundaryDegrees || 360-delta <= nodeBoundaryDegrees || math.Abs(delta-180) <= nodeBoundaryDegrees {
			boundary++
		} else if delta < 180 {
			side = "rahu_to_ketu"
			first++
		} else {
			side = "ketu_to_rahu"
			second++
		}
		out.Planets = append(out.Planets, KaalSarpPlanet{Planet: name, Longitude: p.Longitude, FromRahu: delta, Side: side})
	}
	if first > 0 && second > 0 {
		return out
	}
	if boundary > 0 {
		out.Status = "boundary"
	} else {
		out.Status = "present"
		out.Type = []string{"ananta", "kulik", "vasuki", "shankhpal", "padma", "mahapadma", "takshak", "karkotak", "shankhchud", "ghatak", "vishdhar", "sheshnag"}[out.RahuHouse-1]
	}
	if first > 0 {
		out.Arc = "rahu_to_ketu"
	} else if second > 0 {
		out.Arc = "ketu_to_rahu"
	} else {
		out.Arc = "node_boundary"
	}
	return out
}

// fullGrahaAspect applies the same whole-sign classical graha drishti as the
// planet analysis: seventh for all seven planets, plus Mars/Jupiter/Saturn specials.
func fullGrahaAspect(planet string, from, to int) bool {
	distance := relativeSign(to, from)
	return distance == 7 || planet == "Mars" && (distance == 4 || distance == 8) || planet == "Jupiter" && (distance == 5 || distance == 9) || planet == "Saturn" && (distance == 3 || distance == 10)
}

// rajyogAssessment checks distinct Kendra/Trikona lords for conjunction,
// mutual full aspect and sign exchange. Dual lordship is separate evidence;
// owning Lagna alone never creates a Yogakaraka or a self-pair Rajyog.
func rajyogAssessment(positions map[string]Graha, lagna int) RajyogAssessment {
	out := RajyogAssessment{Pairs: []RajyogPair{}, Yogakarakas: []YogakarakaLord{}}
	for _, kendra := range []int{1, 4, 7, 10} {
		kendraSign := (lagna+kendra-2)%12 + 1
		kendraLord := matchingLords[matchingSignLords[kendraSign-1]]
		for _, trikona := range []int{1, 5, 9} {
			if kendra == trikona {
				continue
			}
			trikonaSign := (lagna+trikona-2)%12 + 1
			trikonaLord := matchingLords[matchingSignLords[trikonaSign-1]]
			if kendraLord == trikonaLord {
				if kendra != 1 && trikona != 1 {
					out.Yogakarakas = append(out.Yogakarakas, YogakarakaLord{Planet: kendraLord, KendraHouse: kendra, TrikonaHouse: trikona})
				}
				continue
			}
			p, q := positions[kendraLord], positions[trikonaLord]
			pair := RajyogPair{KendraHouse: kendra, TrikonaHouse: trikona, KendraLord: kendraLord, TrikonaLord: trikonaLord, KendraSign: p.RashiNumber, TrikonaSign: q.RashiNumber, Links: []string{}}
			if p.RashiNumber == q.RashiNumber {
				pair.Links = append(pair.Links, "conjunction")
			}
			if fullGrahaAspect(kendraLord, p.RashiNumber, q.RashiNumber) && fullGrahaAspect(trikonaLord, q.RashiNumber, p.RashiNumber) {
				pair.Links = append(pair.Links, "mutual_aspect")
			}
			// Exchange requires each planet to occupy any sign owned by the other.
			if matchingLords[matchingSignLords[p.RashiNumber-1]] == trikonaLord && matchingLords[matchingSignLords[q.RashiNumber-1]] == kendraLord {
				pair.Links = append(pair.Links, "sign_exchange")
			}
			out.Matches = out.Matches || len(pair.Links) > 0
			out.Pairs = append(out.Pairs, pair)
		}
	}
	return out
}

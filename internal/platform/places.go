package platform

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// GeoNames, CC BY 4.0. See data/README.md for attribution and rebuild steps.
//
//go:embed data/india.tsv.gz
var indiaPlaces []byte

type indexedPlace struct {
	place  Place
	names  string // normalized alternative names, separated and bounded by |
	search string
}

type placeCatalog struct {
	entries []indexedPlace
	byID    map[string]int
}

var catalog = loadPlaceCatalog()

// normalizePlace normalizes accents, case and separators for birthplace catalogue searching.
func normalizePlace(s string) string {
	s = strings.Map(func(r rune) rune {
		// Fold Latin accents without stripping vowel signs from Indian scripts.
		if r >= 0x0300 && r <= 0x036f {
			return -1
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) || unicode.IsMark(r) {
			return unicode.ToLower(r)
		}
		return ' '
	}, norm.NFD.String(s))
	return strings.Join(strings.Fields(s), " ")
}

// loadPlaceCatalog loads the bundled place records and builds searchable names and ID lookups once.
func loadPlaceCatalog() placeCatalog {
	c := placeCatalog{entries: make([]indexedPlace, 0, 550000), byID: make(map[string]int, 550000)}
	// add adds a catalogue record and its searchable name variants to the place indexes.
	add := func(p Place, aliases, region string) {
		names := []string{normalizePlace(strings.Split(p.Name, ",")[0]), normalizePlace(strings.Split(p.Hindi, ",")[0])}
		for _, name := range strings.Split(aliases, "|") {
			if name = normalizePlace(name); name != "" {
				names = append(names, name)
			}
		}
		joined := "|" + strings.Join(names, "|") + "|"
		c.byID[p.ID] = len(c.entries)
		c.entries = append(c.entries, indexedPlace{p, joined, joined + " " + normalizePlace(region)})
	}
	// Preserve existing IDs and coordinates so previous calculations remain stable.
	for _, p := range Places {
		add(p, "", p.Name)
	}
	legacy := map[string]string{
		"1253405": "varanasi", "1259229": "pune", "1260086": "patna", "1261481": "delhi",
		"1264527": "chennai", "1264733": "lucknow", "1269515": "jaipur", "1269743": "indore",
		"1269843": "hyderabad", "1275004": "kolkata", "1275339": "mumbai", "1275841": "bhopal",
		"1277333": "bengaluru", "1278994": "prayagraj",
	}
	z, err := gzip.NewReader(bytes.NewReader(indiaPlaces))
	if err != nil {
		panic(fmt.Errorf("open bundled India locations: %w", err))
	}
	defer z.Close()
	r := csv.NewReader(z)
	r.Comma, r.FieldsPerRecord, r.ReuseRecord = '\t', 9, true
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Errorf("read bundled India locations: %w", err))
		}
		if old, ok := legacy[row[0]]; ok {
			i := c.byID[old]
			for _, alias := range strings.Split(row[8], "|") {
				c.entries[i].names += normalizePlace(alias) + "|"
			}
			c.entries[i].search = c.entries[i].names + " " + normalizePlace(c.entries[i].place.Name)
			continue
		}
		lat, e1 := strconv.ParseFloat(row[5], 64)
		lon, e2 := strconv.ParseFloat(row[6], 64)
		if e1 != nil || e2 != nil || lat < -66 || lat > 66 || lon < -180 || lon > 180 {
			panic("invalid bundled location coordinates")
		}
		parts := []string{row[1]}
		if row[4] != "" && normalizePlace(row[4]) != normalizePlace(row[1]) {
			parts = append(parts, row[4])
		}
		parts = append(parts, row[3], "India")
		name := strings.Join(parts, ", ")
		hindi := strings.Clone(row[2])
		if hindi == "" {
			hindi = strings.Clone(row[1])
		}
		add(Place{"geonames-" + row[0], name, hindi, lat, lon, "Asia/Kolkata"}, row[8], row[4]+" "+row[3]+" India")
	}
	return c
}

// findPlace resolves a trusted birthplace ID to the same coordinates and timezone used for calculations.
func findPlace(id string) (Place, bool) {
	i, ok := catalog.byID[id]
	if !ok {
		return Place{}, false
	}
	return catalog.entries[i].place, true
}

// Search is bounded; the full gazetteer never travels to the browser. Source
// population order breaks ties, with exact names ahead of partial matches.
func searchPlaces(query string) []Place {
	q := normalizePlace(query)
	if q == "" {
		return append([]Place{}, Places...)
	}
	terms := strings.Fields(q)
	var ranked [4][]Place
	for _, entry := range catalog.entries {
		matches := true
		for _, term := range terms {
			if !strings.Contains(entry.search, term) {
				matches = false
				break
			}
		}
		if !matches {
			continue
		}
		score := 3
		if strings.Contains(entry.names, "|"+q+"|") {
			score = 0
		} else if strings.Contains(entry.names, "|"+terms[0]) {
			score = 1
		} else if strings.Contains(entry.names, terms[0]) {
			score = 2
		}
		if len(ranked[score]) < 25 {
			ranked[score] = append(ranked[score], entry.place)
		}
	}
	result := []Place{}
	for _, group := range ranked {
		for _, p := range group {
			if len(result) == 25 {
				return result
			}
			result = append(result, p)
		}
	}
	return result
}

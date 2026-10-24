package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"astroapp/backend/internal/swisseph"
)

// main runs a command-line Kundali calculation for inspecting native engine output.
func main() {
	epheDir := flag.String("ephe", "./ephe", "ephemeris folder")
	birthText := flag.String("birth", "1995-08-15T10:30:00+05:30", "birth instant in RFC3339 format with UTC offset")
	latitude := flag.Float64("lat", 28.6139, "latitude: north positive, south negative")
	longitude := flag.Float64("lon", 77.2090, "longitude: east positive, west negative")
	asJSON := flag.Bool("json", false, "print complete JSON instead of a readable summary")
	flag.Parse()
	birth, err := time.Parse(time.RFC3339, *birthText)
	if err != nil {
		fail(fmt.Errorf("use a birth instant such as 1995-08-15T10:30:00+05:30: %w", err))
	}
	chart, err := swisseph.CalculateKundali(*epheDir, birth, *latitude, *longitude)
	if err != nil {
		fail(err)
	}
	if *asJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(chart); err != nil {
			fail(err)
		}
		return
	}
	fmt.Println("Birth:", chart.BirthTime, "| UTC:", chart.UTC)
	fmt.Printf("Location: %.4f, %.4f\n", chart.Latitude, chart.Longitude)
	fmt.Println("Lahiri sidereal | Mean Rahu | Whole-sign D1 | Swiss", chart.SwissVersion)
	fmt.Printf("Lagna: %s %s\n\n", chart.Lagna.Rashi, chart.Lagna.DMS)
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "Planet\tRashi\tDegree\tNakshatra\tPada\tHouse\tRetrograde")
	for _, p := range chart.Planets {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%t\n",
			p.Name, p.Rashi, p.DMS, p.Nakshatra, p.Pada, p.House, p.Retrograde)
	}
	if err := w.Flush(); err != nil {
		fail(err)
	}
}

// fail prints a diagnostic and exits the calculation-check command with a failure status.
func fail(err error) {
	fmt.Fprintln(os.Stderr, "Error:", err)
	os.Exit(1)
}

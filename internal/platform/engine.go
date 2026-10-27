package platform

import (
	"astroapp/backend/internal/numerology"
	"astroapp/backend/internal/swisseph"
	"fmt"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

var Places = []Place{
	{"delhi", "New Delhi, Delhi, India", "नई दिल्ली, भारत", 28.6139, 77.2090, "Asia/Kolkata"},
	{"varanasi", "Varanasi, Uttar Pradesh, India", "वाराणसी, उत्तर प्रदेश", 25.3176, 82.9739, "Asia/Kolkata"},
	{"lucknow", "Lucknow, Uttar Pradesh, India", "लखनऊ, उत्तर प्रदेश", 26.8467, 80.9462, "Asia/Kolkata"},
	{"prayagraj", "Prayagraj, Uttar Pradesh, India", "प्रयागराज, उत्तर प्रदेश", 25.4358, 81.8463, "Asia/Kolkata"},
	{"mumbai", "Mumbai, Maharashtra, India", "मुंबई, महाराष्ट्र", 19.0760, 72.8777, "Asia/Kolkata"},
	{"jaipur", "Jaipur, Rajasthan, India", "जयपुर, राजस्थान", 26.9124, 75.7873, "Asia/Kolkata"},
	{"patna", "Patna, Bihar, India", "पटना, बिहार", 25.5941, 85.1376, "Asia/Kolkata"},
	{"kolkata", "Kolkata, West Bengal, India", "कोलकाता, पश्चिम बंगाल", 22.5726, 88.3639, "Asia/Kolkata"},
	{"bengaluru", "Bengaluru, Karnataka, India", "बेंगलुरु, कर्नाटक", 12.9716, 77.5946, "Asia/Kolkata"},
	{"bhopal", "Bhopal, Madhya Pradesh, India", "भोपाल, मध्य प्रदेश", 23.2599, 77.4126, "Asia/Kolkata"},
	{"indore", "Indore, Madhya Pradesh, India", "इंदौर, मध्य प्रदेश", 22.7196, 75.8577, "Asia/Kolkata"},
	{"pune", "Pune, Maharashtra, India", "पुणे, महाराष्ट्र", 18.5204, 73.8567, "Asia/Kolkata"},
	{"chennai", "Chennai, Tamil Nadu, India", "चेन्नई, तमिलनाडु", 13.0827, 80.2707, "Asia/Kolkata"},
	{"hyderabad", "Hyderabad, Telangana, India", "हैदराबाद, तेलंगाना", 17.3850, 78.4867, "Asia/Kolkata"},
	{"kathmandu", "Kathmandu, Nepal", "काठमांडू, नेपाल", 27.7172, 85.3240, "Asia/Kathmandu"},
	{"london", "London, United Kingdom", "लंदन, यूनाइटेड किंगडम", 51.5074, -0.1278, "Europe/London"},
	{"new-york", "New York, United States", "न्यूयॉर्क, अमेरिका", 40.7128, -74.0060, "America/New_York"},
	{"sydney", "Sydney, Australia", "सिडनी, ऑस्ट्रेलिया", -33.8688, 151.2093, "Australia/Sydney"},
	{"dubai", "Dubai, United Arab Emirates", "दुबई, संयुक्त अरब अमीरात", 25.2048, 55.2708, "Asia/Dubai"},
	{"toronto", "Toronto, Canada", "टोरंटो, कनाडा", 43.6532, -79.3832, "America/Toronto"},
}

// ResolveWall rejects nonexistent times and requires an explicit offset for a DST fold.
func ResolveWall(date, clock, zone string, offset *int) (time.Time, error) {
	// Validate before multiplying: an untrusted integer could otherwise wrap
	// when converted from minutes to seconds and match a genuine zone offset.
	if offset != nil && (*offset < -24*60 || *offset > 24*60) {
		return time.Time{}, fmt.Errorf("invalid UTC offset in minutes")
	}
	clockLayout := "15:04"
	if len(clock) == 8 {
		clockLayout = "15:04:05"
	}
	wall, e := time.Parse("2006-01-02 "+clockLayout, date+" "+clock)
	if e != nil || wall.Format("2006-01-02 "+clockLayout) != date+" "+clock {
		return time.Time{}, fmt.Errorf("जन्म तारीख और समय सही प्रारूप में दें")
	}
	loc, e := time.LoadLocation(zone)
	if e != nil {
		return time.Time{}, fmt.Errorf("unsupported IANA timezone")
	}
	offsets := map[int]bool{}
	for h := -36; h <= 36; h++ {
		_, o := wall.Add(time.Duration(h) * time.Hour).In(loc).Zone()
		offsets[o] = true
	}
	candidates := []time.Time{}
	for o := range offsets {
		t := wall.Add(-time.Duration(o) * time.Second)
		if t.In(loc).Format("2006-01-02 "+clockLayout) == date+" "+clock {
			if offset == nil || *offset*60 == o {
				candidates = append(candidates, t)
			}
		}
	}
	if len(candidates) == 0 {
		return time.Time{}, fmt.Errorf("यह स्थानीय समय अस्तित्व में नहीं है या UTC offset गलत है; daylight saving जाँचें")
	}
	if len(candidates) > 1 {
		return time.Time{}, fmt.Errorf("इस समय के दो UTC रूप हैं (daylight saving); सही UTC offset मिनटों में चुनें")
	}
	return candidates[0].In(loc), nil
}

// calculate builds a complete Kundli report, including the expanded birth calculations.
func calculate(c Config, b Birth, rules map[string]Rule, now time.Time) (*Report, error) {
	return calculateReport(c, b, rules, now, true)
}

// calculateCore builds the core birth chart and readings without the expensive expanded calculations.
func calculateCore(c Config, b Birth, rules map[string]Rule, now time.Time) (*Report, error) {
	return calculateReport(c, b, rules, now, false)
}

// calculateReport validates birthplace-local birth inputs, calculates native charts and adds approved readings.
func calculateReport(c Config, b Birth, rules map[string]Rule, now time.Time, expanded bool) (*Report, error) {
	b.Name = strings.TrimSpace(b.Name)
	if len([]rune(b.Name)) < 2 || len([]rune(b.Name)) > 100 {
		return nil, fmt.Errorf("कृपया 2 से 100 अक्षरों में नाम लिखें")
	}
	d, e := time.Parse("2006-01-02", b.Date)
	if e != nil || d.Year() < 1940 || d.Year() > 2099 {
		return nil, fmt.Errorf("जन्म तारीख 1940 से आज के बीच चुनें")
	}
	p, ok := findPlace(b.PlaceID)
	if !ok {
		return nil, fmt.Errorf("खोज सूची से समर्थित जन्म स्थान चुनें")
	}
	loc, e := time.LoadLocation(p.Zone)
	if e != nil {
		return nil, fmt.Errorf("unsupported IANA timezone")
	}
	// Birth dates belong to the birthplace's civil calendar. Comparing an
	// artificial UTC midnight to now rejects today's births east of UTC and
	// admits tomorrow's dates west of UTC around local midnight.
	if b.Date > now.In(loc).Format("2006-01-02") {
		return nil, fmt.Errorf("birth date must not be in the future at the selected birthplace")
	}
	r := &Report{ID: id("r"), Birth: b, Place: p, Created: now, Engine: "m4y-kundli-v4", Rules: []Rule{}, Demo: !c.Production, Limitations: []string{"खगोलीय गणना और ज्योतिषीय व्याख्या अलग हैं। व्याख्या सांस्कृतिक चिंतन है, किसी परिणाम की गारंटी नहीं।"}}
	// An older cached form may still send a separate spelling. New reports derive Naamank only from the main name.
	r.Birth.NumerologyName = ""
	numbers, err := numerology.CalculateFull(b.Date, b.Name)
	if err != nil {
		return nil, err
	}
	r.Numerology = &numbers
	if b.Unknown {
		r.Birth.Time = ""
		r.Limitations = append(r.Limitations, "जन्म समय अज्ञात: लग्न, भाव, ग्रहों की निश्चित स्थिति, नक्षत्र और दशा विश्वसनीय रूप से नहीं दिए जा सकते। कोई काल्पनिक समय नहीं चुना गया है।")
		return r, nil
	}
	t, e := ResolveWall(b.Date, b.Time, p.Zone, b.Offset)
	if e != nil {
		return nil, e
	}
	if t.After(now) {
		return nil, fmt.Errorf("birth time must not be in the future")
	}
	// Public births begin on the birthplace's 1940-01-01. Positive offsets
	// can put that civil midnight on the preceding UTC date; the native
	// engine includes that one-day buffer rather than rejecting those births.
	if t.UTC().Before(time.Date(1939, time.December, 31, 0, 0, 0, 0, time.UTC)) || t.UTC().Year() > 2099 {
		return nil, fmt.Errorf("इस जन्म का समय समर्थित खगोलीय सीमा के बाहर है")
	}
	chart, e := swisseph.CalculateKundali(c.EpheDir, t, p.Lat, p.Lon)
	if e != nil {
		return nil, fmt.Errorf("गणना अभी उपलब्ध नहीं है; ephemeris data जाँचें")
	}
	chart.Settings["timezone"] = p.Zone
	chart.Settings["time_database"] = "Go embedded IANA tzdata"
	chart.Settings["engine"] = r.Engine
	chart.Settings["navamsa"] = "Parashari D9, whole-sign houses"
	chart.Settings["vimshottari"] = "Moon nakshatra; 365.25-day Julian year; start-inclusive/end-exclusive"
	chart.Settings["transits"] = "Report-creation snapshot; houses from natal Lagna / Moon"
	r.Chart = &chart
	navamsa := swisseph.CalculateNavamsa(chart)
	r.Navamsa = &navamsa
	if expanded {
		details, err := expandKundli(c, chart, t, now, p.Zone)
		if err != nil {
			return nil, fmt.Errorf("complete birth calculations are unavailable: %w", err)
		}
		r.Details = details
	}
	for _, planet := range chart.Planets {
		if planet.Name == "Moon" {
			dasha := swisseph.CalculateVimshottari(planet.Longitude, t, now)
			r.Dasha = &dasha
		}
	}
	r.Checks = placementChecks(chart)
	// A temporary transit failure must not discard a valid natal report.
	if current, err := swisseph.CalculateKundali(c.EpheDir, now, p.Lat, p.Lon); err == nil {
		snapshot := transitSnapshot(chart, current, now)
		r.Transits = &snapshot
	}
	r.Rules = matchRules(r, rules, !c.Production)
	if len(r.Rules) == 0 {
		r.Limitations = append(r.Limitations, "इस कुंडली के लिए गुरुजी की स्वीकृत व्याख्या अभी उपलब्ध नहीं है। ग्रह-सारणी वास्तविक खगोलीय गणना है।")
	}
	return r, nil
}

// matchRules selects the latest approved interpretation versions and resolves priority/conflict groups.
func matchRules(r *Report, rules map[string]Rule, allowDemo bool) []Rule {
	out := []Rule{}
	latest := map[string]Rule{}
	for _, v := range rules {
		if v.Demo && !allowDemo {
			continue
		}
		if !v.Demo && v.Status != "approved" && v.Status != "retired" {
			continue
		}
		if previous, ok := latest[v.ID]; !ok || v.Version > previous.Version {
			latest[v.ID] = v
		}
	}
	for _, v := range latest {
		if v.Demo {
			if !allowDemo {
				continue
			}
		} else if v.Status != "approved" || v.Approver == "" || v.ApprovedAt == nil {
			continue
		}
		if r.Chart == nil {
			continue
		}
		match := v.Planet == "" && v.Sign == 0 && v.House == 0
		for _, p := range r.Chart.Planets {
			if p.Name == v.Planet && (v.Sign == 0 || p.RashiNumber == v.Sign) && (v.House == 0 || p.House == v.House) {
				match = true
			}
		}
		match = match && (v.LagnaSign == 0 || r.Chart.Lagna.RashiNumber == v.LagnaSign)
		if v.Nakshatra != 0 {
			moonMatches := false
			for _, p := range r.Chart.Planets {
				if p.Name == "Moon" && p.NakshatraNumber == v.Nakshatra {
					moonMatches = true
				}
			}
			match = match && moonMatches
		}
		if v.DashaLord != "" {
			match = match && r.Dasha != nil && r.Dasha.CurrentLord == v.DashaLord
		}
		if match {
			out = append(out, v)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Priority == out[j].Priority {
			if out[i].ID == out[j].ID {
				return out[i].Version > out[j].Version
			}
			return out[i].ID < out[j].ID
		}
		return out[i].Priority > out[j].Priority
	})
	result := []Rule{}
	groups := map[string]bool{}
	ids := map[string]bool{}
	for _, v := range out {
		if ids[v.ID] || (v.Conflict != "" && groups[v.Conflict]) {
			continue
		}
		result = append(result, v)
		ids[v.ID] = true
		if v.Conflict != "" {
			groups[v.Conflict] = true
		}
	}
	return result
}

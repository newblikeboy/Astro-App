package swisseph

import "math"

type PlanetCondition struct {
	Name          string   `json:"name"`
	SignLord      string   `json:"sign_lord"`
	Dignity       string   `json:"dignity"`
	Baladi        string   `json:"baladi"`
	SunSeparation float64  `json:"sun_separation"`
	Combust       *bool    `json:"combust"`
	CombustLimit  *float64 `json:"combust_limit"`
}
type PlanetFriendship struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Natural   string `json:"natural"`
	Temporary string `json:"temporary"`
	Compound  string `json:"compound"`
}
type GrahaAspect struct {
	Planet    string   `json:"planet"`
	Distance  int      `json:"distance"`
	Sign      int      `json:"sign"`
	House     int      `json:"house"`
	Occupants []string `json:"occupants"`
}
type YogaPattern struct {
	Code      string   `json:"code"`
	Name      string   `json:"name"`
	Matches   bool     `json:"matches"`
	Planets   []string `json:"planets"`
	Condition string   `json:"condition"`
}
type PlanetAnalysis struct {
	VargaStrength []VargaStrength    `json:"varga_strength"`
	Method        string             `json:"method"`
	Conditions    []PlanetCondition  `json:"conditions"`
	Friendships   []PlanetFriendship `json:"friendships"`
	Aspects       []GrahaAspect      `json:"aspects"`
	Patterns      []YogaPattern      `json:"patterns"`
}

// Natural relationships are directional, in Sun/Moon/Mars/Mercury/Jupiter/
// Venus/Saturn order. Combined friendship is separate from marriage points.
var naturalRelations = [7][7]int{
	{0, 1, 1, 0, 1, -1, -1}, {1, 0, 0, 1, 0, 0, 0}, {1, 1, 0, -1, 1, 0, 0},
	{1, -1, 0, 0, 0, 1, 0}, {1, 1, 1, -1, 0, -1, 0}, {-1, -1, 0, 1, 0, 0, 1}, {-1, -1, -1, 1, 0, 1, 0},
}
var exaltationSigns = []int{1, 2, 10, 6, 4, 12, 7}

// relationName labels the declared planetary friendship category for report output.
func relationName(n int) string {
	if n > 0 {
		return "Friend"
	}
	if n < 0 {
		return "Enemy"
	}
	return "Neutral"
}

// CalculatePlanetAnalysis derives dignities, combustion, age states, aspects and friendship evidence from native positions.
func CalculatePlanetAnalysis(chart Kundali) PlanetAnalysis {
	out := PlanetAnalysis{Method: "m4y-parashari-conditions-v1; sign dignity; Baladi six-degree portions; full whole-sign graha drishti; no node aspects", Conditions: []PlanetCondition{}, Friendships: []PlanetFriendship{}, Aspects: []GrahaAspect{}, Patterns: []YogaPattern{}}
	byName := map[string]Graha{}
	for _, p := range chart.Planets {
		byName[p.Name] = p
	}
	sun := byName["Sun"]
	for _, p := range chart.Planets {
		owner := matchingSignLords[p.RashiNumber-1]
		c := PlanetCondition{Name: p.Name, SignLord: matchingLords[owner], Dignity: "Not assigned", Baladi: "Not assigned", SunSeparation: math.Abs(math.Remainder(p.Longitude-sun.Longitude, 360))}
		idx := -1
		for i, name := range matchingLords {
			if name == p.Name {
				idx = i
			}
		}
		if idx >= 0 {
			c.Dignity = relationName(naturalRelations[idx][owner]) + " sign"
			if idx == owner {
				c.Dignity = "Own sign"
			}
			if p.RashiNumber == exaltationSigns[idx] {
				c.Dignity = "Exaltation sign"
			}
			if p.RashiNumber == (exaltationSigns[idx]+5)%12+1 {
				c.Dignity = "Debilitation sign"
			}
			portion := min(4, int(p.Degree/6))
			if p.RashiNumber%2 == 0 {
				portion = 4 - portion
			}
			c.Baladi = []string{"Bala", "Kumara", "Yuva", "Vriddha", "Mrita"}[portion]
			if limit, applicable := combustionLimit(p.Name, p.Retrograde); applicable {
				combust := c.SunSeparation <= limit
				c.Combust = &combust
				c.CombustLimit = &limit
			}
			distances := []int{7}
			switch p.Name {
			case "Mars":
				distances = []int{4, 7, 8}
			case "Jupiter":
				distances = []int{5, 7, 9}
			case "Saturn":
				distances = []int{3, 7, 10}
			}
			for _, distance := range distances {
				sign := (p.RashiNumber+distance-2)%12 + 1
				occupants := []string{}
				for _, target := range chart.Planets {
					if target.RashiNumber == sign {
						occupants = append(occupants, target.Name)
					}
				}
				out.Aspects = append(out.Aspects, GrahaAspect{p.Name, distance, sign, (sign-chart.Lagna.RashiNumber+12)%12 + 1, occupants})
			}
		}
		out.Conditions = append(out.Conditions, c)
	}
	for i, from := range matchingLords {
		for j, to := range matchingLords {
			if i == j {
				continue
			}
			a, aok := byName[from]
			b, bok := byName[to]
			if !aok || !bok {
				continue
			}
			distance := (b.RashiNumber-a.RashiNumber+12)%12 + 1
			temporary := -1
			switch distance {
			case 2, 3, 4, 10, 11, 12:
				temporary = 1
			}
			compound := []string{"Great enemy", "Enemy", "Neutral", "Friend", "Great friend"}[naturalRelations[i][j]+temporary+2]
			out.Friendships = append(out.Friendships, PlanetFriendship{from, to, relationName(naturalRelations[i][j]), relationName(temporary), compound})
		}
	}
	for _, pattern := range []struct{ name, planet string }{{"Ruchaka", "Mars"}, {"Bhadra", "Mercury"}, {"Hamsa", "Jupiter"}, {"Malavya", "Venus"}, {"Shasha", "Saturn"}} {
		p, ok := byName[pattern.planet]
		qualifies := false
		for _, c := range out.Conditions {
			if c.Name == pattern.planet {
				qualifies = c.Dignity == "Own sign" || c.Dignity == "Exaltation sign"
			}
		}
		angular := p.House == 1 || p.House == 4 || p.House == 7 || p.House == 10
		out.Patterns = append(out.Patterns, YogaPattern{Code: "mahapurusha-" + pattern.planet, Name: pattern.name, Matches: ok && qualifies && angular, Planets: []string{pattern.planet}, Condition: "Planet in an own or exaltation sign and a whole-sign kendra from Lagna; combustion and other context shown separately"})
	}
	// Sun/Moon neighbour patterns exclude lunar nodes and their named luminary.
	for _, ref := range []string{"Sun", "Moon"} {
		before, after := []string{}, []string{}
		for _, name := range []string{"Mars", "Mercury", "Jupiter", "Venus", "Saturn"} {
			p, ok := byName[name]
			if !ok {
				continue
			}
			distance := (p.RashiNumber-byName[ref].RashiNumber+12)%12 + 1
			if distance == 12 {
				before = append(before, name)
			}
			if distance == 2 {
				after = append(after, name)
			}
		}
		names := []string{"Vasi", "Vesi", "Ubhayachari"}
		if ref == "Moon" {
			names = []string{"Anapha", "Sunapha", "Durudhara"}
		}
		for i, name := range names {
			matches := []bool{len(before) > 0, len(after) > 0, len(before) > 0 && len(after) > 0}[i]
			planets := append(append([]string{}, before...), after...)
			out.Patterns = append(out.Patterns, YogaPattern{Code: ref + "-neighbours-" + name, Name: name, Matches: matches, Planets: planets, Condition: "Classical planets other than the Sun/Moon in adjacent whole signs; presence pattern, not an outcome or strength assessment"})
		}
	}
	return out
}

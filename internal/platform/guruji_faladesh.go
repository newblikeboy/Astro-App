package platform

import (
	"astroapp/backend/internal/swisseph"
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed data/guruji-faladesh.json
var gurujiFaladeshJSON []byte

//go:embed data/guruji-lagna-roles.json
var gurujiLagnaRolesJSON []byte

//go:embed data/faladesh-reading-copy.json
var faladeshReadingCopyJSON []byte

type GurujiWording struct {
	Hindi   string `json:"hi"`
	English string `json:"en"`
}

type GurujiSource struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Archive  string `json:"archive"`
	SHA256   string `json:"sha256"`
}

type GurujiLocator struct {
	SourceID  string `json:"source_id"`
	PageStart int    `json:"page_start"`
	PageEnd   int    `json:"page_end"`
}

type GurujiPassage struct {
	HindiExcerpt string        `json:"hindi_excerpt"`
	Source       GurujiLocator `json:"source"`
}

type GurujiPlacement struct {
	Planet       string        `json:"planet"`
	House        int           `json:"house"`
	Title        GurujiWording `json:"title"`
	Summary      GurujiWording `json:"summary"`
	HindiExcerpt string        `json:"hindi_excerpt"`
	Source       GurujiLocator `json:"source"`
}

type GurujiRemedy struct {
	Planet string `json:"planet"`
	GurujiPassage
}

type GurujiHouseReference struct {
	House    int           `json:"house"`
	Meaning  GurujiWording `json:"meaning"`
	Positive GurujiPassage `json:"positive"`
	Negative GurujiPassage `json:"negative"`
}

type GurujiLagnaRoles struct {
	Lagna   int      `json:"lagna"`
	Karak   []string `json:"karak"`
	Marak   []string `json:"marak"`
	Neutral []string `json:"neutral"`
}

type GurujiReading struct {
	GurujiPlacement
	Narrative  *GurujiNarrative  `json:"narrative,omitempty"`
	Role       string            `json:"role"`
	Dignity    string            `json:"dignity"`
	SignNumber int               `json:"sign_number,omitempty"`
	SignStatus *GurujiSignStatus `json:"sign_status,omitempty"`
	RoleEffect *GurujiRoleEffect `json:"role_effect,omitempty"`
	Indication string            `json:"indication"`
	Remedy     GurujiRemedy      `json:"remedy"`
}

type GurujiNarrative struct {
	Version string        `json:"version"`
	Meaning GurujiWording `json:"meaning"`
	Support GurujiWording `json:"support"`
	Caution GurujiWording `json:"caution"`
}

type GurujiSignStatus struct {
	Primary     string `json:"primary"`
	Exalted     bool   `json:"exalted"`
	Debilitated bool   `json:"debilitated"`
	OwnSign     bool   `json:"own_sign"`
}

type GurujiRoleEffect struct {
	Direction   string        `json:"direction"`
	Explanation GurujiWording `json:"explanation"`
}

type GurujiFaladesh struct {
	ClassificationVersion string                 `json:"classification_version,omitempty"`
	HouseSystem           string                 `json:"house_system,omitempty"`
	Analysis              *FaladeshSynthesis     `json:"analysis,omitempty"`
	Version               string                 `json:"version"`
	Lagna                 int                    `json:"lagna"`
	Sources               []GurujiSource         `json:"sources"`
	Readings              []GurujiReading        `json:"readings"`
	HouseReferences       []GurujiHouseReference `json:"house_references"`
}

type gurujiCatalog struct {
	Version         string                 `json:"version"`
	Sources         []GurujiSource         `json:"sources"`
	Placements      []GurujiPlacement      `json:"placements"`
	Remedies        []GurujiRemedy         `json:"remedies"`
	HouseReferences []GurujiHouseReference `json:"house_references"`
}

var gurujiReferences = loadGurujiCatalog()
var gurujiRoles = loadGurujiRoles()
var gurujiNarratives = loadGurujiNarratives()

// loadGurujiNarratives validates source-based bilingual summaries used only in newly created snapshots.
func loadGurujiNarratives() map[string]GurujiNarrative {
	var data struct {
		Version string `json:"version"`
		Entries []struct {
			Planet string `json:"planet"`
			House  int    `json:"house"`
			GurujiNarrative
		} `json:"entries"`
	}
	if err := json.Unmarshal(faladeshReadingCopyJSON, &data); err != nil || len(data.Entries) != 108 {
		panic("invalid Faladesh reading summaries")
	}
	result := map[string]GurujiNarrative{}
	for _, entry := range data.Entries {
		value := entry.GurujiNarrative
		value.Version = data.Version
		key := entry.Planet + ":" + fmt.Sprint(entry.House)
		if _, exists := result[key]; exists || value.Meaning.Hindi == "" || value.Meaning.English == "" || value.Support.Hindi == "" || value.Support.English == "" || value.Caution.Hindi == "" || value.Caution.English == "" {
			panic("incomplete Faladesh reading summary")
		}
		result[key] = value
	}
	return result
}

// loadGurujiCatalog decodes the supplied source passages once; it never creates approval records.
func loadGurujiCatalog() gurujiCatalog {
	var catalog gurujiCatalog
	if err := json.Unmarshal(gurujiFaladeshJSON, &catalog); err != nil || len(catalog.Placements) != 108 || len(catalog.Remedies) != 9 || len(catalog.HouseReferences) != 12 {
		panic("invalid supplied Guruji reference catalogue")
	}
	return catalog
}

// loadGurujiRoles decodes the lagna-specific whiteboard transcription separately from universal planet strength.
func loadGurujiRoles() []GurujiLagnaRoles {
	var roles []GurujiLagnaRoles
	if err := json.Unmarshal(gurujiLagnaRolesJSON, &roles); err != nil || len(roles) != 12 {
		panic("invalid supplied Guruji lagna roles")
	}
	seenLagna := map[int]bool{}
	for _, row := range roles {
		if row.Lagna < 1 || row.Lagna > 12 || seenLagna[row.Lagna] {
			panic("invalid supplied Guruji lagna row")
		}
		seenLagna[row.Lagna] = true
		seenPlanet := map[string]bool{}
		for _, group := range [][]string{row.Karak, row.Marak, row.Neutral} {
			for _, name := range group {
				if !synthesisContainsPlanet([]string{"Sun", "Moon", "Mars", "Mercury", "Jupiter", "Venus", "Saturn"}, name) || seenPlanet[name] {
					panic("invalid or duplicate supplied Guruji role")
				}
				seenPlanet[name] = true
			}
		}
	}
	return roles
}

// gurujiRole uses only an explicitly listed whiteboard role; omitted planets remain unclassified.
// It retains the source's Marak label without inferring its intended technical scope or a mortality assessment.
func gurujiRole(lagna int, planet string) string {
	for _, row := range gurujiRoles {
		if row.Lagna != lagna {
			continue
		}
		for role, planets := range map[string][]string{"karak": row.Karak, "marak": row.Marak, "neutral": row.Neutral} {
			for _, name := range planets {
				if planet == name {
					return role
				}
			}
		}
	}
	return "unlisted"
}

// gurujiDignity applies the supplied sign-level uchcha/neecha table, including its two-sign node convention.
// This reference convention does not replace the native classical-planet dignity or imply total strength.
func gurujiDignity(planet string, sign int) string {
	exalted := map[string][]int{"Sun": {1}, "Moon": {2}, "Mars": {10}, "Mercury": {6}, "Jupiter": {4}, "Venus": {12}, "Saturn": {7}, "Rahu": {2, 3}, "Ketu": {8, 9}}
	debilitated := map[string][]int{"Sun": {7}, "Moon": {8}, "Mars": {4}, "Mercury": {12}, "Jupiter": {10}, "Venus": {6}, "Saturn": {1}, "Rahu": {8, 9}, "Ketu": {2, 3}}
	for _, s := range exalted[planet] {
		if sign == s {
			return "exalted"
		}
	}
	for _, s := range debilitated[planet] {
		if sign == s {
			return "debilitated"
		}
	}
	return "ordinary"
}

// suppliedGurujiFaladesh snapshots the nine D1 planet/house passages and separate role/strength evidence.
// No timing, aspects, combustion or automatic gemstone treatment is inferred from these source references.
func suppliedGurujiFaladesh(chart *swisseph.Kundali) *GurujiFaladesh {
	if chart == nil || chart.Lagna.RashiNumber < 1 || chart.Lagna.RashiNumber > 12 {
		return nil
	}
	result := &GurujiFaladesh{Version: gurujiReferences.Version, Lagna: chart.Lagna.RashiNumber, Sources: append([]GurujiSource(nil), gurujiReferences.Sources...), Readings: []GurujiReading{}, HouseReferences: append([]GurujiHouseReference(nil), gurujiReferences.HouseReferences...)}
	for _, p := range chart.Planets {
		if p.RashiNumber < 1 || p.RashiNumber > 12 {
			continue
		}
		for _, placement := range gurujiReferences.Placements {
			if p.Name != placement.Planet || p.House != placement.House {
				continue
			}
			role, dignity := gurujiRole(result.Lagna, p.Name), gurujiDignity(p.Name, p.RashiNumber)
			narrative := gurujiNarratives[p.Name+":"+fmt.Sprint(p.House)]
			status := gurujiSignStatus(p.Name, p.RashiNumber)
			effect := gurujiRoleEffect(role, status)
			indication := map[string]string{"support": "supportive", "attention": "challenging", "context": "neutral"}[effect.Direction]
			if role == "unlisted" {
				indication = "unclassified"
			}
			reading := GurujiReading{GurujiPlacement: placement, Narrative: &narrative, Role: role, Dignity: dignity, SignNumber: p.RashiNumber, SignStatus: &status, RoleEffect: &effect, Indication: indication}
			for _, remedy := range gurujiReferences.Remedies {
				if remedy.Planet == p.Name {
					reading.Remedy = remedy
					break
				}
			}
			result.Readings = append(result.Readings, reading)
			break
		}
	}
	result.ClassificationVersion = "m4y-board-role-sign-v1"
	result.HouseSystem = "D1 whole-sign"
	return result
}

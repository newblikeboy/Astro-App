package platform

import "time"

// Legacy snapshot schemas remain decodable; current output never renders their authored prose.
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

type FaladeshEvidence struct {
	ID        string        `json:"id"`
	Kind      string        `json:"kind"`
	Planet    string        `json:"planet"`
	House     int           `json:"house,omitempty"`
	Direction string        `json:"direction"`
	Text      GurujiWording `json:"text"`
}

type FaladeshPoint struct {
	Tone        string        `json:"tone"`
	Text        GurujiWording `json:"text"`
	EvidenceIDs []string      `json:"evidence_ids"`
}

type FaladeshWindow struct {
	Lord  string        `json:"lord"`
	Start time.Time     `json:"start"`
	End   time.Time     `json:"end"`
	Text  GurujiWording `json:"text"`
}

type FaladeshTopic struct {
	Reader         []ReaderParagraph     `json:"reader,omitempty"`
	Remedy         *RemedyRecommendation `json:"remedy_recommendation,omitempty"`
	Resolution     []ReadingResolution   `json:"resolution,omitempty"`
	Category       string                `json:"category"`
	Assessment     string                `json:"assessment"`
	Conclusion     GurujiWording         `json:"conclusion"`
	Present        GurujiWording         `json:"present"`
	PeriodRelevant bool                  `json:"period_relevant,omitempty"`
	Potential      GurujiWording         `json:"potential"`
	Findings       []FaladeshFinding     `json:"findings,omitempty"`
	MainPoints     []FaladeshPoint       `json:"main_points"`
	Evidence       []FaladeshEvidence    `json:"evidence"`
	RelatedPlanets []string              `json:"related_planets"`
	NextPeriod     *FaladeshWindow       `json:"next_period,omitempty"`
}

type FaladeshSynthesis struct {
	ReaderVersion string          `json:"reader_version,omitempty"`
	Version       string          `json:"version"`
	AsOf          time.Time       `json:"as_of"`
	Topics        []FaladeshTopic `json:"topics"`
	Coverage      []string        `json:"coverage"`
	Limits        []GurujiWording `json:"limits"`
}

type ReaderParagraph struct {
	ID          string        `json:"id"`
	Scope       string        `json:"scope"`
	Text        GurujiWording `json:"text"`
	EvidenceIDs []string      `json:"evidence_ids"`
}

type ReadingResolution struct {
	Factor      string   `json:"factor"`
	Direction   string   `json:"direction"`
	EvidenceIDs []string `json:"evidence_ids"`
	Rule        string   `json:"rule"`
}

type RemedyRecommendation struct {
	Method        string         `json:"method"`
	Status        string         `json:"status"`
	Planet        string         `json:"planet,omitempty"`
	Approach      string         `json:"approach"`
	Reason        GurujiWording  `json:"reason"`
	Practice      GurujiWording  `json:"practice"`
	EvidenceIDs   []string       `json:"evidence_ids"`
	Source        *GurujiLocator `json:"source,omitempty"`
	ApprovedRules []Rule         `json:"approved_rules,omitempty"`
}

type FaladeshFinding struct {
	ID            string        `json:"id"`
	Kind          string        `json:"kind"`
	Scope         string        `json:"scope"`
	Tone          string        `json:"tone"`
	Title         GurujiWording `json:"title"`
	Text          GurujiWording `json:"text"`
	Basis         GurujiWording `json:"basis"`
	Qualification GurujiWording `json:"qualification"`
	EvidenceIDs   []string      `json:"evidence_ids"`
	SourceIDs     []string      `json:"source_ids"`
}

type RajyogReading struct {
	Method     string          `json:"method"`
	Summary    GurujiWording   `json:"summary"`
	Paragraphs []GurujiWording `json:"paragraphs"`
}

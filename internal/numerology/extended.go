package numerology

import (
	"fmt"
	"strconv"
	"strings"
)

const ExtendedMethod = "m4y-numerology-name-arithmetic-only-v2"

// Bilingual preserves original authored wording in both supported report languages.
type Bilingual struct {
	Hindi   string `json:"hi"`
	English string `json:"en"`
}

type Fact struct {
	ID    string    `json:"id"`
	Label Bilingual `json:"label"`
	Value Bilingual `json:"value"`
}

type Guidance struct {
	ID    string    `json:"id"`
	Title Bilingual `json:"title"`
	Text  Bilingual `json:"text"`
}

type Profile struct {
	Number   int        `json:"number"`
	Facts    []Fact     `json:"facts"`
	Guidance []Guidance `json:"guidance"`
	Yantra   [][]int    `json:"yantra"`
}

type NameReading struct {
	Spelling string    `json:"spelling"`
	Letters  []string  `json:"letters"`
	Number   Number    `json:"number"`
	Meaning  Bilingual `json:"meaning"`
}

type ExtendedReading struct {
	Method       string       `json:"method"`
	NameStatus   string       `json:"name_status"`
	Naamank      *NameReading `json:"naamank,omitempty"`
	LuckyNumbers []int        `json:"lucky_numbers"`
	LuckyDates   []int        `json:"lucky_dates"`
	Profile      Profile      `json:"profile"`
}

// CalculateFull retains date and name arithmetic without authored profiles, lucky dates or guidance.
func CalculateFull(date, name string) (Reading, error) {
	r, err := Calculate(date)
	if err != nil {
		return r, err
	}
	ext := &ExtendedReading{Method: ExtendedMethod, NameStatus: "needs_latin_spelling"}
	if named, err := CalculateName(name); err == nil {
		ext.Naamank, ext.NameStatus = &named, "available"
	}
	r.Extended = ext
	return r, nil
}

// CalculateName uses all A-Z letters in the supplied spelling, preserving their individual values.
// Spaces, apostrophes, hyphens and periods are separators; digits and unsupported scripts reject the entire spelling.
func CalculateName(name string) (NameReading, error) {
	spelling := strings.Join(strings.Fields(name), " ")
	if len([]rune(spelling)) > 100 {
		return NameReading{}, fmt.Errorf("name spelling exceeds 100 characters")
	}
	letters, digits := []string{}, ""
	for _, c := range spelling {
		if strings.ContainsRune(" -'.", c) {
			continue
		}
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		if c < 'A' || c > 'Z' {
			return NameReading{}, fmt.Errorf("name spelling requires only A-Z letters and separators")
		}
		value := 0
		for i, group := range []string{"AIJQY", "BKR", "CGLS", "DMT", "EHNX", "UVW", "OZ", "FP"} {
			if strings.ContainsRune(group, c) {
				value = i + 1
				break
			}
		}
		if value == 0 {
			return NameReading{}, fmt.Errorf("name spelling requires only A-Z letters and separators")
		}
		letters = append(letters, string(c))
		digits += strconv.Itoa(value)
	}
	if len(letters) < 2 {
		return NameReading{}, fmt.Errorf("name spelling requires at least two letters")
	}
	n := reduce(digits)
	return NameReading{Spelling: spelling, Letters: letters, Number: n}, nil
}

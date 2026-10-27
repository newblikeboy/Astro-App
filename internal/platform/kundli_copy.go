package platform

import (
	_ "embed"
	"encoding/json"
)

//go:embed data/kundli-copy.json
var kundliCopyJSON []byte

//go:embed data/yoga-conclusions.json
var yogaConclusionsJSON []byte

//go:embed data/numerology-copy.json
var numerologyCopyJSON []byte

//go:embed data/pdf-book-copy.json
var pdfBookCopyJSON []byte

type kundliWording struct {
	Hindi   string `json:"hi"`
	English string `json:"en"`
}

var kundliCopy = func() map[string]kundliWording {
	var wording map[string]kundliWording
	if err := json.Unmarshal(kundliCopyJSON, &wording); err != nil {
		panic("invalid Kundli wording catalogue")
	}
	if err := json.Unmarshal(yogaConclusionsJSON, &wording); err != nil {
		panic("invalid Yoga conclusion catalogue")
	}
	if err := json.Unmarshal(numerologyCopyJSON, &wording); err != nil {
		panic("invalid numerology wording catalogue")
	}
	if err := json.Unmarshal(pdfBookCopyJSON, &wording); err != nil {
		panic("invalid PDF reading catalogue")
	}
	return wording
}()

// Package gurujitext exposes only the Hindi passages supplied in Guruji's PDFs.
package gurujitext

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
)

const Version = "guruji-pdf-hindi-2026-11-24-v1"

//go:embed data/catalog.json.gz
var catalogGZIP []byte

type Source struct {
	File     string `json:"file"`
	SHA256   string `json:"sha256"`
	Pages    int    `json:"pages"`
	Preamble string `json:"preamble"`
}

type Passage struct {
	Title     string `json:"title"`
	Source    int    `json:"source"`
	Pages     [2]int `json:"pages"`
	Text      string `json:"text"`
	House     int    `json:"house,omitempty"`
	Kind      string `json:"kind,omitempty"`
	Condition string `json:"condition,omitempty"`
}

type Reference struct {
	Passage
	Category string `json:"category"`
	Key      string `json:"key"`
	File     string `json:"file"`
	SHA256   string `json:"sha256"`
}

type catalog struct {
	sources    []Source
	categories map[string]map[string][]Passage
}

var supplied = load()

// load checks the bundled source catalogue once and never synthesizes missing text.
func load() catalog {
	reader, err := gzip.NewReader(bytes.NewReader(catalogGZIP))
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		panic(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		panic(err)
	}
	out := catalog{categories: map[string]map[string][]Passage{}}
	if err := json.Unmarshal(fields["sources"], &out.sources); err != nil || len(out.sources) != 7 {
		panic("invalid Guruji PDF sources")
	}
	delete(fields, "sources")
	for category, raw := range fields {
		var entries map[string][]Passage
		if err := json.Unmarshal(raw, &entries); err != nil {
			panic(err)
		}
		for _, passages := range entries {
			for _, passage := range passages {
				if passage.Source < 0 || passage.Source >= len(out.sources) || passage.Title == "" || passage.Text == "" || passage.Pages[0] < 1 || passage.Pages[1] < passage.Pages[0] || passage.Pages[1] > out.sources[passage.Source].Pages {
					panic("invalid Guruji PDF passage")
				}
			}
		}
		out.categories[category] = entries
	}
	return out
}

// Lookup copies original passages and their source locators; an absent key returns no invented fallback.
func Lookup(category, key string) []Reference {
	out := []Reference{}
	for _, passage := range supplied.categories[category][key] {
		source := supplied.sources[passage.Source]
		out = append(out, Reference{Passage: passage, Category: category, Key: key, File: source.File, SHA256: source.SHA256})
	}
	return out
}

// Sources returns a copy so report callers cannot mutate the embedded source inventory.
func Sources() []Source { return append([]Source(nil), supplied.sources...) }

package gurujitext

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

// TestCatalogEvidence ties the embedded JSON to the retained reconstruction audit rather than a regenerated fixture.
func TestCatalogEvidence(t *testing.T) {
	reader, err := gzip.NewReader(bytes.NewReader(catalogGZIP))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	auditData, err := os.ReadFile("../../references/guruji/2026-11-24/catalog-validation.json")
	if err != nil {
		t.Fatal(err)
	}
	var audit struct {
		JSONSHA256 string `json:"json_sha256"`
		Articles   int    `json:"article_count"`
	}
	if err := json.Unmarshal(auditData, &audit); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != audit.JSONSHA256 {
		t.Fatal("catalog differs from source reconstruction audit")
	}
	count := 0
	for _, keys := range supplied.categories {
		for _, entries := range keys {
			count += len(entries)
		}
	}
	if count != 246 || count != audit.Articles {
		t.Fatal("incomplete supplied catalogue", count)
	}
}

// TestSuppliedCoverage verifies the complete source inventory, every house, the missing number and repeated source article.
func TestSuppliedCoverage(t *testing.T) {
	pages := 0
	for _, source := range Sources() {
		data, err := os.ReadFile(filepath.Join("../../references/guruji/2026-11-24", source.File))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(data)
		if hex.EncodeToString(hash[:]) != source.SHA256 {
			t.Fatalf("source changed: %s", source.File)
		}
		pages += source.Pages
	}
	if pages != 1131 {
		t.Fatal("incomplete source pages", pages)
	}
	if len(Lookup("भाग्यांक", "3")) != 0 || len(Lookup("तिथिफल", "चतुर्दशी")) != 2 {
		t.Fatal("missing or repeated source text was replaced")
	}
	for _, planet := range []string{"सूर्य", "चंद्रमा", "मंगल", "बुध", "गुरु", "शुक्र", "शनि", "राहु", "केतु"} {
		seen := map[int]bool{}
		for _, entry := range Lookup("भावफल", planet) {
			if entry.House > 0 {
				seen[entry.House] = true
			}
		}
		for house := 1; house <= 12; house++ {
			if !seen[house] {
				t.Fatalf("missing %s house %d", planet, house)
			}
		}
		if len(Lookup("उपाय", planet)) == 0 {
			t.Fatal("missing remedy source", planet)
		}
	}
	for n := 1; n <= 12; n++ {
		if len(Lookup("ग्रहों का फला देश", strconv.Itoa(n))) != 2 {
			t.Fatal("lost conditional branch", n)
		}
	}
}

// TestLookupIsolation prevents a report from altering the shared source text or its provenance.
func TestLookupIsolation(t *testing.T) {
	first := Lookup("मूलांक", "1")
	want := first[0]
	first[0].Text = "changed"
	first[0].Pages[0] = 999
	if Lookup("मूलांक", "1")[0] != want {
		t.Fatal("mutable source catalogue")
	}
	if len(Lookup("नामांक", "1")) != 0 {
		t.Fatal("invented name-number meaning")
	}
}

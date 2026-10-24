package platform

import (
	"astroapp/backend/internal/swisseph"
	"fmt"
	"html/template"
	"strings"
)

// pdfChart draws D1 directly from the immutable snapshot without reclassifying rounded longitudes.
func pdfChart(r *Report) template.HTML {
	var signs [12]int
	placements := []swisseph.DivisionalPlacement{}
	for i := range signs {
		signs[i] = (r.Chart.Lagna.RashiNumber+i-1)%12 + 1
	}
	for _, p := range r.Chart.Planets {
		placements = append(placements, swisseph.DivisionalPlacement{Name: p.Name, House: p.House})
	}
	return pdfPlacementChart(signs, placements)
}

// pdfPlacementChart draws whole-sign or Chalit houses using only numeric placements and owned glyphs.
// Three labels per line keep crowded conjunctions inside their house rather than overflowing one text line.
func pdfPlacementChart(signs [12]int, planets []swisseph.DivisionalPlacement) template.HTML {
	centers := [][2]int{{150, 82}, {77, 30}, {28, 82}, {75, 152}, {28, 225}, {77, 275}, {150, 229}, {223, 275}, {272, 225}, {225, 152}, {272, 82}, {223, 30}}
	short := map[string]string{"Sun": "सू", "Moon": "चं", "Mars": "मं", "Mercury": "बु", "Jupiter": "गु", "Venus": "शु", "Saturn": "श", "Rahu": "रा", "Ketu": "के"}
	var s strings.Builder
	s.WriteString(`<svg class="chart" viewBox="-3 -3 306 306" role="img" aria-label="उत्तर भारतीय जन्म कुंडली"><rect width="300" height="300" fill="#fff8ea" stroke="#b36c30"/><path d="M0 0L300 300M300 0L0 300M150 0L300 150L150 300L0 150Z" fill="none" stroke="#b36c30"/><g font-family="Noto" text-anchor="middle" fill="#76241e">`)
	for i, xy := range centers {
		names := []string{}
		for _, p := range planets {
			if p.House == i+1 {
				if label, ok := short[p.Name]; ok {
					names = append(names, label)
				}
			}
		}
		fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="11">%d</text>`, xy[0], xy[1]-12, signs[i])
		fontSize, startY, lineHeight := 11, xy[1]+3, 12
		// Compact multiple rows in triangular houses; the last row must remain inside the narrowing diagonal.
		if len(names) > 3 {
			fontSize, startY, lineHeight = 8, xy[1]-1, 10
		}
		for start := 0; start < len(names); start += 3 {
			fmt.Fprintf(&s, `<text x="%d" y="%d" font-size="%d">%s</text>`, xy[0], startY+(start/3)*lineHeight, fontSize, strings.Join(names[start:min(start+3, len(names))], " · "))
		}
	}
	s.WriteString(`</g></svg>`)
	return template.HTML(s.String())
}

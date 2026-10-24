package swisseph

import (
	"fmt"
	"sort"
)

// Varga signs are traditional divisions of a natal longitude, not new sky
// positions. In particular they must not acquire physical speed/nakshatra data.
type Varga struct {
	Factor    int                   `json:"factor"`
	Name      string                `json:"name"`
	Method    string                `json:"method"`
	LagnaSign int                   `json:"lagna_sign"`
	Planets   []DivisionalPlacement `json:"planets"`
}

var vargaNames = map[int]string{1: "Rashi", 2: "Hora", 3: "Drekkana", 4: "Chaturthamsha", 7: "Saptamsha", 9: "Navamsha", 10: "Dashamsha", 12: "Dwadashamsha", 16: "Shodashamsha", 20: "Vimshamsha", 24: "Chaturvimshamsha", 27: "Saptavimshamsha", 30: "Trimshamsha", 40: "Khavedamsha", 45: "Akshavedamsha", 60: "Shashtiamsha"}
var VargaFactors = []int{1, 2, 3, 4, 7, 9, 10, 12, 16, 20, 24, 27, 30, 40, 45, 60}

// vargaPart locates an unrounded degree within the selected divisional-chart segment.
func vargaPart(longitude float64, factor int) int {
	base := float64(int(longitude/30) * 30)
	return min(factor-1, sort.Search(factor, func(i int) bool { return longitude < base+float64(i+1)*30/float64(factor) }))
}

// vargaSign applies the declared Parashari mapping for the requested divisional chart.
func vargaSign(longitude float64, factor int) (int, error) {
	if !finite(longitude) || longitude < 0 || longitude >= 360 {
		return 0, fmt.Errorf("invalid natal longitude")
	}
	if _, ok := vargaNames[factor]; !ok {
		return 0, fmt.Errorf("unsupported varga")
	}
	sign := int(longitude / 30)
	// Division by 30 precedes integer classification without display rounding.
	degree := longitude - float64(sign)*30
	part := vargaPart(longitude, factor)
	even := sign%2 == 1 // one-based even zodiac signs
	mapped := 0
	switch factor {
	case 1:
		mapped = sign
	case 2:
		mapped = 4 // traditional solar Hora: Leo; lunar Hora: Cancer
		if (even && part == 0) || (!even && part == 1) {
			mapped = 3
		}
	case 3:
		mapped = sign + 4*part
	case 4:
		mapped = sign + 3*part
	case 7:
		mapped = sign + part
		if even {
			mapped += 6
		}
	case 9:
		return navamsaSign(longitude), nil
	case 10:
		mapped = sign + part
		if even {
			mapped += 8
		}
	case 12:
		mapped = sign + part
	case 16, 45:
		mapped = part + 4*(sign%3)
	case 20:
		mapped = part + []int{0, 8, 4}[sign%3]
	case 24:
		mapped = 4 + part
		if even {
			mapped = 3 + part
		}
	case 27:
		mapped = part + 3*(sign%4)
	case 30:
		// Unequal Parashari Trimshamshas. Every interval is [start,end),
		// including 5/10/18/25 and 5/12/20/25 degree boundaries.
		bounds := []float64{5, 10, 18, 25, 30}
		signs := []int{0, 10, 8, 2, 6}
		if even {
			bounds = []float64{5, 12, 20, 25, 30}
			signs = []int{1, 5, 11, 9, 7}
		}
		for i, end := range bounds {
			if degree < end {
				mapped = signs[i]
				break
			}
		}
	case 40:
		mapped = part
		if even {
			mapped += 6
		}
	case 60:
		mapped = sign + part // declared Parashari, counted forward from natal sign
	}
	return mapped%12 + 1, nil
}

// CalculateVargas derives the sixteen supported vargas from the chart's unrounded Lagna and graha longitudes.
func CalculateVargas(chart Kundali) ([]Varga, error) {
	out := make([]Varga, 0, len(VargaFactors))
	for _, factor := range VargaFactors {
		lagna, err := vargaSign(chart.Lagna.Longitude, factor)
		if err != nil {
			return nil, err
		}
		method := "Parashari; forward counting; whole-sign divisional houses"
		if factor == 2 {
			method = "Parashari Sun/Moon Hora: Leo/Cancer"
		}
		if factor == 30 {
			method = "Parashari unequal Trimshamsha; half-open intervals"
		}
		if factor == 60 {
			method = "Parashari Shashtiamsha counted forward from natal sign"
		}
		v := Varga{factor, vargaNames[factor], method, lagna, []DivisionalPlacement{}}
		for _, p := range chart.Planets {
			sign, err := vargaSign(p.Longitude, factor)
			if err != nil {
				return nil, err
			}
			v.Planets = append(v.Planets, DivisionalPlacement{p.Name, sign, (sign-lagna+12)%12 + 1, factor == 9 && sign == p.RashiNumber})
		}
		out = append(out, v)
	}
	return out, nil
}

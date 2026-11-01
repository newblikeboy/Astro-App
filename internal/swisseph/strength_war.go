package swisseph

import "math"

// strengthWarTransfers applies the declared latitude convention to the five
// starry planets. Every transfer uses the unchanged pre-war totals, including
// a three-planet cluster; equal latitude stays unresolved rather than guessed.
func strengthWarTransfers(planets [7]Graha, latitude [7]float64, totals [7]float64) ([]PlanetWar, [7]float64) {
	wars := []PlanetWar{}
	adjustment := [7]float64{}
	for i := 2; i < 7; i++ {
		for j := i + 1; j < 7; j++ {
			sep := shortestAngle(planets[i].Longitude, planets[j].Longitude)
			if sep >= 1 {
				continue
			}
			war := PlanetWar{First: planets[i].Name, Second: planets[j].Name, Separation: sep, Status: "resolved"}
			if math.Abs(latitude[i]-latitude[j]) < 1e-10 {
				war.Status = "unresolved"
			} else {
				winner, loser := i, j
				if latitude[j] > latitude[i] {
					winner, loser = j, i
				}
				war.Winner = planets[winner].Name
				war.Transfer = math.Abs(totals[i] - totals[j])
				adjustment[winner] += war.Transfer
				adjustment[loser] -= war.Transfer
			}
			wars = append(wars, war)
		}
	}
	return wars, adjustment
}

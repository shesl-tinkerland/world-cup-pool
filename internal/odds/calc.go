package odds

import "math"

// ConsensusOdds averages h2h prices across all bookmakers for one event.
// Returns (homeOdds, drawOdds, awayOdds, ok). ok is false when no h2h data
// is available or the home/away names can't be matched.
func ConsensusOdds(event OddsEvent) (homeOdds, drawOdds, awayOdds float64, ok bool) {
	var sumH, sumD, sumA float64
	var n int
	for _, bk := range event.Bookmakers {
		for _, mkt := range bk.Markets {
			if mkt.Key != "h2h" {
				continue
			}
			var h, d, a float64
			for _, o := range mkt.Outcomes {
				switch o.Name {
				case event.HomeTeam:
					h = o.Price
				case event.AwayTeam:
					a = o.Price
				default:
					// draw has a different name ("Draw") in h2h
					d = o.Price
				}
			}
			if h > 0 && d > 0 && a > 0 {
				sumH += h
				sumD += d
				sumA += a
				n++
			}
		}
	}
	if n == 0 {
		return 0, 0, 0, false
	}
	return sumH / float64(n), sumD / float64(n), sumA / float64(n), true
}

// MatchProbs converts three decimal odds into true win probabilities by
// removing the bookmaker overround (margin). Each value is in [0,1].
func MatchProbs(homeOdds, drawOdds, awayOdds float64) (pHome, pDraw, pAway float64) {
	if homeOdds <= 0 || drawOdds <= 0 || awayOdds <= 0 {
		return 0, 0, 0
	}
	ih := 1.0 / homeOdds
	id := 1.0 / drawOdds
	ia := 1.0 / awayOdds
	total := ih + id + ia
	return ih / total, id / total, ia / total
}

// RankingProbs computes synthetic H/D/A probabilities from FIFA world rankings.
// A higher rank (smaller number) means a stronger team.
// The draw weight constant (0.30) approximates typical football draw frequency.
func RankingProbs(homeRank, awayRank int) (pHome, pDraw, pAway float64) {
	const drawWeight = 0.30
	if homeRank <= 0 {
		homeRank = 200
	}
	if awayRank <= 0 {
		awayRank = 200
	}
	sh := 1.0 / math.Sqrt(float64(homeRank))
	sa := 1.0 / math.Sqrt(float64(awayRank))
	total := sh + drawWeight + sa
	return sh / total, drawWeight / total, sa / total
}

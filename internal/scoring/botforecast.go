package scoring

import (
	"math/rand"
	"sort"

	"github.com/pocketbase/pocketbase/core"
)

var stageRank = map[string]int{
	"R32": 0, "R16": 1, "QF": 2, "SF": 3, "3RD": 4, "FINAL": 5,
}

// RandomForecast builds a fully self-consistent random Forecast (group order,
// 8 best thirds, and a bracket whose every winner is one of that match's
// resolved participants) using the same resolver the scorer uses — so bot
// players score coherently. Used by the dev bot generator.
func RandomForecast(app core.App, rng *rand.Rand) (
	order map[string][]string,
	thirds map[string]string,
	bracket map[string]string,
	err error,
) {
	groups, err := app.FindRecordsByFilter("tournament_groups", "id != ''", "letter", 0, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	order = map[string][]string{}
	letters := make([]string, 0, len(groups))
	for _, g := range groups {
		ids := append([]string{}, g.GetStringSlice("teams")...)
		rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })
		order[g.GetString("letter")] = ids
		letters = append(letters, g.GetString("letter"))
	}

	// Pick 8 of the 12 groups whose 3rd-placed team advances.
	rng.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	thirds = map[string]string{}
	for _, l := range letters[:8] {
		if len(order[l]) >= 3 {
			thirds[l] = order[l][2]
		}
	}

	koList, err := app.FindRecordsByFilter("matches", "stage != 'group'", "num", 0, 0)
	if err != nil {
		return nil, nil, nil, err
	}
	koByNum := map[int]*core.Record{}
	for _, m := range koList {
		if n := m.GetInt("num"); n > 0 {
			koByNum[n] = m
		}
	}
	// Process feeders before dependents: by stage, then match number.
	sort.SliceStable(koList, func(i, j int) bool {
		si, sj := stageRank[koList[i].GetString("stage")], stageRank[koList[j].GetString("stage")]
		if si != sj {
			return si < sj
		}
		return koList[i].GetInt("num") < koList[j].GetInt("num")
	})

	bracket = map[string]string{}
	r := &fcResolver{
		order:      order,
		thirdByNum: assignThirds(koList, thirds),
		bracket:    bracket,
		ko:         koByNum,
	}
	for _, m := range koList {
		h := r.resolve(m.GetString("homeLabel"), m.GetInt("num"), map[int]bool{})
		a := r.resolve(m.GetString("awayLabel"), m.GetInt("num"), map[int]bool{})
		var pick string
		switch {
		case h != "" && a != "":
			if rng.Intn(2) == 0 {
				pick = h
			} else {
				pick = a
			}
		case h != "":
			pick = h
		case a != "":
			pick = a
		}
		if pick != "" {
			bracket[koStableKey(m)] = pick
		}
	}
	return order, thirds, bracket, nil
}

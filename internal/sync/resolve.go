package sync

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/bracket"
	"github.com/oyvhov/world-cup-pool/internal/standings"
)

// ResolveBracket fills knockout matches' homeTeam/awayTeam from their
// placeholder labels once the referenced results are known. This is what makes
// a knockout Tip become available (Phase 3): a Tip opens as soon as both teams
// of a matchup are resolved.
//
// Resolvable labels:
//   - "1A".."2L"      group winner / runner-up (once that group is complete)
//   - "3A/B/C/D/F"     a best-third slot (see note)
//   - "W73" / "L101"   winner / loser of a finished knockout match
//
// Group order (and so who is 1A/2A/3A) follows the full FIFA tiebreaker chain,
// including head-to-head, via internal/standings — shared with Forecast scoring.
//
// NOTE: once all 8 best thirds are known, the best-third -> R32 slot mapping
// uses FIFA's official Annex C combination table (internal/bracket). While the
// group stage is still incomplete it falls back to a deterministic greedy fill,
// which only runs when the bracket can't be resolved yet anyway.
func ResolveBracket(app core.App) error {
	matches, err := app.FindRecordsByFilter("matches", "id != ''", "num", 0, 0)
	if err != nil {
		return err
	}

	byNum := map[int]*core.Record{}
	for _, m := range matches {
		if n := m.GetInt("num"); n > 0 {
			byNum[n] = m
		}
	}

	first, second, thirds := groupStandings(matches)

	// Resolve the 8 R32 third-slots. With all 8 best thirds known, use FIFA's
	// official Annex C table; otherwise fall back to a deterministic greedy
	// fill (only hit while the group stage is still incomplete, when the
	// bracket can't be resolved yet anyway).
	quals := make([]string, 0, len(thirds))
	for _, st := range thirds {
		quals = append(quals, st.group)
	}
	thirdByNum := map[int]string{}
	if tbl, ok := bracket.Lookup(quals); ok {
		for _, m := range matches {
			if m.GetString("stage") != "R32" {
				continue
			}
			home, away := m.GetString("homeLabel"), m.GetString("awayLabel")
			isSlot := (strings.HasPrefix(home, "3") && strings.Contains(home, "/")) ||
				(strings.HasPrefix(away, "3") && strings.Contains(away, "/"))
			if !isSlot {
				continue
			}
			if w, ok := bracket.WinnerLetter(home, away); ok {
				thirdByNum[m.GetInt("num")] = thirdTeam[tbl[w]]
			}
		}
	} else {
		thirdQueue := make([]string, len(quals))
		copy(thirdQueue, quals)
		r32 := []*core.Record{}
		for _, m := range matches {
			if m.GetString("stage") == "R32" {
				r32 = append(r32, m)
			}
		}
		sort.Slice(r32, func(i, j int) bool {
			return r32[i].GetInt("num") < r32[j].GetInt("num")
		})
		for _, m := range r32 {
			for _, lbl := range []string{m.GetString("homeLabel"), m.GetString("awayLabel")} {
				if !strings.HasPrefix(lbl, "3") || !strings.Contains(lbl, "/") {
					continue
				}
				allowed := strings.Split(strings.TrimPrefix(lbl, "3"), "/")
				for i, g := range thirdQueue {
					if g == "" {
						continue
					}
					ok := false
					for _, a := range allowed {
						if g == a {
							ok = true
							break
						}
					}
					if ok {
						thirdByNum[m.GetInt("num")] = thirdTeam[g]
						thirdQueue[i] = ""
						break
					}
				}
			}
		}
	}

	resolve := func(label string, num int) string {
		if label == "" {
			return ""
		}
		switch label[0] {
		case '1':
			return first[label[1:]]
		case '2':
			return second[label[1:]]
		case '3':
			return thirdByNum[num]
		case 'W', 'L':
			n, err := strconv.Atoi(label[1:])
			if err != nil {
				return ""
			}
			src, ok := byNum[n]
			if !ok || src.GetString("finalizedAt") == "" {
				return ""
			}
			adv := src.GetString("advancer")
			if label[0] == 'W' {
				return adv
			}
			// loser = the side that is not the advancer
			h, a := src.GetString("homeTeam"), src.GetString("awayTeam")
			if adv == h {
				return a
			}
			if adv == a {
				return h
			}
			return ""
		}
		return ""
	}

	for _, m := range matches {
		if m.GetString("stage") == "group" {
			continue
		}
		changed := false
		num := m.GetInt("num")
		if m.GetString("homeTeam") == "" {
			if id := resolve(m.GetString("homeLabel"), num); id != "" {
				m.Set("homeTeam", id)
				changed = true
			}
		}
		if m.GetString("awayTeam") == "" {
			if id := resolve(m.GetString("awayLabel"), num); id != "" {
				m.Set("awayTeam", id)
				changed = true
			}
		}
		if changed {
			if err := app.Save(m); err != nil {
				return err
			}
		}
	}
	return nil
}

// thirdTeam maps a group letter to that group's third-placed team id; filled
// by groupStandings and read by the greedy best-third allocation above.
var thirdTeam = map[string]string{}

type standing struct {
	group string
	team  string
}

// groupStandings computes, from finished group matches only, the 1st/2nd team
// id per group letter (only when that group's 6 matches are all finished) plus
// the globally ranked list of the best third-placed teams (top 8). It delegates
// the FIFA tiebreaker order (including head-to-head) to internal/standings so
// bracket resolution and Forecast scoring always agree, and logs any group that
// needed a non-official tiebreak so an admin can verify/override.
func groupStandings(matches []*core.Record) (first, second map[string]string, thirds []standing) {
	first = map[string]string{}
	second = map[string]string{}

	order, ranked, ambiguous := standings.GroupTables(standings.FromRecords(matches))
	for g, ids := range order {
		if len(ids) >= 2 {
			first[g] = ids[0]
			second[g] = ids[1]
		}
	}
	thirdTeam = map[string]string{}
	for _, r := range ranked {
		thirdTeam[r.Group] = r.TeamID
	}
	for _, r := range ranked {
		thirds = append(thirds, standing{group: r.Group, team: r.TeamID})
	}
	if len(thirds) > 8 {
		thirds = thirds[:8]
	}
	if len(ambiguous) > 0 {
		log.Printf("[sync] group ranking needed a non-official tiebreak (fair play / lots) for %v — verify and override if needed", ambiguous)
	}
	return first, second, thirds
}

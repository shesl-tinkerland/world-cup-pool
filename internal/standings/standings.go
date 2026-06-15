// Package standings computes World Cup group tables from finished group
// matches, applying FIFA's official 2026 tiebreaker order. It is the single
// source of truth shared by bracket resolution (internal/sync) and Forecast
// scoring (internal/scoring) so the two never disagree on who finished where.
//
// FIFA tiebreaker order within a group (Regulations art. on ranking):
//  1. points (all group matches)
//  2. goal difference (all group matches)
//  3. goals scored (all group matches)
//  if two or more teams are still equal, then AMONG THOSE TEAMS ONLY:
//  4. points in the matches between them (head-to-head)
//  5. goal difference in those matches
//  6. goals scored in those matches
//  7. fair-play points (we have no disciplinary data)
//  8. drawing of lots
//
// We implement 1–6. When teams remain tied after head-to-head (steps 7–8 we
// cannot reproduce), we fall back to a deterministic order by team id for
// stability and report the group as ambiguous so an admin can verify/override.
package standings

import "sort"

// Match is a single finished group match (full-time score). Knockout matches
// and unfinished group matches must be filtered out before constructing these.
type Match struct {
	Group     string
	Home      string
	Away      string
	HomeGoals int
	AwayGoals int
}

// Row is one team's aggregate within its group.
type Row struct {
	Group string
	TeamID string
	Pts    int
	GD     int
	GF     int
	Games  int
}

// AmbiguousThirds is the sentinel added to the ambiguous list when the best-8
// third-placed cut is decided by a tie the official numeric criteria cannot
// break.
const AmbiguousThirds = "best-thirds"

// GroupTables ranks every COMPLETE group (4 teams, each having played 3) using
// the FIFA tiebreaker order. It returns:
//   - order: group letter -> ordered team ids (1st..4th)
//   - thirds: every complete group's third-placed team, globally ranked
//     (not truncated; callers take the top 8)
//   - ambiguous: group letters whose order needed a non-official fallback,
//     plus AmbiguousThirds if the best-8 third cut is tied
func GroupTables(ms []Match) (order map[string][]string, thirds []Row, ambiguous []string) {
	order = map[string][]string{}

	type agg struct {
		pts, gd, gf, games int
	}
	groups := map[string]map[string]*agg{}
	for _, m := range ms {
		g := groups[m.Group]
		if g == nil {
			g = map[string]*agg{}
			groups[m.Group] = g
		}
		for _, id := range []string{m.Home, m.Away} {
			if g[id] == nil {
				g[id] = &agg{}
			}
		}
		h, a := g[m.Home], g[m.Away]
		h.games++
		a.games++
		h.gf += m.HomeGoals
		a.gf += m.AwayGoals
		h.gd += m.HomeGoals - m.AwayGoals
		a.gd += m.AwayGoals - m.HomeGoals
		switch {
		case m.HomeGoals > m.AwayGoals:
			h.pts += 3
		case m.AwayGoals > m.HomeGoals:
			a.pts += 3
		default:
			h.pts++
			a.pts++
		}
	}

	// matchesAmong indexes finished matches by group for head-to-head lookups.
	byGroup := map[string][]Match{}
	for _, m := range ms {
		byGroup[m.Group] = append(byGroup[m.Group], m)
	}

	letters := make([]string, 0, len(groups))
	for g := range groups {
		letters = append(letters, g)
	}
	sort.Strings(letters)

	for _, g := range letters {
		tbl := groups[g]
		// Only rank complete groups: 4 teams, each having played 3 matches.
		if len(tbl) < 4 {
			continue
		}
		complete := true
		rows := make([]Row, 0, len(tbl))
		for id, v := range tbl {
			if v.games < 3 {
				complete = false
			}
			rows = append(rows, Row{Group: g, TeamID: id, Pts: v.pts, GD: v.gd, GF: v.gf, Games: v.games})
		}
		if !complete {
			continue
		}

		ordered, amb := rankGroup(rows, byGroup[g])
		ids := make([]string, len(ordered))
		for i, r := range ordered {
			ids[i] = r.TeamID
		}
		order[g] = ids
		if amb {
			ambiguous = append(ambiguous, g)
		}
		thirds = append(thirds, ordered[2])
	}

	// Global best-third ranking (no head-to-head — these teams never met).
	sort.SliceStable(thirds, func(i, j int) bool { return lessOverall(thirds[i], thirds[j]) })
	if thirdsCutAmbiguous(thirds) {
		ambiguous = append(ambiguous, AmbiguousThirds)
	}
	return order, thirds, ambiguous
}

// rankGroup orders one group's rows by the full FIFA tiebreaker chain and
// reports whether any cluster needed a deterministic (non-official) fallback.
func rankGroup(rows []Row, matches []Match) (ordered []Row, ambiguous bool) {
	sort.SliceStable(rows, func(i, j int) bool { return lessOverall(rows[i], rows[j]) })

	// Walk maximal clusters equal on (pts, gd, gf) and break them with
	// head-to-head among the cluster's teams.
	for i := 0; i < len(rows); {
		j := i + 1
		for j < len(rows) && equalOverall(rows[i], rows[j]) {
			j++
		}
		if j-i > 1 {
			if breakTie(rows[i:j], matches) {
				ambiguous = true
			}
		}
		i = j
	}
	return rows, ambiguous
}

type h2h struct{ pts, gd, gf int }

func equalH2H(a, b h2h) bool { return a.pts == b.pts && a.gd == b.gd && a.gf == b.gf }

// breakTie orders a set of teams equal on the overall criteria using FIFA's
// head-to-head rules. Returns true if any teams remain tied after head-to-head
// (official steps 7–8, fair play / lots, are unavailable to us).
func breakTie(group []Row, matches []Match) bool {
	return rankSubgroup(group, matches)
}

// rankSubgroup reorders group in place by the head-to-head results among ONLY
// the teams in group, then recurses into any residual subgroup still tied after
// that pass (re-applying head-to-head exclusively to the matches between the
// teams still concerned, per FIFA). It falls back to a deterministic team-id
// order when teams cannot be separated, reporting that as ambiguous.
func rankSubgroup(group []Row, matches []Match) (ambiguous bool) {
	if len(group) <= 1 {
		return false
	}
	in := make(map[string]bool, len(group))
	for _, r := range group {
		in[r.TeamID] = true
	}
	mini := make(map[string]h2h, len(group))
	for _, m := range matches {
		if !in[m.Home] || !in[m.Away] {
			continue
		}
		h, a := mini[m.Home], mini[m.Away]
		h.gf += m.HomeGoals
		a.gf += m.AwayGoals
		h.gd += m.HomeGoals - m.AwayGoals
		a.gd += m.AwayGoals - m.HomeGoals
		switch {
		case m.HomeGoals > m.AwayGoals:
			h.pts += 3
		case m.AwayGoals > m.HomeGoals:
			a.pts += 3
		default:
			h.pts++
			a.pts++
		}
		mini[m.Home], mini[m.Away] = h, a
	}

	sort.SliceStable(group, func(i, j int) bool {
		x, y := mini[group[i].TeamID], mini[group[j].TeamID]
		if x.pts != y.pts {
			return x.pts > y.pts
		}
		if x.gd != y.gd {
			return x.gd > y.gd
		}
		if x.gf != y.gf {
			return x.gf > y.gf
		}
		return group[i].TeamID < group[j].TeamID
	})

	// Re-apply head-to-head to any residual cluster still equal on the h2h
	// triple, but only when it is strictly smaller than the current group
	// (otherwise we have a true circular tie that nothing here can break).
	for i := 0; i < len(group); {
		j := i + 1
		for j < len(group) && equalH2H(mini[group[i].TeamID], mini[group[j].TeamID]) {
			j++
		}
		if j-i > 1 {
			if j-i == len(group) {
				ambiguous = true
			} else if rankSubgroup(group[i:j], matches) {
				ambiguous = true
			}
		}
		i = j
	}
	return ambiguous
}

func equalOverall(a, b Row) bool {
	return a.Pts == b.Pts && a.GD == b.GD && a.GF == b.GF
}

func lessOverall(a, b Row) bool {
	if a.Pts != b.Pts {
		return a.Pts > b.Pts
	}
	if a.GD != b.GD {
		return a.GD > b.GD
	}
	if a.GF != b.GF {
		return a.GF > b.GF
	}
	return a.TeamID < b.TeamID
}

// thirdsCutAmbiguous reports whether the 8th and 9th best thirds are equal on
// the official numeric criteria (so who advances would be decided by fair play
// or lots). Only meaningful once enough groups are complete.
func thirdsCutAmbiguous(thirds []Row) bool {
	if len(thirds) <= 8 {
		return false
	}
	a, b := thirds[7], thirds[8]
	return a.Pts == b.Pts && a.GD == b.GD && a.GF == b.GF
}

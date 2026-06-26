package sync

import (
	"log"
	"sort"
	"strconv"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/bracket"
	"github.com/oyvhov/world-cup-pool/internal/standings"
	wmtips "github.com/oyvhov/world-cup-pool/internal/tips"
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
// NOTE: the best-third -> R32 slot mapping uses FIFA's official Annex C
// combination table (internal/bracket), but only after every group is complete.
// Until then those slots are deliberately unresolved because the top-8
// third-placed teams can still change.
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

	first, second, thirdTeams, thirds, completeGroups := groupStandings(matches)

	// Resolve the 8 R32 third-slots. With all 8 best thirds known, use FIFA's
	// official Annex C table. The best-third cut is not authoritative until all
	// groups are complete, so never fill these slots from a partial group stage.
	thirdByNum := bestThirdAssignments(matches, thirdTeams, thirds, completeGroups)

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
		tipsInvalidated := false
		num := m.GetInt("num")
		home := resolve(m.GetString("homeLabel"), num)
		if r := reconcileParticipant(m, "homeTeam", m.GetString("homeLabel"), home); r.changed {
			changed = true
			if r.invalidatesTips {
				tipsInvalidated = true
			}
		}
		away := resolve(m.GetString("awayLabel"), num)
		if r := reconcileParticipant(m, "awayTeam", m.GetString("awayLabel"), away); r.changed {
			changed = true
			if r.invalidatesTips {
				tipsInvalidated = true
			}
		}
		if changed {
			if tipsInvalidated {
				if _, err := deleteTipsForMatch(app, m.Id); err != nil {
					return err
				}
			}
			if err := app.Save(m); err != nil {
				return err
			}
		}
	}
	return nil
}

type standing struct {
	group string
	team  string
}

func totalGroupLetters(matches []*core.Record) int {
	seen := map[string]bool{}
	for _, m := range matches {
		if m.GetString("stage") != "group" {
			continue
		}
		if g := m.GetString("groupLetter"); g != "" {
			seen[g] = true
		}
	}
	return len(seen)
}

func bestThirdAssignments(matches []*core.Record, thirdTeams map[string]string, thirds []standing, completeGroups int) map[int]string {
	out := map[int]string{}
	if completeGroups != totalGroupLetters(matches) {
		return out
	}
	quals := make([]string, 0, len(thirds))
	for _, st := range thirds {
		quals = append(quals, st.group)
	}
	return thirdAssignments(matches, quals, thirdTeams)
}

func thirdAssignments(matches []*core.Record, quals []string, thirdTeams map[string]string) map[int]string {
	out := map[int]string{}
	if len(quals) != 8 {
		return out
	}
	if tbl, ok := bracket.Lookup(quals); ok {
		for _, m := range matches {
			if m.GetString("stage") != "R32" {
				continue
			}
			home, away := m.GetString("homeLabel"), m.GetString("awayLabel")
			isSlot := isThirdSlotLabel(home) || isThirdSlotLabel(away)
			if !isSlot {
				continue
			}
			if w, ok := bracket.WinnerLetter(home, away); ok {
				out[m.GetInt("num")] = thirdTeams[tbl[w]]
			}
		}
		return out
	}

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
			if !isThirdSlotLabel(lbl) {
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
					out[m.GetInt("num")] = thirdTeams[g]
					thirdQueue[i] = ""
					break
				}
			}
		}
	}
	return out
}

func isThirdSlotLabel(label string) bool {
	return strings.HasPrefix(label, "3") && strings.Contains(label, "/")
}

func isDynamicLabel(label string) bool {
	if label == "" {
		return false
	}
	switch label[0] {
	case '1', '2', '3', 'W', 'L':
		return true
	default:
		return false
	}
}

func canRewriteParticipant(m *core.Record) bool {
	status := m.GetString("status")
	return status == "" || status == "scheduled"
}

type participantReconcileResult struct {
	changed         bool
	invalidatesTips bool
}

func reconcileParticipant(m *core.Record, field, label, resolved string) participantReconcileResult {
	current := m.GetString(field)
	if resolved != "" {
		if current == resolved {
			return participantReconcileResult{}
		}
		if current == "" || canRewriteParticipant(m) {
			m.Set(field, resolved)
			return participantReconcileResult{
				changed:         true,
				invalidatesTips: current != "",
			}
		}
		return participantReconcileResult{}
	}
	if current != "" && isDynamicLabel(label) && canRewriteParticipant(m) {
		m.Set(field, "")
		return participantReconcileResult{changed: true, invalidatesTips: true}
	}
	return participantReconcileResult{}
}

func deleteTipsForMatch(app core.App, matchID string) (int, error) {
	tips, err := app.FindRecordsByFilter("tips",
		"match = {:m}", "", 0, 0, map[string]any{"m": matchID})
	if err != nil {
		return 0, err
	}
	if len(tips) == 0 {
		return 0, nil
	}

	wmtips.SetBypass(true)
	defer wmtips.SetBypass(false)

	for _, tip := range tips {
		if err := app.Delete(tip); err != nil {
			return 0, err
		}
	}
	return len(tips), nil
}

// groupStandings computes, from finished group matches only, the 1st/2nd team
// id per group letter (only when that group's 6 matches are all finished) plus
// the globally ranked list of the best third-placed teams (top 8). It delegates
// the FIFA tiebreaker order (including head-to-head) to internal/standings so
// bracket resolution and Forecast scoring always agree, and logs any group that
// needed a non-official tiebreak so an admin can verify/override.
func groupStandings(matches []*core.Record) (first, second map[string]string, thirdTeams map[string]string, thirds []standing, completeGroups int) {
	first = map[string]string{}
	second = map[string]string{}
	thirdTeams = map[string]string{}

	order, ranked, ambiguous := standings.GroupTables(standings.FromRecords(matches))
	completeGroups = len(order)
	for g, ids := range order {
		if len(ids) >= 2 {
			first[g] = ids[0]
			second[g] = ids[1]
		}
	}
	for _, r := range ranked {
		thirdTeams[r.Group] = r.TeamID
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
	return first, second, thirdTeams, thirds, completeGroups
}

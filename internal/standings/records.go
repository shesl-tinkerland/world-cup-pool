package standings

import "github.com/pocketbase/pocketbase/core"

// FromRecords converts match records into the finished group matches that
// GroupTables consumes: stage == "group" and finalized, with both teams set.
func FromRecords(recs []*core.Record) []Match {
	out := make([]Match, 0, len(recs))
	for _, m := range recs {
		if m.GetString("stage") != "group" || m.GetString("finalizedAt") == "" {
			continue
		}
		h, a := m.GetString("homeTeam"), m.GetString("awayTeam")
		if h == "" || a == "" {
			continue
		}
		out = append(out, Match{
			Group:     m.GetString("groupLetter"),
			Home:      h,
			Away:      a,
			HomeGoals: m.GetInt("ftHome"),
			AwayGoals: m.GetInt("ftAway"),
		})
	}
	return out
}

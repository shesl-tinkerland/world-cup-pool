package notifications

import (
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/topscorer"
)

// firstKickoff returns the earliest match kickoff (the tournament start / global
// Forecast deadline).
func firstKickoff(app core.App) (time.Time, error) {
	ms, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 1, 0)
	if err != nil {
		return time.Time{}, err
	}
	if len(ms) == 0 {
		return time.Time{}, errUnknownEvent
	}
	return ms[0].GetDateTime("kickoff").Time(), nil
}

// tippedMatchIDs returns the set of match ids the user has a tip for.
func tippedMatchIDs(app core.App, userID string) map[string]bool {
	out := map[string]bool{}
	tips, err := app.FindRecordsByFilter("tips", "user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return out
	}
	for _, t := range tips {
		out[t.GetString("match")] = true
	}
	return out
}

// hasAllGroupTips reports whether the user has tipped every group-stage match.
func hasAllGroupTips(app core.App, userID string, tipped map[string]bool) bool {
	groups, err := app.FindRecordsByFilter("matches", "stage = 'group'", "", 0, 0)
	if err != nil || len(groups) == 0 {
		return false
	}
	for _, m := range groups {
		if !tipped[m.Id] {
			return false
		}
	}
	return true
}

// koKey is the bracket key for a knockout match: its number, or the stage for
// the number-less Final / third-place matches. Mirrors the frontend koKey().
func koKey(m *core.Record) string {
	if n := m.GetInt("num"); n > 0 {
		return itoa(n)
	}
	return m.GetString("stage")
}

// hasCompleteForecast mirrors forecastStore.isComplete: at least 8 best-thirds
// chosen, a golden boot pick, and a winner for every knockout match.
func hasCompleteForecast(app core.App, userID string) bool {
	fc, err := app.FindFirstRecordByFilter("forecasts", "user = {:u}", map[string]any{"u": userID})
	if err != nil {
		return false
	}

	var thirds map[string]string
	_ = fc.UnmarshalJSONField("thirdQualifiers", &thirds)
	if len(thirds) < 8 {
		return false
	}

	if topscorer.PickFromForecast(fc) == "" {
		return false
	}

	var bracket map[string]string
	_ = fc.UnmarshalJSONField("bracket", &bracket)
	ko, err := app.FindRecordsByFilter("matches", "stage != 'group'", "num", 0, 0)
	if err != nil || len(ko) == 0 {
		return false
	}
	for _, m := range ko {
		if bracket[koKey(m)] == "" {
			return false
		}
	}
	return true
}

// hasSubmittedEverything is the "levert alt" rule for the pre-kickoff reminder:
// all group tips AND a complete forecast (groups, thirds, bracket, golden boot).
func hasSubmittedEverything(app core.App, userID string) bool {
	tipped := tippedMatchIDs(app, userID)
	return hasAllGroupTips(app, userID, tipped) && hasCompleteForecast(app, userID)
}

// countUpcomingUntipped returns how many tippable matches kick off in
// (now, now+within] that the user has not tipped. Only matches with both teams
// assigned (group matches, or resolved knockout ties) are counted.
func countUpcomingUntipped(app core.App, userID string, now time.Time, within time.Duration) int {
	until := now.Add(within)
	matches, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	if err != nil {
		return 0
	}
	tipped := tippedMatchIDs(app, userID)
	count := 0
	for _, m := range matches {
		ko := m.GetDateTime("kickoff").Time()
		if !ko.After(now) || ko.After(until) {
			continue
		}
		if m.GetString("homeTeam") == "" || m.GetString("awayTeam") == "" {
			continue // knockout placeholder not yet resolved
		}
		if !tipped[m.Id] {
			count++
		}
	}
	return count
}

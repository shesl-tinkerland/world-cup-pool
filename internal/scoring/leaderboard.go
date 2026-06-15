package scoring

import (
	"encoding/json"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/topscorer"
)

const leaderboardCacheTTL = 30 * time.Second

type leaderboardCacheEntry struct {
	version uint64
	expires time.Time
	data    map[string]any
}

var leaderboardCache = struct {
	sync.RWMutex
	entries map[string]leaderboardCacheEntry
}{entries: map[string]leaderboardCacheEntry{}}

var leaderboardCacheVersion atomic.Uint64

func avatarURL(user *core.Record) *string {
	file := user.GetString("avatar")
	if file == "" {
		return nil
	}
	url := "/api/files/users/" + user.Id + "/" + file
	return &url
}

// InvalidateLeaderboardCache clears the short-lived standings cache after
// writes that can affect ranks, totals, member lists or Golden Boot tables.
func InvalidateLeaderboardCache() {
	leaderboardCacheVersion.Add(1)
	leaderboardCache.Lock()
	leaderboardCache.entries = map[string]leaderboardCacheEntry{}
	leaderboardCache.Unlock()
}

func leaderboardCacheKey(leagueID string, cfgID string) string {
	return leagueID + ":" + cfgID
}

func cloneLeaderboardPayload(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func cachedLeaderboard(key string, version uint64, now time.Time) (map[string]any, bool) {
	leaderboardCache.RLock()
	entry, ok := leaderboardCache.entries[key]
	leaderboardCache.RUnlock()
	if !ok || entry.version != version || !now.Before(entry.expires) {
		return nil, false
	}
	return cloneLeaderboardPayload(entry.data), true
}

func cacheLeaderboard(key string, version uint64, now time.Time, data map[string]any) {
	leaderboardCache.Lock()
	leaderboardCache.entries[key] = leaderboardCacheEntry{
		version: version,
		expires: now.Add(leaderboardCacheTTL),
		data:    cloneLeaderboardPayload(data),
	}
	leaderboardCache.Unlock()
}

// Row is one player's standing in a League.
type Row struct {
	UserID         string  `json:"userId"`
	Name           string  `json:"name"`
	AvatarURL      *string `json:"avatarUrl"`
	Total          int     `json:"total"`
	TipsPoints     int     `json:"tipsPoints"`
	ForecastPoints int     `json:"forecastPoints"`
	Predicted      int     `json:"predicted"` // # matches the user has tipped
	// Tiebreakers (also returned for transparency).
	ExactScores    int `json:"exactScores"`
	CorrectWinners int `json:"correctWinners"`
	GdDeviation    int `json:"gdDeviation"`
	// Forecast correct-pick counts (groups/advance/champion + R32..FINAL).
	Forecast  map[string]int `json:"forecast"`
	RankDelta int            `json:"rankDelta"` // +N = moved up N spots since last matchday, 0 = unchanged/no data
	lastEdit  string         // earliest-wins; not serialized
	prevTotal int            // for delta computation only, not serialized
}

// Leaderboard builds a League's standings using its scoring config and the
// agreed tiebreakers: points → #exact → #correct winners → smaller aggregate
// goal-difference deviation → fewer tips submitted → earliest last edit.
// Users who never submitted a tip are sorted to the bottom regardless.
//
// Note: the sort order below is hardcoded — the scoring_configs.tiebreakers
// list is consumed only by the frontend legend for display. Keep the two in
// sync when changing tiebreakers (update this function, the seeded default
// in internal/seed, and add a migration for existing DBs).
func Leaderboard(app core.App, leagueID string) (map[string]any, error) {
	league, err := app.FindRecordById("leagues", leagueID)
	if err != nil {
		return nil, err
	}
	cfgID := league.GetString("scoringConfig")
	if cfgID == "" {
		if def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true"); err == nil {
			cfgID = def.Id
		}
	}
	now := time.Now()
	cacheVersion := leaderboardCacheVersion.Load()
	cacheKey := leaderboardCacheKey(leagueID, cfgID)
	if cached, ok := cachedLeaderboard(cacheKey, cacheVersion, now); ok {
		return cached, nil
	}

	members, err := app.FindRecordsByFilter("league_members",
		"league = {:l}", "", 0, 0, map[string]any{"l": leagueID})
	if err != nil {
		return nil, err
	}

	// Identify the most recently finalized matchday so we can compute rank
	// movement since those results came in.
	latestBatchIDs := map[string]bool{}
	if lm, err := app.FindRecordsByFilter("matches", "finalizedAt != ''", "-finalizedAt", 1, 0); err == nil && len(lm) > 0 {
		fa := lm[0].GetDateTime("finalizedAt").Time().UTC()
		dayStart := time.Date(fa.Year(), fa.Month(), fa.Day(), 0, 0, 0, 0, time.UTC)
		dayEnd := dayStart.Add(24 * time.Hour)
		if batch, err := app.FindRecordsByFilter("matches",
			"finalizedAt >= {:s} && finalizedAt < {:e}", "", 0, 0,
			map[string]any{"s": dayStart.Format(time.RFC3339), "e": dayEnd.Format(time.RFC3339)}); err == nil {
			for _, bm := range batch {
				latestBatchIDs[bm.Id] = true
			}
		}
	}

	rows := make([]Row, 0, len(members))
	for _, m := range members {
		uid := m.GetString("user")
		u, err := app.FindRecordById("users", uid)
		if err != nil {
			continue
		}
		row := Row{UserID: uid, Name: u.GetString("name"), AvatarURL: avatarURL(u)}

		ms, _ := app.FindRecordsByFilter("match_scores",
			"user = {:u} && config = {:c}", "", 0, 0,
			map[string]any{"u": uid, "c": cfgID})
		var prevTipsPoints int
		for _, s := range ms {
			pts := s.GetInt("points")
			row.TipsPoints += pts
			if !latestBatchIDs[s.GetString("match")] {
				prevTipsPoints += pts
			}
			var comp tipComponents
			_ = json.Unmarshal([]byte(s.GetString("components")), &comp)
			if comp.Exact > 0 {
				row.ExactScores++
			}
			if comp.Tendency > 0 {
				row.CorrectWinners++
			}
			row.GdDeviation += comp.GdDev
		}

		if fs, err := app.FindFirstRecordByFilter("forecast_scores",
			"user = {:u} && config = {:c}",
			map[string]any{"u": uid, "c": cfgID}); err == nil {
			row.ForecastPoints = fs.GetInt("points")
			var bd struct {
				GroupsCorrect     int            `json:"groupsCorrect"`
				AdvanceCorrect    int            `json:"advanceCorrect"`
				RoundCorrect      map[string]int `json:"roundCorrect"`
				ChampionCorrect   int            `json:"championCorrect"`
				GoldenBoot        int            `json:"goldenBoot"`
				GoldenBootCorrect int            `json:"goldenBootCorrect"`
			}
			if json.Unmarshal([]byte(fs.GetString("breakdown")), &bd) == nil {
				f := map[string]int{
					"groups":           bd.GroupsCorrect,
					"advance":          bd.AdvanceCorrect,
					"champion":         bd.ChampionCorrect,
					"goldenBoot":       bd.GoldenBootCorrect,
					"goldenBootPoints": bd.GoldenBoot,
				}
				for k, v := range bd.RoundCorrect {
					f[k] = v
				}
				row.Forecast = f
			}
		}

		row.Total = row.TipsPoints + row.ForecastPoints
		row.prevTotal = prevTipsPoints + row.ForecastPoints

		if n, err := app.CountRecords("tips", dbx.HashExp{"user": uid}); err == nil {
			row.Predicted = int(n)
		}

		// Earliest last-edit across this user's tips (earlier = better).
		if tps, _ := app.FindRecordsByFilter("tips",
			"user = {:u}", "-updated", 1, 0,
			map[string]any{"u": uid}); len(tps) > 0 {
			row.lastEdit = tps[0].GetString("updated")
		}
		rows = append(rows, row)
	}

	// Compute previous ranks (before the latest matchday) so we can emit deltas.
	prevRows := make([]Row, len(rows))
	copy(prevRows, rows)
	sort.SliceStable(prevRows, func(i, j int) bool {
		a, b := prevRows[i], prevRows[j]
		if (a.Predicted == 0) != (b.Predicted == 0) {
			return a.Predicted != 0
		}
		return a.prevTotal > b.prevTotal
	})
	prevRankOf := make(map[string]int, len(prevRows))
	for i, r := range prevRows {
		prevRankOf[r.UserID] = i + 1
	}

	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		aNone, bNone := a.Predicted == 0, b.Predicted == 0
		if aNone != bNone {
			return !aNone
		}
		if a.Total != b.Total {
			return a.Total > b.Total
		}
		if a.ExactScores != b.ExactScores {
			return a.ExactScores > b.ExactScores
		}
		if a.CorrectWinners != b.CorrectWinners {
			return a.CorrectWinners > b.CorrectWinners
		}
		if a.GdDeviation != b.GdDeviation {
			return a.GdDeviation < b.GdDeviation
		}
		if a.Predicted != b.Predicted {
			return a.Predicted < b.Predicted
		}
		return a.lastEdit < b.lastEdit
	})

	// Assign rank deltas: positive = moved up, negative = dropped.
	for i := range rows {
		if prev, ok := prevRankOf[rows[i].UserID]; ok {
			rows[i].RankDelta = prev - (i + 1)
		}
	}

	out := map[string]any{
		"league": map[string]any{"id": league.Id, "name": league.GetString("name")},
		"rows":   rows,
	}
	if goldenBoot, err := topscorer.LeagueTableFor(app, leagueID); err == nil {
		out["goldenBoot"] = goldenBoot
	}
	cacheLeaderboard(cacheKey, cacheVersion, now, out)
	return cloneLeaderboardPayload(out), nil
}

package account

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// scoreComponents mirrors the JSON blob stored in match_scores.components.
// Kept in sync with scoring.tipComponents (unexported there); we only need
// the fields the player-card stats consume.
type scoreComponents struct {
	Tendency int `json:"tendency"`
	Exact    int `json:"exact"`
	GdDev    int `json:"gdDev"`
}

// PlayerStats is the payload for GET /api/player/me/stats.
type PlayerStats struct {
	TipsPredicted  int          `json:"tipsPredicted"`
	TipsScored     int          `json:"tipsScored"`
	HitRate        HitRate      `json:"hitRate"`
	LongestStreak  int          `json:"longestStreak"`
	CurrentStreak  int          `json:"currentStreak"`
	LargestMiss    *LargestMiss `json:"largestMiss,omitempty"`
}

type HitRate struct {
	Count int     `json:"count"`
	Total int     `json:"total"`
	Pct   float64 `json:"pct"` // 0..1
}

type LargestMiss struct {
	MatchID    string `json:"matchId"`
	Kickoff    string `json:"kickoff"`
	Stage      string `json:"stage"`
	HomeTeam   string `json:"homeTeam"` // team id
	AwayTeam   string `json:"awayTeam"` // team id
	HomeLabel  string `json:"homeLabel"`
	AwayLabel  string `json:"awayLabel"`
	TipHome    int    `json:"tipHome"`
	TipAway    int    `json:"tipAway"`
	ActualHome int    `json:"actualHome"`
	ActualAway int    `json:"actualAway"`
	GdDev      int    `json:"gdDev"`
}

// computePlayerStats is the pure aggregation for testability.
type scoredMatch struct {
	matchID  string
	kickoff  string // RFC3339; only used for ordering
	points   int
	exact    bool
	gdDev    int
}

func computePlayerStats(scored []scoredMatch) PlayerStats {
	// Order chronologically for streak computation. Ties on kickoff fall back
	// to the match id so the result is deterministic.
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].kickoff != scored[j].kickoff {
			return scored[i].kickoff < scored[j].kickoff
		}
		return scored[i].matchID < scored[j].matchID
	})

	stats := PlayerStats{TipsScored: len(scored)}
	var maxMissIdx = -1
	var maxMiss int
	var currentRun, bestRun int
	for i, s := range scored {
		if s.exact {
			stats.HitRate.Count++
		}
		if s.points > 0 {
			currentRun++
			if currentRun > bestRun {
				bestRun = currentRun
			}
		} else {
			currentRun = 0
		}
		if maxMissIdx == -1 || s.gdDev > maxMiss {
			maxMissIdx = i
			maxMiss = s.gdDev
		}
	}
	stats.HitRate.Total = len(scored)
	if len(scored) > 0 {
		stats.HitRate.Pct = float64(stats.HitRate.Count) / float64(len(scored))
	}
	stats.LongestStreak = bestRun
	stats.CurrentStreak = currentRun
	return stats
}

// RegisterStats wires the GET /api/player/me/stats endpoint.
//
// Returns global per-user stats (hit rate, longest scoring streak, largest
// goal-difference miss). All numbers are aggregated across every scored
// match the requesting user has tipped on under the default scoring config.
func RegisterStats(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/player/me/stats", func(e *core.RequestEvent) error {
		uid := e.Auth.Id

		def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		if err != nil || def == nil {
			return e.JSON(http.StatusOK, PlayerStats{})
		}

		scores, err := app.FindRecordsByFilter("match_scores",
			"user = {:u} && config = {:c}", "", 0, 0,
			map[string]any{"u": uid, "c": def.Id})
		if err != nil {
			return err
		}

		userTips, err := app.FindRecordsByFilter("tips",
			"user = {:u}", "", 0, 0, map[string]any{"u": uid})
		if err != nil {
			return err
		}
		tipsPredicted := len(userTips)
		tipByMatch := make(map[string]*core.Record, len(userTips))
		for _, t := range userTips {
			tipByMatch[t.GetString("match")] = t
		}

		scored := make([]scoredMatch, 0, len(scores))
		matchCache := map[string]*core.Record{}
		// Track which score row was the worst so we can re-fetch its match record below.
		worstScoreIdx := -1
		var worstGdDev int
		for i, s := range scores {
			matchID := s.GetString("match")
			m, ok := matchCache[matchID]
			if !ok {
				rec, err := app.FindRecordById("matches", matchID)
				if err != nil {
					continue
				}
				m = rec
				matchCache[matchID] = m
			}
			// Only count scored matches (have a final result + finalizedAt set).
			if m.GetString("finalizedAt") == "" {
				continue
			}
			var comp scoreComponents
			_ = json.Unmarshal([]byte(s.GetString("components")), &comp)
			scored = append(scored, scoredMatch{
				matchID: matchID,
				kickoff: m.GetDateTime("kickoff").Time().UTC().Format("2006-01-02T15:04:05Z"),
				points:  s.GetInt("points"),
				exact:   comp.Exact > 0,
				gdDev:   comp.GdDev,
			})
			if worstScoreIdx == -1 || comp.GdDev > worstGdDev {
				worstScoreIdx = i
				worstGdDev = comp.GdDev
			}
		}

		stats := computePlayerStats(scored)
		stats.TipsPredicted = tipsPredicted

		// Largest miss detail (needs match + tip records).
		if worstScoreIdx >= 0 {
			s := scores[worstScoreIdx]
			matchID := s.GetString("match")
			if m, ok := matchCache[matchID]; ok {
				lm := &LargestMiss{
					MatchID:    matchID,
					Kickoff:    m.GetDateTime("kickoff").Time().UTC().Format("2006-01-02T15:04:05Z"),
					Stage:      m.GetString("stage"),
					HomeTeam:   m.GetString("homeTeam"),
					AwayTeam:   m.GetString("awayTeam"),
					HomeLabel:  m.GetString("homeLabel"),
					AwayLabel:  m.GetString("awayLabel"),
					ActualHome: m.GetInt("ftHome"),
					ActualAway: m.GetInt("ftAway"),
					GdDev:      worstGdDev,
				}
				// Fall back to the team's display name for group matches where
				// the slot label is empty.
				if lm.HomeLabel == "" && lm.HomeTeam != "" {
					if rec, err := app.FindRecordById("teams", lm.HomeTeam); err == nil {
						lm.HomeLabel = rec.GetString("name")
					}
				}
				if lm.AwayLabel == "" && lm.AwayTeam != "" {
					if rec, err := app.FindRecordById("teams", lm.AwayTeam); err == nil {
						lm.AwayLabel = rec.GetString("name")
					}
				}
				if t := tipByMatch[matchID]; t != nil {
					lm.TipHome = t.GetInt("ftHome")
					lm.TipAway = t.GetInt("ftAway")
				}
				stats.LargestMiss = lm
			}
		}

		return e.JSON(http.StatusOK, stats)
	}).Bind(apis.RequireAuth())
}

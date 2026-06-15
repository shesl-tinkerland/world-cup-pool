package odds

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// cronExprs run twice daily at 07:00 UTC and 18:30 UTC (20:30 CEST during
// the tournament) — ~62 requests for the tournament month, comfortably within
// The Odds API free tier.
var cronExprs = []string{"0 7 * * *", "30 18 * * *"}

// Register wires the odds cron (when ODDS_API_KEY is set) and the public
// GET /api/odds endpoint.
func Register(app core.App, se *core.ServeEvent) {
	key := os.Getenv("ODDS_API_KEY")

	if key != "" {
		client := New(key)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := SyncOdds(ctx, app, client); err != nil {
				log.Printf("[odds] startup sync error: %v", err)
			}
		}()
		for i, cronExpr := range cronExprs {
			entryID := "odds-sync-" + string(rune('1'+i))
			app.Cron().MustAdd(entryID, cronExpr, func() {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				if err := SyncOdds(ctx, app, client); err != nil {
					log.Printf("[odds] cron sync error: %v", err)
				}
			})
		}
		log.Printf("[odds] auto-sync enabled (%v)", cronExprs)
	} else {
		log.Printf("[odds] ODDS_API_KEY not set — using FIFA ranking fallback")
	}

	se.Router.GET("/api/odds", func(e *core.RequestEvent) error {
		if key != "" {
			return serveStoredOdds(app, e)
		}
		return serveRankingOdds(app, e)
	})
}

type oddsEntry struct {
	MatchID  string  `json:"matchId"`
	PHome    float64 `json:"pHome"`
	PDraw    float64 `json:"pDraw"`
	PAway    float64 `json:"pAway"`
	HomeOdds float64 `json:"homeOdds"`
	DrawOdds float64 `json:"drawOdds"`
	AwayOdds float64 `json:"awayOdds"`
}

type oddsResponse struct {
	Source    string      `json:"source"`
	UpdatedAt string      `json:"updatedAt,omitempty"`
	Odds      []oddsEntry `json:"odds"`
}

// serveStoredOdds returns odds from the match_odds collection (real bookmaker data).
func serveStoredOdds(app core.App, e *core.RequestEvent) error {
	records, err := app.FindRecordsByFilter("match_odds", "id != ''", "-syncedAt", 0, 0)
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}
	if len(records) == 0 {
		return serveRankingOdds(app, e)
	}
	entries := make([]oddsEntry, 0, len(records))
	var latestSync time.Time
	for _, r := range records {
		entries = append(entries, oddsEntry{
			MatchID:  r.GetString("match"),
			PHome:    r.GetFloat("pHome"),
			PDraw:    r.GetFloat("pDraw"),
			PAway:    r.GetFloat("pAway"),
			HomeOdds: r.GetFloat("homeOdds"),
			DrawOdds: r.GetFloat("drawOdds"),
			AwayOdds: r.GetFloat("awayOdds"),
		})
		if t := r.GetDateTime("syncedAt").Time(); t.After(latestSync) {
			latestSync = t
		}
	}
	updatedAt := ""
	if !latestSync.IsZero() {
		updatedAt = latestSync.UTC().Format(time.RFC3339)
	}
	return e.JSON(200, oddsResponse{
		Source:    "odds_api",
		UpdatedAt: updatedAt,
		Odds:      entries,
	})
}

// serveRankingOdds computes synthetic probabilities from teams.fifaRanking.
func serveRankingOdds(app core.App, e *core.RequestEvent) error {
	matches, err := app.FindRecordsByFilter(
		"matches", "status = 'scheduled'", "kickoff", 0, 0,
	)
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	teams, err := app.FindAllRecords("teams")
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}
	rankByID := map[string]int{}
	for _, t := range teams {
		rankByID[t.Id] = t.GetInt("fifaRanking")
	}

	entries := make([]oddsEntry, 0, len(matches))
	for _, m := range matches {
		homeID := m.GetString("homeTeam")
		awayID := m.GetString("awayTeam")
		if homeID == "" || awayID == "" {
			continue
		}
		homeRank := rankByID[homeID]
		awayRank := rankByID[awayID]
		pHome, pDraw, pAway := RankingProbs(homeRank, awayRank)
		entries = append(entries, oddsEntry{
			MatchID:  m.Id,
			PHome:    round4(pHome),
			PDraw:    round4(pDraw),
			PAway:    round4(pAway),
			HomeOdds: 0,
			DrawOdds: 0,
			AwayOdds: 0,
		})
	}
	return e.JSON(200, oddsResponse{
		Source: "rankings",
		Odds:   entries,
	})
}

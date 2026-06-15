package odds

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/football"
)

// oddsAliases maps The Odds API team names (normalised) to the names used in
// the openfootball seed. Add entries here as mismatches are discovered via the
// [odds] unmatched log lines.
var oddsAliases = map[string]string{
	football.NormalizeName("United States"):        football.NormalizeName("USA"),
	football.NormalizeName("Korea Republic"):        football.NormalizeName("South Korea"),
	football.NormalizeName("Bosnia and Herzegovina"): football.NormalizeName("Bosnia & Herzegovina"),
	football.NormalizeName("Cape Verde Islands"):    football.NormalizeName("Cape Verde"),
	football.NormalizeName("Congo DR"):              football.NormalizeName("DR Congo"),
	football.NormalizeName("IR Iran"):               football.NormalizeName("Iran"),
	football.NormalizeName("Czechia"):               football.NormalizeName("Czech Republic"),
}

func canonName(s string) string {
	n := football.NormalizeName(s)
	if a, ok := oddsAliases[n]; ok {
		return a
	}
	return n
}

// SyncOdds fetches h2h odds and upserts them into the match_odds collection.
func SyncOdds(ctx context.Context, app core.App, client *Client) error {
	events, err := client.FetchOdds(ctx)
	if err != nil {
		return fmt.Errorf("fetch odds: %w", err)
	}

	matches, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	if err != nil {
		return fmt.Errorf("load matches: %w", err)
	}

	teamName := map[string]string{}
	teams, _ := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0)
	for _, t := range teams {
		teamName[t.Id] = canonName(t.GetString("name"))
	}

	// Index matches by normalised "homeCanon|awayCanon" pair.
	byPair := map[string]*core.Record{}
	for _, m := range matches {
		h := teamName[m.GetString("homeTeam")]
		a := teamName[m.GetString("awayTeam")]
		if h != "" && a != "" {
			byPair[h+"|"+a] = m
		}
	}

	oddsCol, err := app.FindCollectionByNameOrId("match_odds")
	if err != nil {
		return fmt.Errorf("match_odds collection: %w", err)
	}

	now := time.Now().UTC()
	updated, skipped := 0, 0

	for _, ev := range events {
		homeOdds, drawOdds, awayOdds, ok := ConsensusOdds(ev)
		if !ok {
			log.Printf("[odds] no h2h data for %q vs %q", ev.HomeTeam, ev.AwayTeam)
			continue
		}

		ch := canonName(ev.HomeTeam)
		ca := canonName(ev.AwayTeam)
		mrec, found := byPair[ch+"|"+ca]
		if !found {
			log.Printf("[odds] unmatched event %q vs %q (canon: %q|%q)", ev.HomeTeam, ev.AwayTeam, ch, ca)
			skipped++
			continue
		}

		pHome, pDraw, pAway := MatchProbs(homeOdds, drawOdds, awayOdds)

		// Find existing record or create new.
		existing, _ := app.FindFirstRecordByFilter("match_odds", "match = {:id}", map[string]any{"id": mrec.Id})
		var rec *core.Record
		if existing != nil {
			rec = existing
		} else {
			rec = core.NewRecord(oddsCol)
			rec.Set("match", mrec.Id)
		}
		rec.Set("pHome", round4(pHome))
		rec.Set("pDraw", round4(pDraw))
		rec.Set("pAway", round4(pAway))
		rec.Set("homeOdds", round2(homeOdds))
		rec.Set("drawOdds", round2(drawOdds))
		rec.Set("awayOdds", round2(awayOdds))
		rec.Set("syncedAt", now)
		if err := app.Save(rec); err != nil {
			log.Printf("[odds] save %s vs %s: %v", ev.HomeTeam, ev.AwayTeam, err)
			continue
		}
		updated++
	}

	log.Printf("[odds] sync done: %d updated, %d unmatched", updated, skipped)
	return nil
}

func round2(f float64) float64 {
	v, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", f), 64)
	return v
}

func round4(f float64) float64 {
	v, _ := strconv.ParseFloat(fmt.Sprintf("%.4f", f), 64)
	return v
}

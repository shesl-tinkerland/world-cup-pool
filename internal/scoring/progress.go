package scoring

import (
	"encoding/json"

	"github.com/pocketbase/pocketbase/core"
)

type ProgressSummary struct {
	TipsPoints        int `json:"tipsPoints"`
	Last5Points       int `json:"last5Points"`
	FinishedMatches   int `json:"finishedMatches"`
	TippedFinished    int `json:"tippedFinished"`
	ExactScores       int `json:"exactScores"`
	MatchesWithPoints int `json:"matchesWithPoints"`
	BestPoints        int `json:"bestPoints"`
}

type ProgressEvent struct {
	MatchID        string `json:"matchId"`
	Kickoff        string `json:"kickoff"`
	Stage          string `json:"stage"`
	HomeTeam       string `json:"homeTeam"`
	AwayTeam       string `json:"awayTeam"`
	HomeLabel      string `json:"homeLabel"`
	AwayLabel      string `json:"awayLabel"`
	FTHome         int    `json:"ftHome"`
	FTAway         int    `json:"ftAway"`
	ETHome         int    `json:"etHome"`
	ETAway         int    `json:"etAway"`
	PenHome        int    `json:"penHome"`
	PenAway        int    `json:"penAway"`
	Points         int    `json:"points"`
	TotalAfter     int    `json:"totalAfter"`
	Tipped         bool   `json:"tipped"`
	Exact          bool   `json:"exact"`
	CorrectWinner  bool   `json:"correctWinner"`
	CorrectTotalGoals bool `json:"correctTotalGoals"`
	CorrectGoalDiff bool  `json:"correctGoalDiff"`
}

// LeagueProgress returns a stable, league-scoped match-points timeline for a
// single user. It is derived from match_scores under the league's scoring
// config, so it survives page refreshes and matches the active league rules.
func LeagueProgress(app core.App, leagueID, userID string, limit int) (map[string]any, error) {
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

	if limit <= 0 {
		limit = 5
	}

	finished, err := app.FindRecordsByFilter(
		"matches",
		"finalizedAt != ''",
		"kickoff",
		0,
		0,
	)
	if err != nil {
		return nil, err
	}

	tips, err := app.FindRecordsByFilter(
		"tips",
		"user = {:u}",
		"",
		0,
		0,
		map[string]any{"u": userID},
	)
	if err != nil {
		return nil, err
	}
	tipByMatch := make(map[string]*core.Record, len(tips))
	for _, tip := range tips {
		tipByMatch[tip.GetString("match")] = tip
	}

	scores, err := app.FindRecordsByFilter(
		"match_scores",
		"user = {:u} && config = {:c}",
		"",
		0,
		0,
		map[string]any{"u": userID, "c": cfgID},
	)
	if err != nil {
		return nil, err
	}
	scoreByMatch := make(map[string]*core.Record, len(scores))
	for _, score := range scores {
		scoreByMatch[score.GetString("match")] = score
	}

	events := make([]ProgressEvent, 0, len(finished))
	runningTotal := 0
	exactScores := 0
	tippedFinished := 0
	matchesWithPoints := 0
	bestPoints := 0

	for _, match := range finished {
		tip := tipByMatch[match.Id]
		tipped := tip != nil
		if tipped {
			tippedFinished++
		}

		points := 0
		exact := false
		correctWinner := false
		correctTotalGoals := false
		correctGoalDiff := false
		if score := scoreByMatch[match.Id]; score != nil {
			points = score.GetInt("points")
			var comp tipComponents
			if err := json.Unmarshal([]byte(score.GetString("components")), &comp); err == nil {
				exact = comp.Exact > 0
				correctWinner = comp.Tendency > 0
				correctTotalGoals = comp.TotalGoals > 0
				correctGoalDiff = comp.GoalDiff > 0
				if exact {
					exactScores++
				}
			}
		}

		runningTotal += points
		if points > 0 {
			matchesWithPoints++
		}
		if points > bestPoints {
			bestPoints = points
		}
		events = append(events, ProgressEvent{
			MatchID:         match.Id,
			Kickoff:         match.GetString("kickoff"),
			Stage:           match.GetString("stage"),
			HomeTeam:        match.GetString("homeTeam"),
			AwayTeam:        match.GetString("awayTeam"),
			HomeLabel:       match.GetString("homeLabel"),
			AwayLabel:       match.GetString("awayLabel"),
			FTHome:          match.GetInt("ftHome"),
			FTAway:          match.GetInt("ftAway"),
			ETHome:          match.GetInt("etHome"),
			ETAway:          match.GetInt("etAway"),
			PenHome:         match.GetInt("penHome"),
			PenAway:         match.GetInt("penAway"),
			Points:          points,
			TotalAfter:      runningTotal,
			Tipped:          tipped,
			Exact:           exact,
			CorrectWinner:   correctWinner,
			CorrectTotalGoals: correctTotalGoals,
			CorrectGoalDiff: correctGoalDiff,
		})
	}

	start := 0
	if len(events) > limit {
		start = len(events) - limit
	}
	recent := make([]ProgressEvent, 0, len(events)-start)
	for i := len(events) - 1; i >= start; i-- {
		recent = append(recent, events[i])
	}
	// last-3 points: a fixed recent window for the trend chip, independent of `limit`.
	last5Points := 0
	for i := len(events) - 1; i >= 0 && i >= len(events)-3; i-- {
		last5Points += events[i].Points
	}

	return map[string]any{
		"league": map[string]any{
			"id":   league.Id,
			"name": league.GetString("name"),
		},
		"summary": ProgressSummary{
			TipsPoints:        runningTotal,
			Last5Points:       last5Points,
			FinishedMatches:   len(finished),
			TippedFinished:    tippedFinished,
			ExactScores:       exactScores,
			MatchesWithPoints: matchesWithPoints,
			BestPoints:        bestPoints,
		},
		"events": recent,
	}, nil
}
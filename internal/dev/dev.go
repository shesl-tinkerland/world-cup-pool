// Package dev is a test harness, only active when WMP_DEV=1. It can pin a
// virtual clock and simulate match results up to a timestamp, reusing the
// exact production paths (ApplyResult -> ResolveBracket -> Recompute) so the
// simulation also exercises the real logic. The mutating routes are not
// registered unless WMP_DEV=1, so it can never run in production.
package dev

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
	"github.com/oyvhov/world-cup-pool/internal/football"
	"github.com/oyvhov/world-cup-pool/internal/forecast"
	wmleagues "github.com/oyvhov/world-cup-pool/internal/leagues"
	"github.com/oyvhov/world-cup-pool/internal/scoring"
	wmsync "github.com/oyvhov/world-cup-pool/internal/sync"
	"github.com/oyvhov/world-cup-pool/internal/tips"
)

var botNames = []string{
	"Bot Alex", "Bot Robin", "Bot Sam", "Bot Casey", "Bot Jordan",
	"Bot Riley", "Bot Quinn", "Bot Skylar", "Bot Morgan", "Bot Drew",
	"Bot Pat", "Bot Lee", "Bot Noor", "Bot Kai", "Bot Remy",
}

var botChatLines = []string{
	"Dette ser lovande ut.",
	"Eg star ved tipset mitt.",
	"Her luktar det ekstraomgangar.",
	"Stor sjanse for overrasking no.",
	"Den scoringa der snudde alt.",
	"Eg jaktar tabelltoppen no.",
	"Dette var smart tippa.",
	"No vil eg sjå fleire mal.",
	"Forsvaret held ikkje lenge til.",
	"Chatten vaknar endeleg til liv.",
}

const leagueMessagesCollection = "league_messages"
const devBotEmailSuffix = "@dev.local"

var errNoBotPlayers = errors.New("no bot players found in the selected private league(s)")

// Match windows: a result is "finished" once sim time passes kickoff+window,
// "live" between kickoff and that, otherwise still scheduled.
const (
	groupWindow = 120 * time.Minute // 90' + half-time + stoppage buffer
	koWindow    = 140 * time.Minute // + extra time + penalties
)

func Enabled() bool { return os.Getenv("WMP_DEV") == "1" }

func intp(v int) *int { return &v }

// rngFor returns a deterministic RNG for a match so re-advancing to the same
// timestamp yields stable results.
func rngFor(extID string) *rand.Rand {
	h := fnv.New64a()
	h.Write([]byte(extID))
	return rand.New(rand.NewSource(int64(h.Sum64())))
}

func windowFor(stage string) time.Duration {
	if stage == "group" {
		return groupWindow
	}
	return koWindow
}

func isGlobalLeague(app core.App, leagueID string) bool {
	if leagueID == "" {
		return false
	}
	league, err := app.FindRecordById("leagues", leagueID)
	if err != nil {
		return false
	}
	return league.GetString("inviteCode") == wmleagues.GlobalInviteCode
}

// resetMatch returns a match to its pre-result state.
func resetMatch(m *core.Record) {
	m.Set("status", "scheduled")
	for _, f := range []string{"ftHome", "ftAway", "etHome", "etAway", "penHome", "penAway"} {
		m.Set(f, 0)
	}
	m.Set("penWinner", "")
	m.Set("advancer", "")
	m.Set("finalizedAt", "")
	if m.GetString("stage") != "group" {
		// Knockout teams are only filled by the resolver from results.
		m.Set("homeTeam", "")
		m.Set("awayTeam", "")
	}
}

// simulate brings all matches to the state they'd be in at simNow.
func simulate(app core.App, simNow time.Time) error {
	all, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	if err != nil {
		return err
	}
	// Reset anything now in the future (supports moving the clock back).
	for _, m := range all {
		if simNow.Before(m.GetDateTime("kickoff").Time()) {
			resetMatch(m)
			if err := app.Save(m); err != nil {
				return err
			}
		}
	}

	// Converge: resolve the bracket, finalize/mark-live everything due, repeat
	// until stable (knockout matches need their feeders finished first).
	for pass := 0; pass < 12; pass++ {
		if err := wmsync.ResolveBracket(app); err != nil {
			return err
		}
		matches, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
		if err != nil {
			return err
		}
		changed := false
		for _, m := range matches {
			if m.GetString("status") == "finished" {
				continue
			}
			ko := m.GetDateTime("kickoff").Time()
			if simNow.Before(ko) {
				continue
			}
			if simNow.Before(ko.Add(windowFor(m.GetString("stage")))) {
				if m.GetString("status") != "live" {
					m.Set("status", "live")
					m.Set("finalizedAt", "")
					if err := app.Save(m); err != nil {
						return err
					}
					changed = true
				}
				continue
			}
			// Finished. Knockout needs both teams resolved first.
			if m.GetString("stage") != "group" &&
				(m.GetString("homeTeam") == "" || m.GetString("awayTeam") == "") {
				continue
			}
			r := rngFor(m.GetString("extId"))
			if m.GetString("stage") == "group" {
				wmsync.ApplyResult(m, "finished",
					intp(r.Intn(5)), intp(r.Intn(5)), nil, nil, nil, nil)
			} else {
				h, a := r.Intn(4), r.Intn(4)
				if h != a {
					wmsync.ApplyResult(m, "finished",
						intp(h), intp(a), nil, nil, nil, nil)
				} else {
					// Drawn at 90' -> decided in extra time.
					wmsync.ApplyResult(m, "finished",
						intp(h), intp(a),
						intp(h+1), intp(a), nil, nil)
				}
			}
			if err := app.Save(m); err != nil {
				return err
			}
			changed = true
		}
		if !changed {
			break
		}
	}
	return scoring.Recompute(app)
}

// makeBots creates `count` bot users, each with a complete consistent
// Forecast and a Tip on every match, joined to the given leagues. Uses the
// dev-only validation bypass so it works even after the clock is advanced.
func makeBots(app core.App, count int, leagueIDs []string) ([]string, error) {
	usersCol, err := app.FindCollectionByNameOrId("users")
	if err != nil {
		return nil, err
	}
	fcCol, err := app.FindCollectionByNameOrId("forecasts")
	if err != nil {
		return nil, err
	}
	tipsCol, err := app.FindCollectionByNameOrId("tips")
	if err != nil {
		return nil, err
	}
	lmCol, err := app.FindCollectionByNameOrId("league_members")
	if err != nil {
		return nil, err
	}
	matches, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	if err != nil {
		return nil, err
	}

	forecast.SetBypass(true)
	tips.SetBypass(true)
	defer forecast.SetBypass(false)
	defer tips.SetBypass(false)

	created := []string{}
	used := map[string]int{}
	for i := 0; i < count; i++ {
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(i*7919)))
		name := botNames[rng.Intn(len(botNames))]
		if used[name]++; used[name] > 1 {
			name = fmt.Sprintf("%s %d", name, used[name])
		}

		u := core.NewRecord(usersCol)
		u.SetEmail(fmt.Sprintf("bot-%d@dev.local", time.Now().UnixNano()+int64(i)))
		u.SetRandomPassword()
		u.Set("name", name)
		u.Set("verified", true)
		if err := app.Save(u); err != nil {
			return created, err
		}

		order, thirds, bracket, err := scoring.RandomForecast(app, rng)
		if err != nil {
			return created, err
		}
		f := core.NewRecord(fcCol)
		f.Set("user", u.Id)
		f.Set("groupOrder", order)
		f.Set("thirdQualifiers", thirds)
		f.Set("bracket", bracket)
		if err := app.Save(f); err != nil {
			return created, err
		}

		for _, m := range matches {
			t := core.NewRecord(tipsCol)
			t.Set("user", u.Id)
			t.Set("match", m.Id)
			t.Set("ftHome", rng.Intn(5))
			t.Set("ftAway", rng.Intn(5))
			if err := app.Save(t); err != nil {
				return created, err
			}
		}

		for _, lid := range leagueIDs {
			lm := core.NewRecord(lmCol)
			lm.Set("league", lid)
			lm.Set("user", u.Id)
			lm.Set("role", "member")
			if err := app.Save(lm); err != nil {
				return created, err
			}
		}
		created = append(created, name)
	}
	return created, nil
}

func privateLeagueIDs(app core.App, userID, explicitLeagueID string) ([]string, error) {
	if explicitLeagueID != "" {
		if isGlobalLeague(app, explicitLeagueID) {
			return nil, nil
		}
		return []string{explicitLeagueID}, nil
	}

	mems, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	leagueIDs := make([]string, 0, len(mems))
	for _, m := range mems {
		leagueID := m.GetString("league")
		if seen[leagueID] || isGlobalLeague(app, leagueID) {
			continue
		}
		seen[leagueID] = true
		leagueIDs = append(leagueIDs, leagueID)
	}
	return leagueIDs, nil
}

func botMembersInLeague(app core.App, leagueID string) ([]*core.Record, error) {
	members, err := app.FindRecordsByFilter("league_members",
		"league = {:l}", "", 0, 0, map[string]any{"l": leagueID})
	if err != nil {
		return nil, err
	}
	bots := make([]*core.Record, 0, len(members))
	seen := map[string]bool{}
	for _, member := range members {
		userID := member.GetString("user")
		if seen[userID] {
			continue
		}
		user, err := app.FindRecordById("users", userID)
		if err != nil {
			continue
		}
		email := strings.ToLower(strings.TrimSpace(user.GetString("email")))
		if !strings.HasSuffix(email, devBotEmailSuffix) {
			continue
		}
		seen[userID] = true
		bots = append(bots, user)
	}
	return bots, nil
}

func sendBotChat(app core.App, count int, leagueIDs []string) (int, error) {
	if count <= 0 {
		count = 1
	}
	if count > 50 {
		count = 50
	}
	col, err := app.FindCollectionByNameOrId(leagueMessagesCollection)
	if err != nil {
		return 0, err
	}

	total := 0
	for idx, leagueID := range leagueIDs {
		bots, err := botMembersInLeague(app, leagueID)
		if err != nil {
			return total, err
		}
		if len(bots) == 0 {
			continue
		}
		rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64((idx+1)*3571)))
		for i := 0; i < count; i++ {
			rec := core.NewRecord(col)
			rec.Set("league", leagueID)
			rec.Set("user", bots[rng.Intn(len(bots))].Id)
			rec.Set("text", botChatLines[rng.Intn(len(botChatLines))])
			if err := app.Save(rec); err != nil {
				return total, err
			}
			total++
		}
	}
	if total == 0 {
		return 0, errNoBotPlayers
	}
	return total, nil
}

func parseTimestamp(s string) (time.Time, bool) {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), true
		}
	}
	return time.Time{}, false
}

func state(app core.App) map[string]any {
	now := clock.Now(app)
	sim, isSim := clock.Sim(app)
	out := map[string]any{
		"dev":       Enabled(),
		"now":       now.UnixMilli(),
		"simulated": isSim,
	}
	if isSim {
		out["simTime"] = sim.Format(time.RFC3339)
	}
	return out
}

// Register wires /api/now (always) and, only when WMP_DEV=1, the dev
// mutating endpoints.
func Register(app core.App, se *core.ServeEvent) {
	se.Router.GET("/api/now", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, state(app))
	})

	if !Enabled() {
		return
	}

	g := se.Router.Group("/api/dev")
	g.Bind(apis.RequireAuth())

	g.GET("/state", func(e *core.RequestEvent) error {
		return e.JSON(http.StatusOK, state(app))
	})

	// GET /api/dev/apicheck?season=2026 — validate the live API: plan/quota,
	// schema parse, team-name mapping vs our seed, our-match coverage, and
	// the status/ET/penalty distribution. Point season at a finished World
	// Cup (e.g. 2022) to exercise the results path before 2026 kicks off.
	g.GET("/apicheck", func(e *core.RequestEvent) error {
		key := os.Getenv("API_FOOTBALL_KEY")
		if key == "" {
			return e.JSON(400, map[string]string{"error": "API_FOOTBALL_KEY not set"})
		}
		yr := 2026
		if s := e.Request.URL.Query().Get("season"); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				yr = n
			}
		}
		ctx, cancel := context.WithTimeout(e.Request.Context(), 30*time.Second)
		defer cancel()
		client := football.New(key)
		out := map[string]any{}
		if st, err := client.Status(ctx); err == nil {
			out["account"] = st
		} else {
			out["statusError"] = err.Error()
		}
		rep, err := wmsync.APICheck(ctx, app, client, yr)
		if err != nil {
			out["error"] = err.Error()
			return e.JSON(502, out)
		}
		for k, v := range rep {
			out[k] = v
		}
		return e.JSON(http.StatusOK, out)
	})

	// POST /api/dev/advance { "timestamp": "2026-06-20T16:20:00Z" }
	g.POST("/advance", func(e *core.RequestEvent) error {
		var body struct {
			Timestamp string `json:"timestamp"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(400, map[string]string{"error": err.Error()})
		}
		ts, ok := parseTimestamp(body.Timestamp)
		if !ok {
			return e.JSON(400, map[string]string{"error": "bad timestamp"})
		}
		scoring.SetAutoRecomputeSuspended(true)
		defer scoring.SetAutoRecomputeSuspended(false)
		if err := app.RunInTransaction(func(tx core.App) error {
			if err := clock.Set(tx, ts); err != nil {
				return err
			}
			return simulate(tx, ts)
		}); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, state(app))
	})

	// POST /api/dev/matches/{id}/result { "status": "1H", "ftHome": 1, "ftAway": 0 }
	// lets a regular dev user push a match through live-score states while
	// another tab stays open and listens for PocketBase realtime updates.
	g.POST("/matches/{id}/result", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		rec, err := app.FindRecordById("matches", id)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "match not found"})
		}

		var body struct {
			FTHome  *int   `json:"ftHome"`
			FTAway  *int   `json:"ftAway"`
			ETHome  *int   `json:"etHome"`
			ETAway  *int   `json:"etAway"`
			PenHome *int   `json:"penHome"`
			PenAway *int   `json:"penAway"`
			Status  string `json:"status"`
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": err.Error()})
		}

		status := body.Status
		if status == "" {
			status = rec.GetString("status")
			if status == "" {
				status = "scheduled"
			}
		}

		wmsync.ApplyResult(rec, status, body.FTHome, body.FTAway, body.ETHome, body.ETAway, body.PenHome, body.PenAway)
		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		if err := wmsync.ResolveBracket(app); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		return e.JSON(http.StatusOK, map[string]any{
			"status": "ok",
			"match": map[string]any{
				"id":          rec.Id,
				"status":      rec.GetString("status"),
				"ftHome":      rec.GetInt("ftHome"),
				"ftAway":      rec.GetInt("ftAway"),
				"etHome":      rec.GetInt("etHome"),
				"etAway":      rec.GetInt("etAway"),
				"penHome":     rec.GetInt("penHome"),
				"penAway":     rec.GetInt("penAway"),
				"advancer":    rec.GetString("advancer"),
				"penWinner":   rec.GetString("penWinner"),
				"finalizedAt": rec.GetString("finalizedAt"),
			},
		})
	})

	// POST /api/dev/bots { "count": 3, "leagueId": "" } — create bot players
	// with a full Forecast + a Tip on every match. Joins the given league, or
	// every league the caller is in if omitted.
	g.POST("/bots", func(e *core.RequestEvent) error {
		var body struct {
			Count    int    `json:"count"`
			LeagueID string `json:"leagueId"`
		}
		_ = e.BindBody(&body)
		if body.Count <= 0 {
			body.Count = 1
		}
		if body.Count > 20 {
			body.Count = 20
		}
		leagueIDs, err := privateLeagueIDs(app, e.Auth.Id, body.LeagueID)
		if err != nil {
			return e.JSON(500, map[string]any{"error": err.Error()})
		}
		names, err := makeBots(app, body.Count, leagueIDs)
		if err != nil {
			return e.JSON(500, map[string]any{"error": err.Error(), "created": names})
		}
		if err := scoring.Recompute(app); err != nil {
			return e.JSON(500, map[string]any{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]any{"created": names})
	})

	// POST /api/dev/bot-chat { "count": 6, "leagueId": "" } — send chat
	// messages from existing bot members into the selected private league, or
	// all of the caller's private leagues when omitted.
	g.POST("/bot-chat", func(e *core.RequestEvent) error {
		var body struct {
			Count    int    `json:"count"`
			LeagueID string `json:"leagueId"`
		}
		_ = e.BindBody(&body)
		leagueIDs, err := privateLeagueIDs(app, e.Auth.Id, body.LeagueID)
		if err != nil {
			return e.JSON(500, map[string]any{"error": err.Error()})
		}
		sent, err := sendBotChat(app, body.Count, leagueIDs)
		if err != nil {
			if errors.Is(err, errNoBotPlayers) {
				return e.JSON(400, map[string]any{"error": err.Error()})
			}
			return e.JSON(500, map[string]any{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, map[string]any{"sent": sent})
	})

	// POST /api/dev/reset — clear all results and the dev clock.
	g.POST("/reset", func(e *core.RequestEvent) error {
		matches, err := app.FindRecordsByFilter("matches", "id != ''", "", 0, 0)
		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		for _, m := range matches {
			resetMatch(m)
			if err := app.Save(m); err != nil {
				return e.JSON(500, map[string]string{"error": err.Error()})
			}
		}
		if err := clock.Clear(app); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		if err := scoring.Recompute(app); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(http.StatusOK, state(app))
	})

	g.GET("/topscorers", func(e *core.RequestEvent) error {
		records, err := app.FindRecordsByFilter("golden_boot_players", "seeded = true || goals > 0", "name", 0, 0)
		if err != nil {
			return err
		}
		var out []map[string]any
		for _, r := range records {
			out = append(out, map[string]any{
				"id":    r.Id,
				"name":  r.GetString("name"),
				"goals": r.GetInt("goals"),
			})
		}
		return e.JSON(http.StatusOK, map[string]any{"players": out})
	})

	g.POST("/topscorers", func(e *core.RequestEvent) error {
		var body struct {
			Players map[string]int `json:"players"` // id -> goals
		}
		if err := e.BindBody(&body); err != nil {
			return err
		}
		records, err := app.FindRecordsByFilter("golden_boot_players", "id != ''", "", 0, 0)
		if err != nil {
			return err
		}
		for _, r := range records {
			if goals, ok := body.Players[r.Id]; ok {
				r.Set("goals", goals)
			}
		}

		sort.Slice(records, func(i, j int) bool {
			if records[i].GetInt("goals") != records[j].GetInt("goals") {
				return records[i].GetInt("goals") > records[j].GetInt("goals")
			}
			return records[i].GetString("name") < records[j].GetString("name")
		})

		rank := 0
		lastGoals := -1
		for i, r := range records {
			g := r.GetInt("goals")
			newRank := 0
			if g > 0 {
				if g != lastGoals {
					rank = i + 1
					lastGoals = g
				}
				newRank = rank
			}

			_, inBody := body.Players[r.Id]
			if inBody || r.GetInt("rank") != newRank {
				r.Set("rank", newRank)
				r.Set("syncedAt", time.Now().UTC())
				if err := app.Save(r); err != nil {
					return err
				}
			}
		}
		return e.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
}

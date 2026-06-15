// Package tips enforces the per-match prediction rules server-side:
//   - a Tip can only be created/edited while now < match.kickoff (lock)
//   - knockout Tips are only allowed once both teams are resolved
//   - the knockout advancer is derived from the phased prediction
//   - other players' Tips are visible only AFTER kickoff and only to people
//     who share a League (the /api/tips/others/{matchId} endpoint)
package tips

import (
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
)

func matchKickoff(m *core.Record) time.Time {
	return m.GetDateTime("kickoff").Time()
}

func isLocked(now, kickoff time.Time) bool {
	return !now.Before(kickoff)
}

func locked(app core.App, m *core.Record) bool {
	return isLocked(clock.Now(app), matchKickoff(m))
}

// bypass lets the dev bot generator insert tips for every match regardless
// of lock / knockout-resolution. Never set in production (dev-only path).
var bypass atomic.Bool

// SetBypass toggles the dev-only validation bypass.
func SetBypass(b bool) { bypass.Store(b) }

// validateAndDerive applies lock + validation and sets the derived advancer.
func validateAndDerive(app core.App, tip *core.Record) error {
	if bypass.Load() {
		return nil
	}
	match, err := app.FindRecordById("matches", tip.GetString("match"))
	if err != nil {
		return apis.NewBadRequestError("unknown match", nil)
	}
	if locked(app, match) {
		return apis.NewBadRequestError("this match is locked (kickoff passed)", nil)
	}

	ftH := tip.GetInt("ftHome")
	ftA := tip.GetInt("ftAway")
	if tip.Get("ftHome") == nil || tip.Get("ftAway") == nil {
		return apis.NewBadRequestError("full-time score is required", nil)
	}
	if ftH < 0 || ftA < 0 || ftH > 99 || ftA > 99 {
		return apis.NewBadRequestError("scores out of range", nil)
	}

	if match.GetString("stage") == "group" {
		tip.Set("etHome", 0)
		tip.Set("etAway", 0)
		tip.Set("penWinner", "")
		tip.Set("advancer", "")
		return nil
	}

	// Knockout.
	home := match.GetString("homeTeam")
	away := match.GetString("awayTeam")
	if home == "" || away == "" {
		return apis.NewBadRequestError("this matchup is not set yet", nil)
	}

	if ftH != ftA {
		if ftH > ftA {
			tip.Set("advancer", home)
		} else {
			tip.Set("advancer", away)
		}
		tip.Set("etHome", 0)
		tip.Set("etAway", 0)
		tip.Set("penWinner", "")
		return nil
	}

	// Drawn after 90' -> extra time required (cumulative >= FT).
	etH := tip.GetInt("etHome")
	etA := tip.GetInt("etAway")
	if tip.Get("etHome") == nil || tip.Get("etAway") == nil {
		return apis.NewBadRequestError("predict the score after extra time", nil)
	}
	if etH < ftH || etA < ftA {
		return apis.NewBadRequestError("extra-time score must include the 90' goals", nil)
	}
	if etH != etA {
		if etH > etA {
			tip.Set("advancer", home)
		} else {
			tip.Set("advancer", away)
		}
		tip.Set("penWinner", "")
		return nil
	}

	// Still level -> penalty winner required.
	pw := tip.GetString("penWinner")
	if pw != home && pw != away {
		return apis.NewBadRequestError("pick who wins the penalty shootout", nil)
	}
	tip.Set("advancer", pw)
	return nil
}

// Register wires the Tip validation hooks and the friends-tips endpoint.
func Register(app core.App, se *core.ServeEvent) {
	app.OnRecordCreate("tips").BindFunc(func(e *core.RecordEvent) error {
		if err := validateAndDerive(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordUpdate("tips").BindFunc(func(e *core.RecordEvent) error {
		if err := validateAndDerive(e.App, e.Record); err != nil {
			return err
		}
		return e.Next()
	})
	app.OnRecordDelete("tips").BindFunc(func(e *core.RecordEvent) error {
		if m, err := e.App.FindRecordById("matches", e.Record.GetString("match")); err == nil && locked(e.App, m) {
			return apis.NewBadRequestError("this match is locked", nil)
		}
		return e.Next()
	})

	// GET /api/tips/scores — the signed-in user's points per match under the
	// default scoring config (for the per-match "+N pt" badge).
	se.Router.GET("/api/tips/scores", func(e *core.RequestEvent) error {
		out := map[string]int{}
		def, err := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		if err == nil {
			rows, _ := app.FindRecordsByFilter("match_scores",
				"user = {:u} && config = {:c}", "", 0, 0,
				map[string]any{"u": e.Auth.Id, "c": def.Id})
			for _, r := range rows {
				out[r.GetString("match")] = r.GetInt("points")
			}
		}
		return e.JSON(http.StatusOK, map[string]any{"scores": out})
	}).Bind(apis.RequireAuth())

	// GET /api/tips/others/{matchId} — all league members' Tips for a match,
	// but only after kickoff. The requesting user's own tip is included first
	// (isMe: true). Each row includes the points earned under the default config.
	se.Router.GET("/api/tips/others/{matchId}", func(e *core.RequestEvent) error {
		matchID := e.Request.PathValue("matchId")
		match, err := app.FindRecordById("matches", matchID)
		if err != nil {
			return apis.NewNotFoundError("match not found", nil)
		}
		if !locked(app, match) {
			// Not started: never reveal anyone's picks.
			return e.JSON(http.StatusOK, map[string]any{"locked": false, "tips": []any{}})
		}

		// Default scoring config for points display.
		def, _ := app.FindFirstRecordByFilter("scoring_configs", "isDefault = true")
		pointsFor := func(uid string) int {
			if def == nil {
				return -1
			}
			s, err := app.FindFirstRecordByFilter("match_scores",
				"user = {:u} && match = {:m} && config = {:c}",
				map[string]any{"u": uid, "m": matchID, "c": def.Id})
			if err != nil {
				return -1
			}
			return s.GetInt("points")
		}

		leagueID := strings.TrimSpace(e.Request.URL.Query().Get("leagueId"))
		coMembers, err := visibleLeagueUserIDs(app, e.Auth.Id, leagueID)
		if err != nil {
			return err
		}
		allTips, err := app.FindRecordsByFilter("tips",
			"match = {:m}", "", 0, 0, map[string]any{"m": matchID})
		if err != nil {
			return err
		}

		var myRow *map[string]any
		otherRows := make([]map[string]any, 0, len(allTips))
		for _, t := range allTips {
			uid := t.GetString("user")
			isMe := uid == e.Auth.Id
			if !isMe && !coMembers[uid] {
				continue
			}
			u, err := app.FindRecordById("users", uid)
			if err != nil {
				continue
			}
			row := map[string]any{
				"userId":    uid,
				"name":      u.GetString("name"),
				"isMe":      isMe,
				"ftHome":    t.GetInt("ftHome"),
				"ftAway":    t.GetInt("ftAway"),
				"etHome":    t.GetInt("etHome"),
				"etAway":    t.GetInt("etAway"),
				"penWinner": t.GetString("penWinner"),
				"advancer":  t.GetString("advancer"),
				"points":    pointsFor(uid),
			}
			if isMe {
				r := row
				myRow = &r
			} else {
				otherRows = append(otherRows, row)
			}
		}
		out := make([]map[string]any, 0, len(otherRows)+1)
		if myRow != nil {
			out = append(out, *myRow)
		}
		out = append(out, otherRows...)
		return e.JSON(http.StatusOK, map[string]any{"locked": true, "tips": out})
	}).Bind(apis.RequireAuth())

	// GET /api/tips/crowd/{matchId} — global tip distribution (Home/Draw/Away)
	// across ALL users for a single match. Revealed only after kickoff so we
	// never leak picks before tips lock.
	se.Router.GET("/api/tips/crowd/{matchId}", func(e *core.RequestEvent) error {
		matchID := e.Request.PathValue("matchId")
		match, err := app.FindRecordById("matches", matchID)
		if err != nil {
			return apis.NewNotFoundError("match not found", nil)
		}
		if !locked(app, match) {
			return e.JSON(http.StatusOK, map[string]any{"locked": false})
		}
		dist, err := crowdDistribution(app, match)
		if err != nil {
			return err
		}
		dist["locked"] = true
		return e.JSON(http.StatusOK, dist)
	}).Bind(apis.RequireAuth())
}

// crowdDistribution aggregates every tip for the given match into
// Home / Draw / Away buckets and returns counts plus integer percentages
// that always sum to 100 (largest bucket absorbs any rounding drift).
//
// Group stage: outcome = sign(ftHome - ftAway).
// Knockout: outcome = advancer == homeTeam ? home : away (no draws possible).
func crowdDistribution(app core.App, match *core.Record) (map[string]any, error) {
	tips, err := app.FindRecordsByFilter("tips",
		"match = {:m}", "", 0, 0, map[string]any{"m": match.Id})
	if err != nil {
		return nil, err
	}
	stage := match.GetString("stage")
	isKO := stage != "group"
	home := match.GetString("homeTeam")
	away := match.GetString("awayTeam")
	var hC, dC, aC int
	for _, t := range tips {
		if isKO {
			adv := t.GetString("advancer")
			switch adv {
			case home:
				hC++
			case away:
				aC++
			}
			continue
		}
		// Group stage. Skip malformed rows (missing FT score).
		if t.Get("ftHome") == nil || t.Get("ftAway") == nil {
			continue
		}
		ftH := t.GetInt("ftHome")
		ftA := t.GetInt("ftAway")
		switch {
		case ftH > ftA:
			hC++
		case ftH < ftA:
			aC++
		default:
			dC++
		}
	}
	total := hC + dC + aC
	hP, dP, aP := pctSplit(hC, dC, aC, total)
	return map[string]any{
		"total": total,
		"isKO":  isKO,
		"outcomes": map[string]any{
			"home": map[string]any{"count": hC, "pct": hP},
			"draw": map[string]any{"count": dC, "pct": dP},
			"away": map[string]any{"count": aC, "pct": aP},
		},
	}, nil
}

// pctSplit returns integer percentages for (home, draw, away) that sum to 100
// when total > 0; returns zeros when total == 0. The largest raw bucket
// absorbs any rounding drift so the bar always renders cleanly.
func pctSplit(h, d, a, total int) (int, int, int) {
	if total <= 0 {
		return 0, 0, 0
	}
	hP := (h * 100) / total
	dP := (d * 100) / total
	aP := (a * 100) / total
	diff := 100 - (hP + dP + aP)
	if diff != 0 {
		// Give the leftover to whichever bucket has the most votes.
		switch {
		case h >= d && h >= a:
			hP += diff
		case d >= h && d >= a:
			dP += diff
		default:
			aP += diff
		}
	}
	return hP, dP, aP
}

// visibleLeagueUserIDs returns the set of user ids whose tips the given user
// may see. If leagueID is set, visibility is limited to that one league after
// verifying the requester is a member; otherwise it falls back to all shared
// leagues for backwards-compatible callers.
func visibleLeagueUserIDs(app core.App, userID, leagueID string) (map[string]bool, error) {
	if leagueID != "" {
		if _, err := app.FindFirstRecordByFilter("league_members",
			"league = {:l} && user = {:u}",
			map[string]any{"l": leagueID, "u": userID}); err != nil {
			return nil, apis.NewForbiddenError("not a member of this league", nil)
		}
		members, err := app.FindRecordsByFilter("league_members",
			"league = {:l}", "", 0, 0, map[string]any{"l": leagueID})
		if err != nil {
			return nil, err
		}
		out := map[string]bool{}
		for _, m := range members {
			out[m.GetString("user")] = true
		}
		return out, nil
	}

	mine, err := app.FindRecordsByFilter("league_members",
		"user = {:u}", "", 0, 0, map[string]any{"u": userID})
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, lm := range mine {
		peers, err := app.FindRecordsByFilter("league_members",
			"league = {:l}", "", 0, 0, map[string]any{"l": lm.GetString("league")})
		if err != nil {
			return nil, err
		}
		for _, p := range peers {
			out[p.GetString("user")] = true
		}
	}
	return out, nil
}

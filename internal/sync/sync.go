// Package sync keeps the matches collection up to date: a cron job pulls
// results from API-Football (one request per run), a superuser endpoint
// forces a refresh, and another superuser endpoint applies manual results
// when the provider is wrong or no API key is configured.
package sync

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/clock"
	"github.com/oyvhov/world-cup-pool/internal/football"
)

const (
	intervalLive = 2 * time.Minute
	intervalSoon = 5 * time.Minute
	intervalIdle = 30 * time.Minute
	providerKickoffGrace = 3 * time.Hour
)

var syncMutex sync.Mutex

// liveStatuses matches the set that football.Fixture.Live() returns true for,
// plus the legacy plain `live` value still present in older test data.
var liveStatuses = dbx.NewExp("status IN ('live','1H','2H','HT','ET','BT','P','LIVE','INT')")
var liveStatusFilter = "status = 'live' || status = '1H' || status = '2H' || status = 'HT' || status = 'ET' || status = 'BT' || status = 'P' || status = 'LIVE' || status = 'INT'"

const matchEventsCollection = "match_events"

func kickoffStillNeedsSoonPolling(now, kickoff time.Time) bool {
	if kickoff.After(now.Add(2 * time.Hour)) {
		return false
	}
	return kickoff.After(now.Add(-providerKickoffGrace))
}

func nextSyncInterval(app core.App) time.Duration {
	now := clock.Now(app)

	live, _ := app.CountRecords("matches", liveStatuses)
	if live > 0 {
		return intervalLive
	}

	scheduled, _ := app.FindRecordsByFilter("matches", "status = 'scheduled'", "kickoff", 0, 0)
	for _, match := range scheduled {
		kickoff := match.GetDateTime("kickoff").Time()
		if kickoff.After(now.Add(2 * time.Hour)) {
			break
		}
		// Keep polling tightly for a grace window after kickoff: providers can lag a
		// few minutes before flipping NS/scheduled to 1H, and without this the loop
		// would fall back to 30 minutes exactly when the match has just started.
		if kickoffStillNeedsSoonPolling(now, kickoff) {
			return intervalSoon
		}
	}

	return intervalIdle
}

func startSmartLoop(app core.App, run func(context.Context) error) {
	go func() {
		for {
			interval := nextSyncInterval(app)
			log.Printf("[sync] next sync in %v", interval)
			time.Sleep(interval)

			syncMutex.Lock()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := run(ctx); err != nil {
				log.Printf("[sync] %v", err)
			}
			cancel()
			syncMutex.Unlock()
		}
	}()
}

// nameAliases maps API-Football names that differ from the openfootball seed
// names to the seeded team name. This only matters for the optional
// API-Football results path (a paid plan): the default openfootball sync maps
// matches by the deterministic ExtID and needs no aliasing at all.
//
// Both the provider name and our seed name are run through canonName, so an
// entry unifies the two spellings onto one canonical key. Entries are inert
// until the provider actually sends that exact spelling, so unused ones are
// harmless. Verify real coverage with APICheck (which reports unmappedTeams)
// before trusting the API-Football path — these are best-effort for the known
// FIFA/provider naming differences among the 2026 participants.
var nameAliases = map[string]string{
	football.NormalizeName("Korea Republic"):               football.NormalizeName("South Korea"),
	football.NormalizeName("Czechia"):                      football.NormalizeName("Czech Republic"),
	football.NormalizeName("USA"):                          football.NormalizeName("United States"),
	football.NormalizeName("IR Iran"):                      football.NormalizeName("Iran"),
	football.NormalizeName("Bosnia and Herzegovina"):       football.NormalizeName("Bosnia & Herzegovina"),
	football.NormalizeName("Côte d'Ivoire"):                football.NormalizeName("Ivory Coast"),
	football.NormalizeName("Congo DR"):                     football.NormalizeName("DR Congo"),
	football.NormalizeName("Democratic Republic of Congo"): football.NormalizeName("DR Congo"),
	football.NormalizeName("Cape Verde Islands"):           football.NormalizeName("Cape Verde"),
	football.NormalizeName("Türkiye"):                      football.NormalizeName("Turkey"),
}

func canonName(s string) string {
	n := football.NormalizeName(s)
	if a, ok := nameAliases[n]; ok {
		return a
	}
	return n
}

// pickProvider decides the live-results source: API-Football when its key can
// actually reach WC2026 (a paid plan — free can't), otherwise the free
// openfootball JSON. RESULTS_SOURCE=apifootball|openfootball forces it.
// Returns a label and a sync function (nil = none / manual-only).
func pickProvider(app core.App) (string, func(context.Context) error, *football.Client) {
	key := os.Getenv("API_FOOTBALL_KEY")
	mode := os.Getenv("RESULTS_SOURCE")
	apiClient := football.New(key)

	apiFn := func(ctx context.Context) error {
		return SyncOnce(ctx, app, apiClient)
	}
	ofFn := func(ctx context.Context) error {
		return openfootballSync(ctx, app)
	}

	if mode == "openfootball" {
		return "openfootball", ofFn, nil
	}
	if mode == "apifootball" {
		if key == "" {
			return "", nil, nil
		}
		return "api-football", apiFn, apiClient
	}
	// auto: prefer API-Football only if the key can actually fetch 2026.
	if key != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if fx, err := apiClient.Fixtures(ctx); err == nil && len(fx) > 0 {
			return "api-football", apiFn, apiClient
		}
		log.Printf("[sync] API-Football key can't reach WC2026 (free plan?) — using openfootball")
	}
	return "openfootball", ofFn, nil
}

var lastManualCheck atomic.Value

// Register wires the live-results cron + manual override endpoints.
// Called from the OnServe hook.
func Register(app core.App, se *core.ServeEvent) {
	source, run, eventClient := pickProvider(app)
	if run != nil && eventClient != nil {
		baseRun := run
		run = func(ctx context.Context) error {
			if err := baseRun(ctx); err != nil {
				return err
			}
			// Give event sync its own budget — independent of how long baseRun took.
			evCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := syncLiveEvents(evCtx, app, eventClient); err != nil {
				log.Printf("[sync] live events: %v", err)
			}
			return nil
		}
	}

	if run != nil {
		startSmartLoop(app, run)
		log.Printf("[sync] auto-sync enabled via %s (dynamic interval)", source)
	} else {
		log.Printf("[sync] no results source — manual override only")
	}

	// User-triggered sync (rate-limited to 30s). Any authenticated user may call this.
	se.Router.POST("/api/sync/check", func(e *core.RequestEvent) error {
		if run == nil {
			return e.JSON(400, map[string]string{"error": "no results source configured"})
		}
		if last := lastManualCheck.Load(); last != nil {
			if time.Since(last.(time.Time)) < 30*time.Second {
				return e.JSON(429, map[string]string{"error": "rate limited"})
			}
		}
		lastManualCheck.Store(time.Now())

		syncMutex.Lock()
		ctx, cancel := context.WithTimeout(e.Request.Context(), 30*time.Second)
		err := run(ctx)
		cancel()
		syncMutex.Unlock()

		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]string{"status": "ok"})
	})

	// Force a sync now (superuser).
	se.Router.POST("/api/sync/refresh", func(e *core.RequestEvent) error {
		if run == nil {
			return e.JSON(400, map[string]string{"error": "no results source configured"})
		}
		syncMutex.Lock()
		ctx, cancel := context.WithTimeout(e.Request.Context(), 30*time.Second)
		err := run(ctx)
		cancel()
		syncMutex.Unlock()
		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]string{"status": "ok", "source": source})
	}).Bind(apis.RequireSuperuserAuth())

	// Manual result override (superuser). Body: ftHome,ftAway,etHome,etAway,
	// penHome,penAway (ints, et/pen optional) and status.
	se.Router.POST("/api/admin/matches/{id}/result", func(e *core.RequestEvent) error {
		id := e.Request.PathValue("id")
		rec, err := app.FindRecordById("matches", id)
		if err != nil {
			return e.JSON(404, map[string]string{"error": "match not found"})
		}
		var body struct {
			FTHome, FTAway   *int
			ETHome, ETAway   *int
			PenHome, PenAway *int
			Status           string
		}
		if err := e.BindBody(&body); err != nil {
			return e.JSON(400, map[string]string{"error": err.Error()})
		}
		applyResult(rec, body.Status, body.FTHome, body.FTAway, body.ETHome, body.ETAway, body.PenHome, body.PenAway)
		if err := app.Save(rec); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		if err := ResolveBracket(app); err != nil {
			log.Printf("[sync] resolve after manual override: %v", err)
		}
		return e.JSON(200, map[string]any{"status": "ok", "id": rec.Id})
	}).Bind(apis.RequireSuperuserAuth())

	se.Router.GET("/api/live/events", func(e *core.RequestEvent) error {
		liveMatches, err := app.FindRecordsByFilter("matches", liveStatusFilter, "kickoff", 0, 0)
		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}

		byMatch := map[string][]map[string]any{}
		if len(liveMatches) == 0 {
			return e.JSON(200, map[string]any{"events": byMatch})
		}

		filterParts := make([]string, 0, len(liveMatches))
		params := map[string]any{}
		resolvers := make(map[string]func(string) string, len(liveMatches))
		for i, match := range liveMatches {
			byMatch[match.Id] = []map[string]any{}
			resolvers[match.Id] = eventTeamIDResolver(app, match)
			key := fmt.Sprintf("m%d", i)
			filterParts = append(filterParts, "match = {:"+key+"}")
			params[key] = match.Id
		}

		events, err := app.FindRecordsByFilter(matchEventsCollection,
			strings.Join(filterParts, " || "), "elapsed,extra,created", 0, 0, params)
		if err != nil {
			// During rolling deploys the frontend may ask before the migration is present.
			return e.JSON(200, map[string]any{"events": byMatch})
		}
		for _, ev := range events {
			mid := ev.GetString("match")
			row := liveEventRecordJSON(ev)
			if resolve := resolvers[mid]; resolve != nil {
				row["teamId"] = resolve(ev.GetString("team"))
			}
			byMatch[mid] = append(byMatch[mid], row)
		}
		return e.JSON(200, map[string]any{"events": byMatch})
	}).Bind(apis.RequireAuth())

	// Per-match events for one fixture, in timeline order. Unlike /api/live/events
	// this is not gated to live matches, so the frontend can show a goals/red-card
	// summary after full-time — the match_events rows persist once captured.
	se.Router.GET("/api/matches/{id}/events", func(e *core.RequestEvent) error {
		matchID := e.Request.PathValue("id")
		if matchID == "" {
			return e.JSON(400, map[string]string{"error": "missing match id"})
		}
		events, err := app.FindRecordsByFilter(matchEventsCollection,
			"match = {:match}", "elapsed,extra,created", 0, 0, map[string]any{"match": matchID})
		if err != nil {
			// Collection missing mid-deploy, or an unknown id — either way there is
			// nothing to show, so degrade to empty and let the card show just the score.
			return e.JSON(200, map[string]any{"events": []any{}})
		}
		resolve := func(string) string { return "" }
		if match, err := app.FindRecordById("matches", matchID); err == nil {
			resolve = eventTeamIDResolver(app, match)
		}
		out := make([]map[string]any, 0, len(events))
		for _, ev := range events {
			row := liveEventRecordJSON(ev)
			row["teamId"] = resolve(ev.GetString("team"))
			out = append(out, row)
		}
		return e.JSON(200, map[string]any{"events": out})
	}).Bind(apis.RequireAuth())
}

// SyncOnce pulls all fixtures once and updates matched records.
func SyncOnce(ctx context.Context, app core.App, client *football.Client) error {
	fixtures, err := client.Fixtures(ctx)
	if err != nil {
		return fmt.Errorf("fetch fixtures: %w", err)
	}

	matches, err := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	if err != nil {
		return fmt.Errorf("load matches: %w", err)
	}

	// Index our matches by the normalized team-name pair (group stage) so we
	// can line them up with provider fixtures regardless of fixture ids.
	teamName := map[string]string{} // teamId -> normalized name
	teams, _ := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0)
	for _, t := range teams {
		teamName[t.Id] = canonName(t.GetString("name"))
	}

	byPair := map[string]*core.Record{}
	for _, mrec := range matches {
		h := teamName[mrec.GetString("homeTeam")]
		a := teamName[mrec.GetString("awayTeam")]
		if h != "" && a != "" {
			byPair[h+"|"+a] = mrec
		}
	}

	updated := 0
	for _, f := range fixtures {
		key := canonName(f.HomeName) + "|" + canonName(f.AwayName)
		rec, ok := byPair[key]
		if !ok {
			// KO matches resolve via ResolveBracket; unmatched group names
			// usually mean an alias is missing. Log only actionable live/final
			// misses so the 30-minute cron does not spam future KO placeholders.
			if f.Live() || f.Finished() {
				log.Printf("[sync] unmatched API-Football fixture %q vs %q (status=%s)", f.HomeName, f.AwayName, f.Status)
			}
			continue
		}
		status := rec.GetString("status")
		if status == "" {
			status = "scheduled"
		}
		if f.Live() {
			status = f.Status
		}
		if f.Finished() {
			status = "finished"
		}
		// API `score.extratime` is the ET-only delta; our model (and Tips /
		// scoring) use the cumulative after-120 score, which is exactly the
		// provider `goals` field once a match has gone to extra time.
		var etH, etA *int
		if f.ETHome != nil || f.ETAway != nil {
			etH, etA = f.HomeGoals, f.AwayGoals
		}
		fixtureIDChanged := f.ID != 0 && rec.GetInt("apiFootballFixtureId") != f.ID
		resultApplied := resultAlreadyApplied(rec, status, f.FTHome, f.FTAway, etH, etA, f.PenHome, f.PenAway)
		if resultApplied {
			if !fixtureIDChanged {
				continue
			}
			rec.Set("apiFootballFixtureId", f.ID)
			if app.Save(rec) == nil {
				updated++
			}
			continue
		}
		if f.ID != 0 {
			rec.Set("apiFootballFixtureId", f.ID)
		}
		applyResult(rec, status, f.FTHome, f.FTAway, etH, etA, f.PenHome, f.PenAway)
		if app.Save(rec) == nil {
			updated++
		}
	}

	if err := ResolveBracket(app); err != nil {
		log.Printf("[sync] resolve bracket: %v", err)
	}
	log.Printf("[sync] fixtures=%d updated=%d", len(fixtures), updated)
	return nil
}

func syncLiveEvents(ctx context.Context, app core.App, client *football.Client) error {
	eventsCol, err := app.FindCollectionByNameOrId(matchEventsCollection)
	if err != nil {
		return nil
	}

	liveMatches, err := app.FindRecordsByFilter("matches", liveStatusFilter, "kickoff", 0, 0)
	if err != nil {
		return err
	}

	for _, match := range liveMatches {
		if ctx.Err() != nil {
			log.Printf("[sync] live events: context expired after %d/%d matches", len(liveMatches), len(liveMatches))
			break
		}

		fixtureID := match.GetInt("apiFootballFixtureId")
		if fixtureID == 0 {
			continue
		}

		events, err := client.FixtureEvents(ctx, fixtureID)
		if err != nil {
			log.Printf("[sync] events for match %s fixture %d: %v", match.Id, fixtureID, err)
			continue
		}

		for _, ev := range events {
			key := eventProviderKey(fixtureID, ev)
			existing, _ := app.FindFirstRecordByFilter(matchEventsCollection,
				"match = {:m} && providerKey = {:k}",
				map[string]any{"m": match.Id, "k": key})
			if existing != nil {
				// Update mutable fields (e.g. detail changes after VAR review).
				setLiveEventFields(existing, ev)
				if err := app.Save(existing); err != nil {
					log.Printf("[sync] update event for match %s: %v", match.Id, err)
				}
				continue
			}

			rec := core.NewRecord(eventsCol)
			rec.Set("match", match.Id)
			rec.Set("providerKey", key)
			setLiveEventFields(rec, ev)
			if err := app.Save(rec); err != nil {
				log.Printf("[sync] save event for match %s: %v", match.Id, err)
			}
		}
	}
	return nil
}

func eventProviderKey(fixtureID int, ev football.Event) string {
	// Detail is intentionally excluded: VAR corrections change Detail (e.g.
	// "Normal Goal" → "Goal Disallowed") without changing event identity.
	// Including it would cause a corrected event to be inserted as a duplicate
	// alongside the original rather than updating it.
	parts := []string{
		strconv.Itoa(fixtureID),
		strconv.Itoa(ev.Elapsed),
		strconv.Itoa(ev.Extra),
		strings.ToLower(strings.TrimSpace(ev.Type)),
		eventIdentityPart(ev.TeamID, ev.Team),
		eventIdentityPart(ev.PlayerID, ev.Player),
		eventIdentityPart(ev.AssistID, ev.Assist),
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:])
}

func eventIdentityPart(id int, fallback string) string {
	if id != 0 {
		return strconv.Itoa(id)
	}
	return football.NormalizeName(fallback)
}

func setLiveEventFields(rec *core.Record, ev football.Event) {
	rec.Set("elapsed", ev.Elapsed)
	rec.Set("extra", ev.Extra)
	rec.Set("type", strings.TrimSpace(ev.Type))
	rec.Set("detail", strings.TrimSpace(ev.Detail))
	rec.Set("player", strings.TrimSpace(ev.Player))
	rec.Set("assist", strings.TrimSpace(ev.Assist))
	rec.Set("team", strings.TrimSpace(ev.Team))
	rec.Set("comments", strings.TrimSpace(ev.Comments))
}

// eventTeamIDResolver returns a function that maps an event's provider team name
// to this match's home or away team id, alias-aware via canonName (so "Korea
// Republic" resolves to our "South Korea"). The client uses the id to show the
// scoring country's flag. Returns "" when the name matches neither side.
func eventTeamIDResolver(app core.App, match *core.Record) func(string) string {
	type sideTeam struct{ id, canon string }
	sides := make([]sideTeam, 0, 2)
	for _, field := range []string{"homeTeam", "awayTeam"} {
		id := match.GetString(field)
		if id == "" {
			continue
		}
		if team, err := app.FindRecordById("teams", id); err == nil {
			sides = append(sides, sideTeam{id: id, canon: canonName(team.GetString("name"))})
		}
	}
	return func(teamName string) string {
		want := canonName(teamName)
		if want == "" {
			return ""
		}
		for _, side := range sides {
			if side.canon == want {
				return side.id
			}
		}
		return ""
	}
}

func liveEventRecordJSON(ev *core.Record) map[string]any {
	return map[string]any{
		"id":          ev.Id,
		"match":       ev.GetString("match"),
		"providerKey": ev.GetString("providerKey"),
		"created":     ev.GetString("created"),
		"elapsed":     ev.GetInt("elapsed"),
		"extra":       ev.GetInt("extra"),
		"type":        ev.GetString("type"),
		"detail":      ev.GetString("detail"),
		"player":      ev.GetString("player"),
		"assist":      ev.GetString("assist"),
		"team":        ev.GetString("team"),
		"comments":    ev.GetString("comments"),
	}
}

// APICheck is a dev diagnostic: fetch a season's fixtures from API-Football
// and report parse health, team-name mapping coverage against our seed, how
// many of our match rows resolve, and the status / ET / penalty distribution
// (point it at a finished season like 2022 to validate the results path).
func APICheck(ctx context.Context, app core.App, client *football.Client, yr int) (map[string]any, error) {
	fixtures, err := client.FixturesForSeason(ctx, yr)
	if err != nil {
		return nil, err
	}

	teams, _ := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0)
	seedCanon := map[string]string{} // canonName -> seeded display name
	teamName := map[string]string{}  // teamId -> canonName
	for _, t := range teams {
		c := canonName(t.GetString("name"))
		seedCanon[c] = t.GetString("name")
		teamName[t.Id] = c
	}

	matches, _ := app.FindRecordsByFilter("matches", "id != ''", "kickoff", 0, 0)
	byPair := map[string]*core.Record{}
	for _, m := range matches {
		h, a := teamName[m.GetString("homeTeam")], teamName[m.GetString("awayTeam")]
		if h != "" && a != "" {
			byPair[h+"|"+a] = m
		}
	}

	statusHist := map[string]int{}
	unmapped := map[string]bool{}
	matchedRows := map[string]bool{}
	etCount, penCount := 0, 0
	var sample []map[string]any

	for _, f := range fixtures {
		statusHist[f.Status]++
		for _, nm := range []string{f.HomeName, f.AwayName} {
			if _, ok := seedCanon[canonName(nm)]; !ok {
				unmapped[nm] = true
			}
		}
		if rec, ok := byPair[canonName(f.HomeName)+"|"+canonName(f.AwayName)]; ok {
			matchedRows[rec.Id] = true
		}
		if f.ETHome != nil || f.ETAway != nil {
			etCount++
		}
		if f.PenHome != nil || f.PenAway != nil {
			penCount++
		}
		// Prefer extra-time / penalty fixtures in the sample — that's the
		// path most worth eyeballing.
		if (f.ETHome != nil || f.PenHome != nil) && len(sample) < 6 {
			sample = append(sample, map[string]any{
				"round": f.Round, "status": f.Status,
				"home": f.HomeName, "away": f.AwayName,
				"ft":                []any{f.FTHome, f.FTAway},
				"et":                []any{f.ETHome, f.ETAway},
				"pen":               []any{f.PenHome, f.PenAway},
				"advancerDerivable": f.Finished(),
			})
		}
	}
	unm := make([]string, 0, len(unmapped))
	for n := range unmapped {
		unm = append(unm, n)
	}
	sort.Strings(unm)

	return map[string]any{
		"season":           yr,
		"fixtures":         len(fixtures),
		"statusHistogram":  statusHist,
		"unmappedTeams":    unm,
		"ourMatchesTotal":  len(matches),
		"ourMatchesMapped": len(matchedRows),
		"withExtraTime":    etCount,
		"withPenalties":    penCount,
		"sample":           sample,
	}, nil
}

func ip(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func sameOptionalInt(rec *core.Record, field string, v *int) bool {
	if v == nil {
		return rec.GetInt(field) == 0
	}
	return rec.GetInt(field) == *v
}

func expectedDerivedState(rec *core.Record, status string, ftH, ftA, etH, etA, penH, penA *int) (advancer, penWinner string) {
	if rec.GetString("stage") == "group" || status != "finished" {
		return "", ""
	}

	home := rec.GetString("homeTeam")
	away := rec.GetString("awayTeam")
	switch {
	case penH != nil && penA != nil && *penH != *penA:
		if *penH > *penA {
			return home, home
		}
		return away, away
	case etH != nil && etA != nil && *etH != *etA:
		if *etH > *etA {
			return home, ""
		}
		return away, ""
	case ftH != nil && ftA != nil && *ftH != *ftA:
		if *ftH > *ftA {
			return home, ""
		}
		return away, ""
	}
	return "", ""
}

func resultAlreadyApplied(rec *core.Record, status string, ftH, ftA, etH, etA, penH, penA *int) bool {
	if status != "" && rec.GetString("status") != status {
		return false
	}
	if ftH != nil && rec.GetInt("ftHome") != *ftH {
		return false
	}
	if ftA != nil && rec.GetInt("ftAway") != *ftA {
		return false
	}
	if !(sameOptionalInt(rec, "etHome", etH) &&
		sameOptionalInt(rec, "etAway", etA) &&
		sameOptionalInt(rec, "penHome", penH) &&
		sameOptionalInt(rec, "penAway", penA)) {
		return false
	}

	advancer, penWinner := expectedDerivedState(rec, status, ftH, ftA, etH, etA, penH, penA)
	return rec.GetString("advancer") == advancer && rec.GetString("penWinner") == penWinner
}

// ApplyResult is the exported entry point (used by the dev simulator) that
// writes a result onto a match record using the same logic as live sync /
// manual override.
func ApplyResult(rec *core.Record, status string, ftH, ftA, etH, etA, penH, penA *int) {
	applyResult(rec, status, ftH, ftA, etH, etA, penH, penA)
}

// applyResult writes scores/status onto a match record and, for knockout
// matches, derives the advancer (ET > penalties > regulation).
func applyResult(rec *core.Record, status string, ftH, ftA, etH, etA, penH, penA *int) {
	if status != "" {
		rec.Set("status", status)
	}
	if ftH != nil {
		rec.Set("ftHome", *ftH)
	}
	if ftA != nil {
		rec.Set("ftAway", *ftA)
	}
	rec.Set("etHome", ip(etH))
	rec.Set("etAway", ip(etA))
	rec.Set("penHome", ip(penH))
	rec.Set("penAway", ip(penA))

	finished := rec.GetString("status") == "finished"
	if finished {
		rec.Set("finalizedAt", time.Now().UTC())
	} else {
		rec.Set("finalizedAt", "")
	}

	if rec.GetString("stage") == "group" {
		rec.Set("advancer", "")
		rec.Set("penWinner", "")
		return
	}

	rec.Set("advancer", "")
	rec.Set("penWinner", "")
	if !finished {
		return
	}
	// Knockout advancer resolution.
	home := rec.GetString("homeTeam")
	away := rec.GetString("awayTeam")
	switch {
	case penH != nil && penA != nil && *penH != *penA:
		if *penH > *penA {
			rec.Set("penWinner", home)
			rec.Set("advancer", home)
		} else {
			rec.Set("penWinner", away)
			rec.Set("advancer", away)
		}
	case etH != nil && etA != nil && *etH != *etA:
		if *etH > *etA {
			rec.Set("advancer", home)
		} else {
			rec.Set("advancer", away)
		}
	case ftH != nil && ftA != nil && *ftH != *ftA:
		if *ftH > *ftA {
			rec.Set("advancer", home)
		} else {
			rec.Set("advancer", away)
		}
	}
}

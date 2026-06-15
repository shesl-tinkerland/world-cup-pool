package topscorer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/oyvhov/world-cup-pool/internal/football"
)

const (
	collectionName  = "golden_boot_players"
	activeSyncEvery = 5 * time.Minute
	idleSyncEvery   = time.Hour
	settleWindow    = 2 * time.Hour
	rateWindow      = time.Minute
	searchLimit     = 30
	ensureLimit     = 10
)

// liveStatusFilter matches the matches.status values the provider uses while a
// game is in progress, so the Golden Boot table can refresh quickly during play
// instead of waiting for the next idle tick.
const liveStatusFilter = "status='live'||status='1H'||status='2H'||status='HT'||status='ET'||status='BT'||status='P'||status='LIVE'||status='INT'"

type rateLimiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

var (
	searchLimiter = &rateLimiter{hits: map[string][]time.Time{}}
	ensureLimiter = &rateLimiter{hits: map[string][]time.Time{}}
)

func (r *rateLimiter) allow(key string, limit int, window time.Duration) bool {
	now := time.Now()
	cutoff := now.Add(-window)

	r.mu.Lock()
	defer r.mu.Unlock()

	hits := r.hits[key]
	kept := hits[:0]
	for _, hit := range hits {
		if hit.After(cutoff) {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= limit {
		r.hits[key] = kept
		return false
	}
	kept = append(kept, now)
	r.hits[key] = kept
	return true
}

func rateKey(e *core.RequestEvent, action string) string {
	if e.Auth != nil && e.Auth.Id != "" {
		return action + ":user:" + e.Auth.Id
	}
	return action + ":ip:" + e.Request.RemoteAddr
}

type apiClient interface {
	TopScorers(context.Context) ([]football.TopScorer, error)
	SearchPlayers(context.Context, string) ([]football.PlayerSearchResult, error)
}

type Player struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	TeamID   string `json:"teamId"`
	TeamName string `json:"teamName"`
	PhotoURL string `json:"photoUrl,omitempty"`
	Goals    int    `json:"goals"`
	Assists  int    `json:"assists"`
	Rank     int    `json:"rank"`
	Eligible bool   `json:"eligible"`
	Seeded   bool   `json:"seeded"`
	SyncedAt string `json:"syncedAt,omitempty"`
}

type PickUser struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatarUrl"`
}

type LeaguePlayer struct {
	Player
	Picks []PickUser `json:"picks"`
}

type ForecastData struct {
	Shortlist []Player `json:"shortlist"`
	Leaders   []Player `json:"leaders"`
	UpdatedAt string   `json:"updatedAt,omitempty"`
}

type SearchPlayer struct {
	Key        string `json:"key"`
	ID         string `json:"id,omitempty"`
	ProviderID int    `json:"providerId"`
	Name       string `json:"name"`
	TeamID     string `json:"teamId"`
	TeamName   string `json:"teamName"`
	PhotoURL   string `json:"photoUrl,omitempty"`
	Goals      int    `json:"goals"`
	Assists    int    `json:"assists"`
	Rank       int    `json:"rank"`
	Eligible   bool   `json:"eligible"`
	Existing   bool   `json:"existing"`
}

type LeagueTable struct {
	Players   []LeaguePlayer `json:"players"`
	UpdatedAt string         `json:"updatedAt,omitempty"`
}

// Provide hardcoded default photos for curated players so the shortlist
// doesn't look empty before the tournament kicks off and syncs data.
var curated = []struct {
	Name       string
	Team       string
	ProviderID int
	PhotoURL   string
}{
	{Name: "Kylian Mbappé", Team: "France", ProviderID: 278, PhotoURL: "https://media.api-sports.io/football/players/278.png"},
	{Name: "Harry Kane", Team: "England", ProviderID: 184, PhotoURL: "https://media.api-sports.io/football/players/184.png"},
	{Name: "Erling Haaland", Team: "Norway", ProviderID: 1100, PhotoURL: "https://media.api-sports.io/football/players/1100.png"},
	{Name: "Lionel Messi", Team: "Argentina", ProviderID: 154, PhotoURL: "https://media.api-sports.io/football/players/154.png"},
	{Name: "Cristiano Ronaldo", Team: "Portugal", ProviderID: 874, PhotoURL: "https://media.api-sports.io/football/players/874.png"},
	{Name: "Vinícius Júnior", Team: "Brazil", ProviderID: 738, PhotoURL: "https://media.api-sports.io/football/players/738.png"},
	{Name: "Lautaro Martínez", Team: "Argentina", ProviderID: 2560, PhotoURL: "https://media.api-sports.io/football/players/2560.png"},
	{Name: "Lamine Yamal", Team: "Spain", ProviderID: 386828, PhotoURL: "https://media.api-sports.io/football/players/386828.png"},
	{Name: "Jamal Musiala", Team: "Germany", ProviderID: 118432, PhotoURL: "https://media.api-sports.io/football/players/118432.png"},
	{Name: "Santiago Giménez", Team: "Mexico", ProviderID: 161861, PhotoURL: "https://media.api-sports.io/football/players/161861.png"},
}

var teamAliases = map[string]string{
	football.NormalizeName("United States"):                football.NormalizeName("USA"),
	football.NormalizeName("Korea Republic"):               football.NormalizeName("South Korea"),
	football.NormalizeName("Bosnia and Herzegovina"):       football.NormalizeName("Bosnia & Herzegovina"),
	football.NormalizeName("Cape Verde Islands"):           football.NormalizeName("Cape Verde"),
	football.NormalizeName("Congo DR"):                     football.NormalizeName("DR Congo"),
	football.NormalizeName("IR Iran"):                      football.NormalizeName("Iran"),
	football.NormalizeName("Czechia"):                      football.NormalizeName("Czech Republic"),
	football.NormalizeName("Türkiye"):                      football.NormalizeName("Turkey"),
	football.NormalizeName("Côte d'Ivoire"):                football.NormalizeName("Ivory Coast"),
	football.NormalizeName("Democratic Republic of Congo"): football.NormalizeName("DR Congo"),
}

var accentReplacer = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ä", "a", "ã", "a", "å", "a",
	"Á", "a", "À", "a", "Â", "a", "Ä", "a", "Ã", "a", "Å", "a",
	"ç", "c", "Ç", "c",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"É", "e", "È", "e", "Ê", "e", "Ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"Í", "i", "Ì", "i", "Î", "i", "Ï", "i",
	"ñ", "n", "Ñ", "n",
	"ó", "o", "ò", "o", "ô", "o", "ö", "o", "õ", "o", "ø", "o",
	"Ó", "o", "Ò", "o", "Ô", "o", "Ö", "o", "Õ", "o", "Ø", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"Ú", "u", "Ù", "u", "Û", "u", "Ü", "u",
)

func Register(app core.App, serveEvent *core.ServeEvent) {
	if err := EnsureCurated(app); err != nil {
		log.Printf("[topscorer] curated shortlist seed: %v", err)
	}

	key := os.Getenv("API_FOOTBALL_KEY")
	var client apiClient
	if key != "" {
		client = football.New(key)
		startSmartSync(app, client)
		log.Printf("[topscorer] auto-sync enabled (active %v / idle %v)", activeSyncEvery, idleSyncEvery)
	} else {
		log.Printf("[topscorer] API_FOOTBALL_KEY not set — using curated shortlist only")
	}

	serveEvent.Router.POST("/api/admin/topscorers/refresh", func(requestEvent *core.RequestEvent) error {
		if key == "" {
			return requestEvent.JSON(400, map[string]string{"error": "API_FOOTBALL_KEY is not set"})
		}
		ctx, cancel := context.WithTimeout(requestEvent.Request.Context(), 30*time.Second)
		defer cancel()
		if err := Sync(ctx, app, football.New(key)); err != nil {
			return requestEvent.JSON(500, map[string]string{"error": err.Error()})
		}
		return requestEvent.JSON(200, map[string]string{"status": "ok"})
	}).Bind(apis.RequireSuperuserAuth())

	serveEvent.Router.GET("/api/forecast/topscorers/search", func(requestEvent *core.RequestEvent) error {
		query := strings.TrimSpace(requestEvent.Request.URL.Query().Get("q"))
		if len([]rune(query)) >= 2 && !searchLimiter.allow(rateKey(requestEvent, "search"), searchLimit, rateWindow) {
			return requestEvent.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
		}
		ctx, cancel := context.WithTimeout(requestEvent.Request.Context(), 20*time.Second)
		defer cancel()
		players, err := Search(ctx, app, client, query)
		if err != nil {
			return requestEvent.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}
		return requestEvent.JSON(http.StatusOK, map[string]any{
			"players":      players,
			"apiAvailable": client != nil,
		})
	}).Bind(apis.RequireAuth())

	serveEvent.Router.POST("/api/forecast/topscorers/ensure", func(requestEvent *core.RequestEvent) error {
		if !ensureLimiter.allow(rateKey(requestEvent, "ensure"), ensureLimit, rateWindow) {
			return requestEvent.JSON(http.StatusTooManyRequests, map[string]string{"error": "rate limited"})
		}
		var body SearchPlayer
		if err := requestEvent.BindBody(&body); err != nil {
			return apis.NewBadRequestError("invalid Golden Boot player body", err)
		}
		player, err := EnsureAPIPlayer(app, body)
		if err != nil {
			return apis.NewBadRequestError(err.Error(), nil)
		}
		return requestEvent.JSON(http.StatusOK, map[string]any{"player": player})
	}).Bind(apis.RequireAuth())
}

// startSmartSync refreshes the Golden Boot table on a dynamic cadence: quickly
// while matches are in progress so the standings track goals as they happen, and
// slowly when the schedule is idle. This mirrors the dynamic-interval loop the
// results sync already uses and replaces the old fixed 6-hour cron that left the
// table stale for hours during match days.
func startSmartSync(app core.App, client apiClient) {
	runOnce := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := Sync(ctx, app, client); err != nil {
			log.Printf("[topscorer] sync error: %v", err)
		}
	}
	go func() {
		runOnce() // prime immediately on startup
		for {
			time.Sleep(syncInterval(app))
			runOnce()
		}
	}()
}

func syncInterval(app core.App) time.Duration {
	if liveMatchInProgress(app) || matchFinishedRecently(app) {
		return activeSyncEvery
	}
	return idleSyncEvery
}

func liveMatchInProgress(app core.App) bool {
	records, err := app.FindRecordsByFilter("matches", liveStatusFilter, "", 1, 0)
	return err == nil && len(records) > 0
}

// matchFinishedRecently reports whether any match reached full time within the
// settle window. Provider goal/assist stats keep settling for a while after the
// final whistle, so we hold the active cadence through that window to catch the
// final Golden Boot numbers instead of dropping straight to the idle interval —
// which is what left the table stale for hours after the last match.
func matchFinishedRecently(app core.App) bool {
	records, err := app.FindRecordsByFilter("matches", "finalizedAt != ''", "-finalizedAt", 1, 0)
	if err != nil || len(records) == 0 {
		return false
	}
	return records[0].GetDateTime("finalizedAt").Time().After(time.Now().UTC().Add(-settleWindow))
}

func EnsureCurated(app core.App) error {
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}
	teamByName, err := teamsByCanon(app)
	if err != nil {
		return err
	}
	for _, candidate := range curated {
		teamRecord := teamByName[canonTeam(candidate.Team)]
		if teamRecord == nil {
			log.Printf("[topscorer] curated player %s skipped: team %s not found", candidate.Name, candidate.Team)
			continue
		}
		existing, _ := findByPlayerTeam(app, candidate.Name, teamRecord.Id)
		var record *core.Record
		if existing != nil {
			record = existing
		} else {
			record = core.NewRecord(collection)
			record.Set("providerKey", manualProviderKey(candidate.Name, candidate.Team))
		}
		record.Set("name", candidate.Name)
		record.Set("team", teamRecord.Id)
		record.Set("eligible", true)
		record.Set("seeded", true)
		// Set photo and ID if missing so seeded candidates look good before sync.
		if candidate.ProviderID > 0 {
			if record.GetInt("providerId") == 0 {
				record.Set("providerId", candidate.ProviderID)
				record.Set("providerKey", fmt.Sprintf("api:%d", candidate.ProviderID))
			}
			if record.GetString("photoUrl") == "" {
				record.Set("photoUrl", candidate.PhotoURL)
			}
		}
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save curated player %s: %w", candidate.Name, err)
		}
	}
	return nil
}

func Sync(ctx context.Context, app core.App, client apiClient) error {
	scorers, err := client.TopScorers(ctx)
	if err != nil {
		return fmt.Errorf("fetch top scorers: %w", err)
	}
	if len(scorers) == 0 {
		log.Printf("[topscorer] sync returned no scorers yet")
		return nil
	}
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return err
	}
	teamByName, err := teamsByCanon(app)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	updated := 0
	unmatched := []string{}
	for _, scorer := range scorers {
		teamRecord := teamByName[canonTeam(scorer.TeamName)]
		if teamRecord == nil {
			unmatched = append(unmatched, scorer.TeamName)
			continue
		}
		providerKey := fmt.Sprintf("api:%d", scorer.ProviderID)
		existing, _ := app.FindFirstRecordByFilter(collectionName, "providerKey = {:key}", map[string]any{"key": providerKey})
		if existing == nil {
			existing, _ = findByPlayerTeam(app, scorer.Name, teamRecord.Id)
		}

		var record *core.Record
		if existing != nil {
			record = existing
		} else {
			record = core.NewRecord(collection)
		}
		record.Set("providerKey", providerKey)
		record.Set("providerId", scorer.ProviderID)
		record.Set("name", scorer.Name)
		record.Set("team", teamRecord.Id)
		record.Set("photoUrl", scorer.PhotoURL)
		record.Set("goals", scorer.Goals)
		record.Set("assists", scorer.Assists)
		record.Set("rank", scorer.Rank)
		record.Set("eligible", record.GetBool("eligible") || scorer.Rank <= 10)
		record.Set("syncedAt", now)
		if err := app.Save(record); err != nil {
			return fmt.Errorf("save scorer %s: %w", scorer.Name, err)
		}
		updated++
	}

	if len(unmatched) > 0 {
		// Partial success: matched scorers are already saved. Log the gaps as a
		// warning instead of failing the whole sync so the cron / admin refresh
		// reports success and an unmapped country name doesn't error every 6h.
		log.Printf("[topscorer] sync: %d top-scorer teams were not mapped: %s", len(unmatched), strings.Join(uniqueStrings(unmatched), ", "))
	}
	log.Printf("[topscorer] sync done: %d updated, %d unmatched", updated, len(unmatched))
	return nil
}

func ForecastPayload(app core.App) (ForecastData, error) {
	shortlist, err := Shortlist(app)
	if err != nil {
		return ForecastData{}, err
	}
	leaders, err := DisplayPlayers(app, 10)
	if err != nil {
		return ForecastData{}, err
	}
	return ForecastData{
		Shortlist: shortlist,
		Leaders:   leaders,
		UpdatedAt: latestSync(append(shortlist, leaders...)),
	}, nil
}

func Shortlist(app core.App) ([]Player, error) {
	records, err := app.FindRecordsByFilter(collectionName, "eligible = true", "", 0, 0)
	if err != nil {
		return nil, err
	}
	players := make([]Player, 0, len(records))
	for _, record := range records {
		players = append(players, view(app, record))
	}
	sortPlayers(players)
	return players, nil
}

// DisplayPlayers returns the public Golden Boot standings, ordered purely by
// goals (then assists, then name) and ranked with ties sharing a position. It
// only includes players who have actually scored, so a provider quirk — a
// zero-goal player carrying a low array index, or a stale rank left behind by a
// player who dropped out of the feed — can never surface at the top. When no one
// has scored yet the result is empty and the UI shows its "no goals" state.
func DisplayPlayers(app core.App, limit int) ([]Player, error) {
	records, err := app.FindRecordsByFilter(collectionName, "goals > 0", "-goals", 0, 0)
	if err != nil {
		return nil, err
	}
	players := make([]Player, 0, len(records))
	for _, record := range records {
		players = append(players, view(app, record))
	}
	orderByGoals(players)
	assignCompetitionRanks(players)
	if limit > 0 && len(players) > limit {
		players = players[:limit]
	}
	return players, nil
}

func Search(ctx context.Context, app core.App, client apiClient, query string) ([]SearchPlayer, error) {
	query = strings.TrimSpace(query)
	if len([]rune(query)) < 2 {
		return []SearchPlayer{}, nil
	}

	local, err := searchLocal(app, query, 8)
	if err != nil {
		return nil, err
	}

	results := make([]SearchPlayer, 0, 12)
	seen := map[string]bool{}
	appendUnique := func(player SearchPlayer) {
		if player.Key == "" {
			switch {
			case player.ID != "":
				player.Key = player.ID
			case player.ProviderID > 0:
				player.Key = fmt.Sprintf("api:%d", player.ProviderID)
			default:
				player.Key = manualProviderKey(player.Name, player.TeamName)
			}
		}
		if seen[player.Key] {
			return
		}
		seen[player.Key] = true
		results = append(results, player)
	}

	for _, player := range local {
		appendUnique(player)
	}

	if client != nil {
		teamByName, err := teamsByCanon(app)
		if err != nil {
			return nil, err
		}
		remote, err := client.SearchPlayers(ctx, query)
		if err != nil {
			return nil, err
		}
		for _, hit := range remote {
			teamRecord := teamByName[canonTeam(hit.TeamName)]
			if teamRecord == nil {
				continue
			}

			player := SearchPlayer{
				Key:        fmt.Sprintf("api:%d", hit.ProviderID),
				ProviderID: hit.ProviderID,
				Name:       hit.Name,
				TeamID:     teamRecord.Id,
				TeamName:   teamRecord.GetString("name"),
				PhotoURL:   hit.PhotoURL,
				Goals:      hit.Goals,
				Assists:    hit.Assists,
			}

			providerKey := player.Key
			existing, _ := app.FindFirstRecordByFilter(collectionName, "providerKey = {:key}", map[string]any{"key": providerKey})
			if existing == nil {
				existing, _ = findByPlayerTeam(app, hit.Name, teamRecord.Id)
			}
			if existing != nil {
				localPlayer := view(app, existing)
				player.Key = localPlayer.ID
				player.ID = localPlayer.ID
				player.TeamID = localPlayer.TeamID
				player.TeamName = localPlayer.TeamName
				player.PhotoURL = firstNonEmpty(localPlayer.PhotoURL, player.PhotoURL)
				player.Goals = maxInt(localPlayer.Goals, player.Goals)
				player.Assists = maxInt(localPlayer.Assists, player.Assists)
				player.Rank = localPlayer.Rank
				player.Eligible = localPlayer.Eligible
				player.Existing = true
			}

			appendUnique(player)
		}
	}

	sortSearchPlayers(results)
	if len(results) > 12 {
		results = results[:12]
	}
	return results, nil
}

func EnsureAPIPlayer(app core.App, player SearchPlayer) (Player, error) {
	if player.ProviderID <= 0 {
		return Player{}, fmt.Errorf("missing API player id")
	}
	if strings.TrimSpace(player.Name) == "" {
		return Player{}, fmt.Errorf("missing player name")
	}
	teamRecord, err := app.FindRecordById("teams", player.TeamID)
	if err != nil {
		return Player{}, fmt.Errorf("unknown team")
	}
	collection, err := app.FindCollectionByNameOrId(collectionName)
	if err != nil {
		return Player{}, err
	}

	providerKey := fmt.Sprintf("api:%d", player.ProviderID)
	existing, _ := app.FindFirstRecordByFilter(collectionName, "providerKey = {:key}", map[string]any{"key": providerKey})
	if existing == nil {
		existing, _ = findByPlayerTeam(app, player.Name, teamRecord.Id)
	}

	record := existing
	if record == nil {
		record = core.NewRecord(collection)
	}
	record.Set("providerKey", providerKey)
	record.Set("providerId", player.ProviderID)
	record.Set("name", strings.TrimSpace(player.Name))
	record.Set("team", teamRecord.Id)
	if photoURL := strings.TrimSpace(player.PhotoURL); photoURL != "" {
		record.Set("photoUrl", photoURL)
	}
	record.Set("goals", maxInt(record.GetInt("goals"), player.Goals))
	record.Set("assists", maxInt(record.GetInt("assists"), player.Assists))
	if player.Rank > 0 {
		record.Set("rank", player.Rank)
	}
	record.Set("eligible", true)
	if player.Goals > 0 || player.Assists > 0 || player.Rank > 0 {
		record.Set("syncedAt", time.Now().UTC())
	}
	if err := app.Save(record); err != nil {
		return Player{}, err
	}
	return view(app, record), nil
}

func LeagueTableFor(app core.App, leagueID string) (LeagueTable, error) {
	basePlayers, err := DisplayPlayers(app, 10)
	if err != nil {
		return LeagueTable{}, err
	}
	players := make([]LeaguePlayer, 0, len(basePlayers))
	byID := map[string]int{}
	for _, player := range basePlayers {
		byID[player.ID] = len(players)
		players = append(players, LeaguePlayer{Player: player, Picks: []PickUser{}})
	}

	members, err := app.FindRecordsByFilter("league_members", "league = {:league}", "", 0, 0, map[string]any{"league": leagueID})
	if err != nil {
		return LeagueTable{}, err
	}
	for _, member := range members {
		userID := member.GetString("user")
		forecast, err := app.FindFirstRecordByFilter("forecasts", "user = {:user}", map[string]any{"user": userID})
		if err != nil {
			continue
		}
		userRecord, err := app.FindRecordById("users", userID)
		if err != nil {
			continue
		}
		playerID := PickFromForecast(forecast)
		if playerID == "" {
			continue
		}
		index, found := byID[playerID]
		if !found {
			playerRecord, err := app.FindRecordById(collectionName, playerID)
			if err != nil {
				continue
			}
			byID[playerID] = len(players)
			index = len(players)
			players = append(players, LeaguePlayer{Player: view(app, playerRecord), Picks: []PickUser{}})
		}
		players[index].Picks = append(players[index].Picks, PickUser{
			ID:        userID,
			Name:      userRecord.GetString("name"),
			AvatarURL: avatarURL(userRecord),
		})
	}
	sort.SliceStable(players, func(first, second int) bool {
		if players[first].Goals != players[second].Goals {
			return players[first].Goals > players[second].Goals
		}
		if players[first].Assists != players[second].Assists {
			return players[first].Assists > players[second].Assists
		}
		return players[first].Name < players[second].Name
	})
	flat := make([]Player, 0, len(players))
	for _, player := range players {
		flat = append(flat, player.Player)
	}
	return LeagueTable{Players: players, UpdatedAt: latestSync(flat)}, nil
}

func IsEligible(app core.App, playerID string) bool {
	record, err := app.FindRecordById(collectionName, playerID)
	return err == nil && record.GetBool("eligible")
}

func PickFromForecast(record *core.Record) string {
	var picks []string
	_ = record.UnmarshalJSONField("goldenBootPicks", &picks)
	if len(picks) == 0 {
		if legacy := record.GetString("goldenBootPlayer"); legacy != "" {
			return legacy
		}
		return ""
	}
	for _, pick := range picks {
		pick = strings.TrimSpace(pick)
		if pick != "" {
			return pick
		}
	}
	return ""
}

func WinnerID(app core.App) string {
	records, err := app.FindRecordsByFilter(collectionName, "rank = 1", "rank", 1, 0)
	if err != nil || len(records) == 0 {
		return ""
	}
	return records[0].Id
}

func view(app core.App, record *core.Record) Player {
	teamID := record.GetString("team")
	teamName := ""
	if teamRecord, err := app.FindRecordById("teams", teamID); err == nil {
		teamName = teamRecord.GetString("name")
	}
	syncedAt := ""
	if synced := record.GetDateTime("syncedAt").Time(); !synced.IsZero() {
		syncedAt = synced.UTC().Format(time.RFC3339)
	}
	return Player{
		ID:       record.Id,
		Name:     record.GetString("name"),
		TeamID:   teamID,
		TeamName: teamName,
		PhotoURL: record.GetString("photoUrl"),
		Goals:    record.GetInt("goals"),
		Assists:  record.GetInt("assists"),
		Rank:     record.GetInt("rank"),
		Eligible: record.GetBool("eligible"),
		Seeded:   record.GetBool("seeded"),
		SyncedAt: syncedAt,
	}
}

func searchLocal(app core.App, query string, limit int) ([]SearchPlayer, error) {
	records, err := app.FindRecordsByFilter(collectionName, "id != ''", "name", 0, 0)
	if err != nil {
		return nil, err
	}
	want := canonPlayer(query)
	matches := make([]SearchPlayer, 0, minInt(limit, len(records)))
	for _, record := range records {
		player := view(app, record)
		if !strings.Contains(canonPlayer(player.Name), want) && !strings.Contains(canonPlayer(player.TeamName), want) {
			continue
		}
		matches = append(matches, SearchPlayer{
			Key:        player.ID,
			ID:         player.ID,
			ProviderID: record.GetInt("providerId"),
			Name:       player.Name,
			TeamID:     player.TeamID,
			TeamName:   player.TeamName,
			PhotoURL:   player.PhotoURL,
			Goals:      player.Goals,
			Assists:    player.Assists,
			Rank:       player.Rank,
			Eligible:   player.Eligible,
			Existing:   true,
		})
	}
	sortSearchPlayers(matches)
	if len(matches) > limit {
		matches = matches[:limit]
	}
	return matches, nil
}

func teamsByCanon(app core.App) (map[string]*core.Record, error) {
	teams, err := app.FindRecordsByFilter("teams", "id != ''", "", 0, 0)
	if err != nil {
		return nil, err
	}
	indexed := map[string]*core.Record{}
	for _, teamRecord := range teams {
		indexed[canonTeam(teamRecord.GetString("name"))] = teamRecord
	}
	return indexed, nil
}

func findByPlayerTeam(app core.App, playerName, teamID string) (*core.Record, error) {
	records, err := app.FindRecordsByFilter(collectionName, "team = {:team}", "", 0, 0, map[string]any{"team": teamID})
	if err != nil {
		return nil, err
	}
	want := canonPlayer(playerName)
	for _, record := range records {
		if canonPlayer(record.GetString("name")) == want {
			return record, nil
		}
	}
	return nil, nil
}

// orderByGoals sorts a top-scorer slice the way a Golden Boot table should read:
// most goals first, assists as the tie-break, then name for a stable, deterministic
// order. Unlike sortPlayers it never consults the provider rank, so stale or
// duplicate rank values can't distort the standings.
func orderByGoals(players []Player) {
	sort.SliceStable(players, func(first, second int) bool {
		if players[first].Goals != players[second].Goals {
			return players[first].Goals > players[second].Goals
		}
		if players[first].Assists != players[second].Assists {
			return players[first].Assists > players[second].Assists
		}
		return players[first].Name < players[second].Name
	})
}

// assignCompetitionRanks overwrites each player's Rank with a standard
// competition ranking (ties share a rank, the next rank skips): goal totals of
// 5, 3, 3, 1 become ranks 1, 2, 2, 4. It assumes players is already ordered by
// goals descending (call orderByGoals first).
func assignCompetitionRanks(players []Player) {
	for i := range players {
		if i > 0 && players[i].Goals == players[i-1].Goals {
			players[i].Rank = players[i-1].Rank
		} else {
			players[i].Rank = i + 1
		}
	}
}

func sortPlayers(players []Player) {
	sort.SliceStable(players, func(first, second int) bool {
		firstRank := players[first].Rank
		secondRank := players[second].Rank
		if (firstRank == 0) != (secondRank == 0) {
			return firstRank != 0
		}
		if firstRank != 0 && secondRank != 0 && firstRank != secondRank {
			return firstRank < secondRank
		}
		if players[first].Goals != players[second].Goals {
			return players[first].Goals > players[second].Goals
		}
		return players[first].Name < players[second].Name
	})
}

func sortSearchPlayers(players []SearchPlayer) {
	sort.SliceStable(players, func(first, second int) bool {
		firstRank := players[first].Rank
		secondRank := players[second].Rank
		if (firstRank == 0) != (secondRank == 0) {
			return firstRank != 0
		}
		if firstRank != 0 && secondRank != 0 && firstRank != secondRank {
			return firstRank < secondRank
		}
		if players[first].Existing != players[second].Existing {
			return players[first].Existing
		}
		if players[first].Goals != players[second].Goals {
			return players[first].Goals > players[second].Goals
		}
		return players[first].Name < players[second].Name
	})
}

func latestSync(players []Player) string {
	var latest time.Time
	for _, player := range players {
		if player.SyncedAt == "" {
			continue
		}
		synced, err := time.Parse(time.RFC3339, player.SyncedAt)
		if err == nil && synced.After(latest) {
			latest = synced
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339)
}

func canonTeam(name string) string {
	normalized := football.NormalizeName(name)
	if alias, ok := teamAliases[normalized]; ok {
		return alias
	}
	return normalized
}

func canonPlayer(name string) string {
	normalized := accentReplacer.Replace(name)
	var builder strings.Builder
	for _, character := range strings.ToLower(normalized) {
		if (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9') {
			builder.WriteRune(character)
		}
	}
	return builder.String()
}

func manualProviderKey(playerName, teamName string) string {
	return "manual:" + canonPlayer(playerName) + ":" + canonTeam(teamName)
}

func avatarURL(userRecord *core.Record) *string {
	file := userRecord.GetString("avatar")
	if file == "" {
		return nil
	}
	url := "/api/files/users/" + userRecord.Id + "/" + file
	return &url
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	unique := []string{}
	for _, value := range values {
		if seen[value] {
			continue
		}
		seen[value] = true
		unique = append(unique, value)
	}
	sort.Strings(unique)
	return unique
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

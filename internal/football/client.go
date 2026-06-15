// Package football is a thin client for the API-Football (api-sports.io) free
// tier. One /fixtures call returns all 104 WC2026 matches, so a periodic sync
// costs a single request and stays well within the 100/day free limit.
package football

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

const (
	leagueID = 1    // FIFA World Cup
	season   = 2026 // WC 2026
)

var baseURL = "https://v3.football.api-sports.io"

type Client struct {
	key  string
	http *http.Client
}

func New(key string) *Client {
	return &Client{key: key, http: &http.Client{Timeout: 20 * time.Second}}
}

// Fixture is the subset of the API-Football fixture payload we use.
type Fixture struct {
	ID        int       // provider fixture id
	Date      time.Time // kickoff (UTC)
	Round     string    // e.g. "Group A - 1", "Round of 32"
	Status    string    // NS, 1H, HT, 2H, ET, BT, P, FT, AET, PEN, PST, CANC, ...
	HomeName  string
	AwayName  string
	HomeGoals *int // full 90' (nil if not played)
	AwayGoals *int
	FTHome    *int // regulation
	FTAway    *int
	ETHome    *int // after extra time (cumulative)
	ETAway    *int
	PenHome   *int // shootout
	PenAway   *int
}

// TopScorer is the subset of the API-Football top-scorers payload used by the
// Golden Boot prediction.
type TopScorer struct {
	ProviderID int
	Name       string
	PhotoURL   string
	TeamName   string
	Goals      int
	Assists    int
	Rank       int
}

// PlayerSearchResult is a searchable Golden Boot candidate sourced from the
// API-Football player search endpoint. TeamName is the best country/team label
// the API exposes for mapping back to our World Cup teams.
type PlayerSearchResult struct {
	ProviderID int
	Name       string
	PhotoURL   string
	TeamName   string
	Goals      int
	Assists    int
}

// Event is a timeline item from API-Football's /fixtures/events endpoint.
type Event struct {
	Elapsed  int
	Extra    int
	TeamID   int
	Team     string
	PlayerID int
	Player   string
	AssistID int
	Assist   string
	Type     string
	Detail   string
	Comments string
}

// Finished reports whether the provider considers the match complete.
func (f Fixture) Finished() bool {
	switch f.Status {
	case "FT", "AET", "PEN", "WO":
		return true
	}
	return false
}

// Live reports whether the match is currently in progress.
func (f Fixture) Live() bool {
	switch f.Status {
	case "1H", "2H", "HT", "ET", "BT", "P", "LIVE", "INT":
		return true
	}
	return false
}

func regulationScoreForStatus(status string, goalsHome, goalsAway, fulltimeHome, fulltimeAway *int) (*int, *int) {
	if (&Fixture{Status: status}).Live() && (fulltimeHome == nil || fulltimeAway == nil) {
		return goalsHome, goalsAway
	}
	return fulltimeHome, fulltimeAway
}

type apiResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Response []struct {
		Fixture struct {
			ID     int       `json:"id"`
			Date   time.Time `json:"date"`
			Status struct {
				Short string `json:"short"`
			} `json:"status"`
		} `json:"fixture"`
		League struct {
			Round string `json:"round"`
		} `json:"league"`
		Teams struct {
			Home struct {
				Name string `json:"name"`
			} `json:"home"`
			Away struct {
				Name string `json:"name"`
			} `json:"away"`
		} `json:"teams"`
		Goals struct {
			Home *int `json:"home"`
			Away *int `json:"away"`
		} `json:"goals"`
		Score struct {
			Fulltime  scorePair `json:"fulltime"`
			Extratime scorePair `json:"extratime"`
			Penalty   scorePair `json:"penalty"`
		} `json:"score"`
	} `json:"response"`
}

type topScorersResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Response []struct {
		Player struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Photo string `json:"photo"`
		} `json:"player"`
		Statistics []struct {
			Team struct {
				Name string `json:"name"`
			} `json:"team"`
			Goals struct {
				Total   *int `json:"total"`
				Assists *int `json:"assists"`
			} `json:"goals"`
		} `json:"statistics"`
	} `json:"response"`
}

type playerSearchResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Response []struct {
		Player struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Photo       string `json:"photo"`
			Nationality string `json:"nationality"`
		} `json:"player"`
		Statistics []struct {
			Team struct {
				Name string `json:"name"`
			} `json:"team"`
			Goals struct {
				Total   *int `json:"total"`
				Assists *int `json:"assists"`
			} `json:"goals"`
		} `json:"statistics"`
	} `json:"response"`
}

type fixtureEventsResponse struct {
	Errors   json.RawMessage `json:"errors"`
	Results  int             `json:"results"`
	Response []struct {
		Time struct {
			Elapsed int  `json:"elapsed"`
			Extra   *int `json:"extra"`
		} `json:"time"`
		Team struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"team"`
		Player struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"player"`
		Assist struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"assist"`
		Type     string  `json:"type"`
		Detail   string  `json:"detail"`
		Comments *string `json:"comments"`
	} `json:"response"`
}

type scorePair struct {
	Home *int `json:"home"`
	Away *int `json:"away"`
}

// Fixtures returns every WC2026 fixture in a single request.
func (c *Client) Fixtures(ctx context.Context) ([]Fixture, error) {
	return c.FixturesForSeason(ctx, season)
}

// TopScorers returns API-Football's ordered Golden Boot table for WC2026.
func (c *Client) TopScorers(ctx context.Context) ([]TopScorer, error) {
	url := fmt.Sprintf("%s/players/topscorers?league=%d&season=%d", baseURL, leagueID, season)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football top scorers: status %d", resp.StatusCode)
	}

	var ar topScorersResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(ar.Errors)); s != "" && s != "[]" && s != "{}" {
		return nil, fmt.Errorf("api-football top scorers errors: %s", s)
	}
	if ar.Results > 0 && len(ar.Response) == 0 {
		return nil, fmt.Errorf("api-football top scorers: server reports %d results but response array is empty — possible schema change", ar.Results)
	}

	out := make([]TopScorer, 0, len(ar.Response))
	for i, r := range ar.Response {
		if r.Player.ID == 0 || strings.TrimSpace(r.Player.Name) == "" || len(r.Statistics) == 0 {
			continue
		}
		stat := r.Statistics[0]
		goals, assists := 0, 0
		if stat.Goals.Total != nil {
			goals = *stat.Goals.Total
		}
		if stat.Goals.Assists != nil {
			assists = *stat.Goals.Assists
		}
		out = append(out, TopScorer{
			ProviderID: r.Player.ID,
			Name:       r.Player.Name,
			PhotoURL:   r.Player.Photo,
			TeamName:   stat.Team.Name,
			Goals:      goals,
			Assists:    assists,
			Rank:       i + 1,
		})
	}
	return out, nil
}

// FixtureEvents returns the currently available event timeline for a fixture.
func (c *Client) FixtureEvents(ctx context.Context, fixtureID int) ([]Event, error) {
	if fixtureID == 0 {
		return nil, nil
	}
	url := fmt.Sprintf("%s/fixtures/events?fixture=%d", baseURL, fixtureID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		return []Event{}, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football fixture events: status %d", resp.StatusCode)
	}

	var ar fixtureEventsResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(ar.Errors)); s != "" && s != "[]" && s != "{}" {
		return nil, fmt.Errorf("api-football fixture events errors: %s", s)
	}
	if ar.Results > 0 && len(ar.Response) == 0 {
		return nil, fmt.Errorf("api-football fixture events: server reports %d results but response array is empty — possible schema change", ar.Results)
	}

	out := make([]Event, 0, len(ar.Response))
	for _, r := range ar.Response {
		extra := 0
		if r.Time.Extra != nil {
			extra = *r.Time.Extra
		}
		comments := ""
		if r.Comments != nil {
			comments = *r.Comments
		}
		out = append(out, Event{
			Elapsed:  r.Time.Elapsed,
			Extra:    extra,
			TeamID:   r.Team.ID,
			Team:     r.Team.Name,
			PlayerID: r.Player.ID,
			Player:   r.Player.Name,
			AssistID: r.Assist.ID,
			Assist:   r.Assist.Name,
			Type:     r.Type,
			Detail:   r.Detail,
			Comments: comments,
		})
	}
	return out, nil
}

// SearchPlayers looks up a player by name. It prefers the World Cup-scoped
// search first so goals/assists stay tournament-specific, then falls back to a
// generic player search so users can still nominate breakout candidates before
// they appear in the live top-scorer table.
func (c *Client) SearchPlayers(ctx context.Context, query string) ([]PlayerSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	worldCup, worldCupErr := c.searchPlayers(ctx, "/players", url.Values{
		"search": []string{query},
		"league": []string{fmt.Sprintf("%d", leagueID)},
		"season": []string{fmt.Sprintf("%d", season)},
	})
	if worldCupErr == nil && len(worldCup) > 0 {
		sortPlayerSearchResults(worldCup, query)
		return worldCup, nil
	}

	fallback, err := c.searchPlayers(ctx, "/players/profiles", url.Values{"search": []string{query}})
	if err != nil {
		if worldCupErr != nil {
			return nil, worldCupErr
		}
		return nil, err
	}
	sortPlayerSearchResults(fallback, query)
	for index := range fallback {
		fallback[index].Goals = 0
		fallback[index].Assists = 0
	}
	if len(fallback) > 24 {
		fallback = fallback[:24]
	}
	return fallback, nil
}

// FixturesForSeason fetches the World Cup fixtures for any season (used by the
// dev API diagnostic to replay a finished tournament, e.g. 2022).
func (c *Client) FixturesForSeason(ctx context.Context, yr int) ([]Fixture, error) {
	url := fmt.Sprintf("%s/fixtures?league=%d&season=%d", baseURL, leagueID, yr)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football: status %d", resp.StatusCode)
	}

	var ar apiResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(ar.Errors)); s != "" && s != "[]" && s != "{}" {
		return nil, fmt.Errorf("api-football errors: %s", s)
	}
	// Guard against silent schema drift: the server reports N results but the
	// decoded response array is empty, meaning the "response" key was renamed.
	if ar.Results > 0 && len(ar.Response) == 0 {
		return nil, fmt.Errorf("api-football: server reports %d results but response array is empty — possible schema change", ar.Results)
	}

	out := make([]Fixture, 0, len(ar.Response))
	for _, r := range ar.Response {
		ftHome, ftAway := regulationScoreForStatus(
			r.Fixture.Status.Short,
			r.Goals.Home,
			r.Goals.Away,
			r.Score.Fulltime.Home,
			r.Score.Fulltime.Away,
		)
		out = append(out, Fixture{
			ID:        r.Fixture.ID,
			Date:      r.Fixture.Date.UTC(),
			Round:     r.League.Round,
			Status:    r.Fixture.Status.Short,
			HomeName:  r.Teams.Home.Name,
			AwayName:  r.Teams.Away.Name,
			HomeGoals: r.Goals.Home,
			AwayGoals: r.Goals.Away,
			FTHome:    ftHome,
			FTAway:    ftAway,
			ETHome:    r.Score.Extratime.Home,
			ETAway:    r.Score.Extratime.Away,
			PenHome:   r.Score.Penalty.Home,
			PenAway:   r.Score.Penalty.Away,
		})
	}
	return out, nil
}

func (c *Client) searchPlayers(ctx context.Context, path string, params url.Values) ([]PlayerSearchResult, error) {
	endpoint := baseURL + path
	if encoded := params.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football player search: status %d", resp.StatusCode)
	}

	var ar playerSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&ar); err != nil {
		return nil, err
	}
	if s := strings.TrimSpace(string(ar.Errors)); s != "" && s != "[]" && s != "{}" {
		return nil, fmt.Errorf("api-football player search errors: %s", s)
	}

	out := make([]PlayerSearchResult, 0, len(ar.Response))
	seen := map[int]bool{}
	for _, r := range ar.Response {
		if r.Player.ID == 0 || strings.TrimSpace(r.Player.Name) == "" || seen[r.Player.ID] {
			continue
		}
		seen[r.Player.ID] = true

		teamName := strings.TrimSpace(r.Player.Nationality)
		goals, assists := 0, 0
		if len(r.Statistics) > 0 {
			if teamName == "" {
				teamName = strings.TrimSpace(r.Statistics[0].Team.Name)
			}
			if r.Statistics[0].Goals.Total != nil {
				goals = *r.Statistics[0].Goals.Total
			}
			if r.Statistics[0].Goals.Assists != nil {
				assists = *r.Statistics[0].Goals.Assists
			}
		}

		out = append(out, PlayerSearchResult{
			ProviderID: r.Player.ID,
			Name:       r.Player.Name,
			PhotoURL:   r.Player.Photo,
			TeamName:   teamName,
			Goals:      goals,
			Assists:    assists,
		})
	}
	return out, nil
}

func sortPlayerSearchResults(players []PlayerSearchResult, query string) {
	foldedQuery := foldSearchText(query)
	sort.SliceStable(players, func(first, second int) bool {
		firstScore := playerSearchScore(foldedQuery, players[first].Name)
		secondScore := playerSearchScore(foldedQuery, players[second].Name)
		if firstScore != secondScore {
			return firstScore > secondScore
		}
		if len(players[first].Name) != len(players[second].Name) {
			return len(players[first].Name) < len(players[second].Name)
		}
		return players[first].Name < players[second].Name
	})
}

func playerSearchScore(foldedQuery, name string) int {
	if foldedQuery == "" {
		return 0
	}
	foldedName := foldSearchText(name)
	if foldedName == foldedQuery {
		return 500
	}
	for _, token := range strings.Fields(foldedName) {
		if token == foldedQuery {
			return 450
		}
	}
	if strings.HasPrefix(foldedName, foldedQuery+" ") {
		return 400
	}
	for _, token := range strings.Fields(foldedName) {
		if strings.HasPrefix(token, foldedQuery) {
			return 300
		}
	}
	if strings.Contains(foldedName, foldedQuery) {
		return 200
	}
	return 0
}

func foldSearchText(s string) string {
	decomposed := norm.NFD.String(strings.ToLower(strings.TrimSpace(s)))
	var builder strings.Builder
	spacePending := false
	for _, character := range decomposed {
		switch {
		case unicode.Is(unicode.Mn, character):
			continue
		case unicode.IsLetter(character) || unicode.IsDigit(character):
			builder.WriteRune(character)
			spacePending = false
		default:
			if builder.Len() > 0 && !spacePending {
				builder.WriteByte(' ')
				spacePending = true
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

// Status reports the account/plan and request quota (api-football /status).
func (c *Client) Status(ctx context.Context) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/status", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-apisports-key", c.key)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api-football: status %d", resp.StatusCode)
	}
	var out struct {
		Response map[string]any `json:"response"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Response, nil
}

// NormalizeName lowercases and strips non-alphanumerics so provider team names
// can be matched against the openfootball-seeded names despite spelling
// differences ("Korea Republic" vs "South Korea" still need an alias map).
func NormalizeName(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}
